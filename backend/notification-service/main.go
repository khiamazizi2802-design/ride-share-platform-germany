package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

type Config struct {
	Port                    string
	ServiceName             string
	Environment             string
	APIKey                  string
	WebhookSecret           string
	RetentionReadDays       int // DSGVO: 30 Tage fuer gelesene Benachrichtigungen
	RetentionDeliveryDays   int // DSGVO: 90 Tage fuer Zustellungslogs
	MaxRetryAttempts        int
	RetryBackoffBaseSeconds int
	RateLimitPerMinute      int
	WorkerPoolSize          int
	QueueBufferSize         int
}

func loadConfig() *Config {
	return &Config{
		Port:                    getEnv("PORT", "8080"),
		ServiceName:             getEnv("SERVICE_NAME", "notification-service"),
		Environment:             getEnv("ENVIRONMENT", "development"),
		APIKey:                  getEnv("API_KEY", "dev-api-key-change-in-production"),
		WebhookSecret:           getEnv("WEBHOOK_SECRET", "dev-webhook-secret-change-in-production"),
		RetentionReadDays:       getEnvInt("RETENTION_READ_DAYS", 30),
		RetentionDeliveryDays:   getEnvInt("RETENTION_DELIVERY_DAYS", 90),
		MaxRetryAttempts:        getEnvInt("MAX_RETRY_ATTEMPTS", 3),
		RetryBackoffBaseSeconds: getEnvInt("RETRY_BACKOFF_BASE_SECONDS", 5),
		RateLimitPerMinute:      getEnvInt("RATE_LIMIT_PER_MINUTE", 60),
		WorkerPoolSize:          getEnvInt("WORKER_POOL_SIZE", 5),
		QueueBufferSize:         getEnvInt("QUEUE_BUFFER_SIZE", 1000),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return fallback
}

// ---------------------------------------------------------------------------
// Domain Models
// ---------------------------------------------------------------------------

type NotificationStatus string
type NotificationChannel string
type NotificationType string
type DeliveryStatus string

const (
	StatusPending    NotificationStatus = "pending"
	StatusQueued     NotificationStatus = "queued"
	StatusDelivered  NotificationStatus = "delivered"
	StatusFailed     NotificationStatus = "failed"
	StatusRead       NotificationStatus = "read"
	StatusExpired    NotificationStatus = "expired"
	StatusCancelled  NotificationStatus = "cancelled"

	ChannelPush  NotificationChannel = "push"
	ChannelSMS   NotificationChannel = "sms"
	ChannelEmail NotificationChannel = "email"

	TypeRideRequest    NotificationType = "ride_request"
	TypeRideAccepted   NotificationType = "ride_accepted"
	TypeRideCancelled  NotificationType = "ride_cancelled"
	TypeRideCompleted  NotificationType = "ride_completed"
	TypeDriverArrived  NotificationType = "driver_arrived"
	TypePaymentSuccess NotificationType = "payment_success"
	TypePaymentFailed  NotificationType = "payment_failed"
	TypePromotion      NotificationType = "promotion"
	TypeSystemAlert    NotificationType = "system_alert"
	TypeCustom         NotificationType = "custom"

	DeliveryStatusAttempted DeliveryStatus = "attempted"
	DeliveryStatusSuccess   DeliveryStatus = "success"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusRetrying  DeliveryStatus = "retrying"
)

// Notification represents a single notification entity.
// Datenschutz: Personenbezogene Daten werden gemaess DSGVO Art. 5 verarbeitet.
// Datenspeicherung ausschliesslich auf EU-Servern (Deutschland).
type Notification struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	Type         NotificationType       `json:"type"`
	Channel      NotificationChannel    `json:"channel"`
	Title        string                 `json:"title"`
	Body         string                 `json:"body"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Status       NotificationStatus     `json:"status"`
	RetryCount   int                    `json:"retry_count"`
	TemplateID   string                 `json:"template_id,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
	ReadAt       *time.Time             `json:"read_at,omitempty"`
	DeliveredAt  *time.Time             `json:"delivered_at,omitempty"`
	ConsentGiven bool                   `json:"consent_given"` // DSGVO: Einwilligung des Nutzers
}

// NotificationTemplate stores reusable notification templates.
type NotificationTemplate struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Type          NotificationType    `json:"type"`
	Channel       NotificationChannel `json:"channel"`
	TitleTemplate string              `json:"title_template"`
	BodyTemplate  string              `json:"body_template"`
	Variables     []string            `json:"variables"`
	IsActive      bool                `json:"is_active"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

// UserPreferences stores per-user notification preferences.
// DSGVO: Einstellungen zur Datenschutzpraeferenz des Nutzers.
type UserPreferences struct {
	UserID              string    `json:"user_id"`
	EmailEnabled        bool      `json:"email_enabled"`
	SMSEnabled          bool      `json:"sms_enabled"`
	PushEnabled         bool      `json:"push_enabled"`
	QuietHoursStart     string    `json:"quiet_hours_start"` // "HH:MM" format (24h)
	QuietHoursEnd       string    `json:"quiet_hours_end"`   // "HH:MM" format (24h)
	Language            string    `json:"language"`          // de, en
	Timezone            string    `json:"timezone"`          // e.g. "Europe/Berlin"
	ConsentMarketing    bool      `json:"consent_marketing"` // DSGVO: Marketing-Einwilligung
	ConsentTransactional bool     `json:"consent_transactional"` // DSGVO: Transaktions-Einwilligung
	DataRetentionOptOut bool      `json:"data_retention_opt_out"` // DSGVO: Widerspruchsrecht
	UpdatedAt           time.Time `json:"updated_at"`
}

// DeliveryAttempt tracks each delivery attempt for audit and retry logic.
// DSGVO: Protokollierung gemaess Art. 5 Abs. 2 (Rechenschaftspflicht).
type DeliveryAttempt struct {
	ID             string         `json:"id"`
	NotificationID string         `json:"notification_id"`
	Provider       string         `json:"provider"`
	Status         DeliveryStatus `json:"status"`
	AttemptNumber  int            `json:"attempt_number"`
	AttemptedAt    time.Time      `json:"attempted_at"`
	ErrorMessage   string         `json:"error_message,omitempty"`
	ResponseCode   int            `json:"response_code,omitempty"`
	LatencyMs      int64          `json:"latency_ms,omitempty"`
}

// WebhookEvent represents incoming webhook payload from external providers.
type WebhookEvent struct {
	Provider       string                 `json:"provider"`
	EventType      string                 `json:"event_type"`
	NotificationID string                 `json:"notification_id"`
	Status         string                 `json:"status"`
	Timestamp      time.Time              `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// Request / Response DTOs
// ---------------------------------------------------------------------------

type SendNotificationRequest struct {
	UserID     string                 `json:"user_id"`
	Type       NotificationType       `json:"type"`
	Channel    NotificationChannel    `json:"channel"`
	Title      string                 `json:"title,omitempty"`
	Body       string                 `json:"body,omitempty"`
	TemplateID string                 `json:"template_id,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	ExpiresIn  *int                   `json:"expires_in_seconds,omitempty"`
}

type BatchNotificationRequest struct {
	Notifications []SendNotificationRequest `json:"notifications"`
}

type BatchNotificationResponse struct {
	Total     int                  `json:"total"`
	Succeeded int                  `json:"succeeded"`
	Failed    int                  `json:"failed"`
	Results   []BatchNotifResult   `json:"results"`
}

type BatchNotifResult struct {
	Index          int    `json:"index"`
	NotificationID string `json:"notification_id,omitempty"`
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
}

type CreateTemplateRequest struct {
	Name          string              `json:"name"`
	Type          NotificationType    `json:"type"`
	Channel       NotificationChannel `json:"channel"`
	TitleTemplate string              `json:"title_template"`
	BodyTemplate  string              `json:"body_template"`
	Variables     []string            `json:"variables"`
}

type UpdatePreferencesRequest struct {
	EmailEnabled         *bool   `json:"email_enabled,omitempty"`
	SMSEnabled           *bool   `json:"sms_enabled,omitempty"`
	PushEnabled          *bool   `json:"push_enabled,omitempty"`
	QuietHoursStart      *string `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd        *string `json:"quiet_hours_end,omitempty"`
	Language             *string `json:"language,omitempty"`
	Timezone             *string `json:"timezone,omitempty"`
	ConsentMarketing     *bool   `json:"consent_marketing,omitempty"`
	ConsentTransactional *bool   `json:"consent_transactional,omitempty"`
	DataRetentionOptOut  *bool   `json:"data_retention_opt_out,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
	Total  int `json:"total,omitempty"`
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// ---------------------------------------------------------------------------
// In-Memory Store
// ---------------------------------------------------------------------------

type Store struct {
	mu               sync.RWMutex
	notifications    map[string]*Notification
	templates        map[string]*NotificationTemplate
	preferences      map[string]*UserPreferences
	deliveryAttempts map[string][]*DeliveryAttempt // keyed by notification ID
}

func newStore() *Store {
	return &Store{
		notifications:    make(map[string]*Notification),
		templates:        make(map[string]*NotificationTemplate),
		preferences:      make(map[string]*UserPreferences),
		deliveryAttempts: make(map[string][]*DeliveryAttempt),
	}
}

func (s *Store) SaveNotification(n *Notification) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n.UpdatedAt = time.Now().UTC()
	s.notifications[n.ID] = n
}

func (s *Store) GetNotification(id string) (*Notification, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notifications[id]
	if !ok {
		return nil, false
	}
	copy := *n
	return &copy, true
}

func (s *Store) GetUserNotifications(userID string, limit, offset int) ([]*Notification, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []*Notification
	for _, n := range s.notifications {
		if n.UserID == userID {
			copy := *n
			results = append(results, &copy)
		}
	}
	// Sort by CreatedAt descending (simple selection sort for clarity)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].CreatedAt.After(results[i].CreatedAt) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	total := len(results)
	if offset >= total {
		return []*Notification{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return results[offset:end], total
}

func (s *Store) DeleteNotification(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.notifications[id]; !ok {
		return false
	}
	delete(s.notifications, id)
	delete(s.deliveryAttempts, id)
	return true
}

func (s *Store) DeleteUserData(userID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for id, n := range s.notifications {
		if n.UserID == userID {
			delete(s.notifications, id)
			delete(s.deliveryAttempts, id)
			count++
		}
	}
	delete(s.preferences, userID)
	return count
}

func (s *Store) SaveTemplate(t *NotificationTemplate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t.UpdatedAt = time.Now().UTC()
	s.templates[t.ID] = t
}

func (s *Store) GetTemplate(id string) (*NotificationTemplate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.templates[id]
	if !ok {
		return nil, false
	}
	copy := *t
	return &copy, true
}

func (s *Store) ListTemplates() []*NotificationTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []*NotificationTemplate
	for _, t := range s.templates {
		copy := *t
		results = append(results, &copy)
	}
	return results
}

func (s *Store) GetPreferences(userID string) (*UserPreferences, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.preferences[userID]
	if !ok {
		return nil, false
	}
	copy := *p
	return &copy, true
}

func (s *Store) SavePreferences(p *UserPreferences) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.UpdatedAt = time.Now().UTC()
	s.preferences[p.UserID] = p
}

func (s *Store) AddDeliveryAttempt(attempt *DeliveryAttempt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deliveryAttempts[attempt.NotificationID] = append(
		s.deliveryAttempts[attempt.NotificationID], attempt,
	)
}

func (s *Store) GetDeliveryAttempts(notificationID string) []*DeliveryAttempt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	attempts := s.deliveryAttempts[notificationID]
	result := make([]*DeliveryAttempt, len(attempts))
	for i, a := range attempts {
		copy := *a
		result[i] = &copy
	}
	return result
}

// PurgeExpiredData removes data according to DSGVO retention policies.
// DSGVO Art. 5 Abs. 1 lit. e: Speicherbegrenzung
func (s *Store) PurgeExpiredData(readRetentionDays, deliveryRetentionDays int) (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	notifPurged := 0
	attemptsPurged := 0

	for id, n := range s.notifications {
		// Purge expired notifications
		if n.ExpiresAt != nil && now.After(*n.ExpiresAt) {
			delete(s.notifications, id)
			delete(s.deliveryAttempts, id)
			notifPurged++
			continue
		}
		// Purge read notifications older than retention period
		if n.Status == StatusRead && n.ReadAt != nil {
			if now.Sub(*n.ReadAt) > time.Duration(readRetentionDays)*24*time.Hour {
				delete(s.notifications, id)
				delete(s.deliveryAttempts, id)
				notifPurged++
			}
		}
	}

	// Purge old delivery attempts
	retentionCutoff := now.Add(-time.Duration(deliveryRetentionDays) * 24 * time.Hour)
	for notifID, attempts := range s.deliveryAttempts {
		var kept []*DeliveryAttempt
		for _, a := range attempts {
			if a.AttemptedAt.After(retentionCutoff) {
				kept = append(kept, a)
			} else {
				attemptsPurged++
			}
		}
		if len(kept) == 0 {
			delete(s.deliveryAttempts, notifID)
		} else {
			s.deliveryAttempts[notifID] = kept
		}
	}

	return notifPurged, attemptsPurged
}

// ---------------------------------------------------------------------------
// Rate Limiter
// ---------------------------------------------------------------------------

type RateLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

func newRateLimiterStore(perMinute int) *RateLimiterStore {
	rps := rate.Limit(float64(perMinute) / 60.0)
	return &RateLimiterStore{
		limiters: make(map[string]*rate.Limiter),
		rps:      rps,
		burst:    perMinute / 10,
	}
}

func (r *RateLimiterStore) getLimiter(key string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.limiters[key]; ok {
		return l
	}
	burst := r.burst
	if burst < 1 {
		burst = 1
	}
	l := rate.NewLimiter(r.rps, burst)
	r.limiters[key] = l
	return l
}

func (r *RateLimiterStore) Allow(key string) bool {
	return r.getLimiter(key).Allow()
}

// ---------------------------------------------------------------------------
// Notification Queue & Worker Pool
// ---------------------------------------------------------------------------

type QueueJob struct {
	Notification *Notification
	Attempt      int
	ScheduledAt  time.Time
}

type NotificationQueue struct {
	ch     chan QueueJob
	closed chan struct{}
}

func newNotificationQueue(bufferSize int) *NotificationQueue {
	return &NotificationQueue{
		ch:     make(chan QueueJob, bufferSize),
		closed: make(chan struct{}),
	}
}

func (q *NotificationQueue) Enqueue(job QueueJob) error {
	select {
	case q.ch <- job:
		return nil
	case <-q.closed:
		return errors.New("queue is closed")
	default:
		return errors.New("notification queue is full")
	}
}

func (q *NotificationQueue) Close() {
	close(q.closed)
}

// ---------------------------------------------------------------------------
// Delivery Dispatcher (simulates real provider calls)
// ---------------------------------------------------------------------------

type Dispatcher struct {
	store  *Store
	cfg    *Config
	queue  *NotificationQueue
	logger *slog.Logger
	wg     sync.WaitGroup
}

func newDispatcher(store *Store, cfg *Config, queue *NotificationQueue, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{store: store, cfg: cfg, queue: queue, logger: logger}
}

func (d *Dispatcher) Start(ctx context.Context) {
	for i := 0; i < d.cfg.WorkerPoolSize; i++ {
		d.wg.Add(1)
		go d.worker(ctx, i)
	}
}

func (d *Dispatcher) Wait() {
	d.wg.Wait()
}

func (d *Dispatcher) worker(ctx context.Context, workerID int) {
	defer d.wg.Done()
	d.logger.Info("Notification worker started", "worker_id", workerID)
	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Notification worker stopping", "worker_id", workerID)
			return
		case job, ok := <-d.queue.ch:
			if !ok {
				return
			}
			// Wait until scheduled
			if delay := time.Until(job.ScheduledAt); delay > 0 {
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return
				}
			}
			d.process(ctx, job)
		}
	}
}

func (d *Dispatcher) process(ctx context.Context, job QueueJob) {
	n := job.Notification
	// Re-fetch from store for latest status
	current, ok := d.store.GetNotification(n.ID)
	if !ok {
		return
	}
	if current.Status == StatusCancelled || current.Status == StatusRead {
		return
	}
	if current.ExpiresAt != nil && time.Now().UTC().After(*current.ExpiresAt) {
		current.Status = StatusExpired
		d.store.SaveNotification(current)
		return
	}

	startTime := time.Now()
	provider, err := d.deliver(ctx, current)
	latency := time.Since(startTime).Milliseconds()

	attempt := &DeliveryAttempt{
		ID:             uuid.New().String(),
		NotificationID: current.ID,
		Provider:       provider,
		AttemptNumber:  job.Attempt + 1,
		AttemptedAt:    time.Now().UTC(),
		LatencyMs:      latency,
	}

	if err != nil {
		attempt.Status = DeliveryStatusFailed
		attempt.ErrorMessage = err.Error()
		d.store.AddDeliveryAttempt(attempt)

		if job.Attempt+1 < d.cfg.MaxRetryAttempts {
			// Exponential backoff retry
			backoff := time.Duration(float64(d.cfg.RetryBackoffBaseSeconds)*math.Pow(2, float64(job.Attempt))) * time.Second
			current.Status = StatusQueued
			current.RetryCount = job.Attempt + 1
			d.store.SaveNotification(current)
			retryJob := QueueJob{
				Notification: current,
				Attempt:      job.Attempt + 1,
				ScheduledAt:  time.Now().Add(backoff),
			}
			if enqErr := d.queue.Enqueue(retryJob); enqErr != nil {
				d.logger.Error("Failed to enqueue retry", "notification_id", current.ID, "error", enqErr)
				current.Status = StatusFailed
				d.store.SaveNotification(current)
			} else {
				d.logger.Info("Notification retry scheduled",
					"notification_id", current.ID,
					"attempt", job.Attempt+1,
					"backoff_seconds", backoff.Seconds(),
				)
			}
		} else {
			current.Status = StatusFailed
			d.store.SaveNotification(current)
			d.logger.Warn("Notification delivery failed permanently",
				"notification_id", current.ID,
				"attempts", job.Attempt+1,
			)
		}
	} else {
		attempt.Status = DeliveryStatusSuccess
		d.store.AddDeliveryAttempt(attempt)
		now := time.Now().UTC()
		current.Status = StatusDelivered
		current.DeliveredAt = &now
		d.store.SaveNotification(current)
		d.logger.Info("Notification delivered",
			"notification_id", current.ID,
			"channel", current.Channel,
			"provider", provider,
			"latency_ms", latency,
		)
	}
}

// deliver simulates sending via the appropriate channel provider.
// In production, replace with real SDK calls (e.g. Firebase, Twilio, SES).
func (d *Dispatcher) deliver(_ context.Context, n *Notification) (string, error) {
	switch n.Channel {
	case ChannelPush:
		// TODO: integrate Firebase Cloud Messaging or APNS
		d.logger.Debug("Simulating push delivery", "notification_id", n.ID, "user_id", n.UserID)
		return "firebase-fcm", nil
	case ChannelSMS:
		// TODO: integrate Twilio or Vonage (EU data residency required)
		d.logger.Debug("Simulating SMS delivery", "notification_id", n.ID, "user_id", n.UserID)
		return "twilio", nil
	case ChannelEmail:
		// TODO: integrate Amazon SES EU or Postmark
		d.logger.Debug("Simulating email delivery", "notification_id", n.ID, "user_id", n.UserID)
		return "ses-eu-west", nil
	default:
		return "unknown", fmt.Errorf("unsupported channel: %s", n.Channel)
	}
}

// ---------------------------------------------------------------------------
// Template Engine
// ---------------------------------------------------------------------------

func renderTemplate(tmplStr string, data map[string]interface{}) (string, error) {
	if tmplStr == "" || !strings.Contains(tmplStr, "{{") {
		return tmplStr, nil
	}
	tmpl, err := template.New("").Option("missingkey=error").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("template execute error: %w", err)
	}
	return sb.String(), nil
}

// ---------------------------------------------------------------------------
// Quiet Hours Check
// ---------------------------------------------------------------------------

var timeRegexp = regexp.MustCompile(`^([01][0-9]|2[0-3]):([0-5][0-9])$`)

func isInQuietHours(prefs *UserPreferences) bool {
	if prefs == nil {
		return false
	}
	if !timeRegexp.MatchString(prefs.QuietHoursStart) || !timeRegexp.MatchString(prefs.QuietHoursEnd) {
		return false
	}
	loc := time.UTC
	if prefs.Timezone != "" {
		if tz, err := time.LoadLocation(prefs.Timezone); err == nil {
			loc = tz
		}
	}
	now := time.Now().In(loc)
	currentMins := now.Hour()*60 + now.Minute()

	var startH, startM, endH, endM int
	fmt.Sscanf(prefs.QuietHoursStart, "%d:%d", &startH, &startM)
	fmt.Sscanf(prefs.QuietHoursEnd, "%d:%d", &endH, &endM)
	startMins := startH*60 + startM
	endMins := endH*60 + endM

	if startMins <= endMins {
		return currentMins >= startMins && currentMins < endMins
	}
	// Overnight window
	return currentMins >= startMins || currentMins < endMins
}

// ---------------------------------------------------------------------------
// Service Layer
// ---------------------------------------------------------------------------

type NotificationService struct {
	store    *Store
	cfg      *Config
	queue    *NotificationQueue
	rateLimiter *RateLimiterStore
	logger   *slog.Logger
}

func newNotificationService(store *Store, cfg *Config, queue *NotificationQueue, logger *slog.Logger) *NotificationService {
	return &NotificationService{
		store:       store,
		cfg:         cfg,
		queue:       queue,
		rateLimiter: newRateLimiterStore(cfg.RateLimitPerMinute),
		logger:      logger,
	}
}

func (svc *NotificationService) Send(req *SendNotificationRequest) (*Notification, error) {
	if err := validateSendRequest(req); err != nil {
		return nil, err
	}

	// Rate limiting per user+channel
	rateLimitKey := fmt.Sprintf("%s:%s", req.UserID, req.Channel)
	if !svc.rateLimiter.Allow(rateLimitKey) {
		return nil, &AppError{Code: http.StatusTooManyRequests, Message: "rate limit exceeded for this user/channel"}
	}

	// Check user preferences
	prefs, _ := svc.store.GetPreferences(req.UserID)
	if prefs != nil {
		if err := svc.checkPreferences(prefs, req.Channel, req.Type); err != nil {
			return nil, err
		}
	}

	title := req.Title
	body := req.Body

	// Resolve template if provided
	if req.TemplateID != "" {
		tmpl, ok := svc.store.GetTemplate(req.TemplateID)
		if !ok {
			return nil, &AppError{Code: http.StatusNotFound, Message: "template not found"}
		}
		if !tmpl.IsActive {
			return nil, &AppError{Code: http.StatusBadRequest, Message: "template is inactive"}
		}
		var err error
		title, err = renderTemplate(tmpl.TitleTemplate, req.Data)
		if err != nil {
			return nil, &AppError{Code: http.StatusBadRequest, Message: "title template render failed: " + err.Error()}
		}
		body, err = renderTemplate(tmpl.BodyTemplate, req.Data)
		if err != nil {
			return nil, &AppError{Code: http.StatusBadRequest, Message: "body template render failed: " + err.Error()}
		}
	}

	now := time.Now().UTC()
	n := &Notification{
		ID:           uuid.New().String(),
		UserID:       req.UserID,
		Type:         req.Type,
		Channel:      req.Channel,
		Title:        title,
		Body:         body,
		Data:         req.Data,
		Status:       StatusQueued,
		TemplateID:   req.TemplateID,
		CreatedAt:    now,
		UpdatedAt:    now,
		ConsentGiven: prefs != nil && prefs.ConsentTransactional,
	}

	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiry := now.Add(time.Duration(*req.ExpiresIn) * time.Second)
		n.ExpiresAt = &expiry
	}

	svc.store.SaveNotification(n)

	job := QueueJob{
		Notification: n,
		Attempt:      0,
		ScheduledAt:  now,
	}
	if err := svc.queue.Enqueue(job); err != nil {
		n.Status = StatusFailed
		svc.store.SaveNotification(n)
		return nil, &AppError{Code: http.StatusServiceUnavailable, Message: "notification queue is full, try again later"}
	}

	svc.logger.Info("Notification queued",
		"notification_id", n.ID,
		"user_id", n.UserID,
		"channel", n.Channel,
		"type", n.Type,
	)
	return n, nil
}

func (svc *NotificationService) checkPreferences(prefs *UserPreferences, channel NotificationChannel, nType NotificationType) error {
	// Check channel enablement
	switch channel {
	case ChannelEmail:
		if !prefs.EmailEnabled {
			return &AppError{Code: http.StatusForbidden, Message: "email notifications disabled by user preference"}
		}
	case ChannelSMS:
		if !prefs.SMSEnabled {
			return &AppError{Code: http.StatusForbidden, Message: "SMS notifications disabled by user preference"}
		}
	case ChannelPush:
		if !prefs.PushEnabled {
			return &AppError{Code: http.StatusForbidden, Message: "push notifications disabled by user preference"}
		}
	}
	// Check marketing consent (DSGVO Art. 6)
	if nType == TypePromotion && !prefs.ConsentMarketing {
		return &AppError{Code: http.StatusForbidden, Message: "marketing notifications require explicit consent (DSGVO Art. 6)"}
	}
	// Quiet hours - only for non-critical types
	if nType != TypeSystemAlert && nType != TypePaymentFailed {
		if isInQuietHours(prefs) {
			return &AppError{Code: http.StatusForbidden, Message: "notification suppressed during quiet hours"}
		}
	}
	return nil
}

func (svc *NotificationService) MarkRead(id string) (*Notification, error) {
	n, ok := svc.store.GetNotification(id)
	if !ok {
		return nil, &AppError{Code: http.StatusNotFound, Message: "notification not found"}
	}
	if n.Status == StatusRead {
		return n, nil
	}
	now := time.Now().UTC()
	n.Status = StatusRead
	n.ReadAt = &now
	svc.store.SaveNotification(n)
	return n, nil
}

func (svc *NotificationService) GetStatus(id string) (*Notification, []*DeliveryAttempt, error) {
	n, ok := svc.store.GetNotification(id)
	if !ok {
		return nil, nil, &AppError{Code: http.StatusNotFound, Message: "notification not found"}
	}
	attempts := svc.store.GetDeliveryAttempts(id)
	return n, attempts, nil
}

func (svc *NotificationService) GetOrCreatePreferences(userID string) *UserPreferences {
	if p, ok := svc.store.GetPreferences(userID); ok {
		return p
	}
	// Default preferences (DSGVO: privacy by default)
	p := &UserPreferences{
		UserID:               userID,
		EmailEnabled:         true,
		SMSEnabled:           false, // Opt-in required
		PushEnabled:          true,
		QuietHoursStart:      "22:00",
		QuietHoursEnd:        "07:00",
		Language:             "de",
		Timezone:             "Europe/Berlin",
		ConsentMarketing:     false, // Opt-in required (DSGVO)
		ConsentTransactional: true,
		UpdatedAt:            time.Now().UTC(),
	}
	svc.store.SavePreferences(p)
	return p
}

func (svc *NotificationService) UpdatePreferences(userID string, req *UpdatePreferencesRequest) (*UserPreferences, error) {
	p := svc.GetOrCreatePreferences(userID)
	if req.EmailEnabled != nil {
		p.EmailEnabled = *req.EmailEnabled
	}
	if req.SMSEnabled != nil {
		p.SMSEnabled = *req.SMSEnabled
	}
	if req.PushEnabled != nil {
		p.PushEnabled = *req.PushEnabled
	}
	if req.QuietHoursStart != nil {
		if !timeRegexp.MatchString(*req.QuietHoursStart) {
			return nil, &AppError{Code: http.StatusBadRequest, Message: "invalid quiet_hours_start format, use HH:MM"}
		}
		p.QuietHoursStart = *req.QuietHoursStart
	}
	if req.QuietHoursEnd != nil {
		if !timeRegexp.MatchString(*req.QuietHoursEnd) {
			return nil, &AppError{Code: http.StatusBadRequest, Message: "invalid quiet_hours_end format, use HH:MM"}
		}
		p.QuietHoursEnd = *req.QuietHoursEnd
	}
	if req.Language != nil {
		p.Language = *req.Language
	}
	if req.Timezone != nil {
		if _, err := time.LoadLocation(*req.Timezone); err != nil {
			return nil, &AppError{Code: http.StatusBadRequest, Message: "invalid timezone"}
		}
		p.Timezone = *req.Timezone
	}
	if req.ConsentMarketing != nil {
		p.ConsentMarketing = *req.ConsentMarketing
	}
	if req.ConsentTransactional != nil {
		p.ConsentTransactional = *req.ConsentTransactional
	}
	if req.DataRetentionOptOut != nil {
		p.DataRetentionOptOut = *req.DataRetentionOptOut
	}
	svc.store.SavePreferences(p)
	return p, nil
}

func (svc *NotificationService) CreateTemplate(req *CreateTemplateRequest) (*NotificationTemplate, error) {
	if err := validateCreateTemplateRequest(req); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	t := &NotificationTemplate{
		ID:            uuid.New().String(),
		Name:          req.Name,
		Type:          req.Type,
		Channel:       req.Channel,
		TitleTemplate: req.TitleTemplate,
		BodyTemplate:  req.BodyTemplate,
		Variables:     req.Variables,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	svc.store.SaveTemplate(t)
	svc.logger.Info("Template created", "template_id", t.ID, "name", t.Name)
	return t, nil
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

type AppError struct {
	Code    int
	Message string
}

func (e *AppError) Error() string { return e.Message }

func validateSendRequest(req *SendNotificationRequest) error {
	if req.UserID == "" {
		return &AppError{Code: http.StatusBadRequest, Message: "user_id is required"}
	}
	if !isValidChannel(req.Channel) {
		return &AppError{Code: http.StatusBadRequest, Message: fmt.Sprintf("invalid channel: %s", req.Channel)}
	}
	if !isValidType(req.Type) {
		return &AppError{Code: http.StatusBadRequest, Message: fmt.Sprintf("invalid notification type: %s", req.Type)}
	}
	if req.TemplateID == "" && (req.Title == "" || req.Body == "") {
		return &AppError{Code: http.StatusBadRequest, Message: "either template_id or both title and body are required"}
	}
	return nil
}

func validateCreateTemplateRequest(req *CreateTemplateRequest) error {
	if req.Name == "" {
		return &AppError{Code: http.StatusBadRequest, Message: "template name is required"}
	}
	if !isValidChannel(req.Channel) {
		return &AppError{Code: http.StatusBadRequest, Message: fmt.Sprintf("invalid channel: %s", req.Channel)}
	}
	if !isValidType(req.Type) {
		return &AppError{Code: http.StatusBadRequest, Message: fmt.Sprintf("invalid type: %s", req.Type)}
	}
	if req.TitleTemplate == "" {
		return &AppError{Code: http.StatusBadRequest, Message: "title_template is required"}
	}
	if req.BodyTemplate == "" {
		return &AppError{Code: http.StatusBadRequest, Message: "body_template is required"}
	}
	return nil
}

func isValidChannel(c NotificationChannel) bool {
	switch c {
	case ChannelPush, ChannelSMS, ChannelEmail:
		return true
	}
	return false
}

func isValidType(t NotificationType) bool {
	switch t {
	case TypeRideRequest, TypeRideAccepted, TypeRideCancelled, TypeRideCompleted,
		TypeDriverArrived, TypePaymentSuccess, TypePaymentFailed,
		TypePromotion, TypeSystemAlert, TypeCustom:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// HTTP Handlers
// ---------------------------------------------------------------------------

type Handler struct {
	svc    *NotificationService
	store  *Store
	cfg    *Config
	logger *slog.Logger
}

func newHandler(svc *NotificationService, store *Store, cfg *Config, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, store: store, cfg: cfg, logger: logger}
}

// POST /api/v1/notifications/send
func (h *Handler) SendNotification(w http.ResponseWriter, r *http.Request) {
	var req SendNotificationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	n, err := h.svc.Send(&req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, APIResponse{Success: true, Data: n})
}

// POST /api/v1/notifications/send-batch
func (h *Handler) SendBatch(w http.ResponseWriter, r *http.Request) {
	var req BatchNotificationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if len(req.Notifications) == 0 {
		writeError(w, http.StatusBadRequest, "notifications array cannot be empty")
		return
	}
	if len(req.Notifications) > 100 {
		writeError(w, http.StatusBadRequest, "batch size cannot exceed 100 notifications")
		return
	}

	resp := BatchNotificationResponse{
		Total:   len(req.Notifications),
		Results: make([]BatchNotifResult, len(req.Notifications)),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for i, notifReq := range req.Notifications {
		wg.Add(1)
		go func(idx int, nr SendNotificationRequest) {
			defer wg.Done()
			n, err := h.svc.Send(&nr)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				resp.Results[idx] = BatchNotifResult{Index: idx, Success: false, Error: err.Error()}
				resp.Failed++
			} else {
				resp.Results[idx] = BatchNotifResult{Index: idx, NotificationID: n.ID, Success: true}
				resp.Succeeded++
			}
		}(i, notifReq)
	}
	wg.Wait()

	statusCode := http.StatusMultiStatus
	if resp.Failed == 0 {
		statusCode = http.StatusAccepted
	}
	writeJSON(w, statusCode, APIResponse{Success: resp.Failed == 0, Data: resp})
}

// GET /api/v1/notifications/{id}/status
func (h *Handler) GetNotificationStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	n, attempts, err := h.svc.GetStatus(id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"notification":      n,
			"delivery_attempts": attempts,
		},
	})
}

// GET /api/v1/notifications/user/{user_id}
func (h *Handler) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["user_id"]
	limit := parseQueryInt(r, "limit", 20)
	offset := parseQueryInt(r, "offset", 0)
	if limit > 100 {
		limit = 100
	}
	notifications, total := h.store.GetUserNotifications(userID, limit, offset)
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    notifications,
		Meta:    &Meta{Total: total, Limit: limit, Offset: offset},
	})
}

// PUT /api/v1/notifications/{id}/read
func (h *Handler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	n, err := h.svc.MarkRead(id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: n})
}

// DELETE /api/v1/notifications/{id}
func (h *Handler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if !h.store.DeleteNotification(id) {
		writeError(w, http.StatusNotFound, "notification not found")
		return
	}
	h.logger.Info("Notification deleted", "notification_id", id)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"message": "notification deleted"}})
}

// GET /api/v1/notifications/preferences/{user_id}
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["user_id"]
	p := h.svc.GetOrCreatePreferences(userID)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: p})
}

// PUT /api/v1/notifications/preferences/{user_id}
func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["user_id"]
	var req UpdatePreferencesRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	p, err := h.svc.UpdatePreferences(userID, &req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	h.logger.Info("User preferences updated", "user_id", userID)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: p})
}

// POST /api/v1/notifications/templates
func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req CreateTemplateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	t, err := h.svc.CreateTemplate(&req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, APIResponse{Success: true, Data: t})
}

// GET /api/v1/notifications/templates
func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates := h.store.ListTemplates()
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    templates,
		Meta:    &Meta{Total: len(templates)},
	})
}

// POST /api/v1/webhooks/provider
func (h *Handler) ProviderWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify HMAC-SHA256 signature
	sig := r.Header.Get("X-Webhook-Signature")
	if sig == "" {
		writeError(w, http.StatusUnauthorized, "missing webhook signature")
		return
	}

	body := make([]byte, 0)
	decoder := json.NewDecoder(r.Body)
	var rawMsg json.RawMessage
	if err := decoder.Decode(&rawMsg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook payload")
		return
	}
	body = rawMsg

	if !verifyWebhookSignature(body, sig, h.cfg.WebhookSecret) {
		h.logger.Warn("Webhook signature verification failed", "remote_addr", r.RemoteAddr)
		writeError(w, http.StatusUnauthorized, "invalid webhook signature")
		return
	}

	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse webhook event")
		return
	}

	h.processWebhookEvent(&event)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"message": "webhook processed"}})
}

func (h *Handler) processWebhookEvent(event *WebhookEvent) {
	if event.NotificationID == "" {
		h.logger.Warn("Webhook event missing notification_id", "provider", event.Provider)
		return
	}

	n, ok := h.store.GetNotification(event.NotificationID)
	if !ok {
		h.logger.Warn("Webhook event for unknown notification",
			"notification_id", event.NotificationID,
			"provider", event.Provider,
		)
		return
	}

	switch strings.ToLower(event.Status) {
	case "delivered", "success":
		now := time.Now().UTC()
		n.Status = StatusDelivered
		n.DeliveredAt = &now
	case "failed", "bounced", "undelivered":
		n.Status = StatusFailed
	case "opened", "read":
		now := time.Now().UTC()
		n.Status = StatusRead
		n.ReadAt = &now
	}

	h.store.SaveNotification(n)
	h.logger.Info("Webhook event processed",
		"notification_id", event.NotificationID,
		"provider", event.Provider,
		"event_type", event.EventType,
		"status", event.Status,
	)

	// Record delivery attempt from webhook
	attempt := &DeliveryAttempt{
		ID:             uuid.New().String(),
		NotificationID: event.NotificationID,
		Provider:       event.Provider,
		Status:         DeliveryStatusSuccess,
		AttemptNumber:  0,
		AttemptedAt:    event.Timestamp,
	}
	if strings.ToLower(event.Status) == "failed" || strings.ToLower(event.Status) == "bounced" {
		attempt.Status = DeliveryStatusFailed
	}
	h.store.AddDeliveryAttempt(attempt)
}

// DELETE /api/v1/users/{user_id}/data - Right to be forgotten (DSGVO Art. 17)
func (h *Handler) DeleteUserData(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["user_id"]
	count := h.store.DeleteUserData(userID)
	h.logger.Info("User data deleted (DSGVO Art. 17 - Recht auf Vergessenwerden)",
		"user_id", userID,
		"records_deleted", count,
	)
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message":         "Benutzerdaten wurden gemaess DSGVO Art. 17 geloescht",
			"records_deleted": count,
		},
	})
}

// GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.store.mu.RLock()
	notifCount := len(h.store.notifications)
	tmplCount := len(h.store.templates)
	h.store.mu.RUnlock()

	resp := HealthResponse{
		Status:    "healthy",
		Service:   h.cfg.ServiceName,
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
		Checks: map[string]string{
			"store":              "ok",
			"queue":             "ok",
			"notifications":     fmt.Sprintf("%d stored", notifCount),
			"templates":         fmt.Sprintf("%d stored", tmplCount),
			"gdpr_compliance":   "enabled",
			"data_residency":    "EU-Central-1 (Frankfurt)",
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

func loggingMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			logger.Info("HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"latency_ms", time.Since(start).Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"request_id", r.Header.Get("X-Request-ID"),
			)
		})
	}
}

func authMiddleware(apiKey string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health check
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}
			key := r.Header.Get("X-API-Key")
			if key == "" {
				bearer := r.Header.Get("Authorization")
				if strings.HasPrefix(bearer, "Bearer ") {
					key = strings.TrimPrefix(bearer, "Bearer ")
				}
			}
			if !hmac.Equal([]byte(key), []byte(apiKey)) {
				writeError(w, http.StatusUnauthorized, "invalid or missing API key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// DSGVO: Datenschutz-Header
		w.Header().Set("X-Data-Residency", "EU-Central-1")
		w.Header().Set("X-GDPR-Compliant", "true")
		next.ServeHTTP(w, r)
	})
}

func contentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			ct := r.Header.Get("Content-Type")
			if ct != "" && !strings.HasPrefix(ct, "application/json") {
				writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
			r.Header.Set("X-Request-ID", reqID)
		}
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ---------------------------------------------------------------------------
// HTTP Helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, APIResponse{Success: false, Error: msg})
}

func writeAppError(w http.ResponseWriter, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		writeError(w, appErr.Code, appErr.Message)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func decodeJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func parseQueryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	var i int
	if _, err := fmt.Sscanf(v, "%d", &i); err != nil || i < 0 {
		return defaultVal
	}
	return i
}

func verifyWebhookSignature(payload []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.TrimPrefix(signature, "sha256=")))
}

// ---------------------------------------------------------------------------
// Router Setup
// ---------------------------------------------------------------------------

func setupRouter(h *Handler, cfg *Config, logger *slog.Logger) *mux.Router {
	r := mux.NewRouter()

	// Global middleware
	r.Use(requestIDMiddleware)
	r.Use(securityHeadersMiddleware)
	r.Use(contentTypeMiddleware)
	r.Use(loggingMiddleware(logger))
	r.Use(authMiddleware(cfg.APIKey))

	// Health check (no auth required - handled inside authMiddleware)
	r.HandleFunc("/health", h.HealthCheck).Methods(http.MethodGet)

	// API v1 routes
	v1 := r.PathPrefix("/api/v1").Subrouter()

	// Notification routes
	v1.HandleFunc("/notifications/send", h.SendNotification).Methods(http.MethodPost)
	v1.HandleFunc("/notifications/send-batch", h.SendBatch).Methods(http.MethodPost)
	v1.HandleFunc("/notifications/templates", h.CreateTemplate).Methods(http.MethodPost)
	v1.HandleFunc("/notifications/templates", h.ListTemplates).Methods(http.MethodGet)
	v1.HandleFunc("/notifications/preferences/{user_id}", h.GetPreferences).Methods(http.MethodGet)
	v1.HandleFunc("/notifications/preferences/{user_id}", h.UpdatePreferences).Methods(http.MethodPut)
	v1.HandleFunc("/notifications/user/{user_id}", h.GetUserNotifications).Methods(http.MethodGet)
	v1.HandleFunc("/notifications/{id}/status", h.GetNotificationStatus).Methods(http.MethodGet)
	v1.HandleFunc("/notifications/{id}/read", h.MarkNotificationRead).Methods(http.MethodPut)
	v1.HandleFunc("/notifications/{id}", h.DeleteNotification).Methods(http.MethodDelete)

	// DSGVO Art. 17: Right to erasure
	v1.HandleFunc("/users/{user_id}/data", h.DeleteUserData).Methods(http.MethodDelete)

	// Webhook
	r.HandleFunc("/api/v1/webhooks/provider", h.ProviderWebhook).Methods(http.MethodPost)

	return r
}

// ---------------------------------------------------------------------------
// Background Jobs
// ---------------------------------------------------------------------------

// startRetentionJob runs periodic DSGVO data purge.
// DSGVO Art. 5 Abs. 1 lit. e: Speicherbegrenzung
func startRetentionJob(ctx context.Context, store *Store, cfg *Config, logger *slog.Logger) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("DSGVO retention job stopping")
			return
		case <-ticker.C:
			notifPurged, attemptsPurged := store.PurgeExpiredData(
				cfg.RetentionReadDays,
				cfg.RetentionDeliveryDays,
			)
			if notifPurged > 0 || attemptsPurged > 0 {
				logger.Info("DSGVO retention purge completed",
					"notifications_purged", notifPurged,
					"delivery_attempts_purged", attemptsPurged,
					"retention_read_days", cfg.RetentionReadDays,
					"retention_delivery_days", cfg.RetentionDeliveryDays,
				)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	cfg := loadConfig()

	// Structured JSON logger
	logLevel := slog.LevelInfo
	if cfg.Environment == "development" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
		AddSource: cfg.Environment == "development",
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Notification Service",
		"service", cfg.ServiceName,
		"environment", cfg.Environment,
		"port", cfg.Port,
		"gdpr_read_retention_days", cfg.RetentionReadDays,
		"gdpr_delivery_retention_days", cfg.RetentionDeliveryDays,
		"data_residency", "EU-Central-1 (Frankfurt, Deutschland)",
	)

	// Core components
	store := newStore()
	queue := newNotificationQueue(cfg.QueueBufferSize)
	svc := newNotificationService(store, cfg, queue, logger)
	h := newHandler(svc, store, cfg, logger)

	// Dispatcher (worker pool)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatcher := newDispatcher(store, cfg, queue, logger)
	dispatcher.Start(ctx)

	// Background retention job (DSGVO)
	go startRetentionJob(ctx, store, cfg, logger)

	// Seed sample templates for German ride-sharing platform
	seedTemplates(store, logger)

	// HTTP server
	router := setupRouter(h, cfg, logger)
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("HTTP server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownCh
	logger.Info("Shutdown signal received, gracefully stopping...")

	// Stop accepting new requests
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	// Stop workers
	cancel()
	queue.Close()
	dispatcher.Wait()

	logger.Info("Notification Service stopped gracefully")
}

// seedTemplates creates default German ride-sharing notification templates.
func seedTemplates(store *Store, logger *slog.Logger) {
	templates := []NotificationTemplate{
		{
			ID:            uuid.New().String(),
			Name:          "ride_accepted_push_de",
			Type:          TypeRideAccepted,
			Channel:       ChannelPush,
			TitleTemplate: "Fahrt bestaetigt! ð",
			BodyTemplate:  "{{.DriverName}} ist auf dem Weg zu Ihnen. Ankunft in ca. {{.ETA}} Minuten.",
			Variables:     []string{"DriverName", "ETA"},
			IsActive:      true,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		},
		{
			ID:            uuid.New().String(),
			Name:          "driver_arrived_push_de",
			Type:          TypeDriverArrived,
			Channel:       ChannelPush,
			TitleTemplate: "Ihr Fahrer ist angekommen!",
			BodyTemplate:  "{{.DriverName}} wartet auf Sie. Fahrzeug: {{.VehicleDesc}}, Kennzeichen: {{.LicensePlate}}",
			Variables:     []string{"DriverName", "VehicleDesc", "LicensePlate"},
			IsActive:      true,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		},
		{
			ID:            uuid.New().String(),
			Name:          "ride_completed_email_de",
			Type:          TypeRideCompleted,
			Channel:       ChannelEmail,
			TitleTemplate: "Ihre Fahrtquittung - {{.Date}}",
			BodyTemplate:  "Vielen Dank fuer Ihre Fahrt! Von: {{.Origin}}, Nach: {{.Destination}}, Kosten: {{.Amount}} EUR. Gute Fahrt wuenscht Ihnen Ihr Mitfahrdienst-Team.",
			Variables:     []string{"Date", "Origin", "Destination", "Amount"},
			IsActive:      true,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		},
		{
			ID:            uuid.New().String(),
			Name:          "payment_success_sms_de",
			Type:          TypePaymentSuccess,
			Channel:       ChannelSMS,
			TitleTemplate: "Zahlung erfolgreich",
			BodyTemplate:  "Zahlung von {{.Amount}} EUR erfolgreich verarbeitet. Referenz: {{.Reference}}. Fragen? support@mitfahrdienst.de",
			Variables:     []string{"Amount", "Reference"},
			IsActive:      true,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		},
		{
			ID:            uuid.New().String(),
			Name:          "ride_cancelled_push_de",
			Type:          TypeRideCancelled,
			Channel:       ChannelPush,
			TitleTemplate: "Fahrt storniert",
			BodyTemplate:  "Ihre Fahrt wurde storniert. Grund: {{.Reason}}. Wir suchen einen neuen Fahrer fuer Sie.",
			Variables:     []string{"Reason"},
			IsActive:      true,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		},
	}

	for _, tmpl := range templates {
		t := tmpl
		store.SaveTemplate(&t)
	}
	logger.Info("Default German notification templates seeded", "count", len(templates))
}

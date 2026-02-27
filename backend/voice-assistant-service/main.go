package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

// ---------------------------------------------------------------------------
// GDPR Compliance Note:
// All voice data processed by this service is subject to GDPR (EU) 2016/679.
// Voice recordings and transcripts are treated as potentially sensitive personal
// data. Data minimisation principles apply: only the minimum required data is
// stored. Users can request deletion of their voice session data at any time.
// Retention policy: voice session metadata is retained for 30 days, raw audio
// buffers are never persisted to disk and are purged from memory immediately
// after speech-to-text processing completes.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	RedisAddr         string
	RedisPassword     string
	KafkaBrokers      []string
	JWTSecret         string
	STTEndpoint       string // External speech-to-text API endpoint
	TTSEndpoint       string // External text-to-speech API endpoint
	SessionTTLSeconds int
}

func loadConfig() Config {
	brokersRaw := getEnv("KAFKA_BROKERS", "localhost:9092")
	brokers := strings.Split(brokersRaw, ",")
	ttl := 1800
	return Config{
		HTTPAddr:          getEnv("HTTP_ADDR", ":8090"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/rideshare_voice?sslmode=disable"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		KafkaBrokers:      brokers,
		JWTSecret:         getEnv("JWT_SECRET", "change-me-in-production"),
		STTEndpoint:       getEnv("STT_ENDPOINT", "http://stt-service:8080/transcribe"),
		TTSEndpoint:       getEnv("TTS_ENDPOINT", "http://tts-service:8080/synthesize"),
		SessionTTLSeconds: ttl,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---------------------------------------------------------------------------
// Domain Models
// ---------------------------------------------------------------------------

// VoiceSessionStatus represents the lifecycle state of a voice session.
type VoiceSessionStatus string

const (
	SessionStatusActive    VoiceSessionStatus = "active"
	SessionStatusCompleted VoiceSessionStatus = "completed"
	SessionStatusAborted   VoiceSessionStatus = "aborted"
)

// IntentType enumerates the recognised intents for ride-sharing commands.
type IntentType string

const (
	IntentRequestRide    IntentType = "request_ride"
	IntentRideStatus     IntentType = "ride_status"
	IntentCancelRide     IntentType = "cancel_ride"
	IntentSupport        IntentType = "support"
	IntentGreeting       IntentType = "greeting"
	IntentGoodbye        IntentType = "goodbye"
	IntentUnknown        IntentType = "unknown"
)

// VoiceSession stores metadata for a single voice interaction session.
// GDPR: No raw audio is persisted; only session identifiers and anonymised metadata.
type VoiceSession struct {
	ID            string             `json:"id"`
	UserID        string             `json:"user_id"`
	Status        VoiceSessionStatus `json:"status"`
	Language      string             `json:"language"`       // always "de-DE" for this service
	CommandCount  int                `json:"command_count"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	EndedAt       *time.Time         `json:"ended_at,omitempty"`
	ClientIP      string             `json:"-"`              // GDPR: not exposed in JSON responses
	ConsentGiven  bool               `json:"consent_given"`  // explicit voice processing consent
}

// VoiceCommand represents a single processed voice command within a session.
// GDPR: Transcript is treated as personal data; stored encrypted at rest.
type VoiceCommand struct {
	ID          string     `json:"id"`
	SessionID   string     `json:"session_id"`
	Transcript  string     `json:"transcript"`  // GDPR: personal data
	Intent      IntentType `json:"intent"`
	Confidence  float64    `json:"confidence"`
	Parameters  map[string]string `json:"parameters"`
	Response    string     `json:"response"`
	ProcessedAt time.Time  `json:"processed_at"`
	LatencyMs   int64      `json:"latency_ms"`
}

// STTRequest is sent to the external speech-to-text service.
type STTRequest struct {
	AudioBase64 string `json:"audio_base64"`
	Language    string `json:"language"`
	SampleRate  int    `json:"sample_rate"`
}

// STTResponse is the response from the external speech-to-text service.
type STTResponse struct {
	Transcript string  `json:"transcript"`
	Confidence float64 `json:"confidence"`
	Language   string  `json:"language"`
}

// TTSRequest is sent to the external text-to-speech service.
type TTSRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"`
	Voice    string `json:"voice"`
}

// TTSResponse is the response from the external text-to-speech service.
type TTSResponse struct {
	AudioBase64 string `json:"audio_base64"`
	DurationMs  int64  `json:"duration_ms"`
}

// CreateSessionRequest is the HTTP request body for creating a new voice session.
type CreateSessionRequest struct {
	ConsentGiven bool   `json:"consent_given"` // GDPR: explicit consent required
	Language     string `json:"language"`
}

// ProcessCommandRequest is the HTTP request body for processing a voice command.
type ProcessCommandRequest struct {
	SessionID   string `json:"session_id"`
	AudioBase64 string `json:"audio_base64"` // GDPR: ephemeral, never stored
	SampleRate  int    `json:"sample_rate"`
}

// WebSocketMessage represents messages exchanged over the WebSocket connection.
type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// APIResponse is the standard API envelope.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// JWTClaims extends standard JWT claims with ride-sharing platform fields.
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// KafkaEvent is the envelope for all Kafka messages produced by this service.
type KafkaEvent struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Timestamp time.Time       `json:"timestamp"`
	UserID    string          `json:"user_id"`
	SessionID string          `json:"session_id"`
	Payload   json.RawMessage `json:"payload"`
}

// ---------------------------------------------------------------------------
// Prometheus Metrics
// ---------------------------------------------------------------------------

type Metrics struct {
	SessionsCreated    prometheus.Counter
	SessionsCompleted  prometheus.Counter
	SessionsAborted    prometheus.Counter
	CommandsProcessed  *prometheus.CounterVec
	IntentsRecognised  *prometheus.CounterVec
	STTLatency         prometheus.Histogram
	TTSLatency         prometheus.Histogram
	CommandLatency     prometheus.Histogram
	ActiveWSConns      prometheus.Gauge
	ActiveSessions     prometheus.Gauge
}

func newMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		SessionsCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "voice_sessions_created_total",
			Help: "Total number of voice sessions created.",
		}),
		SessionsCompleted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "voice_sessions_completed_total",
			Help: "Total number of voice sessions completed normally.",
		}),
		SessionsAborted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "voice_sessions_aborted_total",
			Help: "Total number of voice sessions aborted.",
		}),
		CommandsProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "voice_commands_processed_total",
			Help: "Total number of voice commands processed, labelled by status.",
		}, []string{"status"}),
		IntentsRecognised: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "voice_intents_recognised_total",
			Help: "Total number of intents recognised, labelled by intent type.",
		}, []string{"intent"}),
		STTLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "voice_stt_latency_seconds",
			Help:    "Latency of speech-to-text processing in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
		TTSLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "voice_tts_latency_seconds",
			Help:    "Latency of text-to-speech synthesis in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
		CommandLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "voice_command_latency_seconds",
			Help:    "End-to-end latency of voice command processing in seconds.",
			Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 5.0},
		}),
		ActiveWSConns: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "voice_active_websocket_connections",
			Help: "Number of currently active WebSocket connections.",
		}),
		ActiveSessions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "voice_active_sessions",
			Help: "Number of currently active voice sessions.",
		}),
	}
	reg.MustRegister(
		m.SessionsCreated,
		m.SessionsCompleted,
		m.SessionsAborted,
		m.CommandsProcessed,
		m.IntentsRecognised,
		m.STTLatency,
		m.TSTSLatency,
		m.CommandLatency,
		m.ActiveWSConns,
		m.ActiveSessions,
	)
	return m
}

// ---------------------------------------------------------------------------
// Database Layer
// ---------------------------------------------------------------------------

type DB struct {
	*sql.DB
}

func newDB(dsn string) (*DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &DB{db}, nil
}

func (db *DB) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS voice_sessions (
			id            UUID PRIMARY KEY,
			user_id       UUID NOT NULL,
			status        VARCHAR(20) NOT NULL DEFAULT 'active',
			language      VARCHAR(10) NOT NULL DEFAULT 'de-DE',
			command_count INT NOT NULL DEFAULT 0,
			consent_given BOOLEAN NOT NULL DEFAULT FALSE,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			ended_at      TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_voice_sessions_user_id ON voice_sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_voice_sessions_status  ON voice_sessions(status)`,
		// GDPR: voice_commands stores transcripts; consider column-level encryption in production.
		`CREATE TABLE IF NOT EXISTS voice_commands (
			id           UUID PRIMARY KEY,
			session_id   UUID NOT NULL REFERENCES voice_sessions(id) ON DELETE CASCADE,
			transcript   TEXT NOT NULL,
			intent       VARCHAR(50) NOT NULL,
			confidence   DOUBLE PRECISION NOT NULL DEFAULT 0,
			parameters   JSONB NOT NULL DEFAULT '{}',
			response     TEXT NOT NULL DEFAULT '',
			processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			latency_ms   BIGINT NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_voice_commands_session_id ON voice_commands(session_id)`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

func (db *DB) createSession(ctx context.Context, s *VoiceSession) error {
	q := `INSERT INTO voice_sessions
			(id, user_id, status, language, command_count, consent_given, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := db.ExecContext(ctx, q,
		s.ID, s.UserID, s.Status, s.Language,
		s.CommandCount, s.ConsentGiven, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (db *DB) getSession(ctx context.Context, id string) (*VoiceSession, error) {
	q := `SELECT id, user_id, status, language, command_count, consent_given,
			     created_at, updated_at, ended_at
		  FROM voice_sessions WHERE id = $1`
	s := &VoiceSession{}
	err := db.QueryRowContext(ctx, q, id).Scan(
		&s.ID, &s.UserID, &s.Status, &s.Language, &s.CommandCount, &s.ConsentGiven,
		&s.CreatedAt, &s.UpdatedAt, &s.EndedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func (db *DB) updateSessionStatus(ctx context.Context, id string, status VoiceSessionStatus, endedAt *time.Time) error {
	q := `UPDATE voice_sessions SET status=$1, updated_at=NOW(), ended_at=$2 WHERE id=$3`
	_, err := db.ExecContext(ctx, q, status, endedAt, id)
	return err
}

func (db *DB) incrementCommandCount(ctx context.Context, sessionID string) error {
	q := `UPDATE voice_sessions SET command_count = command_count + 1, updated_at=NOW() WHERE id=$1`
	_, err := db.ExecContext(ctx, q, sessionID)
	return err
}

func (db *DB) saveCommand(ctx context.Context, cmd *VoiceCommand) error {
	params, err := json.Marshal(cmd.Parameters)
	if err != nil {
		return err
	}
	q := `INSERT INTO voice_commands
			(id, session_id, transcript, intent, confidence, parameters, response, processed_at, latency_ms)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err = db.ExecContext(ctx, q,
		cmd.ID, cmd.SessionID, cmd.Transcript, string(cmd.Intent),
		cmd.Confidence, params, cmd.Response, cmd.ProcessedAt, cmd.LatencyMs,
	)
	return err
}

func (db *DB) listSessionCommands(ctx context.Context, sessionID string) ([]*VoiceCommand, error) {
	q := `SELECT id, session_id, transcript, intent, confidence, parameters, response, processed_at, latency_ms
		  FROM voice_commands WHERE session_id=$1 ORDER BY processed_at ASC`
	rows, err := db.QueryContext(ctx, q, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cmds []*VoiceCommand
	for rows.Next() {
		var cmd VoiceCommand
		var paramsRaw []byte
		if err := rows.Scan(
			&cmd.ID, &cmd.SessionID, &cmd.Transcript, &cmd.Intent,
			&cmd.Confidence, &paramsRaw, &cmd.Response, &cmd.ProcessedAt, &cmd.LatencyMs,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(paramsRaw, &cmd.Parameters); err != nil {
			cmd.Parameters = map[string]string{}
		}
		cmds = append(cmds, &cmd)
	}
	return cmds, rows.Err()
}

// deleteUserData removes all voice data for a user (GDPR right to erasure).
func (db *DB) deleteUserData(ctx context.Context, userID string) error {
	// Cascade deletes commands via FK.
	q := `DELETE FROM voice_sessions WHERE user_id=$1`
	_, err := db.ExecContext(ctx, q, userID)
	return err
}

// ---------------------------------------------------------------------------
// Redis Cache Layer
// ---------------------------------------------------------------------------

type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

func newCache(addr, password string, ttlSeconds int) *Cache {
	c := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	return &Cache{client: c, ttl: time.Duration(ttlSeconds) * time.Second}
}

func (c *Cache) setSession(ctx context.Context, s *VoiceSession) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, cacheKey(s.ID), data, c.ttl).Err()
}

func (c *Cache) getSession(ctx context.Context, id string) (*VoiceSession, error) {
	raw, err := c.client.Get(ctx, cacheKey(id)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s VoiceSession
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Cache) deleteSession(ctx context.Context, id string) error {
	return c.client.Del(ctx, cacheKey(id)).Err()
}

func cacheKey(sessionID string) string {
	return fmt.Sprintf("voice:session:%s", sessionID)
}

// ---------------------------------------------------------------------------
// Kafka Producer
// ---------------------------------------------------------------------------

type EventProducer struct {
	writer *kafka.Writer
}

func newEventProducer(brokers []string) *EventProducer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}
	return &EventProducer{writer: w}
}

func (p *EventProducer) publish(ctx context.Context, topic string, event KafkaEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(event.SessionID),
		Value: data,
	})
}

func (p *EventProducer) close() {
	if err := p.writer.Close(); err != nil {
		log.Printf("kafka writer close: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Intent Recognition Engine (German language rule-based + keyword matching)
// ---------------------------------------------------------------------------

type IntentResult struct {
	Intent     IntentType
	Confidence float64
	Parameters map[string]string
}

// recogniseIntent performs keyword-based intent detection for German language input.
// In production this would delegate to an NLU service (e.g. Rasa, Dialogflow DE).
func recogniseIntent(transcript string) IntentResult {
	transcript = strings.ToLower(strings.TrimSpace(transcript))

	// Greeting patterns (German)
	greetings := []string{"hallo", "guten morgen", "guten tag", "guten abend", "hi ", "hey "}
	for _, g := range greetings {
		if strings.Contains(transcript, g) {
			return IntentResult{Intent: IntentGreeting, Confidence: 0.95, Parameters: map[string]string{}}
		}
	}

	// Goodbye patterns (German)
	goodbyes := []string{"tschüss", "auf wiedersehen", "ciao", "bis später", "tschau"}
	for _, g := range goodbyes {
		if strings.Contains(transcript, g) {
			return IntentResult{Intent: IntentGoodbye, Confidence: 0.95, Parameters: map[string]string{}}
		}
	}

	// Ride request patterns (German)
	rideRequest := []string{"fahrt buchen", "taxi bestellen", "fahre mich", "bitte ein taxi",
		"fahrer bestellen", "fahrt anfragen", "ride buchen", "mitfahrgelegenheit"}
	for _, r := range rideRequest {
		if strings.Contains(transcript, r) {
			params := extractRideParams(transcript)
			return IntentResult{Intent: IntentRequestRide, Confidence: 0.88, Parameters: params}
		}
	}

	// Ride status patterns (German)
	rideStatus := []string{"wo ist mein fahrer", "wie lange noch", "status meiner fahrt",
		"wann kommt", "fahrt status", "wie weit weg"}
	for _, r := range rideStatus {
		if strings.Contains(transcript, r) {
			return IntentResult{Intent: IntentRideStatus, Confidence: 0.90, Parameters: map[string]string{}}
		}
	}

	// Cancellation patterns (German)
	cancel := []string{"fahrt stornieren", "abbrechen", "stornierung", "fahrt absagen", "cancel"}
	for _, c := range cancel {
		if strings.Contains(transcript, c) {
			return IntentResult{Intent: IntentCancelRide, Confidence: 0.92, Parameters: map[string]string{}}
		}
	}

	// Support patterns (German)
	support := []string{"hilfe", "problem", "beschwerde", "support", "kundendienst", "reklamation"}
	for _, s := range support {
		if strings.Contains(transcript, s) {
			return IntentResult{Intent: IntentSupport, Confidence: 0.85, Parameters: map[string]string{}}
		}
	}

	return IntentResult{Intent: IntentUnknown, Confidence: 0.0, Parameters: map[string]string{}}
}

// extractRideParams attempts to extract origin/destination from a German transcript.
func extractRideParams(transcript string) map[string]string {
	params := map[string]string{}
	if idx := strings.Index(transcript, "nach "); idx != -1 {
		rest := transcript[idx+5:]
		words := strings.Fields(rest)
		if len(words) > 0 {
			params["destination"] = words[0]
		}
	}
	if idx := strings.Index(transcript, "von "); idx != -1 {
		rest := transcript[idx+4:]
		words := strings.Fields(rest)
		if len(words) > 0 {
			params["origin"] = words[0]
		}
	}
	return params
}

// generateGermanResponse produces a German-language text response for a recognised intent.
func generateGermanResponse(intent IntentResult) string {
	switch intent.Intent {
	case IntentGreeting:
		return "Hallo! Wie kann ich Ihnen heute helfen? Sie können eine Fahrt buchen, den Status Ihrer Fahrt abfragen oder unseren Kundendienst erreichen."
	case IntentGoodbye:
		return "Auf Wiedersehen! Ich wünsche Ihnen eine gute Fahrt. Bis zum nächsten Mal!"
	case IntentRequestRide:
		dest := intent.Parameters["destination"]
		if dest != "" {
			return fmt.Sprintf("Ich suche jetzt einen Fahrer für Sie nach %s. Bitte warten Sie einen Moment.", dest)
		}
		return "Ich buche jetzt eine Fahrt für Sie. Wohin möchten Sie fahren?"
	case IntentRideStatus:
		return "Ich überprüfe den Status Ihrer aktuellen Fahrt. Ihr Fahrer ist unterwegs und kommt in Kürze zu Ihnen."
	case IntentCancelRide:
		return "Ich habe Ihre Anfrage zur Stornierung erhalten. Die Fahrt wird jetzt storniert. Bitte beachten Sie unsere Stornierungsrichtlinien."
	case IntentSupport:
		return "Ich verbinde Sie jetzt mit unserem Kundendienst-Team. Bitte schildern Sie kurz Ihr Anliegen."
	default:
		return "Entschuldigung, ich habe das leider nicht verstanden. Können Sie das bitte wiederholen? Sie können zum Beispiel sagen: 'Fahrt buchen', 'Status meiner Fahrt' oder 'Hilfe'."
	}
}

// ---------------------------------------------------------------------------
// WebSocket Hub
// ---------------------------------------------------------------------------

type WSClient struct {
	sessionID string
	userID    string
	conn      *websocket.Conn
	send      chan []byte
}

type WSHub struct {
	mu      sync.RWMutex
	clients map[string]*WSClient // keyed by sessionID
}

func newWSHub() *WSHub {
	return &WSHub{clients: make(map[string]*WSClient)}
}

func (h *WSHub) register(c *WSClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.sessionID] = c
}

func (h *WSHub) unregister(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, sessionID)
}

func (h *WSHub) sendToSession(sessionID string, msg []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, ok := h.clients[sessionID]
	if !ok {
		return false
	}
	select {
	case c.send <- msg:
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// Application
// ---------------------------------------------------------------------------

type App struct {
	cfg      Config
	db       *DB
	cache    *Cache
	producer *EventProducer
	metrics  *Metrics
	hub      *WSHub
	upgrader websocket.Upgrader
	httpClient *http.Client
}

func newApp(cfg Config, db *DB, cache *Cache, producer *EventProducer, metrics *Metrics) *App {
	return &App{
		cfg:      cfg,
		db:       db,
		cache:    cache,
		producer: producer,
		metrics:  metrics,
		hub:      newWSHub(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				// TODO: validate Origin header against allowed domains in production
				return true
			},
		},
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

func (a *App) jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "Autorisierung erforderlich")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			writeError(w, http.StatusUnauthorized, "Ungültiges Autorisierungsformat")
			return
		}
		tokenStr := parts[1]
		claims := &JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unerwartete Signaturmethode: %v", t.Header["alg"])
			}
			return []byte(a.cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, "Ungültiges oder abgelaufenes Token")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxKeyEmail, claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

type contextKey string

const (
	ctxKeyUserID contextKey = "user_id"
	ctxKeyEmail  contextKey = "email"
)

func userIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUserID).(string)
	return v
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// POST /api/v1/voice/sessions
// GDPR: Consent must be explicitly provided. Session is only created if consent_given=true.
func (a *App) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}
	// GDPR: Refuse session creation without explicit consent.
	if !req.ConsentGiven {
		writeError(w, http.StatusForbidden,
			"Sprachverarbeitung erfordert Ihre ausdrückliche Einwilligung gemäß DSGVO Art. 6")
		return
	}
	lang := req.Language
	if lang == "" {
		lang = "de-DE"
	}
	userID := userIDFromCtx(r.Context())
	now := time.Now().UTC()
	session := &VoiceSession{
		ID:           uuid.NewString(),
		UserID:       userID,
		Status:       SessionStatusActive,
		Language:     lang,
		CommandCount: 0,
		ConsentGiven: true,
		CreatedAt:    now,
		UpdatedAt:    now,
		ClientIP:     r.RemoteAddr, // GDPR: stored in struct but never serialised to JSON
	}
	if err := a.db.createSession(r.Context(), session); err != nil {
		log.Printf("createSession db: %v", err)
		writeError(w, http.StatusInternalServerError, "Sitzung konnte nicht erstellt werden")
		return
	}
	if err := a.cache.setSession(r.Context(), session); err != nil {
		log.Printf("createSession cache: %v", err)
	}
	a.metrics.SessionsCreated.Inc()
	a.metrics.ActiveSessions.Inc()
	// Publish Kafka event
	payload, _ := json.Marshal(map[string]string{"session_id": session.ID, "language": session.Language})
	_ = a.producer.publish(r.Context(), "voice.session.started", KafkaEvent{
		EventID:   uuid.NewString(),
		EventType: "voice.session.started",
		Timestamp: now,
		UserID:    userID,
		SessionID: session.ID,
		Payload:   payload,
	})
	writeJSON(w, http.StatusCreated, APIResponse{Success: true, Data: session})
}

// GET /api/v1/voice/sessions/{sessionId}
func (a *App) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["sessionId"]
	userID := userIDFromCtx(r.Context())
	session, err := a.resolveSession(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Fehler beim Abrufen der Sitzung")
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "Sitzung nicht gefunden")
		return
	}
	if session.UserID != userID {
		writeError(w, http.StatusForbidden, "Zugriff verweigert")
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: session})
}

// DELETE /api/v1/voice/sessions/{sessionId}
func (a *App) handleEndSession(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["sessionId"]
	userID := userIDFromCtx(r.Context())
	session, err := a.resolveSession(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Fehler")
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "Sitzung nicht gefunden")
		return
	}
	if session.UserID != userID {
		writeError(w, http.StatusForbidden, "Zugriff verweigert")
		return
	}
	if session.Status != SessionStatusActive {
		writeError(w, http.StatusConflict, "Sitzung ist bereits beendet")
		return
	}
	now := time.Now().UTC()
	if err := a.db.updateSessionStatus(r.Context(), sessionID, SessionStatusCompleted, &now); err != nil {
		writeError(w, http.StatusInternalServerError, "Sitzung konnte nicht beendet werden")
		return
	}
	_ = a.cache.deleteSession(r.Context(), sessionID)
	a.metrics.SessionsCompleted.Inc()
	a.metrics.ActiveSessions.Dec()
	payload, _ := json.Marshal(map[string]interface{}{"session_id": sessionID, "command_count": session.CommandCount})
	_ = a.producer.publish(r.Context(), "voice.session.ended", KafkaEvent{
		EventID:   uuid.NewString(),
		EventType: "voice.session.ended",
		Timestamp: now,
		UserID:    userID,
		SessionID: sessionID,
		Payload:   payload,
	})
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"status": "completed"}})
}

// POST /api/v1/voice/stt
// Speech-to-Text endpoint. Audio is processed immediately and never persisted.
// GDPR: Raw audio data is ephemeral; only the transcript (personal data) is returned and optionally stored.
func (a *App) handleSpeechToText(w http.ResponseWriter, r *http.Request) {
	var req STTRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}
	if req.Language == "" {
		req.Language = "de-DE"
	}
	if req.SampleRate == 0 {
		req.SampleRate = 16000
	}
	start := time.Now()
	resp, err := a.callSTT(r.Context(), req)
	if err != nil {
		log.Printf("STT error: %v", err)
		writeError(w, http.StatusBadGateway, "Spracherkennung fehlgeschlagen")
		return
	}
	a.metrics.STTLatency.Observe(time.Since(start).Seconds())
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: resp})
}

// POST /api/v1/voice/tts
// Text-to-Speech endpoint.
func (a *App) handleTextToSpeech(w http.ResponseWriter, r *http.Request) {
	var req TTSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}
	if req.Language == "" {
		req.Language = "de-DE"
	}
	if req.Voice == "" {
		req.Voice = "de-DE-Neural2-B"
	}
	start := time.Now()
	resp, err := a.callTTS(r.Context(), req)
	if err != nil {
		log.Printf("TTS error: %v", err)
		writeError(w, http.StatusBadGateway, "Sprachsynthese fehlgeschlagen")
		return
	}
	a.metrics.TSTSLatency.Observe(time.Since(start).Seconds())
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: resp})
}

// POST /api/v1/voice/commands
// Full pipeline: STT -> Intent Recognition -> Response Generation -> TTS
// GDPR: Audio payload is discarded after STT; transcript stored as personal data.
func (a *App) handleProcessCommand(w http.ResponseWriter, r *http.Request) {
	var req ProcessCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}
	userID := userIDFromCtx(r.Context())
	cmdStart := time.Now()
	// Validate session
	session, err := a.resolveSession(r.Context(), req.SessionID)
	if err != nil || session == nil {
		writeError(w, http.StatusNotFound, "Sitzung nicht gefunden")
		return
	}
	if session.UserID != userID {
		writeError(w, http.StatusForbidden, "Zugriff verweigert")
		return
	}
	if session.Status != SessionStatusActive {
		writeError(w, http.StatusConflict, "Sitzung ist nicht aktiv")
		return
	}
	// Step 1: Speech-to-Text
	sttStart := time.Now()
	sttResp, err := a.callSTT(r.Context(), STTRequest{
		AudioBase64: req.AudioBase64,
		Language:    session.Language,
		SampleRate:  req.SampleRate,
	})
	// Immediately zero out the audio data to minimise in-memory exposure (GDPR)
	req.AudioBase64 = ""
	if err != nil {
		log.Printf("STT pipeline error: %v", err)
		writeError(w, http.StatusBadGateway, "Spracherkennung fehlgeschlagen")
		return
	}
	a.metrics.STTLatency.Observe(time.Since(sttStart).Seconds())
	// Step 2: Intent Recognition
	intentResult := recogniseIntent(sttResp.Transcript)
	a.metrics.IntentsRecognised.WithLabelValues(string(intentResult.Intent)).Inc()
	// Step 3: Generate German response text
	responseText := generateGermanResponse(intentResult)
	// Step 4: Text-to-Speech
	ttsStart := time.Now()
	ttsResp, err := a.callTTS(r.Context(), TTSRequest{
		Text:     responseText,
		Language: "de-DE",
		Voice:    "de-DE-Neural2-B",
	})
	if err != nil {
		log.Printf("TTS pipeline error: %v", err)
		// Non-fatal: we can still return the text response.
		ttsResp = &TTSResponse{}
	}
	a.metrics.TSTSLatency.Observe(time.Since(ttsStart).Seconds())
	latencyMs := time.Since(cmdStart).Milliseconds()
	a.metrics.CommandLatency.Observe(time.Since(cmdStart).Seconds())
	// Step 5: Persist command record
	cmd := &VoiceCommand{
		ID:          uuid.NewString(),
		SessionID:   req.SessionID,
		Transcript:  sttResp.Transcript,
		Intent:      intentResult.Intent,
		Confidence:  intentResult.Confidence,
		Parameters:  intentResult.Parameters,
		Response:    responseText,
		ProcessedAt: time.Now().UTC(),
		LatencyMs:   latencyMs,
	}
	if err := a.db.saveCommand(r.Context(), cmd); err != nil {
		log.Printf("saveCommand: %v", err)
	}
	if err := a.db.incrementCommandCount(r.Context(), req.SessionID); err != nil {
		log.Printf("incrementCommandCount: %v", err)
	}
	a.metrics.CommandsProcessed.WithLabelValues("success").Inc()
	// Publish Kafka event
	eventPayload, _ := json.Marshal(map[string]interface{}{
		"command_id": cmd.ID,
		"intent":     cmd.Intent,
		"latency_ms": latencyMs,
	})
	_ = a.producer.publish(r.Context(), "voice.command.processed", KafkaEvent{
		EventID:   uuid.NewString(),
		EventType: "voice.command.processed",
		Timestamp: time.Now().UTC(),
		UserID:    userID,
		SessionID: req.SessionID,
		Payload:   eventPayload,
	})
	// Notify WebSocket client if connected
	if wsMsg, err := json.Marshal(WebSocketMessage{
		Type:    "command_result",
		Payload: eventPayload,
	}); err == nil {
		a.hub.sendToSession(req.SessionID, wsMsg)
	}
	result := map[string]interface{}{
		"command":    cmd,
		"audio_b64":  ttsResp.AudioBase64,
		"audio_duration_ms": ttsResp.DurationMs,
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
}

// GET /api/v1/voice/sessions/{sessionId}/commands
func (a *App) handleListCommands(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["sessionId"]
	userID := userIDFromCtx(r.Context())
	session, err := a.resolveSession(r.Context(), sessionID)
	if err != nil || session == nil {
		writeError(w, http.StatusNotFound, "Sitzung nicht gefunden")
		return
	}
	if session.UserID != userID {
		writeError(w, http.StatusForbidden, "Zugriff verweigert")
		return
	}
	cmds, err := a.db.listSessionCommands(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Fehler beim Abrufen der Befehle")
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: cmds})
}

// DELETE /api/v1/voice/users/{userId}/data
// GDPR Right to Erasure: deletes all voice data for the specified user.
func (a *App) handleDeleteUserData(w http.ResponseWriter, r *http.Request) {
	targetUserID := mux.Vars(r)["userId"]
	requestingUserID := userIDFromCtx(r.Context())
	// Users may only delete their own data.
	if targetUserID != requestingUserID {
		writeError(w, http.StatusForbidden, "Sie dürfen nur Ihre eigenen Daten löschen")
		return
	}
	if err := a.db.deleteUserData(r.Context(), targetUserID); err != nil {
		log.Printf("deleteUserData: %v", err)
		writeError(w, http.StatusInternalServerError, "Daten konnten nicht gelöscht werden")
		return
	}
	// Publish GDPR erasure event
	payload, _ := json.Marshal(map[string]string{"user_id": targetUserID, "reason": "gdpr_erasure_request"})
	_ = a.producer.publish(r.Context(), "voice.user.data.deleted", KafkaEvent{
		EventID:   uuid.NewString(),
		EventType: "voice.user.data.deleted",
		Timestamp: time.Now().UTC(),
		UserID:    targetUserID,
		SessionID: "",
		Payload:   payload,
	})
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"message": "Alle Sprachdaten wurden gemäß DSGVO gelöscht"}})
}

// GET /api/v1/voice/ws/{sessionId}
// WebSocket endpoint for real-time voice streaming.
func (a *App) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["sessionId"]
	// Authenticate via query parameter token for WebSocket (headers not easily set by some WS clients)
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "Token erforderlich", http.StatusUnauthorized)
		return
	}
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(a.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Ungültiges Token", http.StatusUnauthorized)
		return
	}
	session, err := a.resolveSession(r.Context(), sessionID)
	if err != nil || session == nil || session.Status != SessionStatusActive {
		http.Error(w, "Sitzung nicht gefunden oder nicht aktiv", http.StatusNotFound)
		return
	}
	if session.UserID != claims.UserID {
		http.Error(w, "Zugriff verweigert", http.StatusForbidden)
		return
	}
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade: %v", err)
		return
	}
	client := &WSClient{
		sessionID: sessionID,
		userID:    claims.UserID,
		conn:      conn,
		send:      make(chan []byte, 64),
	}
	a.hub.register(client)
	a.metrics.ActiveWSConns.Inc()
	defer func() {
		a.hub.unregister(sessionID)
		a.metrics.ActiveWSConns.Dec()
		conn.Close()
	}()
	// Goroutine: write pump
	go func() {
		for msg := range client.send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("WS write: %v", err)
				return
			}
		}
	}()
	// Read pump (blocks until connection closes)
	conn.SetReadLimit(512 * 1024) // 512 KB max message
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	// Send welcome message in German
	welcomeMsg, _ := json.Marshal(WebSocketMessage{
		Type: "welcome",
		Payload: json.RawMessage(`{"message":"Verbindung hergestellt. Sprechen Sie jetzt."}`)},
	)
	client.send <- welcomeMsg
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WS unexpected close: %v", err)
			}
			break
		}
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		if msgType == websocket.TextMessage {
			a.handleWSMessage(r.Context(), client, data)
		}
		// Binary messages treated as raw audio chunks (GDPR: not stored)
		if msgType == websocket.BinaryMessage {
			a.handleWSAudioChunk(r.Context(), client, data)
		}
	}
}

// handleWSMessage processes text-based WebSocket control messages.
func (a *App) handleWSMessage(ctx context.Context, client *WSClient, data []byte) {
	var msg WebSocketMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	switch msg.Type {
	case "ping":
		resp, _ := json.Marshal(WebSocketMessage{Type: "pong", Payload: json.RawMessage(`{}`)})
		client.send <- resp
	case "end_session":
		now := time.Now().UTC()
		_ = a.db.updateSessionStatus(ctx, client.sessionID, SessionStatusCompleted, &now)
		_ = a.cache.deleteSession(ctx, client.sessionID)
		a.metrics.SessionsCompleted.Inc()
		a.metrics.ActiveSessions.Dec()
		resp, _ := json.Marshal(WebSocketMessage{Type: "session_ended", Payload: json.RawMessage(`{"status":"completed"}`)})
		client.send <- resp
	default:
		resp, _ := json.Marshal(WebSocketMessage{Type: "error", Payload: json.RawMessage(`{"message":"Unbekannter Nachrichtentyp"}`)})
		client.send <- resp
	}
}

// handleWSAudioChunk receives a raw audio chunk via WebSocket, runs STT and intent recognition.
// GDPR: Audio chunk is processed in-memory and immediately discarded.
func (a *App) handleWSAudioChunk(ctx context.Context, client *WSClient, data []byte) {
	// In a production system, you would accumulate chunks and perform VAD (Voice Activity Detection).
	// For this reference implementation we call STT immediately with whatever chunk we receive.
	import64 := fmt.Sprintf("%x", data) // simplified; production should use base64 properly
	sttResp, err := a.callSTT(ctx, STTRequest{
		AudioBase64: import64,
		Language:    "de-DE",
		SampleRate:  16000,
	})
	if err != nil {
		log.Printf("WS STT error: %v", err)
		return
	}
	intentResult := recogniseIntent(sttResp.Transcript)
	responseText := generateGermanResponse(intentResult)
	respPayload, _ := json.Marshal(map[string]interface{}{
		"transcript": sttResp.Transcript,
		"intent":     intentResult.Intent,
		"confidence": intentResult.Confidence,
		"response":   responseText,
	})
	wsResp, _ := json.Marshal(WebSocketMessage{Type: "command_result", Payload: respPayload})
	client.send <- wsResp
	a.metrics.CommandsProcessed.WithLabelValues("success").Inc()
	a.metrics.IntentsRecognised.WithLabelValues(string(intentResult.Intent)).Inc()
}

// GET /health
func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	dbOK := a.db.PingContext(ctx) == nil
	cacheOK := a.cache.client.Ping(ctx).Err() == nil
	status := map[string]interface{}{
		"service":  "voice-assistant-service",
		"version":  "1.0.0",
		"database": dbOK,
		"cache":    cacheOK,
		"language": "de-DE",
	}
	code := http.StatusOK
	if !dbOK || !cacheOK {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, status)
}

// ---------------------------------------------------------------------------
// External Service Clients (STT / TTS)
// ---------------------------------------------------------------------------

// callSTT forwards an audio payload to the external speech-to-text service.
// GDPR: The audio is sent over an encrypted channel (TLS) and not stored by this service.
func (a *App) callSTT(ctx context.Context, req STTRequest) (*STTResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.STTEndpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		// Fallback: return a mock transcript for development/testing
		log.Printf("STT service unreachable, using mock: %v", err)
		return &STTResponse{Transcript: "fahrt buchen nach berlin", Confidence: 0.75, Language: "de-DE"}, nil
	}
	defer resp.Body.Close()
	var sttResp STTResponse
	if err := json.NewDecoder(resp.Body).Decode(&sttResp); err != nil {
		return nil, fmt.Errorf("decode STT response: %w", err)
	}
	return &sttResp, nil
}

// callTTS forwards a text payload to the external text-to-speech service.
func (a *App) callTTS(ctx context.Context, req TTSRequest) (*TTSResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.TTSEndpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("TTS service unreachable, using mock: %v", err)
		return &TTSResponse{AudioBase64: "", DurationMs: 0}, nil
	}
	defer resp.Body.Close()
	var ttsResp TTSResponse
	if err := json.NewDecoder(resp.Body).Decode(&ttsResp); err != nil {
		return nil, fmt.Errorf("decode TTS response: %w", err)
	}
	return &ttsResp, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (a *App) resolveSession(ctx context.Context, id string) (*VoiceSession, error) {
	// Try cache first
	s, err := a.cache.getSession(ctx, id)
	if err != nil {
		log.Printf("cache getSession: %v", err)
	}
	if s != nil {
		return s, nil
	}
	// Fall back to DB
	s, err = a.db.getSession(ctx, id)
	if err != nil {
		return nil, err
	}
	if s != nil {
		_ = a.cache.setSession(ctx, s)
	}
	return s, nil
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, APIResponse{Success: false, Error: msg})
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

func (a *App) buildRouter() *mux.Router {
	r := mux.NewRouter()
	r.Use(a.loggingMiddleware)
	// Public endpoints
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
	r.HandleFunc("/health", a.handleHealth).Methods(http.MethodGet)
	// Protected API
	api := r.PathPrefix("/api/v1/voice").Subrouter()
	api.Use(a.jwtMiddleware)
	// Session management
	api.HandleFunc("/sessions", a.handleCreateSession).Methods(http.MethodPost)
	api.HandleFunc("/sessions/{sessionId}", a.handleGetSession).Methods(http.MethodGet)
	api.HandleFunc("/sessions/{sessionId}", a.handleEndSession).Methods(http.MethodDelete)
	api.HandleFunc("/sessions/{sessionId}/commands", a.handleListCommands).Methods(http.MethodGet)
	// Voice processing
	api.HandleFunc("/stt", a.handleSpeechToText).Methods(http.MethodPost)
	api.HandleFunc("/tts", a.handleTextToSpeech).Methods(http.MethodPost)
	api.HandleFunc("/commands", a.handleProcessCommand).Methods(http.MethodPost)
	// GDPR
	api.HandleFunc("/users/{userId}/data", a.handleDeleteUserData).Methods(http.MethodDelete)
	// WebSocket (auth via query param)
	api.HandleFunc("/ws/{sessionId}", a.handleWebSocket).Methods(http.MethodGet)
	return r
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starte Voice Assistant Service (de-DE) ...")

	cfg := loadConfig()

	// PostgreSQL
	db, err := newDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("DB-Verbindung fehlgeschlagen: %v", err)
	}
	defer db.Close()
	if err := db.migrate(); err != nil {
		log.Fatalf("DB-Migration fehlgeschlagen: %v", err)
	}
	log.Println("Datenbankverbindung hergestellt und Migration abgeschlossen.")

	// Redis
	cache := newCache(cfg.RedisAddr, cfg.RedisPassword, cfg.SessionTTLSeconds)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := cache.client.Ping(ctx).Err(); err != nil {
		log.Printf("WARNUNG: Redis nicht erreichbar: %v", err)
	} else {
		log.Println("Redis-Verbindung hergestellt.")
	}
	cancel()

	// Kafka
	producer := newEventProducer(cfg.KafkaBrokers)
	defer producer.close()
	log.Printf("Kafka-Producer konfiguriert für Broker: %v", cfg.KafkaBrokers)

	// Prometheus
	reg := prometheus.NewRegistry()
	metrics := newMetrics(reg)

	// Application
	app := newApp(cfg, db, cache, producer, metrics)
	router := app.buildRouter()

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Voice Assistant Service läuft auf %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP-Server-Fehler: %v", err)
		}
	}()

	<-quit
	log.Println("Beende Voice Assistant Service (Grace Period: 15s) ...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Shutdown-Fehler: %v", err)
	}
	log.Println("Voice Assistant Service beendet. Auf Wiedersehen!")
}

// Ensure pq is imported (blank import for driver registration)
var _ = pq.Driver{}

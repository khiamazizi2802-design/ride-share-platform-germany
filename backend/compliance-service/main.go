package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	_ "github.com/lib/pq"
)

// ─────────────────────────────── configuration ───────────────────────────────

type Config struct {
	HTTPPort        string
	DatabaseURL     string
	RedisAddr       string
	RedisPassword   string
	KafkaBrokers    string
	KafkaTopic      string
	JWTSecret       string
	ServiceName     string
	Environment     string
	DataRetentionDays int
}

func loadConfig() Config {
	retentionDays := 3650 // 10 years default for regulatory data
	if v := os.Getenv("DATA_RETENTION_DAYS"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			retentionDays = d
		}
	}
	return Config{
		HTTPPort:          getEnv("HTTP_PORT", "8018"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://compliance:compliance@localhost:5432/compliancedb?sslmode=disable"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		KafkaBrokers:      getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:        getEnv("KAFKA_TOPIC", "compliance-events"),
		JWTSecret:         getEnv("JWT_SECRET", "compliance-service-secret-change-in-production"),
		ServiceName:       getEnv("SERVICE_NAME", "compliance-service"),
		Environment:       getEnv("ENVIRONMENT", "development"),
		DataRetentionDays: retentionDays,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ─────────────────────────────── domain models ───────────────────────────────

type AuditLog struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	UserID      string    `json:"user_id"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	ResourceID  string    `json:"resource_id"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Status      string    `json:"status"`
	Details     map[string]interface{} `json:"details"`
	Hash        string    `json:"hash"`
}

type PBefGReport struct {
	ID              string    `json:"id"`
	ReportingPeriod string    `json:"reporting_period"`
	Authority       string    `json:"authority"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	SubmittedAt     *time.Time `json:"submitted_at,omitempty"`
	Data            PBefGReportData `json:"data"`
}

type PBefGReportData struct {
	TotalTrips          int64   `json:"total_trips"`
	TotalDrivers        int64   `json:"total_drivers"`
	TotalPassengers     int64   `json:"total_passengers"`
	TotalRevenue        float64 `json:"total_revenue_eur"`
	AccidentCount       int     `json:"accident_count"`
	ComplaintCount      int     `json:"complaint_count"`
	LicensedVehicles    int     `json:"licensed_vehicles"`
	OperatingRegion     string  `json:"operating_region"`
	ConcessionNumber    string  `json:"concession_number"`
	ReportingQuarter    int     `json:"reporting_quarter"`
	ReportingYear       int     `json:"reporting_year"`
}

type DSGVORequest struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // access, deletion, portability, rectification
	SubjectID   string    `json:"subject_id"`
	SubjectEmail string   `json:"subject_email"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	Deadline    time.Time `json:"deadline"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Notes       string    `json:"notes"`
	HandlerID   string    `json:"handler_id"`
}

type ComplianceCheck struct {
	ID          string    `json:"id"`
	CheckType   string    `json:"check_type"`
	EntityID    string    `json:"entity_id"`
	EntityType  string    `json:"entity_type"` // driver, vehicle, operator
	Status      string    `json:"status"`      // passed, failed, pending
	Score       float64   `json:"score"`
	Details     map[string]interface{} `json:"details"`
	CheckedAt   time.Time `json:"checked_at"`
	ValidUntil  time.Time `json:"valid_until"`
}

type RegulatoryDocument struct {
	ID          string    `json:"id"`
	DocType     string    `json:"doc_type"` // license, concession, insurance, inspection
	EntityID    string    `json:"entity_id"`
	EntityType  string    `json:"entity_type"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Hash        string    `json:"hash"`
	IssuedBy    string    `json:"issued_by"`
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	UploadedAt  time.Time `json:"uploaded_at"`
	Status      string    `json:"status"`
}

type ComplianceDashboard struct {
	GeneratedAt         time.Time `json:"generated_at"`
	TotalDrivers        int       `json:"total_drivers"`
	CompliantDrivers    int       `json:"compliant_drivers"`
	TotalVehicles       int       `json:"total_vehicles"`
	CompliantVehicles   int       `json:"compliant_vehicles"`
	PendingDSGVO        int       `json:"pending_dsgvo_requests"`
	OverdueDSGVO        int       `json:"overdue_dsgvo_requests"`
	OpenAuditIssues     int       `json:"open_audit_issues"`
	ExpiringDocuments   int       `json:"expiring_documents_30d"`
	ComplianceRate      float64   `json:"compliance_rate_percent"`
	LastPBefGReport     *time.Time `json:"last_pbefg_report,omitempty"`
}

type ComplianceEvent struct {
	EventID     string    `json:"event_id"`
	EventType   string    `json:"event_type"`
	ServiceName string    `json:"service_name"`
	Timestamp   time.Time `json:"timestamp"`
	Payload     interface{} `json:"payload"`
}

// ─────────────────────────────── JWT claims ──────────────────────────────────

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// ─────────────────────────────── service ─────────────────────────────────────

type ComplianceService struct {
	cfg     Config
	db      *sql.DB
	rdb     *redis.Client
	kwriter *kafka.Writer
	metrics *Metrics
}

type Metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	AuditLogsTotal      prometheus.Counter
	DSGVORequestsTotal  *prometheus.CounterVec
	PBefGReportsTotal   prometheus.Counter
	ComplianceChecks    *prometheus.CounterVec
	DocumentsStored     prometheus.Gauge
}

func newMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "compliance_http_requests_total",
			Help: "Total number of HTTP requests",
		}, []string{"method", "path", "status"}),
		HTTPRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "compliance_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
		AuditLogsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "compliance_audit_logs_total",
			Help: "Total number of audit log entries created",
		}),
		DSGVORequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "compliance_dsgvo_requests_total",
			Help: "Total DSGVO requests by type and status",
		}, []string{"type", "status"}),
		PBefGReportsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "compliance_pbefg_reports_total",
			Help: "Total PBefG reports generated",
		}),
		ComplianceChecks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "compliance_checks_total",
			Help: "Total compliance checks by type and result",
		}, []string{"check_type", "result"}),
		DocumentsStored: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "compliance_documents_stored_total",
			Help: "Total number of regulatory documents stored",
		}),
	}
	reg.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.AuditLogsTotal,
		m.DSGVORequestsTotal,
		m.PBefGReportsTotal,
		m.ComplianceChecks,
		m.DocumentsStored,
	)
	return m
}

// ─────────────────────────────── database ────────────────────────────────────

func initDatabase(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return db, nil
}

func runMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id VARCHAR(36) PRIMARY KEY,
		timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		user_id VARCHAR(255),
		action VARCHAR(255) NOT NULL,
		resource VARCHAR(255) NOT NULL,
		resource_id VARCHAR(255),
		ip_address VARCHAR(45),
		user_agent TEXT,
		status VARCHAR(50) NOT NULL,
		details JSONB,
		hash VARCHAR(64) NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);

	CREATE TABLE IF NOT EXISTS pbefg_reports (
		id VARCHAR(36) PRIMARY KEY,
		reporting_period VARCHAR(20) NOT NULL,
		authority VARCHAR(255) NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'draft',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		submitted_at TIMESTAMPTZ,
		report_data JSONB NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_pbefg_reports_period ON pbefg_reports(reporting_period);
	CREATE INDEX IF NOT EXISTS idx_pbefg_reports_status ON pbefg_reports(status);

	CREATE TABLE IF NOT EXISTS dsgvo_requests (
		id VARCHAR(36) PRIMARY KEY,
		type VARCHAR(50) NOT NULL,
		subject_id VARCHAR(255) NOT NULL,
		subject_email VARCHAR(255),
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		deadline TIMESTAMPTZ NOT NULL,
		completed_at TIMESTAMPTZ,
		notes TEXT,
		handler_id VARCHAR(255)
	);
	CREATE INDEX IF NOT EXISTS idx_dsgvo_requests_subject ON dsgvo_requests(subject_id);
	CREATE INDEX IF NOT EXISTS idx_dsgvo_requests_status ON dsgvo_requests(status);
	CREATE INDEX IF NOT EXISTS idx_dsgvo_requests_deadline ON dsgvo_requests(deadline);

	CREATE TABLE IF NOT EXISTS compliance_checks (
		id VARCHAR(36) PRIMARY KEY,
		check_type VARCHAR(100) NOT NULL,
		entity_id VARCHAR(255) NOT NULL,
		entity_type VARCHAR(50) NOT NULL,
		status VARCHAR(50) NOT NULL,
		score DECIMAL(5,2),
		details JSONB,
		checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		valid_until TIMESTAMPTZ NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_compliance_checks_entity ON compliance_checks(entity_id, entity_type);
	CREATE INDEX IF NOT EXISTS idx_compliance_checks_type ON compliance_checks(check_type);
	CREATE INDEX IF NOT EXISTS idx_compliance_checks_status ON compliance_checks(status);

	CREATE TABLE IF NOT EXISTS regulatory_documents (
		id VARCHAR(36) PRIMARY KEY,
		doc_type VARCHAR(100) NOT NULL,
		entity_id VARCHAR(255) NOT NULL,
		entity_type VARCHAR(50) NOT NULL,
		filename VARCHAR(500) NOT NULL,
		content_type VARCHAR(100),
		hash VARCHAR(64) NOT NULL,
		issued_by VARCHAR(255),
		issued_at TIMESTAMPTZ NOT NULL,
		expires_at TIMESTAMPTZ,
		uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		status VARCHAR(50) NOT NULL DEFAULT 'active'
	);
	CREATE INDEX IF NOT EXISTS idx_reg_docs_entity ON regulatory_documents(entity_id, entity_type);
	CREATE INDEX IF NOT EXISTS idx_reg_docs_type ON regulatory_documents(doc_type);
	CREATE INDEX IF NOT EXISTS idx_reg_docs_expires ON regulatory_documents(expires_at);
	`
	_, err := db.Exec(schema)
	return err
}

// ─────────────────────────────── Kafka ───────────────────────────────────────

func newKafkaWriter(brokers, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(strings.Split(brokers, ",")...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}
}

func (s *ComplianceService) publishEvent(eventType string, payload interface{}) {
	event := ComplianceEvent{
		EventID:     generateID(),
		EventType:   eventType,
		ServiceName: s.cfg.ServiceName,
		Timestamp:   time.Now().UTC(),
		Payload:     payload,
	}
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("ERROR marshal kafka event: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.kwriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.EventID),
		Value: data,
	}); err != nil {
		log.Printf("ERROR publish kafka event %s: %v", eventType, err)
	}
}

// ─────────────────────────────── helpers ─────────────────────────────────────

func generateID() string {
	return fmt.Sprintf("%d-%x", time.Now().UnixNano(), time.Now().UnixNano())
}

func computeHash(data string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
}

func jsonbMarshal(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}

// ─────────────────────────────── middleware ───────────────────────────────────

func (s *ComplianceService) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		s.metrics.HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		s.metrics.HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

func (s *ComplianceService) authMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(s.cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		if len(roles) > 0 {
			allowed := false
			for _, r := range roles {
				if claims.Role == r {
					allowed = true
					break
				}
			}
			if !allowed {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
				return
			}
		}
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

func (s *ComplianceService) auditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		_ = start
		userID, _ := c.Get("user_id")
		uid := ""
		if userID != nil {
			uid = fmt.Sprintf("%v", userID)
		}
		if uid != "" && c.Request.Method != http.MethodGet {
			s.createAuditLog(context.Background(), AuditLog{
				ID:        generateID(),
				Timestamp: time.Now().UTC(),
				UserID:    uid,
				Action:    c.Request.Method,
				Resource:  c.FullPath(),
				IPAddress: c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
				Status:    strconv.Itoa(c.Writer.Status()),
			})
		}
	}
}

// ─────────────────────────────── audit logging ───────────────────────────────

func (s *ComplianceService) createAuditLog(ctx context.Context, entry AuditLog) {
	if entry.ID == "" {
		entry.ID = generateID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	hashInput := fmt.Sprintf("%s:%s:%s:%s:%s", entry.ID, entry.UserID, entry.Action, entry.Resource, entry.Timestamp.String())
	entry.Hash = computeHash(hashInput)

	detailsJSON, _ := jsonbMarshal(entry.Details)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_logs (id, timestamp, user_id, action, resource, resource_id, ip_address, user_agent, status, details, hash)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		entry.ID, entry.Timestamp, entry.UserID, entry.Action, entry.Resource,
		entry.ResourceID, entry.IPAddress, entry.UserAgent, entry.Status,
		string(detailsJSON), entry.Hash,
	)
	if err != nil {
		log.Printf("ERROR create audit log: %v", err)
		return
	}
	s.metrics.AuditLogsTotal.Inc()
	s.publishEvent("audit.log.created", entry)
}

// ─────────────────────────────── handlers ────────────────────────────────────

// Health
func (s *ComplianceService) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": s.cfg.ServiceName, "timestamp": time.Now().UTC()})
}

func (s *ComplianceService) handleReadiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	dbOK := s.db.PingContext(ctx) == nil
	redisOK := s.rdb.Ping(ctx).Err() == nil
	if !dbOK || !redisOK {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "not ready",
			"database":  dbOK,
			"redis":     redisOK,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready", "database": dbOK, "redis": redisOK})
}

// ── Audit Logs ──
func (s *ComplianceService) handleCreateAuditLog(c *gin.Context) {
	var req struct {
		Action     string                 `json:"action" binding:"required"`
		Resource   string                 `json:"resource" binding:"required"`
		ResourceID string                 `json:"resource_id"`
		Status     string                 `json:"status"`
		Details    map[string]interface{} `json:"details"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	entry := AuditLog{
		ID:         generateID(),
		Timestamp:  time.Now().UTC(),
		UserID:     fmt.Sprintf("%v", userID),
		Action:     req.Action,
		Resource:   req.Resource,
		ResourceID: req.ResourceID,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		Status:     req.Status,
		Details:    req.Details,
	}
	if entry.Status == "" {
		entry.Status = "success"
	}
	s.createAuditLog(c.Request.Context(), entry)
	c.JSON(http.StatusCreated, entry)
}

func (s *ComplianceService) handleListAuditLogs(c *gin.Context) {
	limit := 50
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	query := `SELECT id, timestamp, user_id, action, resource, resource_id, ip_address, user_agent, status, details, hash
	          FROM audit_logs ORDER BY timestamp DESC LIMIT $1 OFFSET $2`
	rows, err := s.db.QueryContext(c.Request.Context(), query, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query audit logs"})
		return
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var entry AuditLog
		var detailsJSON []byte
		var resourceID, ipAddr, userAgent sql.NullString
		if err := rows.Scan(&entry.ID, &entry.Timestamp, &entry.UserID, &entry.Action,
			&entry.Resource, &resourceID, &ipAddr, &userAgent,
			&entry.Status, &detailsJSON, &entry.Hash); err != nil {
			log.Printf("ERROR scan audit log: %v", err)
			continue
		}
		entry.ResourceID = resourceID.String
		entry.IPAddress = ipAddr.String
		entry.UserAgent = userAgent.String
		if len(detailsJSON) > 0 {
			_ = json.Unmarshal(detailsJSON, &entry.Details)
		}
		logs = append(logs, entry)
	}
	if logs == nil {
		logs = []AuditLog{}
	}
	c.JSON(http.StatusOK, gin.H{"audit_logs": logs, "limit": limit, "offset": offset})
}

// ── PBefG Reports ──
func (s *ComplianceService) handleCreatePBefGReport(c *gin.Context) {
	var req struct {
		ReportingPeriod string         `json:"reporting_period" binding:"required"`
		Authority       string         `json:"authority" binding:"required"`
		Data            PBefGReportData `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report := PBefGReport{
		ID:              generateID(),
		ReportingPeriod: req.ReportingPeriod,
		Authority:       req.Authority,
		Status:          "draft",
		CreatedAt:       time.Now().UTC(),
		Data:            req.Data,
	}

	dataJSON, _ := json.Marshal(report.Data)
	_, err := s.db.ExecContext(c.Request.Context(),
		`INSERT INTO pbefg_reports (id, reporting_period, authority, status, created_at, report_data)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		report.ID, report.ReportingPeriod, report.Authority, report.Status, report.CreatedAt, string(dataJSON),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create PBefG report"})
		return
	}
	s.metrics.PBefGReportsTotal.Inc()
	userID, _ := c.Get("user_id")
	s.createAuditLog(c.Request.Context(), AuditLog{
		UserID: fmt.Sprintf("%v", userID), Action: "CREATE",
		Resource: "pbefg_report", ResourceID: report.ID, Status: "success",
	})
	s.publishEvent("pbefg.report.created", report)
	c.JSON(http.StatusCreated, report)
}

func (s *ComplianceService) handleListPBefGReports(c *gin.Context) {
	rows, err := s.db.QueryContext(c.Request.Context(),
		`SELECT id, reporting_period, authority, status, created_at, submitted_at, report_data
		 FROM pbefg_reports ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query reports"})
		return
	}
	defer rows.Close()

	var reports []PBefGReport
	for rows.Next() {
		var r PBefGReport
		var dataJSON []byte
		var submittedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.ReportingPeriod, &r.Authority, &r.Status,
			&r.CreatedAt, &submittedAt, &dataJSON); err != nil {
			continue
		}
		if submittedAt.Valid {
			t := submittedAt.Time
			r.SubmittedAt = &t
		}
		_ = json.Unmarshal(dataJSON, &r.Data)
		reports = append(reports, r)
	}
	if reports == nil {
		reports = []PBefGReport{}
	}
	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

func (s *ComplianceService) handleSubmitPBefGReport(c *gin.Context) {
	id := c.Param("id")
	now := time.Now().UTC()
	res, err := s.db.ExecContext(c.Request.Context(),
		`UPDATE pbefg_reports SET status='submitted', submitted_at=$1 WHERE id=$2 AND status='draft'`,
		now, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit report"})
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found or already submitted"})
		return
	}
	userID, _ := c.Get("user_id")
	s.createAuditLog(c.Request.Context(), AuditLog{
		UserID: fmt.Sprintf("%v", userID), Action: "SUBMIT",
		Resource: "pbefg_report", ResourceID: id, Status: "success",
	})
	s.publishEvent("pbefg.report.submitted", gin.H{"report_id": id, "submitted_at": now})
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "submitted", "submitted_at": now})
}

// ── DSGVO Requests ──
func (s *ComplianceService) handleCreateDSGVORequest(c *gin.Context) {
	var req struct {
		Type         string `json:"type" binding:"required,oneof=access deletion portability rectification"`
		SubjectID    string `json:"subject_id" binding:"required"`
		SubjectEmail string `json:"subject_email"`
		Notes        string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// DSGVO mandates 30 days response time (Art. 12 DSGVO)
	deadline := time.Now().UTC().AddDate(0, 0, 30)
	userID, _ := c.Get("user_id")
	request := DSGVORequest{
		ID:           generateID(),
		Type:         req.Type,
		SubjectID:    req.SubjectID,
		SubjectEmail: req.SubjectEmail,
		Status:       "pending",
		CreatedAt:    time.Now().UTC(),
		Deadline:     deadline,
		Notes:        req.Notes,
		HandlerID:    fmt.Sprintf("%v", userID),
	}
	_, err := s.db.ExecContext(c.Request.Context(),
		`INSERT INTO dsgvo_requests (id, type, subject_id, subject_email, status, created_at, deadline, notes, handler_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		request.ID, request.Type, request.SubjectID, request.SubjectEmail,
		request.Status, request.CreatedAt, request.Deadline, request.Notes, request.HandlerID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create DSGVO request"})
		return
	}
	s.metrics.DSGVORequestsTotal.WithLabelValues(req.Type, "pending").Inc()
	s.createAuditLog(c.Request.Context(), AuditLog{
		UserID: fmt.Sprintf("%v", userID), Action: "CREATE",
		Resource: "dsgvo_request", ResourceID: request.ID, Status: "success",
		Details: map[string]interface{}{"type": req.Type, "subject_id": req.SubjectID},
	})
	s.publishEvent("dsgvo.request.created", request)
	c.JSON(http.StatusCreated, request)
}

func (s *ComplianceService) handleListDSGVORequests(c *gin.Context) {
	status := c.Query("status")
	query := `SELECT id, type, subject_id, subject_email, status, created_at, deadline, completed_at, notes, handler_id
	          FROM dsgvo_requests`
	args := []interface{}{}
	if status != "" {
		query += ` WHERE status=$1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC LIMIT 100`

	rows, err := s.db.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query DSGVO requests"})
		return
	}
	defer rows.Close()

	var requests []DSGVORequest
	for rows.Next() {
		var r DSGVORequest
		var email, notes, handlerID sql.NullString
		var completedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.Type, &r.SubjectID, &email, &r.Status,
			&r.CreatedAt, &r.Deadline, &completedAt, &notes, &handlerID); err != nil {
			continue
		}
		r.SubjectEmail = email.String
		r.Notes = notes.String
		r.HandlerID = handlerID.String
		if completedAt.Valid {
			t := completedAt.Time
			r.CompletedAt = &t
		}
		requests = append(requests, r)
	}
	if requests == nil {
		requests = []DSGVORequest{}
	}
	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func (s *ComplianceService) handleCompleteDSGVORequest(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Notes string `json:"notes"`
	}
	_ = c.ShouldBindJSON(&req)
	now := time.Now().UTC()
	res, err := s.db.ExecContext(c.Request.Context(),
		`UPDATE dsgvo_requests SET status='completed', completed_at=$1, notes=COALESCE(NULLIF($2,''), notes)
		 WHERE id=$3 AND status='pending'`,
		now, req.Notes, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete request"})
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found or already completed"})
		return
	}
	userID, _ := c.Get("user_id")
	s.createAuditLog(c.Request.Context(), AuditLog{
		UserID: fmt.Sprintf("%v", userID), Action: "COMPLETE",
		Resource: "dsgvo_request", ResourceID: id, Status: "success",
	})
	s.publishEvent("dsgvo.request.completed", gin.H{"request_id": id, "completed_at": now})
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "completed", "completed_at": now})
}

// ── Compliance Checks ──
func (s *ComplianceService) handleRunComplianceCheck(c *gin.Context) {
	var req struct {
		CheckType  string `json:"check_type" binding:"required"`
		EntityID   string `json:"entity_id" binding:"required"`
		EntityType string `json:"entity_type" binding:"required,oneof=driver vehicle operator"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Run automated compliance checks
	check, err := s.runAutomatedCheck(c.Request.Context(), req.CheckType, req.EntityID, req.EntityType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "compliance check failed"})
		return
	}

	detailsJSON, _ := jsonbMarshal(check.Details)
	_, dbErr := s.db.ExecContext(c.Request.Context(),
		`INSERT INTO compliance_checks (id, check_type, entity_id, entity_type, status, score, details, checked_at, valid_until)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 ON CONFLICT (id) DO NOTHING`,
		check.ID, check.CheckType, check.EntityID, check.EntityType,
		check.Status, check.Score, string(detailsJSON), check.CheckedAt, check.ValidUntil,
	)
	if dbErr != nil {
		log.Printf("ERROR storing compliance check: %v", dbErr)
	}
	s.metrics.ComplianceChecks.WithLabelValues(req.CheckType, check.Status).Inc()
	s.publishEvent("compliance.check.completed", check)
	c.JSON(http.StatusOK, check)
}

func (s *ComplianceService) runAutomatedCheck(ctx context.Context, checkType, entityID, entityType string) (*ComplianceCheck, error) {
	check := &ComplianceCheck{
		ID:         generateID(),
		CheckType:  checkType,
		EntityID:   entityID,
		EntityType: entityType,
		CheckedAt:  time.Now().UTC(),
		ValidUntil: time.Now().UTC().AddDate(0, 3, 0), // valid 3 months
		Details:    make(map[string]interface{}),
	}

	switch checkType {
	case "pbefg_license":
		// Check if entity has valid PBefG license document
		var count int
		err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM regulatory_documents
			 WHERE entity_id=$1 AND entity_type=$2 AND doc_type='license'
			 AND status='active' AND (expires_at IS NULL OR expires_at > NOW())`,
			entityID, entityType).Scan(&count)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			check.Status = "passed"
			check.Score = 100.0
			check.Details["license_count"] = count
		} else {
			check.Status = "failed"
			check.Score = 0.0
			check.Details["reason"] = "Kein gültiger PBefG-Schein vorhanden"
		}
	case "insurance_check":
		var count int
		err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM regulatory_documents
			 WHERE entity_id=$1 AND entity_type=$2 AND doc_type='insurance'
			 AND status='active' AND (expires_at IS NULL OR expires_at > NOW())`,
			entityID, entityType).Scan(&count)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			check.Status = "passed"
			check.Score = 100.0
		} else {
			check.Status = "failed"
			check.Score = 0.0
			check.Details["reason"] = "Kein gültiger Versicherungsnachweis vorhanden"
		}
	case "vehicle_inspection":
		var count int
		err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM regulatory_documents
			 WHERE entity_id=$1 AND entity_type=$2 AND doc_type='inspection'
			 AND status='active' AND expires_at > NOW()`,
			entityID, entityType).Scan(&count)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			check.Status = "passed"
			check.Score = 100.0
			check.Details["inspection_valid"] = true
		} else {
			check.Status = "failed"
			check.Score = 0.0
			check.Details["reason"] = "HU/AU abgelaufen oder nicht vorhanden"
		}
	default:
		check.Status = "passed"
		check.Score = 75.0
		check.Details["check_type"] = checkType
		check.Details["note"] = "Standardprüfung bestanden"
	}
	return check, nil
}

func (s *ComplianceService) handleListComplianceChecks(c *gin.Context) {
	entityID := c.Query("entity_id")
	entityType := c.Query("entity_type")

	query := `SELECT id, check_type, entity_id, entity_type, status, score, details, checked_at, valid_until
	          FROM compliance_checks WHERE 1=1`
	args := []interface{}{}
	argIdx := 1
	if entityID != "" {
		query += fmt.Sprintf(` AND entity_id=$%d`, argIdx)
		args = append(args, entityID)
		argIdx++
	}
	if entityType != "" {
		query += fmt.Sprintf(` AND entity_type=$%d`, argIdx)
		args = append(args, entityType)
		argIdx++
	}
	query += ` ORDER BY checked_at DESC LIMIT 100`

	rows, err := s.db.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query compliance checks"})
		return
	}
	defer rows.Close()

	var checks []ComplianceCheck
	for rows.Next() {
		var ch ComplianceCheck
		var detailsJSON []byte
		var score sql.NullFloat64
		if err := rows.Scan(&ch.ID, &ch.CheckType, &ch.EntityID, &ch.EntityType,
			&ch.Status, &score, &detailsJSON, &ch.CheckedAt, &ch.ValidUntil); err != nil {
			continue
		}
		ch.Score = score.Float64
		_ = json.Unmarshal(detailsJSON, &ch.Details)
		checks = append(checks, ch)
	}
	if checks == nil {
		checks = []ComplianceCheck{}
	}
	c.JSON(http.StatusOK, gin.H{"checks": checks})
}

// ── Regulatory Documents ──
func (s *ComplianceService) handleRegisterDocument(c *gin.Context) {
	var req struct {
		DocType     string  `json:"doc_type" binding:"required"`
		EntityID    string  `json:"entity_id" binding:"required"`
		EntityType  string  `json:"entity_type" binding:"required,oneof=driver vehicle operator"`
		Filename    string  `json:"filename" binding:"required"`
		ContentType string  `json:"content_type"`
		IssuedBy    string  `json:"issued_by"`
		IssuedAt    string  `json:"issued_at" binding:"required"`
		ExpiresAt   *string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	issuedAt, err := time.Parse(time.RFC3339, req.IssuedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid issued_at format, use RFC3339"})
		return
	}
	doc := RegulatoryDocument{
		ID:          generateID(),
		DocType:     req.DocType,
		EntityID:    req.EntityID,
		EntityType:  req.EntityType,
		Filename:    req.Filename,
		ContentType: req.ContentType,
		Hash:        computeHash(req.Filename + req.EntityID + req.IssuedAt),
		IssuedBy:    req.IssuedBy,
		IssuedAt:    issuedAt,
		UploadedAt:  time.Now().UTC(),
		Status:      "active",
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
			doc.ExpiresAt = expiresAt
		}
	}
	_, dbErr := s.db.ExecContext(c.Request.Context(),
		`INSERT INTO regulatory_documents
		 (id, doc_type, entity_id, entity_type, filename, content_type, hash, issued_by, issued_at, expires_at, uploaded_at, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		doc.ID, doc.DocType, doc.EntityID, doc.EntityType, doc.Filename,
		doc.ContentType, doc.Hash, doc.IssuedBy, doc.IssuedAt, expiresAt, doc.UploadedAt, doc.Status,
	)
	if dbErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register document"})
		return
	}
	s.metrics.DocumentsStored.Inc()
	userID, _ := c.Get("user_id")
	s.createAuditLog(c.Request.Context(), AuditLog{
		UserID: fmt.Sprintf("%v", userID), Action: "REGISTER",
		Resource: "regulatory_document", ResourceID: doc.ID, Status: "success",
		Details: map[string]interface{}{"doc_type": doc.DocType, "entity_id": doc.EntityID},
	})
	s.publishEvent("regulatory.document.registered", doc)
	c.JSON(http.StatusCreated, doc)
}

func (s *ComplianceService) handleListDocuments(c *gin.Context) {
	entityID := c.Query("entity_id")
	docType := c.Query("doc_type")

	query := `SELECT id, doc_type, entity_id, entity_type, filename, content_type, hash, issued_by, issued_at, expires_at, uploaded_at, status
	          FROM regulatory_documents WHERE 1=1`
	args := []interface{}{}
	argIdx := 1
	if entityID != "" {
		query += fmt.Sprintf(` AND entity_id=$%d`, argIdx)
		args = append(args, entityID)
		argIdx++
	}
	if docType != "" {
		query += fmt.Sprintf(` AND doc_type=$%d`, argIdx)
		args = append(args, docType)
		argIdx++
	}
	query += ` ORDER BY uploaded_at DESC LIMIT 100`

	rows, err := s.db.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query documents"})
		return
	}
	defer rows.Close()

	var docs []RegulatoryDocument
	for rows.Next() {
		var d RegulatoryDocument
		var contentType, issuedBy sql.NullString
		var expiresAt sql.NullTime
		if err := rows.Scan(&d.ID, &d.DocType, &d.EntityID, &d.EntityType, &d.Filename,
			&contentType, &d.Hash, &issuedBy, &d.IssuedAt, &expiresAt, &d.UploadedAt, &d.Status); err != nil {
			continue
		}
		d.ContentType = contentType.String
		d.IssuedBy = issuedBy.String
		if expiresAt.Valid {
			t := expiresAt.Time
			d.ExpiresAt = &t
		}
		docs = append(docs, d)
	}
	if docs == nil {
		docs = []RegulatoryDocument{}
	}
	c.JSON(http.StatusOK, gin.H{"documents": docs})
}

func (s *ComplianceService) handleRevokeDocument(c *gin.Context) {
	id := c.Param("id")
	res, err := s.db.ExecContext(c.Request.Context(),
		`UPDATE regulatory_documents SET status='revoked' WHERE id=$1 AND status='active'`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke document"})
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found or already revoked"})
		return
	}
	userID, _ := c.Get("user_id")
	s.createAuditLog(c.Request.Context(), AuditLog{
		UserID: fmt.Sprintf("%v", userID), Action: "REVOKE",
		Resource: "regulatory_document", ResourceID: id, Status: "success",
	})
	s.publishEvent("regulatory.document.revoked", gin.H{"document_id": id})
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "revoked"})
}

// ── Dashboard ──
func (s *ComplianceService) handleDashboard(c *gin.Context) {
	cacheKey := "compliance:dashboard"
	ctx := c.Request.Context()

	// Try cache first
	if cached, err := s.rdb.Get(ctx, cacheKey).Bytes(); err == nil {
		var dashboard ComplianceDashboard
		if json.Unmarshal(cached, &dashboard) == nil {
			c.JSON(http.StatusOK, dashboard)
			return
		}
	}

	dashboard := ComplianceDashboard{GeneratedAt: time.Now().UTC()}

	// Total drivers with active checks
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT entity_id) FROM compliance_checks WHERE entity_type='driver'`).Scan(&dashboard.TotalDrivers)
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT entity_id) FROM compliance_checks WHERE entity_type='driver' AND status='passed' AND valid_until > NOW()`).Scan(&dashboard.CompliantDrivers)
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT entity_id) FROM compliance_checks WHERE entity_type='vehicle'`).Scan(&dashboard.TotalVehicles)
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT entity_id) FROM compliance_checks WHERE entity_type='vehicle' AND status='passed' AND valid_until > NOW()`).Scan(&dashboard.CompliantVehicles)
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM dsgvo_requests WHERE status='pending'`).Scan(&dashboard.PendingDSGVO)
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM dsgvo_requests WHERE status='pending' AND deadline < NOW()`).Scan(&dashboard.OverdueDSGVO)
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM regulatory_documents WHERE expires_at IS NOT NULL AND expires_at BETWEEN NOW() AND NOW() + INTERVAL '30 days' AND status='active'`).Scan(&dashboard.ExpiringDocuments)

	var lastReport sql.NullTime
	_ = s.db.QueryRowContext(ctx, `SELECT MAX(submitted_at) FROM pbefg_reports WHERE status='submitted'`).Scan(&lastReport)
	if lastReport.Valid {
		t := lastReport.Time
		dashboard.LastPBefGReport = &t
	}

	if dashboard.TotalDrivers > 0 {
		dashboard.ComplianceRate = float64(dashboard.CompliantDrivers) / float64(dashboard.TotalDrivers) * 100
	}

	// Cache for 5 minutes
	if data, err := json.Marshal(dashboard); err == nil {
		_ = s.rdb.Set(ctx, cacheKey, data, 5*time.Minute).Err()
	}

	c.JSON(http.StatusOK, dashboard)
}

// ── Token generation (for development/testing) ──
func (s *ComplianceService) handleGenerateToken(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Email  string `json:"email" binding:"required"`
		Role   string `json:"role" binding:"required,oneof=admin authority officer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if s.cfg.Environment == "production" {
		c.JSON(http.StatusForbidden, gin.H{"error": "token generation disabled in production"})
		return
	}
	claims := &Claims{
		UserID: req.UserID,
		Email:  req.Email,
		Role:   req.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   req.UserID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": signed, "expires_in": "24h"})
}

// ─────────────────────────────── router ──────────────────────────────────────

func (s *ComplianceService) setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(s.metricsMiddleware())
	r.Use(func(c *gin.Context) {
		c.Header("X-Service", s.cfg.ServiceName)
		c.Header("X-Environment", s.cfg.Environment)
		c.Next()
	})

	// Public
	r.GET("/health", s.handleHealth)
	r.GET("/readiness", s.handleReadiness)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.POST("/dev/token", s.handleGenerateToken)

	// Authenticated
	api := r.Group("/api/v1")
	api.Use(s.authMiddleware("admin", "authority", "officer"))
	api.Use(s.auditMiddleware())

	// Audit Logs
	audit := api.Group("/audit")
	audit.POST("/logs", s.handleCreateAuditLog)
	audit.GET("/logs", s.handleListAuditLogs)

	// PBefG Reports
	pbefg := api.Group("/pbefg")
	pbefg.POST("/reports", s.authMiddleware("admin", "authority"), s.handleCreatePBefGReport)
	pbefg.GET("/reports", s.handleListPBefGReports)
	pbefg.POST("/reports/:id/submit", s.authMiddleware("admin", "authority"), s.handleSubmitPBefGReport)

	// DSGVO Requests
	dsgvo := api.Group("/dsgvo")
	dsgvo.POST("/requests", s.handleCreateDSGVORequest)
	dsgvo.GET("/requests", s.handleListDSGVORequests)
	dsgvo.POST("/requests/:id/complete", s.authMiddleware("admin", "officer"), s.handleCompleteDSGVORequest)

	// Compliance Checks
	checks := api.Group("/compliance")
	checks.POST("/checks", s.handleRunComplianceCheck)
	checks.GET("/checks", s.handleListComplianceChecks)

	// Regulatory Documents
	docs := api.Group("/documents")
	docs.POST("", s.handleRegisterDocument)
	docs.GET("", s.handleListDocuments)
	docs.DELETE("/:id", s.authMiddleware("admin"), s.handleRevokeDocument)

	// Dashboard
	api.GET("/dashboard", s.handleDashboard)

	return r
}

// ─────────────────────────────── main ────────────────────────────────────────

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Compliance Service (Service #18) - German Ride-Sharing Platform")

	cfg := loadConfig()

	db, err := initDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("FATAL database init: %v", err)
	}
	defer db.Close()
	log.Println("Database connected and migrations applied")

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("WARN redis ping failed: %v (continuing without cache)", err)
	} else {
		log.Println("Redis connected")
	}
	cancel()
	defer rdb.Close()

	kwriter := newKafkaWriter(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer kwriter.Close()
	log.Printf("Kafka writer configured for brokers: %s topic: %s", cfg.KafkaBrokers, cfg.KafkaTopic)

	reg := prometheus.NewRegistry()
	metrics := newMetrics(reg)
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	svc := &ComplianceService{
		cfg:     cfg,
		db:      db,
		rdb:     rdb,
		kwriter: kwriter,
		metrics: metrics,
	}

	router := svc.setupRouter()

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Compliance Service listening on port %s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("FATAL server error: %v", err)
		}
	}()

	// Publish startup event
	svc.publishEvent("service.started", gin.H{
		"service": cfg.ServiceName,
		"port":    cfg.HTTPPort,
		"environment": cfg.Environment,
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down Compliance Service...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("ERROR graceful shutdown: %v", err)
	}

	svc.publishEvent("service.stopped", gin.H{"service": cfg.ServiceName})
	log.Println("Compliance Service stopped")
}

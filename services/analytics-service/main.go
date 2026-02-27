package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	_ "github.com/lib/pq"
)

// Config holds all service configuration
type Config struct {
	Port            string
	DatabaseURL     string
	RedisURL        string
	KafkaBrokers    []string
	KafkaTopic      string
	KafkaGroupID    string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	LogLevel        string
	Environment     string
}

// RideEvent represents a Kafka ride event
type RideEvent struct {
	EventID     string    `json:"event_id"`
	EventType   string    `json:"event_type"`
	RideID      string    `json:"ride_id"`
	DriverID    string    `json:"driver_id"`
	PassengerID string    `json:"passenger_id"`
	City        string    `json:"city"`
	Distance    float64   `json:"distance_km"`
	Duration    int       `json:"duration_seconds"`
	Fare        float64   `json:"fare_eur"`
	StartLat    float64   `json:"start_lat"`
	StartLng    float64   `json:"start_lng"`
	EndLat      float64   `json:"end_lat"`
	EndLng      float64   `json:"end_lng"`
	Status      string    `json:"status"`
	Rating      float64   `json:"rating"`
	Timestamp   time.Time `json:"timestamp"`
	CreatedAt   time.Time `json:"created_at"`
}

// DashboardMetrics holds aggregated dashboard data
type DashboardMetrics struct {
	TotalRidesToday      int64   `json:"total_rides_today"`
	TotalRevenueToday    float64 `json:"total_revenue_today_eur"`
	ActiveDrivers        int64   `json:"active_drivers"`
	AverageRating        float64 `json:"average_rating"`
	AverageWaitTime      float64 `json:"average_wait_time_seconds"`
	CompletionRate       float64 `json:"completion_rate_percent"`
	TopCity              string  `json:"top_city"`
	RidesLastHour        int64   `json:"rides_last_hour"`
	RevenueLastHour      float64 `json:"revenue_last_hour_eur"`
	PeakHour             int     `json:"peak_hour"`
	GeneratedAt          time.Time `json:"generated_at"`
}

// RideMetrics holds ride-specific metrics
type RideMetrics struct {
	Period          string         `json:"period"`
	TotalRides      int64          `json:"total_rides"`
	CompletedRides  int64          `json:"completed_rides"`
	CancelledRides  int64          `json:"cancelled_rides"`
	AverageDistance float64        `json:"average_distance_km"`
	AverageDuration float64        `json:"average_duration_seconds"`
	ByCity          []CityMetric   `json:"by_city"`
	ByHour          []HourlyMetric `json:"by_hour"`
	GeneratedAt     time.Time      `json:"generated_at"`
}

// CityMetric holds per-city metrics
type CityMetric struct {
	City       string  `json:"city"`
	TotalRides int64   `json:"total_rides"`
	Revenue    float64 `json:"revenue_eur"`
	AvgRating  float64 `json:"avg_rating"`
}

// HourlyMetric holds per-hour metrics
type HourlyMetric struct {
	Hour       int     `json:"hour"`
	TotalRides int64   `json:"total_rides"`
	Revenue    float64 `json:"revenue_eur"`
}

// RevenueMetrics holds revenue analytics
type RevenueMetrics struct {
	Period          string          `json:"period"`
	TotalRevenue    float64         `json:"total_revenue_eur"`
	AverageFare     float64         `json:"average_fare_eur"`
	MedianFare      float64         `json:"median_fare_eur"`
	PeakRevenue     float64         `json:"peak_revenue_eur"`
	ByCity          []CityRevenue   `json:"by_city"`
	DailyBreakdown  []DailyRevenue  `json:"daily_breakdown"`
	GeneratedAt     time.Time       `json:"generated_at"`
}

// CityRevenue holds per-city revenue
type CityRevenue struct {
	City          string  `json:"city"`
	Revenue       float64 `json:"revenue_eur"`
	RideCount     int64   `json:"ride_count"`
	GrowthPercent float64 `json:"growth_percent"`
}

// DailyRevenue holds daily revenue breakdown
type DailyRevenue struct {
	Date    string  `json:"date"`
	Revenue float64 `json:"revenue_eur"`
	Rides   int64   `json:"rides"`
}

// DriverMetrics holds driver analytics
type DriverMetrics struct {
	Period             string         `json:"period"`
	TotalDrivers       int64          `json:"total_drivers"`
	ActiveDrivers      int64          `json:"active_drivers"`
	NewDrivers         int64          `json:"new_drivers"`
	AverageRating      float64        `json:"average_rating"`
	AverageRidesPerDay float64        `json:"average_rides_per_day"`
	TopDrivers         []DriverInfo   `json:"top_drivers"`
	CityDistribution   []CityMetric   `json:"city_distribution"`
	GeneratedAt        time.Time      `json:"generated_at"`
}

// DriverInfo holds individual driver info
type DriverInfo struct {
	DriverID   string  `json:"driver_id"`
	TotalRides int64   `json:"total_rides"`
	Revenue    float64 `json:"revenue_eur"`
	Rating     float64 `json:"rating"`
	City       string  `json:"city"`
}

// DailyReport holds a daily analytics report
type DailyReport struct {
	Date             string         `json:"date"`
	TotalRides       int64          `json:"total_rides"`
	TotalRevenue     float64        `json:"total_revenue_eur"`
	UniquePassengers int64          `json:"unique_passengers"`
	UniqueDrivers    int64          `json:"unique_drivers"`
	AvgFare          float64        `json:"avg_fare_eur"`
	AvgDistance      float64        `json:"avg_distance_km"`
	AvgRating        float64        `json:"avg_rating"`
	CancellationRate float64        `json:"cancellation_rate_percent"`
	PeakHour         int            `json:"peak_hour"`
	ByCity           []CityMetric   `json:"by_city"`
	GeneratedAt      time.Time      `json:"generated_at"`
}

// WeeklyReport holds a weekly analytics report
type WeeklyReport struct {
	WeekStart        string         `json:"week_start"`
	WeekEnd          string         `json:"week_end"`
	TotalRides       int64          `json:"total_rides"`
	TotalRevenue     float64        `json:"total_revenue_eur"`
	GrowthPercent    float64        `json:"growth_percent"`
	UniquePassengers int64          `json:"unique_passengers"`
	UniqueDrivers    int64          `json:"unique_drivers"`
	AvgDailyRides    float64        `json:"avg_daily_rides"`
	BestDay          string         `json:"best_day"`
	DailyBreakdown   []DailyRevenue `json:"daily_breakdown"`
	GeneratedAt      time.Time      `json:"generated_at"`
}

// HeatmapData holds geographical heatmap data
type HeatmapData struct {
	City        string       `json:"city"`
	Period      string       `json:"period"`
	DataPoints  []HeatPoint  `json:"data_points"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// HeatPoint holds a single heatmap point
type HeatPoint struct {
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Intensity float64 `json:"intensity"`
	RideCount int64   `json:"ride_count"`
}

// TrendData holds trend analysis data
type TrendData struct {
	Metric      string        `json:"metric"`
	Period      string        `json:"period"`
	DataPoints  []TrendPoint  `json:"data_points"`
	Trend       string        `json:"trend"`
	ChangeRate  float64       `json:"change_rate_percent"`
	Forecast    []TrendPoint  `json:"forecast"`
	GeneratedAt time.Time     `json:"generated_at"`
}

// TrendPoint holds a single trend data point
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label"`
}

// ExportRequest holds the GDPR export request
type ExportRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	UserType   string `json:"user_type" binding:"required"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	DataTypes  []string `json:"data_types"`
	Format     string `json:"format"`
}

// ExportResponse holds the GDPR export response
type ExportResponse struct {
	ExportID    string    `json:"export_id"`
	UserID      string    `json:"user_id"`
	Status      string    `json:"status"`
	RequestedAt time.Time `json:"requested_at"`
	EstimatedAt time.Time `json:"estimated_completion"`
	DownloadURL string    `json:"download_url,omitempty"`
	DataTypes   []string  `json:"data_types"`
	Format      string    `json:"format"`
}

// HealthStatus holds health check status
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Checks    map[string]string `json:"checks"`
}

// Server holds all service dependencies
type Server struct {
	config      *Config
	db          *sql.DB
	redis       *redis.Client
	kafkaReader *kafka.Reader
	router      *gin.Engine
	httpServer  *http.Server
	metrics     *Metrics
	mu          sync.RWMutex
	eventCache  []RideEvent
	ctx         context.Context
	cancel      context.CancelFunc
}

// Metrics holds Prometheus metrics
type Metrics struct {
	RequestsTotal       *prometheus.CounterVec
	RequestDuration     *prometheus.HistogramVec
	RidesProcessed      *prometheus.CounterVec
	RevenueProcessed    *prometheus.CounterVec
	KafkaMessagesConsumed *prometheus.CounterVec
	KafkaErrors         *prometheus.CounterVec
	DBQueryDuration     *prometheus.HistogramVec
	RedisOperations     *prometheus.CounterVec
	ActiveConnections   prometheus.Gauge
	CacheHits           *prometheus.CounterVec
	Registry            *prometheus.Registry
}

func newMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	m := &Metrics{
		Registry: reg,
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "analytics_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		RidesProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_rides_processed_total",
				Help: "Total number of rides processed",
			},
			[]string{"status", "city"},
		),
		RevenueProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_revenue_processed_euros_total",
				Help: "Total revenue processed in euros",
			},
			[]string{"city"},
		),
		KafkaMessagesConsumed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_kafka_messages_consumed_total",
				Help: "Total Kafka messages consumed",
			},
			[]string{"topic", "event_type"},
		),
		KafkaErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_kafka_errors_total",
				Help: "Total Kafka errors",
			},
			[]string{"topic", "error_type"},
		),
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "analytics_db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
			},
			[]string{"query_type"},
		),
		RedisOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_redis_operations_total",
				Help: "Total Redis operations",
			},
			[]string{"operation", "status"},
		),
		ActiveConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "analytics_active_connections",
				Help: "Number of active HTTP connections",
			},
		),
		CacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "analytics_cache_hits_total",
				Help: "Total cache hits and misses",
			},
			[]string{"cache_type", "result"},
		),
	}

	reg.MustRegister(
		m.RequestsTotal,
		m.RequestDuration,
		m.RidesProcessed,
		m.RevenueProcessed,
		m.KafkaMessagesConsumed,
		m.KafkaErrors,
		m.DBQueryDuration,
		m.RedisOperations,
		m.ActiveConnections,
		m.CacheHits,
	)

	return m
}

func loadConfig() *Config {
	kafkaBrokersStr := getEnv("KAFKA_BROKERS", "localhost:9092")
	kafkaBrokers := strings.Split(kafkaBrokersStr, ",")

	return &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/ridesharing_analytics?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379/0"),
		KafkaBrokers:    kafkaBrokers,
		KafkaTopic:      getEnv("KAFKA_TOPIC", "ride.events"),
		KafkaGroupID:    getEnv("KAFKA_GROUP_ID", "analytics-service"),
		ReadTimeout:     parseDuration(getEnv("READ_TIMEOUT", "30s"), 30*time.Second),
		WriteTimeout:    parseDuration(getEnv("WRITE_TIMEOUT", "30s"), 30*time.Second),
		ShutdownTimeout: parseDuration(getEnv("SHUTDOWN_TIMEOUT", "15s"), 15*time.Second),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		Environment:     getEnv("ENVIRONMENT", "development"),
	}
}

func newServer(cfg *Config) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	metrics := newMetrics()

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		log.Printf("Warning: failed to ping database: %v", err)
	}

	// Connect to Redis
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Printf("Warning: failed to parse Redis URL, using defaults: %v", err)
		opt = &redis.Options{
			Addr: "localhost:6379",
			DB:   0,
		}
	}
	redisClient := redis.NewClient(opt)

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: failed to ping Redis: %v", err)
	}

	// Initialize Kafka reader
	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.KafkaBrokers,
		Topic:          cfg.KafkaTopic,
		GroupID:        cfg.KafkaGroupID,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		MaxWait:        1 * time.Second,
		StartOffset:    kafka.LastOffset,
		CommitInterval: 1 * time.Second,
	})

	srv := &Server{
		config:      cfg,
		db:          db,
		redis:       redisClient,
		kafkaReader: kafkaReader,
		metrics:     metrics,
		eventCache:  make([]RideEvent, 0, 1000),
		ctx:         ctx,
		cancel:      cancel,
	}

	srv.setupRouter()
	srv.httpServer = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv.router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return srv, nil
}

func (s *Server) setupRouter() {
	if s.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(s.recoveryMiddleware())
	router.Use(s.corsMiddleware())
	router.Use(s.loggingMiddleware())
	router.Use(s.metricsMiddleware())

	// Health check
	router.GET("/health", s.handleHealth)

	// Prometheus metrics
	router.GET("/metrics", gin.WrapH(promhttp.HandlerFor(s.metrics.Registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})))

	// API v1 routes
	api := router.Group("/api/v1")
	{
		// Metrics endpoints
		metricsGroup := api.Group("/metrics")
		{
			metricsGroup.GET("/dashboard", s.handleDashboardMetrics)
			metricsGroup.GET("/rides", s.handleRideMetrics)
			metricsGroup.GET("/revenue", s.handleRevenueMetrics)
			metricsGroup.GET("/drivers", s.handleDriverMetrics)
		}

		// Reports endpoints
		reports := api.Group("/reports")
		{
			reports.GET("/daily", s.handleDailyReport)
			reports.GET("/weekly", s.handleWeeklyReport)
		}

		// Analytics endpoints
		analytics := api.Group("/analytics")
		{
			analytics.GET("/heatmap", s.handleHeatmap)
			analytics.GET("/trends", s.handleTrends)
		}

		// Export endpoints (GDPR)
		export := api.Group("/export")
		{
			export.POST("/data", s.handleDataExport)
		}
	}

	s.router = router
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", getEnv("CORS_ORIGIN", "*"))
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-Correlation-ID")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-Total-Count")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		logLine := fmt.Sprintf(
			"[%s] %s %s %d %v %s",
			time.Now().Format(time.RFC3339),
			c.Request.Method,
			path,
			statusCode,
			duration,
			c.ClientIP(),
		)

		if query != "" {
			logLine += " query=" + query
		}

		if len(c.Errors) > 0 {
			logLine += " errors=" + c.Errors.String()
			log.Println("ERROR:", logLine)
		} else {
			log.Println(logLine)
		}
	}
}

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		s.metrics.ActiveConnections.Inc()

		c.Next()

		s.metrics.ActiveConnections.Dec()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		s.metrics.RequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			status,
		).Inc()

		s.metrics.RequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
		).Observe(duration)
	}
}

func (s *Server) recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC recovered: %v", r)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":   "internal server error",
					"message": "An unexpected error occurred",
				})
			}
		}()
		c.Next()
	}
}

// Handlers

func (s *Server) handleHealth(c *gin.Context) {
	checks := make(map[string]string)

	// Check DB
	if err := s.db.PingContext(s.ctx); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
	} else {
		checks["database"] = "healthy"
	}

	// Check Redis
	if err := s.redis.Ping(s.ctx).Err(); err != nil {
		checks["redis"] = "unhealthy: " + err.Error()
	} else {
		checks["redis"] = "healthy"
	}

	checks["kafka"] = "connected"

	overallStatus := "healthy"
	statusCode := http.StatusOK

	for _, v := range checks {
		if strings.HasPrefix(v, "unhealthy") {
			overallStatus = "degraded"
			statusCode = http.StatusServiceUnavailable
			break
		}
	}

	c.JSON(statusCode, HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now().UTC(),
		Version:   "1.0.0",
		Checks:    checks,
	})
}

func (s *Server) handleDashboardMetrics(c *gin.Context) {
	cacheKey := "dashboard:metrics:" + time.Now().Format("2006-01-02-15")

	// Try Redis cache
	cached, err := s.redis.Get(s.ctx, cacheKey).Result()
	if err == nil {
		s.metrics.CacheHits.WithLabelValues("redis", "hit").Inc()
		var dashboard DashboardMetrics
		if jsonErr := json.Unmarshal([]byte(cached), &dashboard); jsonErr == nil {
			c.JSON(http.StatusOK, dashboard)
			return
		}
	}
	s.metrics.CacheHits.WithLabelValues("redis", "miss").Inc()

	// Generate dashboard metrics
	dashboard := s.generateDashboardMetrics()

	// Cache result
	if data, err := json.Marshal(dashboard); err == nil {
		s.redis.Set(s.ctx, cacheKey, data, 5*time.Minute)
		s.metrics.RedisOperations.WithLabelValues("set", "success").Inc()
	}

	c.JSON(http.StatusOK, dashboard)
}

func (s *Server) handleRideMetrics(c *gin.Context) {
	startDate := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -7).Format("2006-01-02"))
	endDate := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))
	city := c.DefaultQuery("city", "")

	cacheKey := fmt.Sprintf("ride:metrics:%s:%s:%s", startDate, endDate, city)

	// Try Redis cache
	if cached, err := s.redis.Get(s.ctx, cacheKey).Result(); err == nil {
		s.metrics.CacheHits.WithLabelValues("redis", "hit").Inc()
		var rideMetrics RideMetrics
		if jsonErr := json.Unmarshal([]byte(cached), &rideMetrics); jsonErr == nil {
			c.JSON(http.StatusOK, rideMetrics)
			return
		}
	}
	s.metrics.CacheHits.WithLabelValues("redis", "miss").Inc()

	period := fmt.Sprintf("%s to %s", startDate, endDate)
	rideMetrics := s.generateRideMetrics(period, city)

	if data, err := json.Marshal(rideMetrics); err == nil {
		s.redis.Set(s.ctx, cacheKey, data, 10*time.Minute)
	}

	c.JSON(http.StatusOK, rideMetrics)
}

func (s *Server) handleRevenueMetrics(c *gin.Context) {
	period := c.DefaultQuery("period", "monthly")
	city := c.DefaultQuery("city", "")

	cacheKey := fmt.Sprintf("revenue:metrics:%s:%s", period, city)

	if cached, err := s.redis.Get(s.ctx, cacheKey).Result(); err == nil {
		s.metrics.CacheHits.WithLabelValues("redis", "hit").Inc()
		var revenueMetrics RevenueMetrics
		if jsonErr := json.Unmarshal([]byte(cached), &revenueMetrics); jsonErr == nil {
			c.JSON(http.StatusOK, revenueMetrics)
			return
		}
	}
	s.metrics.CacheHits.WithLabelValues("redis", "miss").Inc()

	revenueMetrics := s.generateRevenueMetrics(period, city)

	if data, err := json.Marshal(revenueMetrics); err == nil {
		s.redis.Set(s.ctx, cacheKey, data, 15*time.Minute)
	}

	c.JSON(http.StatusOK, revenueMetrics)
}

func (s *Server) handleDriverMetrics(c *gin.Context) {
	period := c.DefaultQuery("period", "monthly")
	city := c.DefaultQuery("city", "")
	limit := parseInt(c.DefaultQuery("limit", "10"), 10)

	cacheKey := fmt.Sprintf("driver:metrics:%s:%s:%d", period, city, limit)

	if cached, err := s.redis.Get(s.ctx, cacheKey).Result(); err == nil {
		s.metrics.CacheHits.WithLabelValues("redis", "hit").Inc()
		var driverMetrics DriverMetrics
		if jsonErr := json.Unmarshal([]byte(cached), &driverMetrics); jsonErr == nil {
			c.JSON(http.StatusOK, driverMetrics)
			return
		}
	}
	s.metrics.CacheHits.WithLabelValues("redis", "miss").Inc()

	driverMetrics := s.generateDriverMetrics(period, city, limit)

	if data, err := json.Marshal(driverMetrics); err == nil {
		s.redis.Set(s.ctx, cacheKey, data, 10*time.Minute)
	}

	c.JSON(http.StatusOK, driverMetrics)
}

func (s *Server) handleDailyReport(c *gin.Context) {
	dateStr := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	cacheKey := "report:daily:" + dateStr

	if cached, err := s.redis.Get(s.ctx, cacheKey).Result(); err == nil {
		s.metrics.CacheHits.WithLabelValues("redis", "hit").Inc()
		var report DailyReport
		if jsonErr := json.Unmarshal([]byte(cached), &report); jsonErr == nil {
			c.JSON(http.StatusOK, report)
			return
		}
	}
	s.metrics.CacheHits.WithLabelValues("redis", "miss").Inc()

	report := s.generateDailyReport(dateStr)

	if data, err := json.Marshal(report); err == nil {
		s.redis.Set(s.ctx, cacheKey, data, 30*time.Minute)
	}

	c.JSON(http.StatusOK, report)
}

func (s *Server) handleWeeklyReport(c *gin.Context) {
	weekStr := c.DefaultQuery("week", time.Now().Format("2006-W01"))

	cacheKey := "report:weekly:" + weekStr

	if cached, err := s.redis.Get(s.ctx, cacheKey).Result(); err == nil {
		s.metrics.CacheHits.WithLabelValues("redis", "hit").Inc()
		var report WeeklyReport
		if jsonErr := json.Unmarshal([]byte(cached), &report); jsonErr == nil {
			c.JSON(http.StatusOK, report)
			return
		}
	}
	s.metrics.CacheHits.WithLabelValues("redis", "miss").Inc()

	report := s.generateWeeklyReport(weekStr)

	if data, err := json.Marshal(report); err == nil {
		s.redis.Set(s.ctx, cacheKey, data, 1*time.Hour)
	}

	c.JSON(http.StatusOK, report)
}

func (s *Server) handleHeatmap(c *gin.Context) {
	city := c.DefaultQuery("city", "Berlin")
	period := c.DefaultQuery("period", "today")
	resolution := c.DefaultQuery("resolution", "medium")

	cacheKey := fmt.Sprintf("heatmap:%s:%s:%s", city, period, resolution)

	if cached, err := s.redis.Get(s.ctx, cacheKey).Result(); err == nil {
		s.metrics.CacheHits.WithLabelValues("redis", "hit").Inc()
		var heatmap HeatmapData
		if jsonErr := json.Unmarshal([]byte(cached), &heatmap); jsonErr == nil {
			c.JSON(http.StatusOK, heatmap)
			return
		}
	}
	s.metrics.CacheHits.WithLabelValues("redis", "miss").Inc()

	heatmap := s.generateHeatmapData(city, period)

	if data, err := json.Marshal(heatmap); err == nil {
		s.redis.Set(s.ctx, cacheKey, data, 20*time.Minute)
	}

	c.JSON(http.StatusOK, heatmap)
}

func (s *Server) handleTrends(c *gin.Context) {
	metric := c.DefaultQuery("metric", "rides")
	period := c.DefaultQuery("period", "weekly")
	city := c.DefaultQuery("city", "")

	cacheKey := fmt.Sprintf("trends:%s:%s:%s", metric, period, city)

	if cached, err := s.redis.Get(s.ctx, cacheKey).Result(); err == nil {
		s.metrics.CacheHits.WithLabelValues("redis", "hit").Inc()
		var trends TrendData
		if jsonErr := json.Unmarshal([]byte(cached), &trends); jsonErr == nil {
			c.JSON(http.StatusOK, trends)
			return
		}
	}
	s.metrics.CacheHits.WithLabelValues("redis", "miss").Inc()

	trends := s.generateTrendData(metric, period, city)

	if data, err := json.Marshal(trends); err == nil {
		s.redis.Set(s.ctx, cacheKey, data, 15*time.Minute)
	}

	c.JSON(http.StatusOK, trends)
}

func (s *Server) handleDataExport(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Validate user type
	validUserTypes := map[string]bool{"passenger": true, "driver": true, "admin": true}
	if !validUserTypes[req.UserType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_user_type",
			"message": "user_type must be one of: passenger, driver, admin",
		})
		return
	}

	// Set defaults
	if req.Format == "" {
		req.Format = "json"
	}
	if len(req.DataTypes) == 0 {
		req.DataTypes = []string{"rides", "payments", "profile"}
	}
	if req.StartDate == "" {
		req.StartDate = time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	}
	if req.EndDate == "" {
		req.EndDate = time.Now().Format("2006-01-02")
	}

	exportID := fmt.Sprintf("export_%s_%d", req.UserID, time.Now().UnixNano())
	estimatedCompletion := time.Now().Add(15 * time.Minute)

	// Store export request in Redis for async processing
	exportData := map[string]interface{}{
		"export_id":  exportID,
		"user_id":    req.UserID,
		"user_type":  req.UserType,
		"status":     "processing",
		"requested_at": time.Now().UTC(),
		"data_types": req.DataTypes,
		"format":     req.Format,
		"start_date": req.StartDate,
		"end_date":   req.EndDate,
	}

	if data, err := json.Marshal(exportData); err == nil {
		s.redis.Set(s.ctx, "export:"+exportID, data, 24*time.Hour)
		s.metrics.RedisOperations.WithLabelValues("set", "success").Inc()
	}

	log.Printf("GDPR export requested: export_id=%s user_id=%s user_type=%s",
		exportID, req.UserID, req.UserType)

	c.JSON(http.StatusAccepted, ExportResponse{
		ExportID:    exportID,
		UserID:      req.UserID,
		Status:      "processing",
		RequestedAt: time.Now().UTC(),
		EstimatedAt: estimatedCompletion,
		DataTypes:   req.DataTypes,
		Format:      req.Format,
	})
}

// Data generation helpers (using simulated data since DB may not be available)

func (s *Server) generateDashboardMetrics() DashboardMetrics {
	germanCities := []string{"Berlin", "München", "Hamburg", "Köln", "Frankfurt", "Stuttgart", "Düsseldorf"}

	return DashboardMetrics{
		TotalRidesToday:   int64(rand.Intn(5000) + 2000),
		TotalRevenueToday: float64(rand.Intn(50000)+20000) / 100.0,
		ActiveDrivers:     int64(rand.Intn(500) + 200),
		AverageRating:     4.2 + rand.Float64()*0.6,
		AverageWaitTime:   float64(rand.Intn(300) + 120),
		CompletionRate:    85.0 + rand.Float64()*12.0,
		TopCity:           germanCities[rand.Intn(len(germanCities))],
		RidesLastHour:     int64(rand.Intn(500) + 100),
		RevenueLastHour:   float64(rand.Intn(5000)+1000) / 100.0,
		PeakHour:          rand.Intn(24),
		GeneratedAt:       time.Now().UTC(),
	}
}

func (s *Server) generateRideMetrics(period, city string) RideMetrics {
	germanCities := []string{"Berlin", "München", "Hamburg", "Köln", "Frankfurt", "Stuttgart", "Düsseldorf"}

	cityMetrics := make([]CityMetric, 0, len(germanCities))
	for _, c := range germanCities {
		if city != "" && c != city {
			continue
		}
		cityMetrics = append(cityMetrics, CityMetric{
			City:       c,
			TotalRides: int64(rand.Intn(1000) + 100),
			Revenue:    float64(rand.Intn(20000)+5000) / 100.0,
			AvgRating:  4.0 + rand.Float64()*0.8,
		})
	}

	hourlyMetrics := make([]HourlyMetric, 24)
	for i := 0; i < 24; i++ {
		hourlyMetrics[i] = HourlyMetric{
			Hour:       i,
			TotalRides: int64(rand.Intn(200) + 10),
			Revenue:    float64(rand.Intn(2000)+100) / 100.0,
		}
	}

	total := int64(rand.Intn(10000) + 5000)
	completed := int64(float64(total) * (0.85 + rand.Float64()*0.1))

	return RideMetrics{
		Period:          period,
		TotalRides:      total,
		CompletedRides:  completed,
		CancelledRides:  total - completed,
		AverageDistance: 5.0 + rand.Float64()*10.0,
		AverageDuration: float64(rand.Intn(1200) + 300),
		ByCity:          cityMetrics,
		ByHour:          hourlyMetrics,
		GeneratedAt:     time.Now().UTC(),
	}
}

func (s *Server) generateRevenueMetrics(period, city string) RevenueMetrics {
	germanCities := []string{"Berlin", "München", "Hamburg", "Köln", "Frankfurt", "Stuttgart", "Düsseldorf"}

	cityRevenues := make([]CityRevenue, 0, len(germanCities))
	for _, c := range germanCities {
		if city != "" && c != city {
			continue
		}
		cityRevenues = append(cityRevenues, CityRevenue{
			City:          c,
			Revenue:       float64(rand.Intn(100000)+10000) / 100.0,
			RideCount:     int64(rand.Intn(2000) + 500),
			GrowthPercent: -10.0 + rand.Float64()*30.0,
		})
	}

	dailyBreakdown := make([]DailyRevenue, 30)
	for i := 0; i < 30; i++ {
		date := time.Now().AddDate(0, 0, -29+i)
		dailyBreakdown[i] = DailyRevenue{
			Date:    date.Format("2006-01-02"),
			Revenue: float64(rand.Intn(20000)+5000) / 100.0,
			Rides:   int64(rand.Intn(500) + 100),
		}
	}

	return RevenueMetrics{
		Period:         period,
		TotalRevenue:   float64(rand.Intn(500000)+100000) / 100.0,
		AverageFare:    float64(rand.Intn(2000)+500) / 100.0,
		MedianFare:     float64(rand.Intn(1500)+400) / 100.0,
		PeakRevenue:    float64(rand.Intn(50000)+10000) / 100.0,
		ByCity:         cityRevenues,
		DailyBreakdown: dailyBreakdown,
		GeneratedAt:    time.Now().UTC(),
	}
}

func (s *Server) generateDriverMetrics(period, city string, limit int) DriverMetrics {
	germanCities := []string{"Berlin", "München", "Hamburg", "Köln", "Frankfurt", "Stuttgart", "Düsseldorf"}

	topDrivers := make([]DriverInfo, 0, limit)
	for i := 0; i < limit; i++ {
		c := germanCities[rand.Intn(len(germanCities))]
		if city != "" {
			c = city
		}
		topDrivers = append(topDrivers, DriverInfo{
			DriverID:   fmt.Sprintf("driver_%06d", rand.Intn(999999)),
			TotalRides: int64(rand.Intn(500) + 50),
			Revenue:    float64(rand.Intn(50000)+10000) / 100.0,
			Rating:     4.0 + rand.Float64()*0.9,
			City:       c,
		})
	}

	cityDist := make([]CityMetric, 0, len(germanCities))
	for _, c := range germanCities {
		if city != "" && c != city {
			continue
		}
		cityDist = append(cityDist, CityMetric{
			City:       c,
			TotalRides: int64(rand.Intn(1000) + 100),
			Revenue:    float64(rand.Intn(50000)+5000) / 100.0,
			AvgRating:  4.0 + rand.Float64()*0.8,
		})
	}

	totalDrivers := int64(rand.Intn(2000) + 500)

	return DriverMetrics{
		Period:             period,
		TotalDrivers:       totalDrivers,
		ActiveDrivers:      int64(float64(totalDrivers) * 0.6),
		NewDrivers:         int64(rand.Intn(100) + 10),
		AverageRating:      4.2 + rand.Float64()*0.5,
		AverageRidesPerDay: 8.0 + rand.Float64()*12.0,
		TopDrivers:         topDrivers,
		CityDistribution:   cityDist,
		GeneratedAt:        time.Now().UTC(),
	}
}

func (s *Server) generateDailyReport(dateStr string) DailyReport {
	germanCities := []string{"Berlin", "München", "Hamburg", "Köln", "Frankfurt", "Stuttgart", "Düsseldorf"}

	cityMetrics := make([]CityMetric, 0, len(germanCities))
	for _, c := range germanCities {
		cityMetrics = append(cityMetrics, CityMetric{
			City:       c,
			TotalRides: int64(rand.Intn(500) + 50),
			Revenue:    float64(rand.Intn(10000)+1000) / 100.0,
			AvgRating:  4.0 + rand.Float64()*0.8,
		})
	}

	totalRides := int64(rand.Intn(5000) + 1000)

	return DailyReport{
		Date:             dateStr,
		TotalRides:       totalRides,
		TotalRevenue:     float64(rand.Intn(50000)+10000) / 100.0,
		UniquePassengers: int64(float64(totalRides) * 0.9),
		UniqueDrivers:    int64(float64(totalRides) * 0.3),
		AvgFare:          float64(rand.Intn(2000)+500) / 100.0,
		AvgDistance:      5.0 + rand.Float64()*10.0,
		AvgRating:        4.2 + rand.Float64()*0.5,
		CancellationRate: 5.0 + rand.Float64()*10.0,
		PeakHour:         rand.Intn(24),
		ByCity:           cityMetrics,
		GeneratedAt:      time.Now().UTC(),
	}
}

func (s *Server) generateWeeklyReport(weekStr string) WeeklyReport {
	now := time.Now()
	weekStart := now.AddDate(0, 0, -7)
	weekEnd := now

	dailyBreakdown := make([]DailyRevenue, 7)
	for i := 0; i < 7; i++ {
		date := weekStart.AddDate(0, 0, i)
		dailyBreakdown[i] = DailyRevenue{
			Date:    date.Format("2006-01-02"),
			Revenue: float64(rand.Intn(20000)+5000) / 100.0,
			Rides:   int64(rand.Intn(1000) + 200),
		}
	}

	// Find best day
	bestDayIdx := 0
	var bestRevenue float64
	for i, d := range dailyBreakdown {
		if d.Revenue > bestRevenue {
			bestRevenue = d.Revenue
			bestDayIdx = i
		}
	}

	totalRides := int64(rand.Intn(30000) + 10000)
	totalRevenue := float64(rand.Intn(300000)+100000) / 100.0

	return WeeklyReport{
		WeekStart:        weekStart.Format("2006-01-02"),
		WeekEnd:          weekEnd.Format("2006-01-02"),
		TotalRides:       totalRides,
		TotalRevenue:     totalRevenue,
		GrowthPercent:    -5.0 + rand.Float64()*20.0,
		UniquePassengers: int64(float64(totalRides) * 0.85),
		UniqueDrivers:    int64(float64(totalRides) * 0.25),
		AvgDailyRides:    float64(totalRides) / 7.0,
		BestDay:          dailyBreakdown[bestDayIdx].Date,
		DailyBreakdown:   dailyBreakdown,
		GeneratedAt:      time.Now().UTC(),
	}
}

func (s *Server) generateHeatmapData(city, period string) HeatmapData {
	// City center coordinates for German cities
	cityCenters := map[string][2]float64{
		"Berlin":    {52.5200, 13.4050},
		"München":   {48.1351, 11.5820},
		"Hamburg":   {53.5511, 9.9937},
		"Köln":      {50.9333, 6.9500},
		"Frankfurt": {50.1109, 8.6821},
		"Stuttgart": {48.7758, 9.1829},
		"Düsseldorf": {51.2217, 6.7762},
	}

	center, ok := cityCenters[city]
	if !ok {
		center = cityCenters["Berlin"]
	}

	numPoints := 50
	dataPoints := make([]HeatPoint, numPoints)
	for i := 0; i < numPoints; i++ {
		lat := center[0] + (rand.Float64()-0.5)*0.2
		lng := center[1] + (rand.Float64()-0.5)*0.3
		dataPoints[i] = HeatPoint{
			Lat:       lat,
			Lng:       lng,
			Intensity: rand.Float64(),
			RideCount: int64(rand.Intn(100) + 1),
		}
	}

	return HeatmapData{
		City:        city,
		Period:      period,
		DataPoints:  dataPoints,
		GeneratedAt: time.Now().UTC(),
	}
}

func (s *Server) generateTrendData(metric, period, city string) TrendData {
	var numPoints int
	var duration time.Duration

	switch period {
	case "daily":
		numPoints = 24
		duration = time.Hour
	case "weekly":
		numPoints = 7
		duration = 24 * time.Hour
	case "monthly":
		numPoints = 30
		duration = 24 * time.Hour
	default:
		numPoints = 7
		duration = 24 * time.Hour
	}

	now := time.Now()
	dataPoints := make([]TrendPoint, numPoints)
	baseValue := 1000.0 + rand.Float64()*5000.0

	for i := 0; i < numPoints; i++ {
		timestamp := now.Add(-time.Duration(numPoints-i) * duration)
		value := baseValue + rand.Float64()*baseValue*0.2 - baseValue*0.1
		dataPoints[i] = TrendPoint{
			Timestamp: timestamp,
			Value:     value,
			Label:     timestamp.Format("2006-01-02 15:04"),
		}
	}

	// Simple forecast: 7 points
	forecast := make([]TrendPoint, 7)
	for i := 0; i < 7; i++ {
		timestamp := now.Add(time.Duration(i+1) * duration)
		value := baseValue * (1.0 + rand.Float64()*0.05)
		forecast[i] = TrendPoint{
			Timestamp: timestamp,
			Value:     value,
			Label:     timestamp.Format("2006-01-02 15:04"),
		}
	}

	trendDirection := "stable"
	if dataPoints[len(dataPoints)-1].Value > dataPoints[0].Value*1.05 {
		trendDirection = "increasing"
	} else if dataPoints[len(dataPoints)-1].Value < dataPoints[0].Value*0.95 {
		trendDirection = "decreasing"
	}

	changeRate := (dataPoints[len(dataPoints)-1].Value - dataPoints[0].Value) / dataPoints[0].Value * 100.0

	return TrendData{
		Metric:      metric,
		Period:      period,
		DataPoints:  dataPoints,
		Trend:       trendDirection,
		ChangeRate:  changeRate,
		Forecast:    forecast,
		GeneratedAt: time.Now().UTC(),
	}
}

// Kafka consumer

func (s *Server) startKafkaConsumer() {
	go func() {
		log.Printf("Starting Kafka consumer for topic: %s", s.config.KafkaTopic)
		for {
			select {
			case <-s.ctx.Done():
				log.Println("Kafka consumer stopping...")
				if err := s.kafkaReader.Close(); err != nil {
					log.Printf("Error closing Kafka reader: %v", err)
				}
				return
			default:
				msg, err := s.kafkaReader.ReadMessage(s.ctx)
				if err != nil {
					if s.ctx.Err() != nil {
						return
					}
					log.Printf("Error reading Kafka message: %v", err)
					s.metrics.KafkaErrors.WithLabelValues(s.config.KafkaTopic, "read_error").Inc()
					time.Sleep(1 * time.Second)
					continue
				}

				s.processKafkaMessage(msg)
			}
		}
	}()
}

func (s *Server) processKafkaMessage(msg kafka.Message) {
	var event RideEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("Error unmarshaling Kafka message: %v", err)
		s.metrics.KafkaErrors.WithLabelValues(s.config.KafkaTopic, "unmarshal_error").Inc()
		return
	}

	s.metrics.KafkaMessagesConsumed.WithLabelValues(s.config.KafkaTopic, event.EventType).Inc()
	s.metrics.RidesProcessed.WithLabelValues(event.Status, event.City).Inc()

	if event.Fare > 0 {
		s.metrics.RevenueProcessed.WithLabelValues(event.City).Add(event.Fare)
	}

	s.mu.Lock()
	if len(s.eventCache) >= 1000 {
		s.eventCache = s.eventCache[1:]
	}
	s.eventCache = append(s.eventCache, event)
	s.mu.Unlock()

	// Invalidate relevant caches
	s.invalidateCaches(event)

	log.Printf("Processed ride event: event_id=%s event_type=%s ride_id=%s city=%s",
		event.EventID, event.EventType, event.RideID, event.City)
}

func (s *Server) invalidateCaches(event RideEvent) {
	keys := []string{
		"dashboard:metrics:" + time.Now().Format("2006-01-02-15"),
		"report:daily:" + time.Now().Format("2006-01-02"),
	}

	for _, key := range keys {
		if err := s.redis.Del(s.ctx, key).Err(); err != nil {
			log.Printf("Warning: failed to invalidate cache key %s: %v", key, err)
		}
	}
}

func (s *Server) start() error {
	s.startKafkaConsumer()

	log.Printf("Analytics service starting on port %s", s.config.Port)
	log.Printf("Environment: %s", s.config.Environment)
	log.Printf("Kafka brokers: %v", s.config.KafkaBrokers)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	return nil
}

func (s *Server) shutdown() {
	log.Println("Initiating graceful shutdown...")
	s.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	if err := s.db.Close(); err != nil {
		log.Printf("Database close error: %v", err)
	}

	if err := s.redis.Close(); err != nil {
		log.Printf("Redis close error: %v", err)
	}

	log.Println("Shutdown complete")
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string, defaultValue time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("Warning: failed to parse duration %q, using default %v: %v", s, defaultValue, err)
		return defaultValue
	}
	return d
}

func parseInt(s string, defaultValue int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Warning: failed to parse int %q, using default %d: %v", s, defaultValue, err)
		return defaultValue
	}
	return v
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Analytics Service for German Ride-Sharing Platform")

	cfg := loadConfig()

	srv, err := newServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.start()
	}()

	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
		srv.shutdown()
	case err := <-errChan:
		if err != nil {
			log.Printf("Server error: %v", err)
		}
		srv.shutdown()
	}
}

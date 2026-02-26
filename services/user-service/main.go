// user-service/main.go
// Production-ready User Service for German ride-sharing platform
// Handles authentication, user profiles, MFA, and GDPR-compliant data management

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Config holds service configuration
type Config struct {
	Port              string
	DatabaseURL       string
	RedisURL          string
	JWTSecret         string
	JWTAccessExpiry   time.Duration
	JWTRefreshExpiry  time.Duration
	Environment       string
	MaxLoginAttempts  int
	LockoutDuration   time.Duration
	MFAEnabled        bool
	GDPRDataRetention time.Duration
}

// Server holds all dependencies
type Server struct {
	config     *Config
	db         *pgxpool.Pool
	redis      *redis.Client
	logger     *zap.Logger
	router     *gin.Engine
	userRepo   *UserRepository
	tokenRepo  *TokenRepository
	authSvc    *AuthService
	userSvc    *UserService
	gdprSvc    *GDPRService
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using environment variables")
	}

	config := loadConfig()
	
	server, err := NewServer(config, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	if err := server.Run(); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}

func loadConfig() *Config {
	return &Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/users?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:         getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		JWTAccessExpiry:   getDurationEnv("JWT_ACCESS_EXPIRY", 15*time.Minute),
		JWTRefreshExpiry:  getDurationEnv("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
		Environment:       getEnv("ENVIRONMENT", "development"),
		MaxLoginAttempts:  getIntEnv("MAX_LOGIN_ATTEMPTS", 5),
		LockoutDuration:   getDurationEnv("LOCKOUT_DURATION", 30*time.Minute),
		MFAEnabled:        getBoolEnv("MFA_ENABLED", true),
		GDPRDataRetention: getDurationEnv("GDPR_DATA_RETENTION", 365*24*time.Hour),
	}
}

func NewServer(config *Config, logger *zap.Logger) (*Server, error) {
	// Database connection
	db, err := pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := runMigrations(config.DatabaseURL); err != nil {
		logger.Warn("Migration failed or already applied", zap.Error(err))
	}

	// Redis connection
	redisOpt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		redisOpt = &redis.Options{Addr: config.RedisURL}
	}
	rdb := redis.NewClient(redisOpt)

	// Initialize repositories
	userRepo := NewUserRepository(db, logger)
	tokenRepo := NewTokenRepository(rdb, logger)

	// Initialize services
	authSvc := NewAuthService(config, userRepo, tokenRepo, logger)
	userSvc := NewUserService(config, userRepo, logger)
	gdprSvc := NewGDPRService(config, userRepo, logger)

	// Setup router
	router := setupRouter(config.Environment)

	server := &Server{
		config:    config,
		db:        db,
		redis:     rdb,
		logger:    logger,
		router:    router,
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		authSvc:   authSvc,
		userSvc:   userSvc,
		gdprSvc:   gdprSvc,
	}

	server.setupRoutes()

	return server, nil
}

func runMigrations(databaseURL string) error {
	m, err := migrate.New(
		"file://migrations",
		databaseURL,
	)
	if err != nil {
		return err
	}
	return m.Up()
}

func setupRouter(env string) *gin.Engine {
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(requestIDMiddleware())
	router.Use(loggingMiddleware())
	router.Use(rateLimitMiddleware())
	return router
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Public routes
	public := s.router.Group("/api/v1")
	{
		public.POST("/auth/register", s.registerHandler)
		public.POST("/auth/login", s.loginHandler)
		public.POST("/auth/refresh", s.refreshHandler)
		public.POST("/auth/logout", s.logoutHandler)
		public.POST("/auth/forgot-password", s.forgotPasswordHandler)
		public.POST("/auth/reset-password", s.resetPasswordHandler)
		public.POST("/auth/verify-email", s.verifyEmailHandler)
		public.POST("/auth/mfa/setup", s.setupMFAHandler)
		public.POST("/auth/mfa/verify", s.verifyMFAHandler)
	}

	// Protected routes
	protected := s.router.Group("/api/v1")
	protected.Use(s.authMiddleware())
	{
		// User management
		protected.GET("/users/me", s.getCurrentUserHandler)
		protected.PUT("/users/me", s.updateUserHandler)
		protected.DELETE("/users/me", s.deleteUserHandler)
		protected.POST("/users/me/avatar", s.uploadAvatarHandler)
		
		// MFA management
		protected.POST("/users/me/mfa/enable", s.enableMFAHandler)
		protected.POST("/users/me/mfa/disable", s.disableMFAHandler)
		protected.GET("/users/me/mfa/backup-codes", s.generateBackupCodesHandler)
		
		// GDPR
		protected.GET("/users/me/data-export", s.exportUserDataHandler)
		protected.POST("/users/me/data-delete", s.requestDataDeletionHandler)
		protected.GET("/users/me/privacy-settings", s.getPrivacySettingsHandler)
		protected.PUT("/users/me/privacy-settings", s.updatePrivacySettingsHandler)
		
		// Sessions
		protected.GET("/users/me/sessions", s.getActiveSessionsHandler)
		protected.DELETE("/users/me/sessions/:sessionId", s.revokeSessionHandler)
		protected.DELETE("/users/me/sessions", s.revokeAllSessionsHandler)
	}

	// Admin routes
	admin := s.router.Group("/api/v1/admin")
	admin.Use(s.authMiddleware())
	admin.Use(s.adminMiddleware())
	{
		admin.GET("/users", s.listUsersHandler)
		admin.GET("/users/:id", s.getUserHandler)
		admin.PUT("/users/:id", s.adminUpdateUserHandler)
		admin.DELETE("/users/:id", s.adminDeleteUserHandler)
		admin.POST("/users/:id/verify", s.verifyUserHandler)
		admin.POST("/users/:id/suspend", s.suspendUserHandler)
	}
}

func (s *Server) Run() error {
	srv := &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		s.logger.Info("Shutting down server...")
		if err := srv.Shutdown(ctx); err != nil {
			s.logger.Error("Server shutdown error", zap.Error(err))
		}
	}()

	s.logger.Info("Starting User Service", zap.String("port", s.config.Port))
	return srv.ListenAndServe()
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
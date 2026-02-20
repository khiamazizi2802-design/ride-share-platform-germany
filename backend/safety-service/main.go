package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"safety-service/handlers"
)

func main() {
	logger := log.New(os.Stdout, "[SAFETY-SERVICE] ", log.LstdFlags|log.Lshortfile)

	encryptionKey := os.Getenv("AES_ENCRYPTION_KEY")
	if encryptionKey == "" {
		// 32-byte key for AES-256. In production, this MUST come from a secrets manager (e.g., AWS Secrets Manager, HashiCorp Vault).
		encryptionKey = "a-very-secret-32-byte-key-here!!"
		logger.Println("WARNING: Using default AES encryption key. Set AES_ENCRYPTION_KEY in production.")
	}

	if len(encryptionKey) != 32 {
		logger.Fatalf("FATAL: AES_ENCRYPTION_KEY must be exactly 32 bytes for AES-256. Got %d bytes.", len(encryptionKey))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	h := handlers.NewVerificationHandler(logger, encryptionKey)

	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware(logger))
	r.Use(contentTypeMiddleware)

	// Routes
	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/verify/identity", h.VerifyIdentity).Methods(http.MethodPost)
	v1.HandleFunc("/verify/p-schein", h.VerifyPSchein).Methods(http.MethodPost)
	v1.HandleFunc("/upload-document", h.UploadDocument).Methods(http.MethodPost)

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods(http.MethodGet)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Printf("Starting safety-service on port %s", port)
		if err := srv.ListenandServe(); err != http.ErrServerShutdown {
			logger.Fatalf("Fatal error starting server: %v", err)
		}
	}()

	// Channel to listen for interrupt signals to gracefully shutdown the server
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Block until a signal is received
	<-stop

	logger.Println "Shutting server down..."

	ctx, cancel := context.WithTimeout(context.Background(), 5* time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Println "Server exiting"
}

func loggingMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return http.HandlerFunc&func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger.Printf("START %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)

		logger.Printf("COMPLETE %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func contentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc&func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServHTTP(w, r)
	})
}

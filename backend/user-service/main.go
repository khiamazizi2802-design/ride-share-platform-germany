package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type UserType string

const (
	Rider  UserType = "RIDER"
	Driver UserType = "DRIVER"
)

type PScheinStatus string

const (
	PScheinPending  PScheinStatus = "PENDING"
	PScheinVerified PScheinStatus = "VERIFIED"
	PScheinRejected PScheinStatus = "REJECTED"
	PScheinExpired  PScheinStatus = "EXPIRED"
)

type User struct {
	ID              string         `json:"id"`
	Email           string         `json:"email"`
	Name            string         `json:"name"`
	Phone           string         `json:"phone"`
	UserType        UserType       `json:"user_type"`
	PScheinNumber   string         `json:"p_schein_number,omitempty"`
	PScheinStatus   PScheinStatus  `json:"p_schein_status,omitempty"`
	PScheinIssuedAt *time.Time     `json:"p_schein_issued_at,omitempty"`
	PScheinExpiresAt *time.Time    `json:"p_schein_expires_at,omitempty"`
	PScheinVerifiedAt *time.Time   `json:"p_schein_verified_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type UserStore struct {
	mu    sync.RWMutex
	users map[string]*User
}

var (
	userStore *UserStore
	logger    *log.Logger
)

func init() {
	userStore = &UserStore{users: make(map[string]*User)}
	logger = log.New(os.Stdout, "[USER-SERVICE] ", log.LstdFlags|log.Lshortfile)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/users", createUserHandler).Methods("POST")
	router.HandleFunc("/users/{id}", getUserHandler).Methods("GET")
	router.HandleFunc("/users/{id}", updateUserHandler).Methods("PUT")
	router.HandleFunc("/users/{id}", deleteUserHandler).Methods("DELETE")
	router.HandleFunc("/users/{id}/p-schein", updatePScheinHandler).Methods("PUT")
	router.HandleFunc("/users/{id}/p-schein/verify", verifyPScheinHandler).Methods("POST")

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Printf("Starting user-service on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server exited")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email           string    `json:"email"`
		Name            string    `json:"name"`
		Phone           string    `json:"phone"`
		UserType        UserType  `json:"user_type"`
		PScheinNumber   string    `json:"p_schein_number,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Name == "" || req.Phone == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if req.UserType != Rider && req.UserType != Driver {
		http.Error(w, "Invalid user type. Must be RIDER or DRIVER", http.StatusBadRequest)
		return
	}

	now := time.Now()
	user := &User{
		ID:        uuid.New().String(),
		Email:     req.Email,
		Name:      req.Name,
		Phone:     req.Phone,
		UserType:  req.UserType,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if req.UserType == Driver {
		if req.PScheinNumber == "" {
			http.Error(w, "P-Schein number required for drivers", http.StatusBadRequest)
			return
		}
		user.PScheinNumber = req.PScheinNumber
		user.PScheinStatus = PScheinPending
	}

	userStore.mu.Lock()
	userStore.users[user.ID] = user
	userStore.mu.Unlock()

	logger.Printf("User created: %s (%s) - %s", user.ID, user.UserType, user.Email)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	userStore.mu.RLock()
	user, exists := userStore.users[id]
	userStore.mu.RUnlock()

	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Email string `json:"email,omitempty"`
		Name  string `json:"name,omitempty"`
		Phone string `json:"phone,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	userStore.mu.Lock()
	user, exists := userStore.users[id]
	if !exists {
		userStore.mu.Unlock()
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	user.UpdatedAt = time.Now()
	userStore.mu.Unlock()

	logger.Printf("User updated: %s", user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	userStore.mu.Lock()
	_, exists := userStore.users[id]
	if !exists {
		userStore.mu.Unlock()
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	delete(userStore.users, id)
	userStore.mu.Unlock()

	logger.Printf("User deleted: %s", id)

	w.WriteHeader(http.StatusNoContent)
}

func updatePScheinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		PScheinNumber   string     `json:"p_schein_number"`
		PScheinIssuedAt *time.Time `json:"p_schein_issued_at,omitempty"`
		PScheinExpiresAt *time.Time `json:"p_schein_expires_at,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	userStore.mu.Lock()
	user, exists := userStore.users[id]
	if !exists {
		userStore.mu.Unlock()
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if user.UserType != Driver {
		userStore.mu.Unlock()
		http.Error(w, "User is not a driver", http.StatusBadRequest)
		return
	}

	if req.PScheinNumber != "" {
		user.PScheinNumber = req.PScheinNumber
		user.PScheinStatus = PScheinPending
		user.PScheinVerifiedAt = nil
	}

	if req.PScheinIssuedAt != nil {
		user.PScheinIssuedAt = req.PScheinIssuedAt
	}

	if req.PScheinExpiresAt != nil {
		user.PScheinExpiresAt = req.PScheinExpiresAt
		if time.Now().After(*req.PScheinExpiresAt) {
			user.PScheinStatus = PScheinExpired
		}
	}

	user.UpdatedAt = time.Now()
	userStore.mu.Unlock()

	logger.Printf("P-Schein updated for user: %s", user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func verifyPScheinHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Verified bool   `json:"verified"`
		Reason   string `json:"reason,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	userStore.mu.Lock()
	user, exists := userStore.users[id]
	if !exists {
		userStore.mu.Unlock()
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if user.UserType != Driver {
		userStore.mu.Unlock()
		http.Error(w, "User is not a driver", http.StatusBadRequest)
		return
	}

	if user.PScheinStatus != PScheinPending {
		userStore.mu.Unlock()
		http.Error(w, "P-Schein is not in pending status", http.StatusBadRequest)
		return
	}

	now := time.Now()
	if req.Verified {
		user.PScheinStatus = PScheinVerified
		user.PScheinVerifiedAt = &now
		logger.Printf("P-Schein verified for user: %s", user.ID)
	} else {
		user.PScheinStatus = PScheinRejected
		logger.Printf("P-Schein rejected for user: %s, reason: %s", user.ID, req.Reason)
	}

	user.UpdatedAt = now
	userStore.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
package main

import (
	"context"
	"encoding/json"
	"fmt"
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

type RideStatus string

const (
	RideRequested RideStatus = "REQUESTED"
	RideMatched   RideStatus = "MATCHED"
	RideStarted   RideStatus = "STARTED"
	RideCompleted RideStatus = "COMPLETED"
)

type Ride struct {
	ID            string     `json:"id"`
	RiderID       string     `json:"rider_id"`
	DriverID      string     `json:"driver_id,omitempty"`
	Status        RideStatus `json:"status"`
	PickupLat     float64    `json:"pickup_lat"`
	PickupLon     float64    `json:"pickup_lon"`
	DropoffLat    float64    `json:"dropoff_lat,omitempty"`
	DropoffLon    float64    `json:"dropoff_lon,omitempty"`
	RequestedAt   time.Time  `json:"requested_at"`
	MatchedAt     *time.Time `json:"matched_at,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	ReturnToBase  bool       `json:"return_to_base"`
}

type ReturnToBaseLog struct {
	ID              string    `json:"id"`
	RideID          string    `json:"ride_id"`
	DriverID        string    `json:"driver_id"`
	ReturnStartedAt time.Time `json:"return_started_at"`
	ReturnEndedAt   *time.Time `json:"return_ended_at,omitempty"`
	BaseLat         float64   `json:"base_lat"`
	BaseLon         float64   `json:"base_lon"`
	Compliance      bool      `json:"compliance"`
}

type RideStore struct {
	mu    sync.RWMutex
	rides map[string]*Ride
}

type ReturnToBaseStore struct {
	mu   sync.RWMutex
	logs map[string]*ReturnToBaseLog
}

var (
	rideStore         *RideStore
	returnToBaseStore *ReturnToBaseStore
	logger            *log.Logger
)

func init() {
	rideStore = &RideStore{rides: make(map[string]*Ride)}
	returnToBaseStore = &ReturnToBaseStore{logs: make(map[string]*ReturnToBaseLog)}
	logger = log.New(os.Stdout, "[RIDE-SERVICE] ", log.LstdFlags|log.Lshortfile)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/rides", createRideHandler).Methods("POST")
	router.HandleFunc("/rides/{id}", getRideHandler).Methods("GET")
	router.HandleFunc("/rides/{id}/match", matchRideHandler).Methods("PUT")
	router.HandleFunc("/rides/{id}/start", startRideHandler).Methods("PUT")
	router.HandleFunc("/rides/{id}/complete", completeRideHandler).Methods("PUT")
	router.HandleFunc("/return-to-base", createReturnToBaseHandler).Methods("POST")
	router.HandleFunc("/return-to-base/{id}/end", endReturnToBaseHandler).Methods("PUT")
	router.HandleFunc("/return-to-base/driver/{driver_id}", getReturnToBaseLogsHandler).Methods("GET")

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Printf("Starting ride-service on port %s", port)
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

func createRideHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RiderID    string  `json:"rider_id"`
		PickupLat  float64 `json:"pickup_lat"`
		PickupLon  float64 `json:"pickup_lon"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.RiderID == "" || req.PickupLat == 0 || req.PickupLon == 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	ride := &Ride{
		ID:          uuid.New().String(),
		RiderID:     req.RiderID,
		Status:      RideRequested,
		PickupLat:   req.PickupLat,
		PickupLon:   req.PickupLon,
		RequestedAt: time.Now(),
	}

	rideStore.mu.Lock()
	rideStore.rides[ride.ID] = ride
	rideStore.mu.Unlock()

	logger.Printf("Ride created: %s for rider: %s", ride.ID, ride.RiderID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ride)
}

func getRideHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rideStore.mu.RLock()
	ride, exists := rideStore.rides[id]
	rideStore.mu.RUnlock()

	if !exists {
		http.Error(w, "Ride not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ride)
}

func matchRideHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		DriverID string `json:"driver_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.DriverID == "" {
		http.Error(w, "Driver ID required", http.StatusBadRequest)
		return
	}

	rideStore.mu.Lock()
	ride, exists := rideStore.rides[id]
	if !exists {
		rideStore.mu.Unlock()
		http.Error(w, "Ride not found", http.StatusNotFound)
		return
	}

	if ride.Status != RideRequested {
		rideStore.mu.Unlock()
		http.Error(w, fmt.Sprintf("Cannot match ride in status: %s", ride.Status), http.StatusBadRequest)
		return
	}

	now := time.Now()
	ride.DriverID = req.DriverID
	ride.Status = RideMatched
	ride.MatchedAt = &now
	rideStore.mu.Unlock()

	logger.Printf("Ride matched: %s with driver: %s", ride.ID, req.DriverID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ride)
}

func startRideHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rideStore.mu.Lock()
	ride, exists := rideStore.rides[id]
	if !exists {
		rideStore.mu.Unlock()
		http.Error(w, "Ride not found", http.StatusNotFound)
		return
	}

	if ride.Status != RideMatched {
		rideStore.mu.Unlock()
		http.Error(w, fmt.Sprintf("Cannot start ride in status: %s", ride.Status), http.StatusBadRequest)
		return
	}

	now := time.Now()
	ride.Status = RideStarted
	ride.StartedAt = &now
	rideStore.mu.Unlock()

	logger.Printf("Ride started: %s", ride.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ride)
}

func completeRideHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		DropoffLat   float64 `json:"dropoff_lat"`
		DropoffLon   float64 `json:"dropoff_lon"`
		ReturnToBase bool    `json:"return_to_base"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	rideStore.mu.Lock()
	ride, exists := rideStore.rides[id]
	if !exists {
		rideStore.mu.Unlock()
		http.Error(w, "Ride not found", http.StatusNotFound)
		return
	}

	if ride.Status != RideStarted {
		rideStore.mu.Unlock()
		http.Error(w, fmt.Sprintf("Cannot complete ride in status: %s", ride.Status), http.StatusBadRequest)
		return
	}

	now := time.Now()
	ride.Status = RideCompleted
	ride.CompletedAt = &now
	ride.DropoffLat = req.DropoffLat
	ride.DropoffLon = req.DropoffLon
	ride.ReturnToBase = req.ReturnToBase
	rideStore.mu.Unlock()

	logger.Printf("Ride completed: %s, return-to-base: %v", ride.ID, req.ReturnToBase)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ride)
}

func createReturnToBaseHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RideID   string  `json:"ride_id"`
		DriverID string  `json:"driver_id"`
		BaseLat  float64 `json:"base_lat"`
		BaseLon  float64 `json:"base_lon"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.RideID == "" || req.DriverID == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	rtbLog := &ReturnToBaseLog{
		ID:              uuid.New().String(),
		RideID:          req.RideID,
		DriverID:        req.DriverID,
		ReturnStartedAt: time.Now(),
		BaseLat:         req.BaseLat,
		BaseLon:         req.BaseLon,
		Compliance:      true,
	}

	returnToBaseStore.mu.Lock()
	returnToBaseStore.logs[rtbLog.ID] = rtbLog
	returnToBaseStore.mu.Unlock()

	logger.Printf("Return-to-base started: %s for ride: %s, driver: %s", rtbLog.ID, req.RideID, req.DriverID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rtbLog)
}

func endReturnToBaseHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	returnToBaseStore.mu.Lock()
	rtbLog, exists := returnToBaseStore.logs[id]
	if !exists {
		returnToBaseStore.mu.Unlock()
		http.Error(w, "Return-to-base log not found", http.StatusNotFound)
		return
	}

	if rtbLog.ReturnEndedAt != nil {
		returnToBaseStore.mu.Unlock()
		http.Error(w, "Return-to-base already ended", http.StatusBadRequest)
		return
	}

	now := time.Now()
	rtbLog.ReturnEndedAt = &now
	returnToBaseStore.mu.Unlock()

	logger.Printf("Return-to-base ended: %s for driver: %s", rtbLog.ID, rtbLog.DriverID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rtbLog)
}

func getReturnToBaseLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	driverID := vars["driver_id"]

	returnToBaseStore.mu.RLock()
	var logs []*ReturnToBaseLog
	for _, log := range returnToBaseStore.logs {
		if log.DriverID == driverID {
			logs = append(logs, log)
		}
	}
	returnToBaseStore.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
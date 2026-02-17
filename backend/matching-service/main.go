package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang/geo/s2"
)

// AuditLogger handles compliant logging for German regulations (GDPR, audit trails)
type AuditLogger struct {
	logger *log.Logger
}

func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		logger: log.New(os.Stdout, "[AUDIT] ", log.LstdFlags|log.Lmicroseconds|log.LUTC),
	}
}

func (a *AuditLogger) LogMatchRequest(riderID, sessionID string, lat, lng float64) {
	a.logger.Printf("MATCH_REQUEST rider_id=%s session_id=%s lat=%.6f lng=%.6f timestamp=%s", riderID, sessionID, lat, lng, time.Now().UTC().Format(time.RFC3339))
}

func (a *AuditLogger) LogMatchResult(riderID, driverID, sessionID string, distance float64, success bool) {
	a.logger.Printf("MATCH_RESULT rider_id=%s driver_id=%s session_id=%s distance_km=%.3f success=%t timestamp=%s", riderID, driverID, sessionID, distance, success, time.Now().UTC().Format(time.RFC3339))
}

func (a *AuditLogger) LogError(action, riderID, sessionID, errMsg string) {
	a.logger.Printf("ERROR action=%s rider_id=%s session_id=%s error=%s timestamp=%s", action, riderID, sessionID, errMsg, time.Now().UTC().Format(time.RFC3339))
}

// Driver represents an available driver with location
type Driver struct {
	ID        string    `json:"id"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Available bool      `json:"available"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MatchRequest represents a rider's match request
type MatchRequest struct {
	RiderID   string  `json:"rider_id"`
	SessionID string  `json:"session_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
}

// MatchResponse represents the matching result
type MatchResponse struct {
	Success    bool    `json:"success"`
	DriverID   string  `json:"driver_id,omitempty"`
	DriverLat  float64 `json:"driver_lat,omitempty"`
	DriverLng  float64 `json:"driver_lng,omitempty"`
	DistanceKM float64 `json:"distance_km,omitempty"`
	Message    string  `json:"message,omitempty"`
}

// DriverStore manages in-memory driver locations with S2 indexing
type DriverStore struct {
	mu      sync.RWMutex
	drivers map[string]*Driver
	// S2 CellID to driver IDs mapping for spatial indexing
	s2Index map[s2.CellID][]string
}

func NewDriverStore() *DriverStore {
	return &DriverStore{
		drivers: make(map[string]*Driver),
		s2Index: make(map[s2.CellID][]string),
	}
}

// AddDriver adds or updates a driver location with S2 indexing
func (ds *DriverStore) AddDriver(driver *Driver) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Remove old S2 index if driver exists
	if oldDriver, exists := ds.drivers[driver.ID]; exists {
		oldCellID := s2.CellIDFromLatLng(s2.LatLngFromDegrees(oldDriver.Lat, oldDriver.Lng)).Parent(15)
		ds.removeFromS2Index(oldCellID, driver.ID)
	}

	// Add to driver map
	ds.drivers[driver.ID] = driver

	// Index using S2 cell at level 15 (~1km cells)
	if driver.Available {
		latLng := s2.LatLngFromDegrees(driver.Lat, driver.Lng)
		cellID := s2.CellIDFromLatLng(latLng).Parent(15)
		ds.s2Index[cellID] = append(ds.s2Index[cellID], driver.ID)
	}
}

func (ds *DriverStore) removeFromS2Index(cellID s2.CellID, driverID string) {
	if ids, exists := ds.s2Index[cellID]; exists {
		for i, id := range ids {
			if id == driverID {
				ds.s2Index[cellID] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
		if len(ds.s2Index[cellID]) == 0 {
			delete(ds.s2Index, cellID)
		}
	}
}

// FindNearestDriver uses S2 geometry to find the closest available driver
func (ds *DriverStore) FindNearestDriver(lat, lng float64, maxDistanceKM float64) (*Driver, float64) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	riderLatLng := s2.LatLngFromDegrees(lat, lng)
	riderCellID := s2.CellIDFromLatLng(riderLatLng).Parent(15)

	// Start with the rider's cell and expand to neighbors
	cellsToCheck := []s2.CellID{riderCellID}
	
	// Add neighbor cells for wider search radius
	for _, neighbor := range riderCellID.EdgeNeighbors() {
		cellsToCheck = append(cellsToCheck, neighbor)
	}

	var nearestDriver *Driver
	minDistance := math.MaxFloat64

	// Check drivers in relevant S2 cells
	for _, cellID := range cellsToCheck {
		if driverIDs, exists := ds.s2Index[cellID]; exists {
			for _, driverID := range driverIDs {
				driver := ds.drivers[driverID]
				if !driver.Available {
					continue
				}

				driverLatLng := s2.LatLngFromDegrees(driver.Lat, driver.Lng)
				distance := riderLatLng.Distance(driverLatLng).Radians() * 6371.0 // Earth radius in km

				if distance < minDistance && distance <= maxDistanceKM {
					minDistance = distance
					nearestDriver = driver
				}
			}
		}
	}

	if nearestDriver == nil {
		return nil, 0
	}

	return nearestDriver, minDistance
}

// MatchingService handles ride matching logic
type MatchingService struct {
	driverStore *DriverStore
	auditLogger *AuditLogger
	appLogger   *log.Logger
}

func NewMatchingService() *MatchingService {
	return &MatchingService{
		driverStore: NewDriverStore(),
		auditLogger: NewAuditLogger(),
		appLogger:   log.New(os.Stdout, "[APP] ", log.LstdFlags),
	}
}

// Initialize with mock drivers for demonstration
func (ms *MatchingService) InitializeMockDrivers() {
	mockDrivers := []*Driver{
		{ID: "driver-001", Lat: 52.5200, Lng: 13.4050, Available: true, UpdatedAt: time.Now()}, // Berlin
		{ID: "driver-002", Lat: 52.5180, Lng: 13.4070, Available: true, UpdatedAt: time.Now()},
		{ID: "driver-003", Lat: 52.5230, Lng: 13.4100, Available: true, UpdatedAt: time.Now()},
		{ID: "driver-004", Lat: 48.1351, Lng: 11.5820, Available: true, UpdatedAt: time.Now()}, // Munich
		{ID: "driver-005", Lat: 50.1109, Lng: 8.6821, Available: true, UpdatedAt: time.Now()},  // Frankfurt
	}

	for _, driver := range mockDrivers {
		ms.driverStore.AddDriver(driver)
	}
	ms.appLogger.Printf("Initialized %d mock drivers", len(mockDrivers))
}

// healthHandler implements health check endpoint
func (ms *MatchingService) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{"){
		"status":    "UP",
		"service":   "matching-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// matchHandler implements the matching endpoint
func (ms *MatchingService) matchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ms.appLogger.Printf("Invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.RiderID == "" || req.SessionID == "" {
		http.Error(w, "rider_id and session_id are required", http.StatusBadRequest)
		return
	}

	if req.Lat < -90 || req.Lat > 90 || req.Lng < -180 || req.Lng > 180 {
		http.Error(w, "Invalid coordinates", http.StatusBadRequest)
		return
	}

	// Audit log for compliance (GDPR requires logging of data processing)
	ms.auditLogger.LogMatchRequest(req.RiderID, req.SessionID, req.Lat, req.Lng)

	// Find nearest driver within 10km radius
	driver, distance := ms.driverStore.FindNearestDriver(req.Lat, req.Lng, 10.0)

	var response MatchResponse

	if driver == nil {
		// No driver found
		response = MatchResponse{
			Success: false,
			Message: "No available drivers found nearby",
		}
		ms.auditLogger.LogMatchResult(req.RiderID, "", req.SessionID, 0, false)
	} else {
		// Successful match
		response = MatchResponse{
			Success:    true,
			DriverID:   driver.ID,
			DriverLat:  driver.Lat,
			DriverLng:  driver.Lng,
			DistanceKM: distance,
			Message:    "Match found",
		}
		ms.auditLogger.LogMatchResult(req.RiderID, driver.ID, req.SessionID, distance, true)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Initialize service
	service := NewMatchingService()
	service.InitializeMockDrivers()

	// Setup HTTP server with timeouts for production
	mux := http.NewServeMux()
	mux.HandleFunc("/health", service.healthHandler)
	mux.HandleFunc("/api/v1/match", service.matchHandler)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	service.appLogger.Println("Matching service starting on port 8080")
	service.appLogger.Println("Endpoints: /health, /api/v1/match")

	// Graceful shutdown handling
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			service.appLogger.Fatalf("Server error: %v", err)
		}
	}()

	service.appLogger.Println("Service ready to accept requests")

	// Block forever (in production, add graceful shutdown with signal handling)
	select {}
}
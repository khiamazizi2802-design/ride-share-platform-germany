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

// Driver represents a real-time driver state
type Driver struct {
	ID        string
	Lat       float64
	Lng       float64
	Available bool
	LastSeen  time.Time
}

// SpatialIndex manages real-time geospatial driver tracking using S2
type SpatialIndex struct {
	mu      sync.RWMutex
	drivers map[string]*Driver
}

func NewSpatialIndex() *SpatialIndex {
	return &SpatialIndex{
		drivers: make(map[string]*Driver),
	}
}

func (s *SpatialIndex) UpdateDriver(id string, lat, lng float64, available bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.drivers[id] = &Driver{
		ID:        id,
		Lat:       lat,
		Lng:       lng,
		Available: available,
		LastSeen:  time.Now(),
	}
}

// findMatch implements the core matching algorithm
func (s *SpatialIndex) findMatch(riderLat, riderLng float64, radiusKm float64) (*Driver, float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	riderLatLng := s2.LatLngFromDegrees(riderLat, riderLng)
	var bestDriver *Driver
	minDist := radiusKm

	for _, d := range s.drivers {
		if !d.Available {
			continue
		}

		// Calculate Haversine distance using S2
		driverLatLng := s2.LatLngFromDegrees(d.Lat, d.Lng)
		dist := riderLatLng.Distance(driverLatLng).Radians() * 6371.0 // Earth radius in km

		if dist < minDist {
			minDist = dist
			bestDriver = d
		}
	}

	return bestDriver, minDist
}

type MatchRequest struct {
	RiderID   string  `json:"rider_id"`
	SessionID string  `json:"session_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
}

type MatchResponse struct {
	Success  bool    `json:"success"`
	DriverID string  `json:"driver_id,omitempty"`
	Distance float64 `json:"distance_km,omitempty"`
	Message  string  `json:"message,omitempty"`
}

func main() {
	audit := NewAuditLogger()
	index := NewSpatialIndex()

	// Mock data for demonstration
	index.UpdateDriver("driver_berlin_01", 52.5200, 13.4050, true)  // Mitte
	index.UpdateDriver("driver_berlin_02", 52.5300, 13.3800, true)  // Wedding
	index.UpdateDriver("driver_berlin_03", 52.4800, 13.4200, true)  // Neukölln

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/match", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req MatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			audit.LogError("DECODE", req.RiderID, req.SessionID, err.Error())
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		audit.LogMatchRequest(req.RiderID, req.SessionID, req.Lat, req.Lng)

		// Find closest driver within 5km
		driver, dist := index.findMatch(req.Lat, req.Lng, 5.0)

		resp := MatchResponse{Success: driver != nil}
		if driver != nil {
			resp.DriverID = driver.ID
			resp.Distance = dist
			resp.Message = "Driver found and dispatched"
			audit.LogMatchResult(req.RiderID, driver.ID, req.SessionID, dist, true)
		} else {
			resp.Message = "No drivers available within 5km"
			audit.LogMatchResult(req.RiderID, "", req.SessionID, 0, false)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Matching Service starting on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

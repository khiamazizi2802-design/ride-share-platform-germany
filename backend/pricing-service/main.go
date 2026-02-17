package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// German PBefG (Personenbeförderungsgesetz) compliance constants
const (
	// MinimumFareEUR represents the absolute minimum fare in EUR as per PBefG §51
	// This prevents price dumping and ensures fair competition
	MinimumFareEUR = 5.00

	// MaxSurgeMultiplier caps the surge pricing to prevent excessive pricing
	// German regulation requires "reasonable" pricing (PBefG §39)
	MaxSurgeMultiplier = 2.0

	// BaseRateEUR is the starting fare for any ride
	BaseRateEUR = 3.50

	// PricePerKmEUR is the cost per kilometer traveled
	PricePerKmEUR = 1.80

	// PricePerMinuteEUR is the cost per minute of ride time
	PricePerMinuteEUR = 0.35

	// MinPricePerKmEUR ensures compliance with §39 PBefG regarding minimum cost coverage
	// Price per km cannot effectively fall below this after surge is applied
	MinPricePerKmEUR = 1.50
)

// PriceRequest represents the incoming pricing calculation request
type PriceRequest struct {
	DistanceKm float64 `json:"distance_km"`
	DurationMin float64 `json:"duration_min"`
	Demand int `json:"demand"` // Current demand in area (e.g., active ride requests)
	Supply int `json:"supply"` // Current supply in area (e.g., available drivers)
}

// PriceResponse represents the pricing calculation response
type PriceResponse struct {
	BasePrice float64 `json:"base_price"`
	DistancePrice float64 `json:"distance_price"`
	TimePrice float64 `json:"time_price"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	Subtotal float64 `json:"subtotal"`
	FinalPrice float64 `json:"final_price"`
	Currency string `json:"currency"`
	ComplianceNote string `json:"compliance_note,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code string `json:"code"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status string `json:"status"`
	Timestamp string `json:"timestamp"`
	Service string `json:"service"`
}

var logger *slog.Logger

func init() {
	// Initialize structured logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}

func main() {
	logger.Info("Starting pricing-service", "version", "1.0.0")

	mux := http.NewServeMux()
	mux.HandleFunc("/price", handlePrice)
	mux.HandleFunc("/health", handleHealth)

	// Wrap mux with logging middleware
	handler := loggingMiddleware(mux)

	srv := &http.Server{
		Addr: ":8080",
		Handler: handler,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	// Graceful shutdown handling
	go func() {
		logger.Info("Server starting", "address", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("Server stopped gracefully")
}

// loggingMiddleware adds request logging
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger.Info("Request started",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		next.ServeHTTP(w, r)

		logger.Info("Request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// handleHealth returns the service health status
func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responseError(w, "Method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status: "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Service: "pricing-service",
	}

	responseJSON(w, response, http.StatusOK)
}

// handlePrice calculates the ride price based on distance, time, and surge
func handlePrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responseError(w, "Method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	req, err := parsePriceRequest(r)
	if err != nil {
		logger.Warn("Invalid price request", "error", err)
		responseError(w, err.Error(), "INVALID_REQUEST", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := validatePriceRequest(req); err != nil {
		logger.Warn("Price request validation failed", "error", err)
		responseError(w, err.Error(), "VALIDATION_ERROR", http.StatusBadRequest)
		return
	}

	// Calculate price
	resp, err := calculatePrice(req)
	if err != nil {
		logger.Error("Price calculation error", "error", err)
		responseError(w, "Failed to calculate price", "CALCULATION_ERROR", http.StatusInternalServerError)
		return
	}

	logger.Info("Price calculated",
		"distance_km", req.DistanceKm,
		"duration_min", req.DurationMin,
		"surge_multiplier", resp.SurgeMultiplier,
		"final_price", resp.FinalPrice,
	)

	responseJSON(w, resp, http.StatusOK)
}

// parsePriceRequest extracts pricing parameters from query string
func parsePriceRequest(r *http.Request) (*PriceRequest, error) {
	query := r.URL.Query()

	distance, err := strconv.ParseFloat(query.Get("distance_km"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid distance_km parameter: %w", err)
	}

	duration, err := strconv.ParseFloat(query.Get("duration_min"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid duration_min parameter: %w", err)
	}

	demand, err := strconv.Atoi(query.Get("demand"))
	if err != nil {
		demand = 10 // Default demand
	}

	supply, err := strconv.Atoi(query.Get("supply"))
	if err != nil {
		supply = 10 // Default supply
	}

	return &PriceRequest{
		DistanceKm: distance,
		DurationMin: duration,
		Demand: demand,
		Supply: supply,
	}, nil
}

// validatePriceRequest ensures request parameters are valid
func validatePriceRequest(req *PriceRequest) error {
	if req.DistanceKm <= 0 {
		return errors.New("distance_km must be greater than 0")
	}

	if req.DistanceKm > 500 {
		return errors.New("distance_km exceeds maximum allowed (500km)")
	}

	if req.DurationMin <= 0 {
		return errors.New("duration_min must be greater than 0")
	}

	if req.DurationMin > 600 {
		return errors.New("duration_min exceeds maximum allowed (600min)")
	}

	if req.Demand < 0 {
		return errors.New("demand cannot be negative")
	}

	if req.Supply < 0 {
		return errors.New("supply cannot be negative")
	}

	return nil
}

// calculatePrice computes the final price with PBefG compliance
func calculatePrice(req *PriceRequest) (*PriceResponse, error) {
	// Base price component
	basePrice := BaseRateEUR

	// Distance-based price component
	distancePrice := req.DistanceKm * PricePerKmEUR

	// Time-based price component
	timePrice := req.DurationMin * PricePerMinuteEUR

	// Calculate surge multiplier based on demand/supply ratio
	surgeMultiplier := calculateSurgeMultiplier(req.Demand, req.Supply)

	// Calculate subtotal before surge
	subtotal := basePrice + distancePrice + timePrice

	// Apply surge multiplier
	finalPrice := subtotal * surgeMultiplier

	// PBefG Compliance checks and adjustments
	complianceNote := ""

	// 1. Enforce minimum fare (PBefG §51 - prevents price dumping)
	if finalPrice < MinimumFareEUR {
		logger.Info("Minimum fare enforced",
			"calculated_price", finalPrice,
			"minimum_fare", MinimumFareEUR,
		)
		finalPrice = MinimumFareEUR
		complianceNote = "Price adjusted to minimum fare per PBefG §51"
	}

	// 2. Ensure effective price per km meets minimum threshold (PBefG §39)
	// This ensures operational costs are covered
	effectivePricePerKm := (finalPrice - basePrice) / req.DistanceKm
	if effectivePricePerKm < MinPricePerKmEUR && req.DistanceKm > 0 {
		// Adjust price to meet minimum per-km rate
		requiredDistancePrice := req.DistanceKm * MinPricePerKmEUR
		adjustedPrice := basePrice + requiredDistancePrice + timePrice
		if adjustedPrice > finalPrice {
			logger.Info("Minimum per-km rate enforced",
				"original_price", finalPrice,
				"adjusted_price", adjustedPrice,
			)
			finalPrice = adjustedPrice
			if complianceNote == "" {
				complianceNote = "Price adjusted to minimum per-km rate per PBefG §39"
			}
		}
	}

	// 3. Round to 2 decimal places (EUR cents)
	finalPrice = math.Round(finalPrice*100) / 100
	subtotal = math.Round(subtotal*100) / 100
	distancePrice = math.Round(distancePrice*100) / 100
	timePrice = math.Round(timePrice*100) / 100

	return &PriceResponse{
		BasePrice: basePrice,
		DistancePrice: distancePrice,
		TimePrice: timePrice,
		SurgeMultiplier: surgeMultiplier,
		Subtotal: subtotal,
		FinalPrice: finalPrice,
		Currency: "EUR",
		ComplianceNote: complianceNote,
	}, nil
}

// calculateSurgeMultiplier computes surge pricing based on demand/supply
// Capped at MaxSurgeMultiplier to comply with PBefG §39 (reasonable pricing)
func calculateSurgeMultiplier(demand, supply int) float64 {
	// Avoid division by zero
	if supply == 0 {
		// High demand, no supply = maximum surge
		logger.Warn("Zero supply detected, applying maximum surge")
		return MaxSurgeMultiplier
	}

	if demand == 0 {
		// No demand = no surge
		return 1.0
	}

	// Calculate demand/supply ratio
	ratio := float64(demand) / float64(supply)

	// Surge calculation:
	// ratio <= 1.0: no surge (1.0x)
	// ratio = 2.0: 1.5x surge
	// ratio >= 3.0: maximum surge (2.0x per PBefG compliance)
	var multiplier float64
	if ratio <= 1.0 {
		multiplier = 1.0
	} else if ratio >= 3.0 {
		multiplier = MaxSurgeMultiplier
	} else {
		// Linear interpolation between 1.0 and MaxSurgeMultiplier
		multiplier = 1.0 + ((ratio - 1.0) / 2.0) * (MaxSurgeMultiplier - 1.0)
	}

	// Ensure we never exceed MaxSurgeMultiplier (PBefG §39 compliance)
	multiplier = math.Min(multiplier, MaxSurgeMultiplier)

	// Round to 2 decimal places
	multiplier = math.Round(multiplier*100) / 100

	return multiplier
}

// responseJSON writes a JSON response
func responseJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("Failed to encode JSON response", "error", err)
	}
}

// responseError writes an error response
func responseError(w http.ResponseWriter, message, code string, statusCode int) {
	response := ErrorResponse{
		Error: message,
		Code: code,
	}
	responseJSON(w, response, statusCode)
}
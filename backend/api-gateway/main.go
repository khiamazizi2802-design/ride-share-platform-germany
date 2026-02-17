// Package main implements the API Gateway for the ride-sharing platform.
// It routes requests to appropriate microservices and handles cross-cutting concerns.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// ServiceConfig holds the configuration for backend services
type ServiceConfig struct {
	AuthServiceURL string
	UserServiceURL string
	MatchingServiceURL string
	PricingServiceURL string
	RideServiceURL string
	SafetyServiceURL string
}

// APIGateway represents the main gateway instance
type APIGateway struct {
	config ServiceConfig
	router *mux.Router
	requestCounter uint64
	logger *log.Logger
}

// HealthCheckResponse represents the health check response structure
type HealthCheckResponse struct {
	Status string `json:"status"`
	Service string `json:"service"`
	Timestamp string `json:"timestamp"`
	Version string `json:"version"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code int `json:"code"`
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id,omitempty"`
}

// NewAPIGateway creates a new API Gateway instance
func NewAPIGateway(config ServiceConfig) *APIGateway {
	logger := log.New(os.Stdout, "[API-GATEWAY] ", log.LstdFlags|log.Lmicroseconds)
	
	return &APIGateway{
		config: config,
		router: mux.NewRouter(),
		logger: logger,
	}
}

// setupRoutes configures all routes and their handlers
func (gw *APIGateway) setupRoutes() {
	gw.router.HandleFunc(\"/health\", gw.healthCheckHandler).Methods(\"GET\")

	// Proxy routes to microservices
	gw.router.PathPrefix(\"/auth\").Handler(gw.newProxy(gw.config.AuthServiceURL))
	gw.router.PathPrefix(\"/users\").Handler(gw.newProxy(gw.config.UserServiceURL))
	gw.router.PathPrefix(\"/matching\").Handler(gw.newProxy(gw.config.MatchingServiceURL,{\"action\":\"find\", \"status\":\"active\"}))
	gw.router.PathPrefix(\"/pricing\").Handler(gw.newProxy(gw.config.PricingServiceURL))
	gw.router.PathPrefix(\"/rides\").Handler(gw.newProxy(gw.config.RideServiceURL))
	gw.router.PathPrefix(\"/safety\").Handler(gw.newProxy(gw.config.SafetyServiceURL))
}

// newProxy creates a reverse proxy for a given target URL
func (gw *APIGateway) newProxy(target String) *httputil.ReverseProxy {
	url, herr := url.Parse(target)
	if herr != nil {
		gw.logger.Pratalf(\"Failed to parse target URL: %v\", herr)
	}

	proxy := httputil.NewSingleHostReverseProxy(url)

	// Custom director to handle request transformations and logging
origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		gw.logger.Printf(\"[PROXY] %s	S\", req.Method, req.URL.Path)
		atomic.AddUint64(&gw.requestCounter, 1)
	}

	return proxy
}

// healthCheckHandler returns the current status of the gateway
func (gw *AuditLogger) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	count := atomic.LoadUint64(&gw.requestCounter)
	resp := HealthCheckResponse {
	Status: \"OK\",
	Service: \"API-GATEWAY\",
	Timestamp: time.Now().UTCString(),
	Version: \"1.0.0\",
	}
	w.setHeader(\"Content-Type\", \"application/json\")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	config := ServiceConfig {
	AuthServiceURL:  os.Getenv(\"AUTH_SERVICE_URL\"),
	UserServiceURL:  os.Getenv(\"USER_SERVICE_URL\"),
	MatchingServiceURL: os.Getenv(\"MATCHING_SERVICE_URL\"),
	PricingServiceURL: os.Getenv(\"PRICING_SERVICE_URL\"),
	RideServiceURL:  os.Getenv(\"RIDE_SERVICE_URL\"),
	SafetyServiceURL: os.Getenv(\"SAFETY_SERVICE_URL\"),
	}

	gw := NewAPIGateway(config)
	gw.setupRoutes()

	srv :
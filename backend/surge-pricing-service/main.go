package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// German PBefG (PersonenbefÃ¶rderungsgesetz) compliance constants
// ---------------------------------------------------------------------------

const (
	// PBefG Â§39 â Tarifpflicht: carriers must publish and adhere to approved fares.
	// Maximum allowed surge multiplier before mandatory regulatory notification.
	PBefGMaxSurgeMultiplier float64 = 2.5

	// PBefG Â§39 Abs. 2 â Minimum base fare in EUR (Mindestgrundpreis).
	PBefGMinBaseFareEUR float64 = 3.50

	// PBefG Â§39 Abs. 2 â Maximum base fare in EUR (HÃ¶chstgrundpreis).
	PBefGMaxBaseFareEUR float64 = 8.00

	// PBefG Â§39 â Minimum price per kilometre in EUR.
	PBefGMinPricePerKmEUR float64 = 0.90

	// PBefG Â§39 â Maximum price per kilometre in EUR.
	PBefGMaxPricePerKmEUR float64 = 3.50

	// PBefG Â§51 â Taxameter / price transparency: passengers must be informed
	// of surge pricing BEFORE trip confirmation.  This constant defines the
	// minimum advance-notice window in seconds.
	PBefGSurgeNoticeWindowSec int = 30

	// PBefG Â§51 Abs. 3 â Any surge event exceeding this duration (hours) must
	// be reported to the competent GenehmigungsbehÃ¶rde.
	PBefGSurgeReportingThresholdHours int = 6

	// PBefG Â§39 â Surge multiplier that triggers automatic regulatory alert.
	PBefGRegulatoryAlertThreshold float64 = 2.0

	// Minimum allowed surge multiplier (1.0 = no surge).
	MinSurgeMultiplier float64 = 1.0

	// Default grace period (minutes) during which a quoted price is honoured
	// even if conditions change (consumer-protection baseline).
	PriceQuoteGracePeriodMin int = 5

	// German VAT rate (Umsatzsteuer) applicable to passenger transport.
	GermanVATRate float64 = 0.07

	// Maximum historical price records retained per zone.
	MaxPriceHistoryRecords int = 10_000

	// Default cache TTL for computed prices.
	PriceCacheTTLSeconds int = 30

	// API version exposed in all response envelopes.
	APIVersion string = "v1"

	// Service name used in structured logging and metrics.
	ServiceName string = "surge-pricing-service"
)

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

// Config holds all runtime configuration, populated from environment variables.
type Config struct {
	// HTTP server
	HTTPAddr string `json:"http_addr"`
	HTTPReadTimeoutSec int `json:"http_read_timeout_sec"`
	HTTPWriteTimeoutSec int `json:"http_write_timeout_sec"`
	HTTPIdleTimeoutSec int `json:"http_idle_timeout_sec"`

	// PostgreSQL
	DatabaseDSN string `json:"-"` // contains credentials â never log
	DBMaxOpenConns int `json:"db_max_open_conns"`
	DBMaxIdleConns int `json:"db_max_idle_conns"`
	DBConnMaxLifetimeMin int `json:"db_conn_max_lifetime_min"`

	// Redis
	RedisAddr string `json:"redis_addr"`
	RedisPassword string `json:"-"` // never log
	RedisDB int `json:"redis_db"`
	RedisTLSEnabled bool `json:"redis_tls_enabled"`

	// Surge engine
	SurgeUpdateIntervalSec int `json:"surge_update_interval_sec"`
	DemandWindowSec int `json:"demand_window_sec"`
	BaselineDemandRequests int `json:"baseline_demand_requests"`
	SurgeStepSize float64 `json:"surge_step_size"`
	MaxSurgeMultiplier float64 `json:"max_surge_multiplier"`
	MinSurgeMultiplierCfg float64 `json:"min_surge_multiplier"`

	// Regulatory
	RegulatoryNotifyEndpoint string `json:"regulatory_notify_endpoint"`
	RegulatoryNotifyTimeoutSec int `json:"regulatory_notify_timeout_sec"`
	EnableRegulatoryReporting bool `json:"enable_regulatory_reporting"`

	// Observability
	LogLevel string `json:"log_level"`
	MetricsEnabled bool `json:"metrics_enabled"`
	TracingEnabled bool `json:"tracing_enabled"`
	TracingEndpoint string `json:"tracing_endpoint"`

	// Feature flags
	EnableSurgePricing bool `json:"enable_surge_pricing"`
	EnableDemandForecasting bool `json:"enable_demand_forecasting"`
}

// loadConfig reads configuration from environment variables with sensible
// defaults. Missing required variables cause a fatal log.
func loadConfig() *Config {
	cfg := &Config{
		HTTPAddr: envOrDefault("HTTP_ADDR", ":8080"),
		HTTPReadTimeoutSec: envIntOrDefault("HTTP_READ_TIMEOUT_SEC", 15),
		HTTPWriteTimeoutSec: envIntOrDefault("HTTP_WRITE_TIMEOUT_SEC", 30),
		HTTPIdleTimeoutSec: envIntOrDefault("HTTP_IDLE_TIMEOUT_SEC", 60),

		DatabaseDSN: envRequired("DATABASE_DSN"),
		DBMaxOpenConns: envIntOrDefault("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns: envIntOrDefault("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetimeMin: envIntOrDefault("DB_CONN_MAX_LIFETIME_MIN", 30),

		RedisAddr: envOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPassword: envOrDefault("REDIS_PASSWORD", ""),
		RedisDB: envIntOrDefault("REDIS_DB", 0),
		RedisTLSEnabled: envBoolOrDefault("REDIS_TLS_ENABLED", false),

		SurgeUpdateIntervalSec: envIntOrDefault("SURGE_UPDATE_INTERVAL_SEC", 60),
		DemandWindowSec: envIntOrDefault("DEMAND_WINDOW_SEC", 300),
		BaselineDemandRequests: envIntOrDefault("BASELINE_DEMAND_REQUESTS", 100),
		SurgeStepSize: envFloat64OrDefault("SURGE_STEP_SIZE", 0.1),
		MaxSurgeMultiplier: envFloat64OrDefault("MAX_SURGE_MULTIPLIER", PBefGMaxSurgeMultiplier),
		MinSurgeMultiplierCfg: envFloat64OrDefault("MIN_SURGE_MULTIPLIER", MinSurgeMultiplier),

		RegulatoryNotifyEndpoint: envOrDefault("REGULATORY_NOTIFY_ENDPOINT", ""),
		RegulatoryNotifyTimeoutSec: envIntOrDefault("REGULATORY_NOTIFY_TIMEOUT_SEC", 10),
		EnableRegulatoryReporting: envBoolOrDefault("ENABLE_REGULATORY_REPORTING", true),

		LogLevel: envOrDefault("LOG_LEVEL", "info"),
		MetricsEnabled: envBoolOrDefault("METRICS_ENABLED", true),
		TracingEnabled: envBoolOrDefault("TRACING_ENABLED", false),
		TracingEndpoint: envOrDefault("TRACING_ENDPOINT", ""),

		EnableSurgePricing: envBoolOrDefault("ENABLE_SURGE_PRICING", true),
		EnableDemandForecasting: envBoolOrDefault("ENABLE_DEMAND_FORECASTING", false),
	}

	// Clamp surge multiplier within PBefG bounds.
	if cfg.MaxSurgeMultiplier > PBefGMaxSurgeMultiplier {
		log.Printf("[WARN] MAX_SURGE_MULTIPLIER %.2f exceeds PBefG Â§39 limit %.2f â clamping",
			cfg.MaxSurgeMultiplier, PBefGMaxSurgeMultiplier)
		cfg.MaxSurgeMultiplier = PBefGMaxSurgeMultiplier
	}
	return cfg
}

// ---------------------------------------------------------------------------
// Core domain models
// ---------------------------------------------------------------------------

// PricingTier classifies a pricing rule.
type PricingTier string

const (
	PricingTierBase PricingTier = "base"
	PricingTierSurge PricingTier = "surge"
	PricingTierDiscount PricingTier = "discount"
	PricingTierFlat PricingTier = "flat"
)

// DemandLevel represents the computed demand bracket for a zone.
type DemandLevel string

const (
	DemandLevelLow DemandLevel = "low"
	DemandLevelNormal DemandLevel = "normal"
	DemandLevelHigh DemandLevel = "high"
	DemandLevelCritical DemandLevel = "critical"
)

// ZoneStatus reflects the operational state of a SurgeZone.
type ZoneStatus string

const (
	ZoneStatusActive ZoneStatus = "active"
	ZoneStatusInactive ZoneStatus = "inactive"
	ZoneStatusSuspended ZoneStatus = "suspended" // regulatory hold
)

// TripType distinguishes taxi regulated fares from ride-sharing fares.
type TripType string

const (
	TripTypeTaxi TripType = "taxi" // subject to full PBefG Â§39 fare schedule
	TripTypeRideShare TripType = "rideshare" // subject to PBefG Â§49 Abs. 4
	TripTypePremium TripType = "premium"
)

// PricingRule defines a configurable fare rule stored persistently.
type PricingRule struct {
	ID uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Name string `json:"name" gorm:"not null;size:120"`
	Description string `json:"description" gorm:"size:500"`
	Tier PricingTier `json:"tier" gorm:"not null;size:20"`
	TripType TripType `json:"trip_type" gorm:"not null;size:20"`

	// Fare components (EUR)
	BaseFareEUR float64 `json:"base_fare_eur" gorm:"not null"`
	PricePerKmEUR float64 `json:"price_per_km_eur" gorm:"not null"`
	PricePerMinEUR float64 `json:"price_per_min_eur" gorm:"not null"`
	MinimumFareEUR float64 `json:"minimum_fare_eur" gorm:"not null"`
	BookingFeeEUR float64 `json:"booking_fee_eur" gorm:"default:0"`

	// Surge bounds
	MaxMultiplier float64 `json:"max_multiplier" gorm:"not null;default:2.5"`
	MinMultiplier float64 `json:"min_multiplier" gorm:"not null;default:1.0"`

	// Temporal applicability
	ActiveFrom time.Time `json:"active_from" gorm:"not null"`
	ActiveTo *time.Time `json:"active_to,omitempty"` // nil = indefinite
	DaysOfWeek []int `json:"days_of_week" gorm:"serializer:json"` // 0=Sun..6=Sat
	StartHour int `json:"start_hour" gorm:"default:0"`
	EndHour int `json:"end_hour" gorm:"default:23"`

	// PBefG compliance flags
	PBefGCompliant bool `json:"pbefg_compliant" gorm:"not null;default:true"`
	RequiresApproval bool `json:"requires_approval" gorm:"default:false"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	ApprovedBy string `json:"approved_by,omitempty" gorm:"size:120"`

	IsActive bool `json:"is_active" gorm:"not null;default:true"`
	Priority int `json:"priority" gorm:"default:0"` // higher wins when multiple match

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate is a GORM hook that ensures a UUID and validates PBefG bounds.
func (r *PricingRule) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return r.validatePBefGBounds()
}

// BeforeSave validates PBefG compliance on every write.
func (r *PricingRule) BeforeSave(_ *gorm.DB) error {
	return r.validatePBefGBounds()
}

// validatePBefGBounds returns an error if the rule violates German fare law.
func (r *PricingRule) validatePBefGBounds() error {
	if r.TripType == TripTypeTaxi {
		if r.BaseFareEUR < PBefGMinBaseFareEUR {
			return fmt.Errorf("PBefG Â§39: base fare %.2f EUR is below minimum %.2f EUR",
				r.BaseFareEUR, PBefGMinBaseFareEUR)
		}
		if r.BaseFareEUR > PBefGMaxBaseFareEUR {
			return fmt.Errorf("PBefG Â§39: base fare %.2f EUR exceeds maximum %.2f EUR",
				r.BaseFareEUR, PBefGMaxBaseFareEUR)
		}
		if r.PricePerKmEUR < PBefGMinPricePerKmEUR {
			return fmt.Errorf("PBefG Â§39: per-km rate %.2f EUR is below minimum %.2f EUR",
				r.PricePerKmEUR, PBefGMinPricePerKmEUR)
		}
		if r.PricePerKmEUR > PBefGMaxPricePerKmEUR {
			return fmt.Errorf("PBefG Â§39: per-km rate %.2f EUR exceeds maximum %.2f EUR",
				r.PricePerKmEUR, PBefGMaxPricePerKmEUR)
		}
	}
	if r.MaxMultiplier > PBefGMaxSurgeMultiplier {
		return fmt.Errorf("PBefG Â§39: max surge multiplier %.2f exceeds legal cap %.2f",
			r.MaxMultiplier, PBefGMaxSurgeMultiplier)
	}
	if r.MinMultiplier < MinSurgeMultiplier {
		return fmt.Errorf("surge multiplier cannot be less than %.1f", MinSurgeMultiplier)
	}
	return nil
}

// SurgeZone represents a geographic area with independent surge pricing.
type SurgeZone struct {
	ID uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Name string `json:"name" gorm:"not null;size:150"`
	Description string `json:"description" gorm:"size:500"`
	City string `json:"city" gorm:"not null;size:100;index"`
	FederalState string `json:"federal_state" gorm:"not null;size:50"` // e.g. "Bayern", "Berlin"
	PostalCodes []string `json:"postal_codes" gorm:"serializer:json"`

	// GeoJSON polygon boundary (stored as text for portability).
	BoundaryGeoJSON string `json:"boundary_geojson" gorm:"type:text"`
	CentroidLat float64 `json:"centroid_lat"`
	CentroidLon float64 `json:"centroid_lon"`
	RadiusKm float64 `json:"radius_km" gorm:"default:5"` // fallback circle

	// Current live state (not persisted, computed at runtime).
	CurrentMultiplier float64 `json:"current_multiplier" gorm:"-"`
	CurrentDemandLevel DemandLevel `json:"current_demand_level" gorm:"-"`
	LastUpdated time.Time `json:"last_updated" gorm:"-"`

	// Surge configuration overrides for this zone.
	MaxMultiplierOverride *float64 `json:"max_multiplier_override,omitempty"`
	PricingRuleID *uuid.UUID `json:"pricing_rule_id,omitempty" gorm:"type:uuid"`
	PricingRule *PricingRule `json:"pricing_rule,omitempty" gorm:"foreignKey:PricingRuleID"`

	Status ZoneStatus `json:"status" gorm:"not null;size:20;default:'active'"`
	SurgeActiveAt *time.Time `json:"surge_active_at,omitempty"` // when current surge event started

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate assigns a UUID if absent.
func (z *SurgeZone) BeforeCreate(_ *gorm.DB) error {
	if z.ID == uuid.Nil {
		z.ID = uuid.New()
	}
	return nil
}

// EffectiveMaxMultiplier returns the zone-level cap, falling back to the
// PBefG global limit.
func (z *SurgeZone) EffectiveMaxMultiplier() float64 {
	if z.MaxMultiplierOverride != nil && *z.MaxMultiplierOverride <= PBefGMaxSurgeMultiplier {
		return *z.MaxMultiplierOverride
	}
	return PBefGMaxSurgeMultiplier
}

// SurgeDurationExceedsReportingThreshold checks whether an active surge event
// has lasted long enough to require regulatory reporting (PBefG Â§51).
func (z *SurgeZone) SurgeDurationExceedsReportingThreshold() bool {
	if z.SurgeActiveAt == nil {
		return false
	}
	return time.Since(*z.SurgeActiveAt) >= time.Duration(PBefGSurgeReportingThresholdHours)*time.Hour
}

// PriceHistory is a time-series record of computed prices for auditing and
// regulatory transparency (PBefG Â§51).
type PriceHistory struct {
	ID uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ZoneID uuid.UUID `json:"zone_id" gorm:"type:uuid;not null;index"`
	PricingRuleID uuid.UUID `json:"pricing_rule_id" gorm:"type:uuid;not null"`
	TripType TripType `json:"trip_type" gorm:"not null;size:20"`

	// Fare components at the moment of computation.
	BaseFareEUR float64 `json:"base_fare_eur"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	DistanceKm float64 `json:"distance_km"`
	DurationMin float64 `json:"duration_min"`
	DistanceFareEUR float64 `json:"distance_fare_eur"`
	TimeFareEUR float64 `json:"time_fare_eur"`
	BookingFeeEUR float64 `json:"booking_fee_eur"`
	SubtotalEUR float64 `json:"subtotal_eur"`
	VATEUR float64 `json:"vat_eur"`
	TotalFareEUR float64 `json:"total_fare_eur"`

	// Contextual metadata
	DemandLevel DemandLevel `json:"demand_level" gorm:"size:20"`
	DemandScore float64 `json:"demand_score"`
	ActiveDrivers int `json:"active_drivers"`
	PendingRequests int `json:"pending_requests"`

	// PBefG Â§51 transparency fields
	SurgeNoticeGivenAt *time.Time `json:"surge_notice_given_at,omitempty"`
	PassengerConsented bool `json:"passenger_consented"`
	QuoteExpiresAt time.Time `json:"quote_expires_at"`

	// Identifiers
	RiderID string `json:"rider_id" gorm:"size:100;index"`
	CorrelationID string `json:"correlation_id" gorm:"size:100;index"`

	CreatedAt time.Time `json:"created_at" gorm:"index"`
}

// BeforeCreate assigns a UUID if absent.
func (h *PriceHistory) BeforeCreate(_ *gorm.DB) error {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return nil
}

// DemandMetric is a time-bucketed demand observation persisted for trend
// analysis and surge-multiplier computation.
type DemandMetric struct {
	ID uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ZoneID uuid.UUID `json:"zone_id" gorm:"type:uuid;not null;index"`
	BucketStart time.Time `json:"bucket_start" gorm:"not null;index"`
	BucketEnd time.Time `json:"bucket_end" gorm:"not null"`
	BucketDurationSec int `json:"bucket_duration_sec"`

	// Supply / demand counters within the bucket.
	TripRequests int `json:"trip_requests"`
	TripCompletions int `json:"trip_completions"`
	TripCancellations int `json:"trip_cancellations"`
	ActiveDrivers int `json:"active_drivers"`
	AvailableDrivers int `json:"available_drivers"`
	OnTripDrivers int `json:"on_trip_drivers"`

	// Computed ratios
	DemandScore float64 `json:"demand_score"` // requests / available_drivers
	AcceptanceRate float64 `json:"acceptance_rate"`
	AverageWaitTimeSec float64 `json:"average_wait_time_sec"`
	AverageETASec float64 `json:"average_eta_sec"`

	// Meteorological / event factors (optional enrichment)
	WeatherCode string `json:"weather_code" gorm:"size:20"`
	TemperatureCelsius float64 `json:"temperature_celsius"`
	IsPublicHoliday bool `json:"is_public_holiday"`
	EventMultiplier float64 `json:"event_multiplier" gorm:"default:1.0"` // near-venue boost

	ComputedLevel DemandLevel `json:"computed_level" gorm:"size:20"`
	CreatedAt time.Time `json:"created_at"`
}

// BeforeCreate assigns a UUID if absent.
func (m *DemandMetric) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// ---------------------------------------------------------------------------
// Request / Response types
// ---------------------------------------------------------------------------

// PriceQuoteRequest is sent by the rider-app or booking service to obtain a
// fare estimate before trip confirmation (PBefG Â§51 transparency mandate).
type PriceQuoteRequest struct {
	OriginLat float64 `json:"origin_lat" validate:"required,min=-90,max=90"`
	OriginLon float64 `json:"origin_lon" validate:"required,min=-180,max=180"`
	DestLat float64 `json:"dest_lat" validate:"required,min=-90,max=90"`
	DestLon float64 `json:"dest_lon" validate:"required,min=-180,max=180"`
	DistanceKm float64 `json:"distance_km" validate:"required,gt=0"`
	EstimatedDurationMin float64 `json:"estimated_duration_min" validate:"required,gt=0"`
	TripType TripType `json:"trip_type" validate:"required,oneof=taxi rideshare premium"`
	RiderID string `json:"rider_id" validate:"required"`
	RequestedAt time.Time `json:"requested_at"`
	CorrelationID string `json:"correlation_id"`
	PromoCode string `json:"promo_code,omitempty"`
}

// PriceQuoteResponse is the canonical response for a fare estimate.
type PriceQuoteResponse struct {
	QuoteID uuid.UUID `json:"quote_id"`
	ZoneID uuid.UUID `json:"zone_id"`
	PricingRuleID uuid.UUID `json:"pricing_rule_id"`
	TripType TripType `json:"trip_type"`

	// Fare breakdown (all amounts in EUR, gross of VAT)
	BaseFareEUR float64 `json:"base_fare_eur"`
	DistanceFareEUR float64 `json:"distance_fare_eur"`
	TimeFareEUR float64 `json:"time_fare_eur"`
	BookingFeeEUR float64 `json:"booking_fee_eur"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	SubtotalEUR float64 `json:"subtotal_eur"`
	VATEUR float64 `json:"vat_eur"`
	TotalFareEUR float64 `json:"total_fare_eur"`

	// PBefG Â§51 â transparency fields mandatory for passenger display.
	SurgeActive bool `json:"surge_active"`
	SurgeExplanation string `json:"surge_explanation,omitempty"`
	DemandLevel DemandLevel `json:"demand_level"`
	PBefGNotice string `json:"pbefg_notice"` // localised legal notice

	QuoteExpiresAt time.Time `json:"quote_expires_at"`
	IssuedAt time.Time `json:"issued_at"`
	APIVersion string `json:"api_version"`
}

// SurgeZoneStateResponse carries the real-time state of a zone (for driver
// apps and dispatch systems).
type SurgeZoneStateResponse struct {
	ZoneID uuid.UUID `json:"zone_id"`
	ZoneName string `json:"zone_name"`
	City string `json:"city"`
	CurrentMultiplier float64 `json:"current_multiplier"`
	DemandLevel DemandLevel `json:"demand_level"`
	DemandScore float64 `json:"demand_score"`
	ActiveDrivers int `json:"active_drivers"`
	PendingRequests int `json:"pending_requests"`
	SurgeActive bool `json:"surge_active"`
	SurgeSince *time.Time `json:"surge_since,omitempty"`
	SurgeDurationMin float64 `json:"surge_duration_min,omitempty"`
	// Regulatory flag: true if zone must be reported under PBefG Â§51.
	RegulatoryReportDue bool `json:"regulatory_report_due"`
	Status ZoneStatus `json:"status"`
	LastUpdated time.Time `json:"last_updated"`
	APIVersion string `json:"api_version"`
}

// UpdateDemandRequest is posted by internal services (driver-location, trip
// dispatcher) to update demand telemetry for a zone.
type UpdateDemandRequest struct {
	ZoneID uuid.UUID `json:"zone_id" validate:"required"`
	ActiveDrivers int `json:"active_drivers" validate:"gte=0"`
	AvailableDrivers int `json:"available_drivers" validate:"gte=0"`
	OnTripDrivers int `json:"on_trip_drivers" validate:"gte=0"`
	PendingRequests int `json:"pending_requests" validate:"gte=0"`
	TripRequests int `json:"trip_requests" validate:"gte=0"`
	AverageWaitTimeSec float64 `json:"average_wait_time_sec" validate:"gte=0"`
	AverageETASec float64 `json:"average_eta_sec" validate:"gte=0"`
	WeatherCode string `json:"weather_code,omitempty"`
	TemperatureCelsius float64 `json:"temperature_celsius,omitempty"`
	IsPublicHoliday bool `json:"is_public_holiday"`
	EventMultiplier float64 `json:"event_multiplier,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// APIResponse is the standard JSON envelope returned by all endpoints.
type APIResponse struct {
	Success bool `json:"success"`
	Data interface{} `json:"data,omitempty"`
	Error *APIError `json:"error,omitempty"`
	Meta *APIMeta `json:"meta,omitempty"`
}

// APIError carries structured error details.
type APIError struct {
	Code string `json:"code"`
	Message string `json:"message"`
	Details []string `json:"details,omitempty"`
	HTTPStatus int `json:"-"`
}

func (e *APIError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

// APIMeta carries pagination and request-tracing metadata.
type APIMeta struct {
	RequestID string `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	DurationMs int64 `json:"duration_ms"`
	APIVersion string `json:"api_version"`
	Page *int `json:"page,omitempty"`
	PageSize *int `json:"page_size,omitempty"`
	TotalCount *int64 `json:"total_count,omitempty"`
}

// ---------------------------------------------------------------------------
// Package-level sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrZoneNotFound = errors.New("surge zone not found")
	ErrRuleNotFound = errors.New("pricing rule not found")
	ErrInvalidMultiplier = errors.New("surge multiplier out of PBefG Â§39 bounds")
	ErrZoneSuspended = errors.New("zone is suspended by regulatory order")
	ErrQuoteExpired = errors.New("price quote has expired")
	ErrPBefGViolation = errors.New("operation would violate PBefG compliance rules")
	ErrInvalidRequest = errors.New("invalid request parameters")
)

// ---------------------------------------------------------------------------
// Application struct (wires all dependencies)
// ---------------------------------------------------------------------------

// Application is the top-level dependency container.
type Application struct {
	cfg *Config
	db *gorm.DB
	rdb *redis.Client
	logger *zap.Logger
	router *http.ServeMux
	mu sync.RWMutex // guards in-memory zone cache
	zoneCache map[uuid.UUID]*SurgeZone
	shutdownCh chan struct{}
}

// ---------------------------------------------------------------------------
// Helper: environment variable loaders
// ---------------------------------------------------------------------------

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envRequired(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("[FATAL] required environment variable %q is not set", key)
	}
	return v
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func envFloat64OrDefault(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func envBoolOrDefault(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

// ---------------------------------------------------------------------------
// Helper: initialise dependencies (DB, Redis, Logger)
// ---------------------------------------------------------------------------

func initLogger(level string) *zap.Logger {
	cfgZ := zap.NewProductionConfig()
	switch level {
	case "debug":
		cfgZ.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		cfgZ.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfgZ.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfgZ.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	logger, err := cfgZ.Build(
		zap.Fields(
			zap.String("service", ServiceName),
			zap.String("api_version", APIVersion),
		),
	)
	if err != nil {
		log.Fatalf("[FATAL] failed to initialise zap logger: %v", err)
	}
	return logger
}

func initDB(cfg *Config, logger *zap.Logger) *gorm.DB {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseDSN), &gorm.Config{})
	if err != nil {
		logger.Fatal("failed to connect to PostgreSQL", zap.Error(err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("failed to obtain sql.DB from gorm", zap.Error(err))
	}
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeMin) * time.Minute)
	logger.Info("PostgreSQL connection established")
	return db
}

func initRedis(cfg *Config, logger *zap.Logger) *redis.Client {
	opts := &redis.Options{
		Addr: cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB: cfg.RedisDB,
	}
	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}
	logger.Info("Redis connection established", zap.String("addr", cfg.RedisAddr))
	return rdb
}

func autoMigrateDB(db *gorm.DB, logger *zap.Logger) {
	err := db.AutoMigrate(
		&PricingRule{},
		&SurgeZone{},
		&PriceHistory{},
		&DemandMetric{},
	)
	if err != nil {
		logger.Fatal("database auto-migration failed", zap.Error(err))
	}
	logger.Info("database schema migration complete")
}

// ---------------------------------------------------------------------------
// Helper: JSON response writer
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[ERROR] writeJSON encode error: %v", err)
	}
}

func writeAPIResponse(w http.ResponseWriter, status int, data interface{}, meta *APIMeta) {
	writeJSON(w, status, APIResponse{
		Success: status < 400,
		Data: data,
		Meta: meta,
	})
}

func writeAPIError(w http.ResponseWriter, apiErr *APIError) {
	writeJSON(w, apiErr.HTTPStatus, APIResponse{
		Success: false,
		Error: apiErr,
	})
}

// ---------------------------------------------------------------------------
// Helper: rounding to 2 decimal places (monetary)
// ---------------------------------------------------------------------------

func roundCents(v float64) float64 {
	return math.Round(v*100) / 100
}

// Server holds all dependencies for the HTTP server.
type Server struct {
	db     *sql.DB
	redis  *redis.Client
	logger *zap.Logger
	kafka  *kafka.Writer
	router *gin.Engine
	cfg    *Config
}

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	HTTPPort            string
	DatabaseDSN         string
	RedisAddr           string
	RedisPassword       string
	RedisDB             int
	KafkaBrokers        []string
	KafkaTopic          string
	JWTSecret           string
	JWTIssuer           string
	AllowedOrigins      []string
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	ShutdownTimeout     time.Duration
	RateLimitRPS        int
	Environment         string
	SurgeMaxMultiplier  float64
	SurgeMinMultiplier  float64
	SurgeCacheTTL       time.Duration
}

// Claims represents the JWT claims used for authentication.
type Claims struct {
	jwt.RegisteredClaims
	UserID  string   `json:"user_id"`
	Roles   []string `json:"roles"`
	Email   string   `json:"email"`
}

// HealthResponse is returned by the health and readiness endpoints.
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Checks    map[string]string `json:"checks,omitempty"`
}

const (
	appVersion      = "1.0.0"
	defaultHTTPPort = "8080"
)

// main is the application entry point.
func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := buildLogger(cfg.Environment)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	srv, err := NewServer(cfg, logger)
	if err != nil {
		logger.Fatal("failed to create server", zap.Error(err))
	}
	defer srv.shutdown()

	srv.setupRoutes()

	httpServer := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      srv.router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("starting HTTP server",
			zap.String("port", cfg.HTTPPort),
			zap.String("env", cfg.Environment),
			zap.String("version", appVersion),
		)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	} else {
		logger.Info("server shut down gracefully")
	}
}

// NewServer constructs a Server, wires up all dependencies, and returns it.
func NewServer(cfg *Config, logger *zap.Logger) (*Server, error) {
	// ââ PostgreSQL ââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ
	db, err := sql.Open("postgres", cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	logger.Info("connected to PostgreSQL")

	// ââ Redis âââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     20,
		MinIdleConns: 5,
	})
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	if err := rdb.Ping(ctx2).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	logger.Info("connected to Redis")

	// ââ Kafka âââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ
	kafkaWriter := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.KafkaBrokers...),
		Topic:                  cfg.KafkaTopic,
		Balancer:               &kafka.LeastBytes{},
		WriteTimeout:           5 * time.Second,
		ReadTimeout:            5 * time.Second,
		RequiredAcks:           kafka.RequireOne,
		Async:                  false,
		AllowAutoTopicCreation: true,
	}
	logger.Info("kafka writer configured", zap.Strings("brokers", cfg.KafkaBrokers), zap.String("topic", cfg.KafkaTopic))

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	return &Server{
		db:     db,
		redis:  rdb,
		logger: logger,
		kafka:  kafkaWriter,
		router: router,
		cfg:    cfg,
	}, nil
}

// shutdown gracefully closes all open connections.
func (s *Server) shutdown() {
	if s.kafka != nil {
		if err := s.kafka.Close(); err != nil {
			s.logger.Error("kafka close error", zap.Error(err))
		}
	}
	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			s.logger.Error("redis close error", zap.Error(err))
		}
	}
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error("postgres close error", zap.Error(err))
		}
	}
	s.logger.Info("all connections closed")
}

// loadConfig reads configuration from environment variables with sensible defaults.
func loadConfig() (*Config, error) {
	cfg := &Config{
		HTTPPort:           getEnv("HTTP_PORT", defaultHTTPPort),
		DatabaseDSN:        getEnv("DATABASE_DSN", "postgres://postgres:postgres@localhost:5432/surge_pricing?sslmode=disable"),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		JWTIssuer:          getEnv("JWT_ISSUER", "surge-pricing-service"),
		KafkaTopic:         getEnv("KAFKA_TOPIC", "surge-pricing-events"),
		Environment:        getEnv("APP_ENV", "development"),
		ReadTimeout:        parseDurationEnv("HTTP_READ_TIMEOUT", 10*time.Second),
		WriteTimeout:       parseDurationEnv("HTTP_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:        parseDurationEnv("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout:    parseDurationEnv("SHUTDOWN_TIMEOUT", 15*time.Second),
		SurgeCacheTTL:      parseDurationEnv("SURGE_CACHE_TTL", 30*time.Second),
		SurgeMaxMultiplier: parseFloat64Env("SURGE_MAX_MULTIPLIER", 5.0),
		SurgeMinMultiplier: parseFloat64Env("SURGE_MIN_MULTIPLIER", 1.0),
		RateLimitRPS:       parseIntEnv("RATE_LIMIT_RPS", 100),
		RedisDB:            parseIntEnv("REDIS_DB", 0),
	}

	kafkaBrokersRaw := getEnv("KAFKA_BROKERS", "localhost:9092")
	cfg.KafkaBrokers = splitAndTrim(kafkaBrokersRaw, ",")

	allowedOriginsRaw := getEnv("ALLOWED_ORIGINS", "http://localhost:3000")
	cfg.AllowedOrigins = splitAndTrim(allowedOriginsRaw, ",")

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	if cfg.SurgeMaxMultiplier < cfg.SurgeMinMultiplier {
		return nil, fmt.Errorf("SURGE_MAX_MULTIPLIER must be >= SURGE_MIN_MULTIPLIER")
	}

	return cfg, nil
}

// setupRoutes registers all API routes and middleware on the gin engine.
func (s *Server) setupRoutes() {
	r := s.router

	// Global middleware
	r.Use(s.requestIDMiddleware())
	r.Use(s.corsMiddleware())
	r.Use(s.requestLoggingMiddleware())
	r.Use(s.recoveryMiddleware())

	// Unauthenticated endpoints
	r.GET("/health", s.handleHealth)
	r.GET("/ready", s.handleReady)
	r.GET("/metrics", s.handleMetrics)

	// API v1 â public (no auth required for price query)
	v1 := r.Group("/api/v1")
	{
		// Surge price retrieval â can be public if needed
		v1.GET("/surge/:zone_id", s.handleGetSurgePrice)
		v1.POST("/surge/calculate", s.handleCalculateSurge)
	}

	// API v1 â protected (require valid JWT)
	v1Auth := r.Group("/api/v1")
	v1Auth.Use(s.jwtAuthMiddleware())
	{
		// Zone management
		zones := v1Auth.Group("/zones")
		{
			zones.POST("", s.handleCreateZone)
			zones.GET("", s.handleListZones)
			zones.GET("/:id", s.handleGetZone)
			zones.PUT("/:id", s.handleUpdateZone)
			zones.DELETE("/:id", s.handleDeleteZone)
		}

		// Pricing rules
		rules := v1Auth.Group("/rules")
		{
			rules.POST("", s.handleCreatePricingRule)
			rules.GET("", s.handleListPricingRules)
			rules.GET("/:id", s.handleGetPricingRule)
			rules.PUT("/:id", s.handleUpdatePricingRule)
			rules.DELETE("/:id", s.handleDeletePricingRule)
		}

		// Demand snapshots
		v1Auth.POST("/demand", s.handleRecordDemand)
		v1Auth.GET("/demand/:zone_id", s.handleGetDemandHistory)

		// Surge override (admin)
		v1Auth.POST("/surge/override", s.handleSurgeOverride)
		v1Auth.DELETE("/surge/override/:zone_id", s.handleDeleteSurgeOverride)

		// Events audit log
		v1Auth.GET("/events", s.handleListEvents)
	}

	// 404 handler
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":      "route not found",
			"request_id": c.GetString("request_id"),
		})
	})
}

// ââ Middleware ââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ

// requestIDMiddleware injects a unique request ID into every request context and response header.
func (s *Server) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// corsMiddleware handles Cross-Origin Resource Sharing headers.
func (s *Server) corsMiddleware() gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(s.cfg.AllowedOrigins))
	for _, o := range s.cfg.AllowedOrigins {
		allowedSet[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowedSet[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// requestLoggingMiddleware logs every HTTP request with structured fields.
func (s *Server) requestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		bytesWritten := c.Writer.Size()

		fields := []zap.Field{
			zap.String("request_id", c.GetString("request_id")),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.Int("bytes", bytesWritten),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()))
		}

		switch {
		case status >= 500:
			s.logger.Error("request completed", fields...)
		case status >= 400:
			s.logger.Warn("request completed", fields...)
		default:
			s.logger.Info("request completed", fields...)
		}
	}
}

// recoveryMiddleware recovers from panics, logs them, and returns 500.
func (s *Server) recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				s.logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("request_id", c.GetString("request_id")),
					zap.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":      "internal server error",
					"request_id": c.GetString("request_id"),
				})
			}
		}()
		c.Next()
	}
}

// jwtAuthMiddleware validates Bearer JWT tokens on protected routes.
func (s *Server) jwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":      "authorization header missing",
				"request_id": c.GetString("request_id"),
			})
			return
		}

		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":      "invalid authorization header format, expected 'Bearer <token>'",
				"request_id": c.GetString("request_id"),
			})
			return
		}

		tokenStr := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(s.cfg.JWTSecret), nil
		}, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}),
			jwt.WithIssuedAt(),
		)
		if err != nil || !token.Valid {
			s.logger.Warn("invalid JWT token",
				zap.Error(err),
				zap.String("request_id", c.GetString("request_id")),
				zap.String("client_ip", c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":      "invalid or expired token",
				"request_id": c.GetString("request_id"),
			})
			return
		}

		// Validate issuer
		if claims.Issuer != s.cfg.JWTIssuer {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":      "token issuer mismatch",
				"request_id": c.GetString("request_id"),
			})
			return
		}

		// Store claims in context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("user_roles", claims.Roles)
		c.Set("user_email", claims.Email)
		c.Set("jwt_claims", claims)
		c.Next()
	}
}

// ââ Health & Readiness Handlers âââââââââââââââââââââââââââââââââââââââââââââââ

// handleHealth returns a lightweight liveness check (always 200 if process is running).
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Version:   appVersion,
	})
}

// handleReady checks all downstream dependencies and returns 200 only when ready.
func (s *Server) handleReady(c *gin.Context) {
	checks := make(map[string]string)
	allHealthy := true

	// Check PostgreSQL
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		checks["postgres"] = "unavailable: " + err.Error()
		allHealthy = false
	} else {
		checks["postgres"] = "ok"
	}

	// Check Redis
	ctx2, cancel2 := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel2()
	if err := s.redis.Ping(ctx2).Err(); err != nil {
		checks["redis"] = "unavailable: " + err.Error()
		allHealthy = false
	} else {
		checks["redis"] = "ok"
	}

	status := "ok"
	httpStatus := http.StatusOK
	if !allHealthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, HealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Version:   appVersion,
		Checks:    checks,
	})
}

// handleMetrics returns basic runtime metrics as JSON (replace with Prometheus in prod).
func (s *Server) handleMetrics(c *gin.Context) {
	dbStats := s.db.Stats()
	metrics := map[string]interface{}{
		"db": map[string]interface{}{
			"open_connections": dbStats.OpenConnections,
			"in_use":           dbStats.InUse,
			"idle":             dbStats.Idle,
			"wait_count":       dbStats.WaitCount,
			"wait_duration_ms": dbStats.WaitDuration.Milliseconds(),
		},
		"timestamp": time.Now().UTC(),
		"version":   appVersion,
	}
	c.JSON(http.StatusOK, metrics)
}

// ââ Helpers âââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ

// buildLogger constructs a zap.Logger tuned to the environment.
func buildLogger(env string) (*zap.Logger, error) {
	var cfg zap.Config
	if env == "production" {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	return cfg.Build()
}

// getEnv returns the environment variable value or a fallback default.
func getEnv(key, defaultVal string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return defaultVal
}

// parseDurationEnv parses a duration from an env variable or returns the default.
func parseDurationEnv(key string, defaultVal time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultVal
	}
	return d
}

// parseFloat64Env parses a float64 from an env variable or returns the default.
func parseFloat64Env(key string, defaultVal float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	var f float64
	if err := json.Unmarshal([]byte(v), &f); err != nil {
		return defaultVal
	}
	return f
}

// parseIntEnv parses an int from an env variable or returns the default.
func parseIntEnv(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	var i int
	if err := json.Unmarshal([]byte(v), &i); err != nil {
		return defaultVal
	}
	return i
}

// splitAndTrim splits a string by sep and trims whitespace from each element.
func splitAndTrim(s, sep string) []string {
	raw := strings.Split(s, sep)
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		if t := strings.TrimSpace(r); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// requireRole is a helper that returns a middleware enforcing a specific role.
func requireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, exists := c.Get("user_roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no roles found"})
			return
		}
		roles, ok := rolesVal.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "malformed roles"})
			return
		}
		for _, r := range roles {
			if strings.EqualFold(r, role) {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":         "insufficient permissions",
			"required_role": role,
		})
	}
}

// getUserIDFromContext is a convenience helper for handlers.
func getUserIDFromContext(c *gin.Context) string {
	v, _ := c.Get("user_id")
	id, _ := v.(string)
	return id
}

// calculatePrice handles POST /api/v1/pricing/calculate
func calculatePrice(c *gin.Context) {
	start := time.Now()

	var req PriceRequest
	if err := c.ShouldBindJSON(
												&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_REQUEST",
			Message: "Invalid request body: " + err.Error(),
			Details: map[string]interface{}{"field_errors": extractValidationErrors(err)},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Validate zone exists
	zonesMu.RLock()
	zone, zoneExists := zones[req.ZoneID]
	zonesMu.RUnlock()

	if !zoneExists {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code: "ZONE_NOT_FOUND",
			Message: fmt.Sprintf("Zone '%s' not found", req.ZoneID),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Validate vehicle type
	validVehicleTypes := map[string]bool{
		"economy": true, "comfort": true, "premium": true,
		"xl": true, "electric": true, "cargo": true,
	}
	if !validVehicleTypes[req.VehicleType] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_VEHICLE_TYPE",
			Message: fmt.Sprintf("Vehicle type '%s' is not supported", req.VehicleType),
			Details: map[string]interface{}{"valid_types": []string{"economy", "comfort", "premium", "xl", "electric", "cargo"}},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Build pricing context
	now := time.Now().In(berlinLocation)
	ctx := PricingContext{
		ZoneID: req.ZoneID,
		VehicleType: req.VehicleType,
		Distance: req.Distance,
		Duration: req.Duration,
		RequestTime: now,
		PassengerCount: req.PassengerCount,
		SpecialConditions: req.SpecialConditions,
	}

	// Compute surge multiplier from demand/supply data
	surgeData := computeSurgeMultiplier(req.ZoneID, req.ActiveDrivers, req.PendingRequests)

	// Compute time-based multiplier
	timeMultiplier, timeLabel := getTimeBasedMultiplier(now)

	// Compute zone-specific multiplier
	zoneMultiplier := getZoneMultiplier(zone)

	// Compute vehicle-type multiplier
	vehicleMultiplier := getVehicleMultiplier(req.VehicleType)

	// Compute special conditions multiplier
	specialMultiplier := getSpecialConditionsMultiplier(req.SpecialConditions)

	// Calculate base price
	baseFare := cfg.Pricing.BaseFareEUR
	perKmRate := cfg.Pricing.PerKmRateEUR * vehicleMultiplier
	perMinRate := cfg.Pricing.PerMinRateEUR * vehicleMultiplier

	distanceComponent := req.Distance * perKmRate
	durationComponent := (req.Duration / 60.0) * perMinRate
	basePrice := baseFare + distanceComponent + durationComponent

	// Apply minimum fare
	if basePrice < cfg.Pricing.MinFareEUR {
		basePrice = cfg.Pricing.MinFareEUR
	}

	// Compose total multiplier
	totalMultiplier := surgeData.Multiplier * timeMultiplier * zoneMultiplier * specialMultiplier

	// Apply hard cap from config
	if totalMultiplier > cfg.Pricing.MaxSurgeMultiplier {
		totalMultiplier = cfg.Pricing.MaxSurgeMultiplier
	}

	// Calculate final price
	finalPrice := basePrice * totalMultiplier

	// Apply maximum fare cap
	if cfg.Pricing.MaxFareEUR > 0 && finalPrice > cfg.Pricing.MaxFareEUR {
		finalPrice = cfg.Pricing.MaxFareEUR
	}

	// Round to 2 decimal places (German monetary standard)
	finalPrice = math.Round(finalPrice*100) / 100
	basePrice = math.Round(basePrice*100) / 100

	// German compliance validation
	complianceResult := validateGermanCompliance(ctx, finalPrice, totalMultiplier, zone)
	if !complianceResult.IsCompliant {
		logger.Warn("German compliance check failed",
			zap.String("zone_id", req.ZoneID),
			zap.Float64("final_price", finalPrice),
			zap.Float64("multiplier", totalMultiplier),
			zap.Strings("violations", complianceResult.Violations),
		)
		if complianceResult.IsCritical {
			c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
				Code: "COMPLIANCE_VIOLATION",
				Message: "Price calculation violates German transport regulations (PBefG)",
				Details: map[string]interface{}{
					"violations": complianceResult.Violations,
					"regulation": "PBefG Â§39, Â§51",
				},
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				RequestID: c.GetString("request_id"),
			})
			return
		}
		// Non-critical: adjust price to compliant value
		finalPrice = complianceResult.AdjustedPrice
	}

	// Compute price breakdown
	taxRate := cfg.Pricing.TaxRatePercent / 100.0
	netPrice := finalPrice / (1 + taxRate)
	taxAmount := finalPrice - netPrice
	netPrice = math.Round(netPrice*100) / 100
	taxAmount = math.Round(taxAmount*100) / 100

	// Build price components
	components := []PriceComponent{
		{Name: "base_fare", Amount: baseFare, Description: "Fixed base fare"},
		{Name: "distance", Amount: math.Round(distanceComponent*vehicleMultiplier*100) / 100, Description: fmt.Sprintf("%.2f km Ã â¬%.4f/km", req.Distance, perKmRate)},
		{Name: "duration", Amount: math.Round(durationComponent*vehicleMultiplier*100) / 100, Description: fmt.Sprintf("%.0f min Ã â¬%.4f/min", req.Duration/60.0, perMinRate)},
	}

	if surgeData.Multiplier > 1.0 {
		components = append(components, PriceComponent{
			Name: "surge",
			Amount: math.Round((finalPrice-(basePrice*timeMultiplier*zoneMultiplier*specialMultiplier))*100) / 100,
			Description: fmt.Sprintf("Surge pricing (%.2fx) â high demand in zone", surgeData.Multiplier),
		})
	}

	if timeMultiplier > 1.0 {
		components = append(components, PriceComponent{
			Name: "time_adjustment",
			Amount: math.Round(basePrice*(timeMultiplier-1.0)*100) / 100,
			Description: fmt.Sprintf("%s adjustment (%.2fx)", timeLabel, timeMultiplier),
		})
	}

	if zoneMultiplier != 1.0 {
		components = append(components, PriceComponent{
			Name: "zone_adjustment",
			Amount: math.Round(basePrice*(zoneMultiplier-1.0)*100) / 100,
			Description: fmt.Sprintf("Zone '%s' adjustment (%.2fx)", zone.Type, zoneMultiplier),
		})
	}

	// Build final response
	priceID := generatePriceID()
	expiresAt := time.Now().UTC().Add(cfg.Pricing.QuoteValidityDuration)

	resp := PriceResponse{
		PriceID: priceID,
		ZoneID: req.ZoneID,
		VehicleType: req.VehicleType,
		BasePrice: basePrice,
		FinalPrice: finalPrice,
		NetPrice: netPrice,
		TaxAmount: taxAmount,
		TaxRate: cfg.Pricing.TaxRatePercent,
		Currency: "EUR",
		SurgeMultiplier: surgeData.Multiplier,
		TotalMultiplier: totalMultiplier,
		Components: components,
		SurgeActive: surgeData.IsActive,
		SurgeReason: surgeData.Reason,
		TimeLabel: timeLabel,
		Compliance: complianceResult,
		CalculatedAt: time.Now().UTC().Format(time.RFC3339),
		ExpiresAt: expiresAt.Format(time.RFC3339),
		RequestID: c.GetString("request_id"),
	}

	// Update metrics
	if metricsEnabled {
		priceCalculationDuration.WithLabelValues(req.ZoneID, req.VehicleType).Observe(time.Since(start).Seconds())
		priceCalculationTotal.WithLabelValues(req.ZoneID, req.VehicleType, "success").Inc()
		currentSurgeMultiplier.WithLabelValues(req.ZoneID).Set(surgeData.Multiplier)
		if surgeData.IsActive {
			surgeEventsTotal.WithLabelValues(req.ZoneID, "activated").Inc()
		}
	}

	// Publish Kafka events asynchronously
	go func() {
		_ = publishKafkaEvent(KafkaEventPriceUpdated, map[string]interface{}{
			"price_id": priceID,
			"zone_id": req.ZoneID,
			"vehicle_type": req.VehicleType,
			"base_price": basePrice,
			"final_price": finalPrice,
			"surge_multiplier": surgeData.Multiplier,
			"total_multiplier": totalMultiplier,
			"surge_active": surgeData.IsActive,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})

		if surgeData.IsActive {
			_ = publishKafkaEvent(KafkaEventSurgeActivated, map[string]interface{}{
				"zone_id": req.ZoneID,
				"multiplier": surgeData.Multiplier,
				"demand_supply_ratio": surgeData.DemandSupplyRatio,
				"pending_requests": req.PendingRequests,
				"active_drivers": req.ActiveDrivers,
				"reason": surgeData.Reason,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
		}

		if float64(req.PendingRequests) >= float64(cfg.Surge.DemandThreshold) {
			_ = publishKafkaEvent(KafkaEventDemandThreshold, map[string]interface{}{
				"zone_id": req.ZoneID,
				"pending_requests": req.PendingRequests,
				"threshold": cfg.Surge.DemandThreshold,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
		}
	}()

	// Cache the price quote in Redis
	go func() {
		if redisClient != nil {
			cacheKey := fmt.Sprintf("price_quote:%s", priceID)
			cacheData, err := json.Marshal(resp)
			if err == nil {
				_ = redisClient.Set(context.Background(), cacheKey, cacheData, cfg.Pricing.QuoteValidityDuration).Err()
			}
		}
	}()

	logger.Info("Price calculated",
		zap.String("price_id", priceID),
		zap.String("zone_id", req.ZoneID),
		zap.String("vehicle_type", req.VehicleType),
		zap.Float64("base_price", basePrice),
		zap.Float64("final_price", finalPrice),
		zap.Float64("surge_multiplier", surgeData.Multiplier),
		zap.Float64("total_multiplier", totalMultiplier),
		zap.Bool("surge_active", surgeData.IsActive),
		zap.Duration("latency", time.Since(start)),
	)

	c.JSON(http.StatusOK, resp)
}

// computeSurgeMultiplier calculates the surge multiplier based on demand/supply dynamics
func computeSurgeMultiplier(zoneID string, activeDrivers, pendingRequests int) SurgeData {
	// Retrieve historical surge data from in-memory store
	surgeStore.mu.RLock()
	historicalData, hasHistory := surgeStore.history[zoneID]
	surgeStore.mu.RUnlock()

	// Guard against division by zero
	if activeDrivers <= 0 {
		if pendingRequests <= 0 {
			return SurgeData{
				Multiplier: 1.0,
				IsActive: false,
				Reason: "no_data",
				DemandSupplyRatio: 0,
			}
		}
		// No drivers but there are requests â maximum surge
		return SurgeData{
			Multiplier: cfg.Pricing.MaxSurgeMultiplier,
			IsActive: true,
			Reason: "no_drivers_available",
			DemandSupplyRatio: float64(pendingRequests),
		}
	}

	// Core demand/supply ratio
	demandSupplyRatio := float64(pendingRequests) / float64(activeDrivers)

	// Smoothing: blend with historical EMA if available
	if hasHistory && len(historicalData) > 0 {
		ema := computeEMA(historicalData, cfg.Surge.EMAAlpha)
		// Weighted blend: 70% current, 30% historical EMA
		demandSupplyRatio = 0.7*demandSupplyRatio + 0.3*ema
	}

	// Update history store
	surgeStore.mu.Lock()
	surgeStore.history[zoneID] = append(historicalData, demandSupplyRatio)
	if len(surgeStore.history[zoneID]) > cfg.Surge.HistoryWindowSize {
		surgeStore.history[zoneID] = surgeStore.history[zoneID][1:]
	}
	surgeStore.mu.Unlock()

	// Tiered surge multiplier calculation
	// Each tier represents increasing demand pressure
	var multiplier float64
	var reason string

	switch {
	case demandSupplyRatio < cfg.Surge.Tier1Threshold:
		// Normal supply-demand balance
		multiplier = 1.0
		reason = "normal"

	case demandSupplyRatio < cfg.Surge.Tier2Threshold:
		// Mild surge: linear interpolation within tier 1
		t := (demandSupplyRatio - cfg.Surge.Tier1Threshold) / (cfg.Surge.Tier2Threshold - cfg.Surge.Tier1Threshold)
		multiplier = 1.0 + t*(cfg.Surge.Tier1Multiplier-1.0)
		reason = "mild_demand"

	case demandSupplyRatio < cfg.Surge.Tier3Threshold:
		// Moderate surge: linear interpolation within tier 2
		t := (demandSupplyRatio - cfg.Surge.Tier2Threshold) / (cfg.Surge.Tier3Threshold - cfg.Surge.Tier2Threshold)
		multiplier = cfg.Surge.Tier1Multiplier + t*(cfg.Surge.Tier2Multiplier-cfg.Surge.Tier1Multiplier)
		reason = "moderate_demand"

	case demandSupplyRatio < cfg.Surge.Tier4Threshold:
		// High surge: linear interpolation within tier 3
		t := (demandSupplyRatio - cfg.Surge.Tier3Threshold) / (cfg.Surge.Tier4Threshold - cfg.Surge.Tier3Threshold)
		multiplier = cfg.Surge.Tier2Multiplier + t*(cfg.Surge.Tier3Multiplier-cfg.Surge.Tier2Multiplier)
		reason = "high_demand"

	default:
		// Critical surge: cap at maximum allowed
		multiplier = cfg.Surge.Tier3Multiplier
		reason = "critical_demand"
	}

	// Apply zone-level surge override if present (e.g., from manual admin override)
	zonesMu.RLock()
	if zone, exists := zones[zoneID]; exists && zone.SurgeOverride > 0 {
		multiplier = zone.SurgeOverride
		reason = "manual_override"
	}
	zonesMu.RUnlock()

	// Round to 2 decimal places for consistent display
	multiplier = math.Round(multiplier*100) / 100

	// Clamp to configured max
	if multiplier > cfg.Pricing.MaxSurgeMultiplier {
		multiplier = cfg.Pricing.MaxSurgeMultiplier
	}

	isActive := multiplier > 1.0

	// Track surge state changes
	trackSurgeStateChange(zoneID, isActive, multiplier)

	return SurgeData{
		Multiplier: multiplier,
		IsActive: isActive,
		Reason: reason,
		DemandSupplyRatio: math.Round(demandSupplyRatio*100) / 100,
		ActiveDrivers: activeDrivers,
		PendingRequests: pendingRequests,
	}
}

// computeEMA computes an Exponential Moving Average over a slice of float64 values
func computeEMA(data []float64, alpha float64) float64 {
	if len(data) == 0 {
		return 0
	}
	ema := data[0]
	for _, v := range data[1:] {
		ema = alpha*v + (1-alpha)*ema
	}
	return ema
}

// trackSurgeStateChange records surge activation/deactivation transitions
func trackSurgeStateChange(zoneID string, isActive bool, multiplier float64) {
	surgeStore.mu.Lock()
	defer surgeStore.mu.Unlock()

	prevActive, exists := surgeStore.activeZones[zoneID]
	surgeStore.activeZones[zoneID] = isActive
	surgeStore.multipliers[zoneID] = multiplier
	surgeStore.lastUpdated[zoneID] = time.Now().UTC()

	if !exists {
		return
	}

	// Publish deactivation event if surge just ended
	if prevActive && !isActive {
		go func() {
			_ = publishKafkaEvent(KafkaEventSurgeDeactivated, map[string]interface{}{
				"zone_id": zoneID,
				"multiplier": multiplier,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
		}()
	}
}

// getTimeBasedMultiplier returns a pricing multiplier and label based on Berlin local time
func getTimeBasedMultiplier(t time.Time) (float64, string) {
	hour := t.Hour()
	weekday := t.Weekday()
	isWeekend := weekday == time.Saturday || weekday == time.Sunday

	// Morning peak hours: 07:00â09:00 (MonâFri)
	if !isWeekend && hour >= 7 && hour < 9 {
		return cfg.Pricing.PeakHourMultiplier, "morning_peak"
	}

	// Evening peak hours: 17:00â19:00 (MonâFri)
	if !isWeekend && hour >= 17 && hour < 19 {
		return cfg.Pricing.PeakHourMultiplier, "evening_peak"
	}

	// Late evening rush: 22:00â23:00 (all days)
	if hour >= 22 && hour < 23 {
		return cfg.Pricing.PeakHourMultiplier * 0.85, "late_evening"
	}

	// Night rate: 23:00â05:00 (Nachttarif per German taxi regulations)
	if hour >= 23 || hour < 5 {
		return cfg.Pricing.NightRateMultiplier, "night_rate"
	}

	// Early morning: 05:00â07:00
	if hour >= 5 && hour < 7 {
		return 1.1, "early_morning"
	}

	// Weekend daytime: higher baseline due to leisure demand
	if isWeekend && hour >= 10 && hour < 22 {
		return cfg.Pricing.WeekendMultiplier, "weekend_day"
	}

	// Weekend morning: 07:00â10:00
	if isWeekend && hour >= 7 && hour < 10 {
		return 1.1, "weekend_morning"
	}

	// Public holiday check (German Feiertage)
	if isGermanPublicHoliday(t) {
		return cfg.Pricing.WeekendMultiplier * 1.1, "public_holiday"
	}

	return 1.0, "standard"
}

// isGermanPublicHoliday checks for national German public holidays (fixed-date Feiertage)
func isGermanPublicHoliday(t time.Time) bool {
	month := t.Month()
	day := t.Day()

	switch {
	case month == time.January && day == 1: // Neujahr
		return true
	case month == time.May && day == 1: // Tag der Arbeit
		return true
	case month == time.October && day == 3: // Tag der Deutschen Einheit
		return true
	case month == time.November && day == 1: // Allerheiligen (Bavaria, Baden-WÃ¼rttemberg, etc.)
		return true
	case month == time.December && day == 25: // 1. Weihnachtstag
		return true
	case month == time.December && day == 26: // 2. Weihnachtstag
		return true
	case month == time.December && day == 31 && t.Hour() >= 20: // Silvester evening
		return true
	}

	// Easter-based holidays (computationally derived)
	easterSunday := computeEasterSunday(t.Year())
	easterMonday := easterSunday.AddDate(0, 0, 1)
	goodFriday := easterSunday.AddDate(0, 0, -2)
	ascensionDay := easterSunday.AddDate(0, 0, 39) // Christi Himmelfahrt
	whitSunday := easterSunday.AddDate(0, 0, 49)   // Pfingstsonntag
	whitMonday := easterSunday.AddDate(0, 0, 50)   // Pfingstmontag

	candidates := []time.Time{easterMonday, goodFriday, ascensionDay, whitSunday, whitMonday}
	for _, candidate := range candidates {
		if t.Month() == candidate.Month() && t.Day() == candidate.Day() {
			return true
		}
	}

	return false
}

// computeEasterSunday implements the Anonymous Gregorian algorithm for Easter
func computeEasterSunday(year int) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, berlinLocation)
}

// getZoneMultiplier returns a pricing multiplier based on zone characteristics
func getZoneMultiplier(zone Zone) float64 {
	baseMultiplier := 1.0

	switch zone.Type {
	case ZoneTypeCityCenter:
		// High demand, traffic congestion, premium catchment area
		baseMultiplier = 1.15

	case ZoneTypeAirport:
		// Fixed airport surcharge â covers pickup fees & terminal routing
		baseMultiplier = 1.25

	case ZoneTypeTrainStation:
		// Moderate premium â high throughput but competitive with other transit
		baseMultiplier = 1.10

	case ZoneTypeSuburb:
		// Lower demand density â slight discount to incentivise drivers
		baseMultiplier = 0.95

	case ZoneTypeIndustrial:
		// Low demand outside shift changes
		baseMultiplier = 0.90

	case ZoneTypeSpecialEvent:
		// Concert venues, stadiums, fairs â temporary high-demand surcharge
		baseMultiplier = 1.35

	default:
		baseMultiplier = 1.0
	}

	// Apply zone-configured custom multiplier if set (overrides type default)
	if zone.CustomMultiplier > 0 {
		baseMultiplier = zone.CustomMultiplier
	}

	// Apply zone capacity pressure factor
	if zone.CapacityFactor > 0 {
		baseMultiplier *= zone.CapacityFactor
	}

	return math.Round(baseMultiplier*1000) / 1000
}

// getVehicleMultiplier returns a pricing multiplier based on vehicle type
func getVehicleMultiplier(vehicleType string) float64 {
	switch vehicleType {
	case "economy":
		return 1.0
	case "comfort":
		return 1.25
	case "premium":
		return 1.75
	case "xl":
		return 1.50
	case "electric":
		// Slight premium â green surcharge partially offset by fuel savings
		return 1.10
	case "cargo":
		return 2.0
	default:
		return 1.0
	}
}

// getSpecialConditionsMultiplier aggregates multipliers from special conditions list
func getSpecialConditionsMultiplier(conditions []string) float64 {
	if len(conditions) == 0 {
		return 1.0
	}

	multiplier := 1.0
	conditionMap := map[string]float64{
		"bad_weather": 1.15,  // Starkregen, Schnee, Glatteis
		"rain": 1.08,
		"snow": 1.20,
		"fog": 1.05,
		"event_nearby": 1.10,
		"construction": 1.05, // Baustelle â longer routes
		"public_transit_strike": 1.30, // Streik â modal shift spike
		"new_years_eve": 1.50,
		"oktoberfest": 1.40,
		"carnival": 1.25,
		"marathon": 1.20,     // Road closures
	}

	for _, cond := range conditions {
		if factor, ok := conditionMap[cond]; ok {
			// Multiplicative stacking with diminishing returns
			multiplier *= (1 + (factor-1)*0.8)
		}
	}

	return math.Round(multiplier*1000) / 1000
}

// validateGermanCompliance validates price against PBefG Â§39 (BefÃ¶rderungspflicht)
// and Â§51 (BefÃ¶rderungsentgelte) â German Passenger Transport Act requirements
func validateGermanCompliance(ctx PricingContext, finalPrice, multiplier float64, zone Zone) ComplianceResult {
	result := ComplianceResult{
		IsCompliant: true,
		IsCritical: false,
		Violations: []string{},
		AdjustedPrice: finalPrice,
		Regulation: "PBefG Â§39, Â§51",
	}

	// Â§39 â BefÃ¶rderungspflicht: cannot refuse service or price discriminate
	// Ensure price is not zero (service must have defined price)
	if finalPrice <= 0 {
		result.IsCompliant = false
		result.IsCritical = true
		result.Violations = append(result.Violations, "PBefG Â§39: BefÃ¶rderungsentgelt must be positive (BefÃ¶rderungspflicht)")
		result.AdjustedPrice = cfg.Pricing.MinFareEUR
	}

	// Â§51 â BefÃ¶rderungsentgelte: tariffs must be approved; surcharges need justification
	// Maximum allowed multiplier per German platform regulation
	maxAllowedMultiplier := cfg.Compliance.MaxAllowedMultiplierPBefG
	if multiplier > maxAllowedMultiplier {
		result.IsCompliant = false
		result.IsCritical = true
		result.Violations = append(result.Violations,
			fmt.Sprintf("PBefG Â§51: Surge multiplier %.2fx exceeds maximum allowed %.2fx", multiplier, maxAllowedMultiplier),
		)
		// Adjust price to compliant level
		result.AdjustedPrice = math.Round((finalPrice/multiplier)*maxAllowedMultiplier*100) / 100
	}

	// Minimum fare check â cannot undercut licensed taxi minimum fares
	if finalPrice < cfg.Compliance.MinimumFarePBefG {
		result.IsCompliant = false
		result.IsCritical = false // Non-critical: adjust up
		result.Violations = append(result.Violations,
			fmt.Sprintf("PBefG Â§51: Final price â¬%.2f is below minimum compliant fare â¬%.2f", finalPrice, cfg.Compliance.MinimumFarePBefG),
		)
		result.AdjustedPrice = cfg.Compliance.MinimumFarePBefG
	}

	// Night surcharge validation (Â§51 Abs. 2 â Nachtzuschlag max 20%)
	hour := ctx.RequestTime.Hour()
	if (hour >= 23 || hour < 5) && multiplier > cfg.Compliance.MaxNightSurchargeMultiplier {
		result.IsCompliant = false
		result.IsCritical = false
		result.Violations = append(result.Violations,
			fmt.Sprintf("PBefG Â§51 Abs.2: Night surcharge multiplier %.2fx exceeds Nachtzuschlag limit %.2fx",
				multiplier, cfg.Compliance.MaxNightSurchargeMultiplier),
		)
		result.AdjustedPrice = math.Round((finalPrice/multiplier)*cfg.Compliance.MaxNightSurchargeMultiplier*100) / 100
	}

	// Passenger count surcharge validation (GroÃraumtaxe)
	if ctx.PassengerCount > 4 && ctx.VehicleType != "xl" && ctx.VehicleType != "cargo" {
		result.IsCompliant = false
		result.IsCritical = false
		result.Violations = append(result.Violations,
			fmt.Sprintf("PBefG Â§51: %d passengers require XL or Cargo vehicle type (GroÃraumtaxe)", ctx.PassengerCount),
		)
	}

	// Airport zone: mandatory airport surcharge disclosure
	if zone.Type == ZoneTypeAirport && !zone.AirportSurchargeDisclosed {
		result.IsCompliant = false
		result.IsCritical = false
		result.Violations = append(result.Violations,
			"PBefG Â§51: Airport surcharge must be disclosed to passenger before booking (Pflichtangabe)",
		)
	}

	// Special event zones: duration limit for surge pricing (max 4 hours per event per BNetzA guidance)
	if zone.Type == ZoneTypeSpecialEvent && zone.SurgeStartedAt != nil {
		surgeDuration := time.Since(*zone.SurgeStartedAt)
		if surgeDuration > cfg.Compliance.MaxEventSurgeDuration {
			result.IsCompliant = false
			result.IsCritical = false
			result.Violations = append(result.Violations,
				fmt.Sprintf("BNetzA guidance: Special event surge has been active for %s, exceeding %s limit",
					surgeDuration.Round(time.Minute), cfg.Compliance.MaxEventSurgeDuration),
			)
			// Revert to standard pricing for compliance
			result.AdjustedPrice = math.Round((finalPrice/multiplier)*100) / 100
		}
	}

	// GDPR: ensure no personal data embedded in price ID or logs (structural check)
	if containsPII(ctx) {
		result.IsCompliant = false
		result.IsCritical = true
		result.Violations = append(result.Violations,
			"DSGVO/GDPR: PII detected in pricing context â must be anonymised before processing",
		)
	}

	return result
}

// containsPII performs a lightweight check for personal data in pricing context
func containsPII(ctx PricingContext) bool {
	// Check if any special condition contains identifiable passenger data patterns
	piiPatterns := []string{"email:", "phone:", "name:", "iban:", "passport:"}
	for _, condition := range ctx.SpecialConditions {
		for _, pattern := range piiPatterns {
			if len(condition) > len(pattern) && condition[:len(pattern)] == pattern {
				return true
			}
		}
	}
	return false
}

// getSurgeStatus returns GET /api/v1/pricing/surge â all zones surge summary
func getSurgeStatus(c *gin.Context) {
	zonesMu.RLock()
	allZones := make([]Zone, 0, len(zones))
	for _, z := range zones {
		allZones = append(allZones, z)
	}
	zonesMu.RUnlock()

	surgeStore.mu.RLock()
	defer surgeStore.mu.RUnlock()

	type ZoneSurgeInfo struct {
		ZoneID string `json:"zone_id"`
		ZoneName string `json:"zone_name"`
		ZoneType string `json:"zone_type"`
		SurgeActive bool `json:"surge_active"`
		Multiplier float64 `json:"multiplier"`
		LastUpdated string `json:"last_updated,omitempty"`
		DemandLevel string `json:"demand_level"`
	}

	surgeInfos := make([]ZoneSurgeInfo, 0, len(allZones))
	activeSurgeCount := 0

	for _, z := range allZones {
		multiplier := surgeStore.multipliers[z.ID]
		if multiplier == 0 {
			multiplier = 1.0
		}
		isActive := surgeStore.activeZones[z.ID]
		if isActive {
			activeSurgeCount++
		}

		lastUpdated := ""
		if ts, ok := surgeStore.lastUpdated[z.ID]; ok {
			lastUpdated = ts.Format(time.RFC3339)
		}

		demandLevel := classifyDemandLevel(multiplier)

		surgeInfos = append(surgeInfos, ZoneSurgeInfo{
			ZoneID: z.ID,
			ZoneName: z.Name,
			ZoneType: string(z.Type),
			SurgeActive: isActive,
			Multiplier: multiplier,
			LastUpdated: lastUpdated,
			DemandLevel: demandLevel,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"zones": surgeInfos,
		"total_zones": len(allZones),
		"active_surge_zones": activeSurgeCount,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"request_id": c.GetString("request_id"),
	})
}

// getSurgeStatusByZone returns GET /api/v1/pricing/surge/:zone_id â specific zone surge
func getSurgeStatusByZone(c *gin.Context) {
	zoneID := c.Param("zone_id")
	if zoneID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "MISSING_ZONE_ID",
			Message: "zone_id path parameter is required",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	zonesMu.RLock()
	zone, exists := zones[zoneID]
	zonesMu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code: "ZONE_NOT_FOUND",
			Message: fmt.Sprintf("Zone '%s' not found", zoneID),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	surgeStore.mu.RLock()
	multiplier := surgeStore.multipliers[zoneID]
	isActive := surgeStore.activeZones[zoneID]
	history := make([]float64, len(surgeStore.history[zoneID]))
	copy(history, surgeStore.history[zoneID])
	lastUpdated := surgeStore.lastUpdated[zoneID]
	surgeStore.mu.RUnlock()

	if multiplier == 0 {
		multiplier = 1.0
	}

	now := time.Now().In(berlinLocation)
	timeMultiplier, timeLabel := getTimeBasedMultiplier(now)
	zoneMultiplier := getZoneMultiplier(zone)
	demandLevel := classifyDemandLevel(multiplier)

	// Compute EMA trend from history
	ema := 1.0
	if len(history) > 0 {
		ema = computeEMA(history, cfg.Surge.EMAAlpha)
		ema = math.Round(ema*100) / 100
	}

	c.JSON(http.StatusOK, gin.H{
		"zone_id": zone.ID,
		"zone_name": zone.Name,
		"zone_type": string(zone.Type),
		"surge_active": isActive,
		"surge_multiplier": multiplier,
		"time_multiplier": timeMultiplier,
		"time_label": timeLabel,
		"zone_multiplier": zoneMultiplier,
		"effective_multiplier": math.Round(multiplier*timeMultiplier*zoneMultiplier*100) / 100,
		"demand_level": demandLevel,
		"demand_history_ema": ema,
		"history_window_size": len(history),
		"last_updated": lastUpdated.Format(time.RFC3339),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"request_id": c.GetString("request_id"),
	})
}

// classifyDemandLevel maps a surge multiplier to a human-readable demand tier
func classifyDemandLevel(multiplier float64) string {
	switch {
	case multiplier >= 3.0:
		return "critical"
	case multiplier >= 2.0:
		return "very_high"
	case multiplier >= 1.5:
		return "high"
	case multiplier >= 1.2:
		return "moderate"
	case multiplier > 1.0:
		return "mild"
	default:
		return "normal"
	}
}

// publishKafkaEvent publishes a structured event to the appropriate Kafka topic
func publishKafkaEvent(eventType KafkaEventType, payload map[string]interface{}) error {
	if kafkaProducer == nil {
		logger.Debug("Kafka producer not initialised â skipping event", zap.String("event_type", string(eventType)))
		return nil
	}

	// Map event type to topic
	topicMap := map[KafkaEventType]string{
		KafkaEventPriceUpdated: cfg.Kafka.Topics.PriceUpdated,
		KafkaEventSurgeActivated: cfg.Kafka.Topics.SurgeActivated,
		KafkaEventSurgeDeactivated: cfg.Kafka.Topics.SurgeDeactivated,
		KafkaEventDemandThreshold: cfg.Kafka.Topics.DemandThreshold,
	}

	topic, ok := topicMap[eventType]
	if !ok {
		return fmt.Errorf("unknown Kafka event type: %s", eventType)
	}

	// Build the envelope
	envelope := KafkaEventEnvelope{
		EventID: generateEventID(),
		EventType: string(eventType),
		Version: "1.0",
		Source: "surge-pricing-service",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload: payload,
		Metadata: map[string]string{
			"service": "surge-pricing",
			"environment": cfg.App.Environment,
			"region": cfg.App.Region,
		},
	}

	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		logger.Error("Failed to marshal Kafka event",
			zap.String("event_type", string(eventType)),
			zap.Error(err),
		)
		return fmt.Errorf("marshal kafka event: %w", err)
	}

	// Use zone_id as partition key for ordered delivery per zone
	partitionKey := ""
	if zoneID, ok := payload["zone_id"].(string); ok {
		partitionKey = zoneID
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(envelopeBytes),
		Timestamp: time.Now().UTC(),
		Headers: []sarama.RecordHeader{
			{Key: []byte("event_type"), Value: []byte(string(eventType))},
			{Key: []byte("content_type"), Value: []byte("application/json")},
			{Key: []byte("source"), Value: []byte("surge-pricing-service")},
			{Key: []byte("version"), Value: []byte("1.0")},
		},
	}

	if partitionKey != "" {
		msg.Key = sarama.StringEncoder(partitionKey)
	}

	// Send with retry logic (sarama sync producer handles this internally)
	partition, offset, err := kafkaProducer.SendMessage(msg)
	if err != nil {
		logger.Error("Failed to publish Kafka event",
			zap.String("event_type", string(eventType)),
			zap.String("topic", topic),
			zap.Error(err),
		)
		// Update metrics
		if metricsEnabled {
			kafkaPublishErrors.WithLabelValues(topic, string(eventType)).Inc()
		}
		return fmt.Errorf("publish kafka event to topic %s: %w", topic, err)
	}

	logger.Debug("Kafka event published",
		zap.String("event_type", string(eventType)),
		zap.String("topic", topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
		zap.String("partition_key", partitionKey),
	)

	if metricsEnabled {
		kafkaPublishTotal.WithLabelValues(topic, string(eventType)).Inc()
	}

	return nil
}

// extractValidationErrors converts binding validation errors to a structured map
func extractValidationErrors(err error) map[string]string {
	result := make(map[string]string)
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, fe := range validationErrors {
			field := strings.ToLower(fe.Field())
			switch fe.Tag() {
			case "required":
				result[field] = fmt.Sprintf("%s is required", field)
			case "min":
				result[field] = fmt.Sprintf("%s must be at least %s", field, fe.Param())
			case "max":
				result[field] = fmt.Sprintf("%s must be at most %s", field, fe.Param())
			case "gt":
				result[field] = fmt.Sprintf("%s must be greater than %s", field, fe.Param())
			case "gte":
				result[field] = fmt.Sprintf("%s must be greater than or equal to %s", field, fe.Param())
			case "oneof":
				result[field] = fmt.Sprintf("%s must be one of: %s", field, fe.Param())
			default:
				result[field] = fmt.Sprintf("%s failed validation: %s", field, fe.Tag())
			}
		}
	}
	return result
}

// generatePriceID generates a unique, time-ordered price quote identifier
func generatePriceID() string {
	return fmt.Sprintf("PQ-%d-%s", time.Now().UnixNano(), generateShortID(8))
}

// generateEventID generates a unique Kafka event identifier
func generateEventID() string {
	return fmt.Sprintf("EVT-%d-%s", time.Now().UnixNano(), generateShortID(6))
}

// generateShortID generates a random alphanumeric string of given length
func generateShortID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			b[i] = charset[0]
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// ============================================================
// PART 4: Admin CRUD Operations 
// Pricing Rules CRUD, Surge Zones CRUD, Demand Reporting
// ============================================================

// âââ Pricing Rules CRUD ââââââââââââââââââââââââââââââââââââââââââââââââââââââââ

// createPricingRule handles POST /api/v1/pricing/rules
// Admin only: creates a new dynamic pricing rule
func (app *Application) createPricingRule(c *gin.Context) {
	const op = "Application.createPricingRule"

	// Extract admin identity from context (set by auth middleware)
	adminID, exists := c.Get("user_id")
	if !exists {
		app.logger.Warn("createPricingRule: missing user_id in context")
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code: "UNAUTHORIZED",
			Message: "authentication required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	var req CreatePricingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		app.logger.Warnw("createPricingRule: invalid request body",
			"error", err,
			"admin_id", adminID,
		)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_REQUEST",
			Message: "invalid request body: " + err.Error(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Validate the request
	if err := app.validator.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		app.logger.Warnw("createPricingRule: validation failed",
			"errors", validationErrors,
			"admin_id", adminID,
		)
		c.JSON(http.StatusUnprocessableEntity, ValidationErrorResponse{
			Code: "VALIDATION_ERROR",
			Message: "request validation failed",
			Errors: validationErrors,
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Additional business rule validations
	if req.MinMultiplier <= 0 || req.MaxMultiplier <= 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_MULTIPLIER",
			Message: "multipliers must be positive values",
			RequestID: c.GetString("request_id"),
		})
		return
	}
	if req.MinMultiplier > req.MaxMultiplier {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_MULTIPLIER_RANGE",
			Message: "min_multiplier must be less than or equal to max_multiplier",
			RequestID: c.GetString("request_id"),
		})
		return
	}
	if req.BaseMultiplier < req.MinMultiplier || req.BaseMultiplier > req.MaxMultiplier {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_BASE_MULTIPLIER",
			Message: "base_multiplier must be within [min_multiplier, max_multiplier]",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	now := time.Now().UTC()
	rule := &PricingRule{
		ID: uuid.New().String(),
		Name: strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		RuleType: req.RuleType,
		Priority: req.Priority,
		MinMultiplier: req.MinMultiplier,
		MaxMultiplier: req.MaxMultiplier,
		BaseMultiplier: req.BaseMultiplier,
		DemandThresholdLow: req.DemandThresholdLow,
		DemandThresholdHigh: req.DemandThresholdHigh,
		SupplyThresholdLow: req.SupplyThresholdLow,
		SupplyThresholdHigh: req.SupplyThresholdHigh,
		TimeWindows: req.TimeWindows,
		DayOfWeekMask: req.DayOfWeekMask,
		WeatherConditions: req.WeatherConditions,
		EventTags: req.EventTags,
		IsActive: true,
		CreatedBy: fmt.Sprintf("%v", adminID),
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	query := `
		INSERT INTO pricing_rules (
			id, name, description, rule_type, priority,
			min_multiplier, max_multiplier, base_multiplier,
			demand_threshold_low, demand_threshold_high,
			supply_threshold_low, supply_threshold_high,
			time_windows, day_of_week_mask,
			weather_conditions, event_tags,
			is_active, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10,
			$11, $12,
			$13, $14,
			$15, $16,
			$17, $18, $19, $20
		) RETURNING id, created_at, updated_at`

	timeWindowsJSON, _ := json.Marshal(rule.TimeWindows)
	weatherCondJSON, _ := json.Marshal(rule.WeatherConditions)
	eventTagsJSON, _ := json.Marshal(rule.EventTags)

	err := app.db.QueryRowContext(ctx, query,
		rule.ID, rule.Name, rule.Description, rule.RuleType, rule.Priority,
		rule.MinMultiplier, rule.MaxMultiplier, rule.BaseMultiplier,
		rule.DemandThresholdLow, rule.DemandThresholdHigh,
		rule.SupplyThresholdLow, rule.SupplyThresholdHigh,
		timeWindowsJSON, rule.DayOfWeekMask,
		weatherCondJSON, eventTagsJSON,
		rule.IsActive, rule.CreatedBy, rule.CreatedAt, rule.UpdatedAt,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)

	if err != nil {
		app.logger.Errorw(op+": failed to insert pricing rule",
			"error", err,
			"rule_name", rule.Name,
			"admin_id", adminID,
		)
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Code: "RULE_ALREADY_EXISTS",
				Message: "a pricing rule with this name already exists",
				RequestID: c.GetString("request_id"),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to create pricing rule",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Invalidate cached rules
	cacheKey := "pricing:rules:all"
	if err := app.cache.Delete(ctx, cacheKey); err != nil {
		app.logger.Warnw(op+": failed to invalidate rules cache", "error", err)
	}

	// Publish Kafka event
	event := KafkaEvent{
		EventID: uuid.New().String(),
		EventType: "pricing.rule.created",
		Timestamp: now,
		Version: "1.0",
		Source: "pricing-service",
		Payload: map[string]interface{}{
			"rule_id": rule.ID,
			"rule_name": rule.Name,
			"rule_type": rule.RuleType,
			"priority": rule.Priority,
			"created_by": rule.CreatedBy,
		},
	}
	if err := app.publishEvent(ctx, app.config.Kafka.Topics.PricingUpdates, rule.ID, event); err != nil {
		app.logger.Warnw(op+": failed to publish pricing.rule.created event",
			"error", err,
			"rule_id", rule.ID,
		)
	}

	app.logger.Infow("pricing rule created",
		"rule_id", rule.ID,
		"rule_name", rule.Name,
		"admin_id", adminID,
	)

	c.JSON(http.StatusCreated, PricingRuleResponse{
		Data: rule,
		Meta: ResponseMeta{
			RequestID: c.GetString("request_id"),
			Timestamp: now,
		},
	})
}

// listPricingRules handles GET /api/v1/pricing/rules
// Admin only: returns paginated list of pricing rules
func (app *Application) listPricingRules(c *gin.Context) {
	const op = "Application.listPricingRules"

	// Parse query params
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	ruleType := c.Query("rule_type")
	isActiveStr := c.Query("is_active")
	sortBy := c.DefaultQuery("sort_by", "priority")
	sortOrder := c.DefaultQuery("sort_order", "asc")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Build cache key based on query params
	cacheKey := fmt.Sprintf("pricing:rules:list:%d:%d:%s:%s:%s:%s",
		page, pageSize, ruleType, isActiveStr, sortBy, sortOrder)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Try cache
	var cached PricingRulesListResponse
	if err := app.cache.Get(ctx, cacheKey, &cached); err == nil {
		c.JSON(http.StatusOK, cached)
		return
	}

	// Build dynamic query
	var conditions []string
	var args []interface{}
	argIdx := 1

	if ruleType != "" {
		conditions = append(conditions, fmt.Sprintf("rule_type = $%d", argIdx))
		args = append(args, ruleType)
		argIdx++
	}
	if isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
			args = append(args, isActive)
			argIdx++
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Validate sort params to prevent SQL injection
	allowedSortFields := map[string]bool{"priority": true, "name": true, "created_at": true, "updated_at": true, "rule_type": true}
	if !allowedSortFields[sortBy] {
		sortBy = "priority"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "asc"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM pricing_rules %s", whereClause)
	var total int
	if err := app.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		app.logger.Errorw(op+": failed to count pricing rules", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to retrieve pricing rules",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	offset := (page - 1) * pageSize
	listQuery := fmt.Sprintf(`
		SELECT
			id, name, description, rule_type, priority,
			min_multiplier, max_multiplier, base_multiplier,
			demand_threshold_low, demand_threshold_high,
			supply_threshold_low, supply_threshold_high,
			time_windows, day_of_week_mask,
			weather_conditions, event_tags,
			is_active, created_by, created_at, updated_at
		FROM pricing_rules
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, sortBy, sortOrder, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := app.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		app.logger.Errorw(op+": failed to query pricing rules", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to retrieve pricing rules",
			RequestID: c.GetString("request_id"),
		})
		return
	}
	defer rows.Close()

	var rules []*PricingRule
	for rows.Next() {
		rule := &PricingRule{}
		var timeWindowsJSON, weatherCondJSON, eventTagsJSON []byte
		if err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Description, &rule.RuleType, &rule.Priority,
			&rule.MinMultiplier, &rule.MaxMultiplier, &rule.BaseMultiplier,
			&rule.DemandThresholdLow, &rule.DemandThresholdHigh,
			&rule.SupplyThresholdLow, &rule.SupplyThresholdHigh,
			&timeWindowsJSON, &rule.DayOfWeekMask,
			&weatherCondJSON, &eventTagsJSON,
			&rule.IsActive, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			app.logger.Errorw(op+": failed to scan pricing rule row", "error", err)
			continue
		}
		_ = json.Unmarshal(timeWindowsJSON, &rule.TimeWindows)
		_ = json.Unmarshal(weatherCondJSON, &rule.WeatherConditions)
		_ = json.Unmarshal(eventTagsJSON, &rule.EventTags)
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		app.logger.Errorw(op+": row iteration error", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "error reading pricing rules",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	if rules == nil {
		rules = []*PricingRule{}
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	resp := PricingRulesListResponse{
		Data: rules,
		Pagination: PaginationMeta{
			Page: page,
			PageSize: pageSize,
			Total: total,
			TotalPages: totalPages,
			HasNext: page < totalPages,
			HasPrev: page > 1,
		},
		Meta: ResponseMeta{
			RequestID: c.GetString("request_id"),
			Timestamp: time.Now().UTC(),
		},
	}

	// Cache the response for 30 seconds
	if err := app.cache.Set(ctx, cacheKey, resp, 30*time.Second); err != nil {
		app.logger.Warnw(op+": failed to cache rules list", "error", err)
	}

	c.JSON(http.StatusOK, resp)
}

// updatePricingRule handles PUT /api/v1/pricing/rules/:id
// Admin only: updates an existing pricing rule
func (app *Application) updatePricingRule(c *gin.Context) {
	const op = "Application.updatePricingRule"

	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code: "UNAUTHORIZED",
			Message: "authentication required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	ruleID := c.Param("id")
	if ruleID == "" || !isValidUUID(ruleID) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_RULE_ID",
			Message: "valid rule ID is required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	var req UpdatePricingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_REQUEST",
			Message: "invalid request body: " + err.Error(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	if err := app.validator.Struct(req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, ValidationErrorResponse{
			Code: "VALIDATION_ERROR",
			Message: "request validation failed",
			Errors: formatValidationErrors(err),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Validate multiplier constraints if provided
	if req.MinMultiplier != nil && req.MaxMultiplier != nil {
		if *req.MinMultiplier > *req.MaxMultiplier {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code: "INVALID_MULTIPLIER_RANGE",
				Message: "min_multiplier must be <= max_multiplier",
				RequestID: c.GetString("request_id"),
			})
			return
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Begin transaction
	tx, err := app.db.BeginTx(ctx, nil)
	if err != nil {
		app.logger.Errorw(op+": failed to begin transaction", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to update pricing rule",
			RequestID: c.GetString("request_id"),
		})
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Fetch existing rule
	var existing PricingRule
	var timeWindowsJSON, weatherCondJSON, eventTagsJSON []byte
	fetchQuery := `
		SELECT id, name, description, rule_type, priority,
			min_multiplier, max_multiplier, base_multiplier,
			demand_threshold_low, demand_threshold_high,
			supply_threshold_low, supply_threshold_high,
			time_windows, day_of_week_mask,
			weather_conditions, event_tags,
			is_active, created_by, created_at, updated_at
		FROM pricing_rules WHERE id = $1 FOR UPDATE`

	err = tx.QueryRowContext(ctx, fetchQuery, ruleID).Scan(
		&existing.ID, &existing.Name, &existing.Description, &existing.RuleType, &existing.Priority,
		&existing.MinMultiplier, &existing.MaxMultiplier, &existing.BaseMultiplier,
		&existing.DemandThresholdLow, &existing.DemandThresholdHigh,
		&existing.SupplyThresholdLow, &existing.SupplyThresholdHigh,
		&timeWindowsJSON, &existing.DayOfWeekMask,
		&weatherCondJSON, &eventTagsJSON,
		&existing.IsActive, &existing.CreatedBy, &existing.CreatedAt, &existing.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Code: "RULE_NOT_FOUND",
				Message: fmt.Sprintf("pricing rule %s not found", ruleID),
				RequestID: c.GetString("request_id"),
			})
			return
		}
		app.logger.Errorw(op+": failed to fetch pricing rule", "error", err, "rule_id", ruleID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to fetch pricing rule",
			RequestID: c.GetString("request_id"),
		})
		return
	}
	_ = json.Unmarshal(timeWindowsJSON, &existing.TimeWindows)
	_ = json.Unmarshal(weatherCondJSON, &existing.WeatherConditions)
	_ = json.Unmarshal(eventTagsJSON, &existing.EventTags)

	// Apply partial updates
	updated := existing
	if req.Name != nil {
		updated.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		updated.Description = strings.TrimSpace(*req.Description)
	}
	if req.Priority != nil {
		updated.Priority = *req.Priority
	}
	if req.MinMultiplier != nil {
		updated.MinMultiplier = *req.MinMultiplier
	}
	if req.MaxMultiplier != nil {
		updated.MaxMultiplier = *req.MaxMultiplier
	}
	if req.BaseMultiplier != nil {
		updated.BaseMultiplier = *req.BaseMultiplier
	}
	if req.DemandThresholdLow != nil {
		updated.DemandThresholdLow = *req.DemandThresholdLow
	}
	if req.DemandThresholdHigh != nil {
		updated.DemandThresholdHigh = *req.DemandThresholdHigh
	}
	if req.SupplyThresholdLow != nil {
		updated.SupplyThresholdLow = *req.SupplyThresholdLow
	}
	if req.SupplyThresholdHigh != nil {
		updated.SupplyThresholdHigh = *req.SupplyThresholdHigh
	}
	if req.TimeWindows != nil {
		updated.TimeWindows = req.TimeWindows
	}
	if req.DayOfWeekMask != nil {
		updated.DayOfWeekMask = *req.DayOfWeekMask
	}
	if req.WeatherConditions != nil {
		updated.WeatherConditions = req.WeatherConditions
	}
	if req.EventTags != nil {
		updated.EventTags = req.EventTags
	}
	if req.IsActive != nil {
		updated.IsActive = *req.IsActive
	}
	updated.UpdatedAt = time.Now().UTC()

	// Validate final state
	if updated.MinMultiplier > updated.MaxMultiplier {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_MULTIPLIER_RANGE",
			Message: "after update, min_multiplier would exceed max_multiplier",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	updTimeWindowsJSON, _ := json.Marshal(updated.TimeWindows)
	updWeatherCondJSON, _ := json.Marshal(updated.WeatherConditions)
	updEventTagsJSON, _ := json.Marshal(updated.EventTags)

	updateQuery := `
		UPDATE pricing_rules SET
			name = $1, description = $2, priority = $3,
			min_multiplier = $4, max_multiplier = $5, base_multiplier = $6,
			demand_threshold_low = $7, demand_threshold_high = $8,
			supply_threshold_low = $9, supply_threshold_high = $10,
			time_windows = $11, day_of_week_mask = $12,
			weather_conditions = $13, event_tags = $14,
			is_active = $15, updated_at = $16
		WHERE id = $17`

	_, err = tx.ExecContext(ctx, updateQuery,
		updated.Name, updated.Description, updated.Priority,
		updated.MinMultiplier, updated.MaxMultiplier, updated.BaseMultiplier,
		updated.DemandThresholdLow, updated.DemandThresholdHigh,
		updated.SupplyThresholdLow, updated.SupplyThresholdHigh,
		updTimeWindowsJSON, updated.DayOfWeekMask,
		updWeatherCondJSON, updEventTagsJSON,
		updated.IsActive, updated.UpdatedAt,
		ruleID,
	)
	if err != nil {
		app.logger.Errorw(op+": failed to update pricing rule", "error", err, "rule_id", ruleID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to update pricing rule",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	if err = tx.Commit(); err != nil {
		app.logger.Errorw(op+": failed to commit transaction", "error", err, "rule_id", ruleID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to commit pricing rule update",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Invalidate caches
	_ = app.cache.Delete(ctx, "pricing:rules:all")
	_ = app.cache.Delete(ctx, fmt.Sprintf("pricing:rule:%s", ruleID))
	app.invalidateCachePattern(ctx, "pricing:rules:list:*")

	// Publish event
	event := KafkaEvent{
		EventID: uuid.New().String(),
		EventType: "pricing.rule.updated",
		Timestamp: updated.UpdatedAt,
		Version: "1.0",
		Source: "pricing-service",
		Payload: map[string]interface{}{
			"rule_id": updated.ID,
			"rule_name": updated.Name,
			"is_active": updated.IsActive,
			"updated_by": fmt.Sprintf("%v", adminID),
		},
	}
	if err := app.publishEvent(ctx, app.config.Kafka.Topics.PricingUpdates, updated.ID, event); err != nil {
		app.logger.Warnw(op+": failed to publish rule updated event", "error", err, "rule_id", updated.ID)
	}

	app.logger.Infow("pricing rule updated",
		"rule_id", updated.ID,
		"admin_id", adminID,
	)

	c.JSON(http.StatusOK, PricingRuleResponse{
		Data: &updated,
		Meta: ResponseMeta{
			RequestID: c.GetString("request_id"),
			Timestamp: updated.UpdatedAt,
		},
	})
}

// deletePricingRule handles DELETE /api/v1/pricing/rules/:id
// Admin only: soft-deletes a pricing rule (marks is_active = false and sets deleted_at)
func (app *Application) deletePricingRule(c *gin.Context) {
	const op = "Application.deletePricingRule"

	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code: "UNAUTHORIZED",
			Message: "authentication required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	ruleID := c.Param("id")
	if ruleID == "" || !isValidUUID(ruleID) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_RULE_ID",
			Message: "valid rule ID is required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	now := time.Now().UTC()

	// Soft delete: mark inactive and set deleted_at
	result, err := app.db.ExecContext(ctx,
		`UPDATE pricing_rules
		 SET is_active = false, deleted_at = $1, updated_at = $1
		 WHERE id = $2 AND deleted_at IS NULL`,
		now, ruleID,
	)
	if err != nil {
		app.logger.Errorw(op+": failed to delete pricing rule", "error", err, "rule_id", ruleID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to delete pricing rule",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code: "RULE_NOT_FOUND",
			Message: fmt.Sprintf("pricing rule %s not found or already deleted", ruleID),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Invalidate caches
	_ = app.cache.Delete(ctx, "pricing:rules:all")
	_ = app.cache.Delete(ctx, fmt.Sprintf("pricing:rule:%s", ruleID))
	app.invalidateCachePattern(ctx, "pricing:rules:list:*")

	// Publish event
	event := KafkaEvent{
		EventID: uuid.New().String(),
		EventType: "pricing.rule.deleted",
		Timestamp: now,
		Version: "1.0",
		Source: "pricing-service",
		Payload: map[string]interface{}{
			"rule_id": ruleID,
			"deleted_by": fmt.Sprintf("%v", adminID),
			"deleted_at": now,
		},
	}
	if err := app.publishEvent(ctx, app.config.Kafka.Topics.PricingUpdates, ruleID, event); err != nil {
		app.logger.Warnw(op+": failed to publish rule deleted event", "error", err, "rule_id", ruleID)
	}

	app.logger.Infow("pricing rule deleted",
		"rule_id", ruleID,
		"admin_id", adminID,
	)

	c.JSON(http.StatusOK, SuccessResponse{
		Message: fmt.Sprintf("pricing rule %s successfully deleted", ruleID),
		Meta: ResponseMeta{
			RequestID: c.GetString("request_id"),
			Timestamp: now,
		},
	})
}

// âââ Surge Zones CRUD ââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ

// createSurgeZone handles POST /api/v1/pricing/zones
// Admin only: creates a new geographic surge zone
func (app *Application) createSurgeZone(c *gin.Context) {
	const op = "Application.createSurgeZone"

	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code: "UNAUTHORIZED",
			Message: "authentication required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	var req CreateSurgeZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_REQUEST",
			Message: "invalid request body: " + err.Error(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	if err := app.validator.Struct(req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, ValidationErrorResponse{
			Code: "VALIDATION_ERROR",
			Message: "request validation failed",
			Errors: formatValidationErrors(err),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Validate GeoJSON geometry
	if err := validateGeoJSON(req.Geometry); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_GEOMETRY",
			Message: "invalid GeoJSON geometry: " + err.Error(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Validate radius if provided (must be positive)
	if req.RadiusMeters != nil && *req.RadiusMeters <= 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_RADIUS",
			Message: "radius_meters must be positive",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	now := time.Now().UTC()
	zone := &SurgeZone{
		ID: uuid.New().String(),
		Name: strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		City: strings.TrimSpace(req.City),
		Country: strings.TrimSpace(req.Country),
		Geometry: req.Geometry,
		CenterLat: req.CenterLat,
		CenterLng: req.CenterLng,
		RadiusMeters: req.RadiusMeters,
		ZoneType: req.ZoneType,
		PricingRuleIDs: req.PricingRuleIDs,
		Metadata: req.Metadata,
		IsActive: true,
		CreatedBy: fmt.Sprintf("%v", adminID),
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	geometryJSON, _ := json.Marshal(zone.Geometry)
	pricingRuleIDsJSON, _ := json.Marshal(zone.PricingRuleIDs)
	metadataJSON, _ := json.Marshal(zone.Metadata)

	query := `
		INSERT INTO surge_zones (
			id, name, description, city, country,
			geometry, center_lat, center_lng, radius_meters,
			zone_type, pricing_rule_ids, metadata,
			is_active, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12,
			$13, $14, $15, $16
		) RETURNING id, created_at, updated_at`

	err := app.db.QueryRowContext(ctx, query,
		zone.ID, zone.Name, zone.Description, zone.City, zone.Country,
		geometryJSON, zone.CenterLat, zone.CenterLng, zone.RadiusMeters,
		zone.ZoneType, pricingRuleIDsJSON, metadataJSON,
		zone.IsActive, zone.CreatedBy, zone.CreatedAt, zone.UpdatedAt,
	).Scan(&zone.ID, &zone.CreatedAt, &zone.UpdatedAt)

	if err != nil {
		app.logger.Errorw(op+": failed to insert surge zone",
			"error", err,
			"zone_name", zone.Name,
			"admin_id", adminID,
		)
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Code: "ZONE_ALREADY_EXISTS",
				Message: "a surge zone with this name already exists in this city",
				RequestID: c.GetString("request_id"),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to create surge zone",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Invalidate zone caches
	_ = app.cache.Delete(ctx, "pricing:zones:all")
	app.invalidateCachePattern(ctx, fmt.Sprintf("pricing:zones:city:%s:*", zone.City))

	// Publish event
	event := KafkaEvent{
		EventID: uuid.New().String(),
		EventType: "pricing.zone.created",
		Timestamp: now,
		Version: "1.0",
		Source: "pricing-service",
		Payload: map[string]interface{}{
			"zone_id": zone.ID,
			"zone_name": zone.Name,
			"city": zone.City,
			"country": zone.Country,
			"zone_type": zone.ZoneType,
			"created_by": zone.CreatedBy,
		},
	}
	if err := app.publishEvent(ctx, app.config.Kafka.Topics.ZoneUpdates, zone.ID, event); err != nil {
		app.logger.Warnw(op+": failed to publish zone created event", "error", err, "zone_id", zone.ID)
	}

	app.logger.Infow("surge zone created",
		"zone_id", zone.ID,
		"zone_name", zone.Name,
		"city", zone.City,
		"admin_id", adminID,
	)

	c.JSON(http.StatusCreated, SurgeZoneResponse{
		Data: zone,
		Meta: ResponseMeta{
			RequestID: c.GetString("request_id"),
			Timestamp: now,
		},
	})
}

// listSurgeZones handles GET /api/v1/pricing/zones
// Admin only: returns paginated list of surge zones with optional filters
func (app *Application) listSurgeZones(c *gin.Context) {
	const op = "Application.listSurgeZones"

	// Parse query params
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	city := strings.TrimSpace(c.Query("city"))
	country := strings.TrimSpace(c.Query("country"))
	zoneType := c.Query("zone_type")
	isActiveStr := c.Query("is_active")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	cacheKey := fmt.Sprintf("pricing:zones:list:%d:%d:%s:%s:%s:%s",
		page, pageSize, city, country, zoneType, isActiveStr)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var cached SurgeZonesListResponse
	if err := app.cache.Get(ctx, cacheKey, &cached); err == nil {
		c.JSON(http.StatusOK, cached)
		return
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	if city != "" {
		conditions = append(conditions, fmt.Sprintf("city ILIKE $%d", argIdx))
		args = append(args, "%"+city+"%")
		argIdx++
	}
	if country != "" {
		conditions = append(conditions, fmt.Sprintf("country = $%d", argIdx))
		args = append(args, country)
		argIdx++
	}
	if zoneType != "" {
		conditions = append(conditions, fmt.Sprintf("zone_type = $%d", argIdx))
		args = append(args, zoneType)
		argIdx++
	}
	if isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
			args = append(args, isActive)
			argIdx++
		}
	}
	// Exclude soft-deleted zones
	conditions = append(conditions, "deleted_at IS NULL")

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := app.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM surge_zones %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		app.logger.Errorw(op+": failed to count surge zones", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to retrieve surge zones",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	offset := (page - 1) * pageSize
	listQuery := fmt.Sprintf(`
		SELECT
			id, name, description, city, country,
			geometry, center_lat, center_lng, radius_meters,
			zone_type, pricing_rule_ids, metadata,
			is_active, created_by, created_at, updated_at
		FROM surge_zones
		%s
		ORDER BY city ASC, name ASC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := app.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		app.logger.Errorw(op+": failed to query surge zones", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to retrieve surge zones",
			RequestID: c.GetString("request_id"),
		})
		return
	}
	defer rows.Close()

	var zones []*SurgeZone
	for rows.Next() {
		zone := &SurgeZone{}
		var geometryJSON, pricingRuleIDsJSON, metadataJSON []byte
		if err := rows.Scan(
			&zone.ID, &zone.Name, &zone.Description, &zone.City, &zone.Country,
			&geometryJSON, &zone.CenterLat, &zone.CenterLng, &zone.RadiusMeters,
			&zone.ZoneType, &pricingRuleIDsJSON, &metadataJSON,
			&zone.IsActive, &zone.CreatedBy, &zone.CreatedAt, &zone.UpdatedAt,
		); err != nil {
			app.logger.Errorw(op+": failed to scan surge zone row", "error", err)
			continue
		}
		_ = json.Unmarshal(geometryJSON, &zone.Geometry)
		_ = json.Unmarshal(pricingRuleIDsJSON, &zone.PricingRuleIDs)
		_ = json.Unmarshal(metadataJSON, &zone.Metadata)
		zones = append(zones, zone)
	}
	if err := rows.Err(); err != nil {
		app.logger.Errorw(op+": row iteration error", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "error reading surge zones",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	if zones == nil {
		zones = []*SurgeZone{}
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	resp := SurgeZonesListResponse{
		Data: zones,
		Pagination: PaginationMeta{
			Page: page,
			PageSize: pageSize,
			Total: total,
			TotalPages: totalPages,
			HasNext: page < totalPages,
			HasPrev: page > 1,
		},
		Meta: ResponseMeta{
			RequestID: c.GetString("request_id"),
			Timestamp: time.Now().UTC(),
		},
	}

	if err := app.cache.Set(ctx, cacheKey, resp, 30*time.Second); err != nil {
		app.logger.Warnw(op+": failed to cache zones list", "error", err)
	}

	c.JSON(http.StatusOK, resp)
}

// updateSurgeZone handles PUT /api/v1/pricing/zones/:id
// Admin only: updates an existing surge zone
func (app *Application) updateSurgeZone(c *gin.Context) {
	const op = "Application.updateSurgeZone"

	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code: "UNAUTHORIZED",
			Message: "authentication required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	zoneID := c.Param("id")
	if zoneID == "" || !isValidUUID(zoneID) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_ZONE_ID",
			Message: "valid zone ID is required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	var req UpdateSurgeZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_REQUEST",
			Message: "invalid request body: " + err.Error(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	if err := app.validator.Struct(req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, ValidationErrorResponse{
			Code: "VALIDATION_ERROR",
			Message: "request validation failed",
			Errors: formatValidationErrors(err),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	if req.Geometry != nil {
		if err := validateGeoJSON(req.Geometry); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code: "INVALID_GEOMETRY",
				Message: "invalid GeoJSON geometry: " + err.Error(),
				RequestID: c.GetString("request_id"),
			})
			return
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	tx, err := app.db.BeginTx(ctx, nil)
	if err != nil {
		app.logger.Errorw(op+": failed to begin transaction", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to update surge zone",
			RequestID: c.GetString("request_id"),
		})
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var existing SurgeZone
	var geometryJSON, pricingRuleIDsJSON, metadataJSON []byte

	err = tx.QueryRowContext(ctx, `
		SELECT id, name, description, city, country,
			geometry, center_lat, center_lng, radius_meters,
			zone_type, pricing_rule_ids, metadata,
			is_active, created_by, created_at, updated_at
		FROM surge_zones
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE`, zoneID,
	).Scan(
		&existing.ID, &existing.Name, &existing.Description, &existing.City, &existing.Country,
		&geometryJSON, &existing.CenterLat, &existing.CenterLng, &existing.RadiusMeters,
		&existing.ZoneType, &pricingRuleIDsJSON, &metadataJSON,
		&existing.IsActive, &existing.CreatedBy, &existing.CreatedAt, &existing.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Code: "ZONE_NOT_FOUND",
				Message: fmt.Sprintf("surge zone %s not found", zoneID),
				RequestID: c.GetString("request_id"),
			})
			return
		}
		app.logger.Errorw(op+": failed to fetch surge zone", "error", err, "zone_id", zoneID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to fetch surge zone",
			RequestID: c.GetString("request_id"),
		})
		return
	}
	_ = json.Unmarshal(geometryJSON, &existing.Geometry)
	_ = json.Unmarshal(pricingRuleIDsJSON, &existing.PricingRuleIDs)
	_ = json.Unmarshal(metadataJSON, &existing.Metadata)

	// Apply partial updates
	updated := existing
	if req.Name != nil {
		updated.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		updated.Description = strings.TrimSpace(*req.Description)
	}
	if req.City != nil {
		updated.City = strings.TrimSpace(*req.City)
	}
	if req.Country != nil {
		updated.Country = strings.TrimSpace(*req.Country)
	}
	if req.Geometry != nil {
		updated.Geometry = req.Geometry
	}
	if req.CenterLat != nil {
		updated.CenterLat = *req.CenterLat
	}
	if req.CenterLng != nil {
		updated.CenterLng = *req.CenterLng
	}
	if req.RadiusMeters != nil {
		updated.RadiusMeters = req.RadiusMeters
	}
	if req.ZoneType != nil {
		updated.ZoneType = *req.ZoneType
	}
	if req.PricingRuleIDs != nil {
		updated.PricingRuleIDs = req.PricingRuleIDs
	}
	if req.Metadata != nil {
		updated.Metadata = req.Metadata
	}
	if req.IsActive != nil {
		updated.IsActive = *req.IsActive
	}
	updated.UpdatedAt = time.Now().UTC()

	updGeometryJSON, _ := json.Marshal(updated.Geometry)
	updPricingRuleIDsJSON, _ := json.Marshal(updated.PricingRuleIDs)
	updMetadataJSON, _ := json.Marshal(updated.Metadata)

	_, err = tx.ExecContext(ctx, `
		UPDATE surge_zones SET
			name = $1, description = $2, city = $3, country = $4,
			geometry = $5, center_lat = $6, center_lng = $7, radius_meters = $8,
			zone_type = $9, pricing_rule_ids = $10, metadata = $11,
			is_active = $12, updated_at = $13
		WHERE id = $14`,
		updated.Name, updated.Description, updated.City, updated.Country,
		updGeometryJSON, updated.CenterLat, updated.CenterLng, updated.RadiusMeters,
		updated.ZoneType, updPricingRuleIDsJSON, updMetadataJSON,
		updated.IsActive, updated.UpdatedAt,
		zoneID,
	)
	if err != nil {
		app.logger.Errorw(op+": failed to update surge zone", "error", err, "zone_id", zoneID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to update surge zone",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	if err = tx.Commit(); err != nil {
		app.logger.Errorw(op+": failed to commit transaction", "error", err, "zone_id", zoneID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to commit surge zone update",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Invalidate caches
	_ = app.cache.Delete(ctx, "pricing:zones:all")
	_ = app.cache.Delete(ctx, fmt.Sprintf("pricing:zone:%s", zoneID))
	app.invalidateCachePattern(ctx, fmt.Sprintf("pricing:zones:city:%s:*", updated.City))
	app.invalidateCachePattern(ctx, "pricing:zones:list:*")

	// Publish event
	event := KafkaEvent{
		EventID: uuid.New().String(),
		EventType: "pricing.zone.updated",
		Timestamp: updated.UpdatedAt,
		Version: "1.0",
		Source: "pricing-service",
		Payload: map[string]interface{}{
			"zone_id": updated.ID,
			"zone_name": updated.Name,
			"city": updated.City,
			"is_active": updated.IsActive,
			"updated_by": fmt.Sprintf("%v", adminID),
		},
	}
	if err := app.publishEvent(ctx, app.config.Kafka.Topics.ZoneUpdates, updated.ID, event); err != nil {
		app.logger.Warnw(op+": failed to publish zone updated event", "error", err, "zone_id", updated.ID)
	}

	app.logger.Infow("surge zone updated",
		"zone_id", updated.ID,
		"admin_id", adminID,
	)

	c.JSON(http.StatusOK, SurgeZoneResponse{
		Data: &updated,
		Meta: ResponseMeta{
			RequestID: c.GetString("request_id"),
			Timestamp: updated.UpdatedAt,
		},
	})
}

// deleteSurgeZone handles DELETE /api/v1/pricing/zones/:id
// Admin only: soft-deletes a surge zone
func (app *Application) deleteSurgeZone(c *gin.Context) {
	const op = "Application.deleteSurgeZone"

	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code: "UNAUTHORIZED",
			Message: "authentication required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	zoneID := c.Param("id")
	if zoneID == "" || !isValidUUID(zoneID) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_ZONE_ID",
			Message: "valid zone ID is required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	now := time.Now().UTC()

	// Retrieve city before deletion for cache invalidation
	var city string
	_ = app.db.QueryRowContext(ctx, "SELECT city FROM surge_zones WHERE id = $1 AND deleted_at IS NULL", zoneID).Scan(&city)

	result, err := app.db.ExecContext(ctx,
		`UPDATE surge_zones
		 SET is_active = false, deleted_at = $1, updated_at = $1
		 WHERE id = $2 AND deleted_at IS NULL`,
		now, zoneID,
	)
	if err != nil {
		app.logger.Errorw(op+": failed to delete surge zone", "error", err, "zone_id", zoneID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to delete surge zone",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code: "ZONE_NOT_FOUND",
			Message: fmt.Sprintf("surge zone %s not found or already deleted", zoneID),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Invalidate caches
	_ = app.cache.Delete(ctx, "pricing:zones:all")
	_ = app.cache.Delete(ctx, fmt.Sprintf("pricing:zone:%s", zoneID))
	if city != "" {
		app.invalidateCachePattern(ctx, fmt.Sprintf("pricing:zones:city:%s:*", city))
	}
	app.invalidateCachePattern(ctx, "pricing:zones:list:*")

	// Also invalidate any surge multiplier caches referencing this zone
	app.invalidateCachePattern(ctx, fmt.Sprintf("pricing:surge:%s:*", zoneID))

	// Publish event
	event := KafkaEvent{
		EventID: uuid.New().String(),
		EventType: "pricing.zone.deleted",
		Timestamp: now,
		Version: "1.0",
		Source: "pricing-service",
		Payload: map[string]interface{}{
			"zone_id": zoneID,
			"city": city,
			"deleted_by": fmt.Sprintf("%v", adminID),
			"deleted_at": now,
		},
	}
	if err := app.publishEvent(ctx, app.config.Kafka.Topics.ZoneUpdates, zoneID, event); err != nil {
		app.logger.Warnw(op+": failed to publish zone deleted event", "error", err, "zone_id", zoneID)
	}

	app.logger.Infow("surge zone deleted",
		"zone_id", zoneID,
		"city", city,
		"admin_id", adminID,
	)

	c.JSON(http.StatusOK, SuccessResponse{
		Message: fmt.Sprintf("surge zone %s successfully deleted", zoneID),
		Meta: ResponseMeta{
			RequestID: c.GetString("request_id"),
			Timestamp: now,
		},
	})
}

// âââ Demand Reporting ââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ

// reportDemand handles POST /api/v1/pricing/demand/report
// Internal endpoint: receives demand signals from other services
func (app *Application) reportDemand(c *gin.Context) {
	const op = "Application.reportDemand"

	// Verify internal service token
	serviceToken := c.GetHeader("X-Service-Token")
	if serviceToken == "" || !app.isValidServiceToken(serviceToken) {
		app.logger.Warnw(op+": invalid or missing service token",
			"remote_addr", c.ClientIP(),
		)
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code: "UNAUTHORIZED",
			Message: "valid service token required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	var req DemandReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_REQUEST",
			Message: "invalid request body: " + err.Error(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	if err := app.validator.Struct(req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, ValidationErrorResponse{
			Code: "VALIDATION_ERROR",
			Message: "request validation failed",
			Errors: formatValidationErrors(err),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Validate zone exists
	if !isValidUUID(req.ZoneID) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_ZONE_ID",
			Message: "valid zone ID is required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	if req.ActiveRiders < 0 || req.AvailableDrivers < 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_COUNTS",
			Message: "active_riders and available_drivers must be non-negative",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	now := time.Now().UTC()
	if req.Timestamp.IsZero() {
		req.Timestamp = now
	}

	// Compute demand ratio
	var demandRatio float64
	if req.AvailableDrivers > 0 {
		demandRatio = float64(req.ActiveRiders) / float64(req.AvailableDrivers)
	} else {
		// No drivers: extreme demand
		demandRatio = float64(req.ActiveRiders) * 10.0
	}

	report := &DemandReport{
		ID: uuid.New().String(),
		ZoneID: req.ZoneID,
		ActiveRiders: req.ActiveRiders,
		AvailableDrivers: req.AvailableDrivers,
		PendingRequests: req.PendingRequests,
		CompletedRides: req.CompletedRides,
		CancelledRides: req.CancelledRides,
		AverageWaitSeconds: req.AverageWaitSeconds,
		DemandRatio: demandRatio,
		WeatherCode: req.WeatherCode,
		TemperatureCelsius: req.TemperatureCelsius,
		IsSpecialEvent: req.IsSpecialEvent,
		EventName: req.EventName,
		SourceService: req.SourceService,
		ReportedAt: req.Timestamp,
		CreatedAt: now,
	}

	// Persist to time-series table
	insertQuery := `
		INSERT INTO demand_reports (
			id, zone_id, active_riders, available_drivers,
			pending_requests, completed_rides, cancelled_rides,
			average_wait_seconds, demand_ratio,
			weather_code, temperature_celsius,
			is_special_event, event_name,
			source_service, reported_at, created_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9,
			$10, $11,
			$12, $13,
			$14, $15, $16
		)`

	_, err := app.db.ExecContext(ctx, insertQuery,
		report.ID, report.ZoneID, report.ActiveRiders, report.AvailableDrivers,
		report.PendingRequests, report.CompletedRides, report.CancelledRides,
		report.AverageWaitSeconds, report.DemandRatio,
		report.WeatherCode, report.TemperatureCelsius,
		report.IsSpecialEvent, report.EventName,
		report.SourceService, report.ReportedAt, report.CreatedAt,
	)
	if err != nil {
		app.logger.Errorw(op+": failed to persist demand report",
			"error", err,
			"zone_id", report.ZoneID,
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to record demand report",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Update real-time demand state in Redis using atomic operations
	demandKey := fmt.Sprintf("demand:zone:%s:current", report.ZoneID)
	demandState := map[string]interface{}{
		"zone_id": report.ZoneID,
		"active_riders": report.ActiveRiders,
		"available_drivers": report.AvailableDrivers,
		"pending_requests": report.PendingRequests,
		"demand_ratio": report.DemandRatio,
		"average_wait_seconds": report.AverageWaitSeconds,
		"weather_code": report.WeatherCode,
		"is_special_event": report.IsSpecialEvent,
		"event_name": report.EventName,
		"updated_at": now.Unix(),
	}
	if err := app.cache.Set(ctx, demandKey, demandState, 5*time.Minute); err != nil {
		app.logger.Warnw(op+": failed to update demand state in cache",
			"error", err,
			"zone_id", report.ZoneID,
		)
	}

	// Invalidate existing surge multiplier cache for this zone so next request recalculates
	app.invalidateCachePattern(ctx, fmt.Sprintf("pricing:surge:%s:*", report.ZoneID))

	// Publish demand signal event
	event := KafkaEvent{
		EventID: uuid.New().String(),
		EventType: "demand.signal.received",
		Timestamp: now,
		Version: "1.0",
		Source: "pricing-service",
		Payload: map[string]interface{}{
			"report_id": report.ID,
			"zone_id": report.ZoneID,
			"active_riders": report.ActiveRiders,
			"available_drivers": report.AvailableDrivers,
			"demand_ratio": report.DemandRatio,
			"is_special_event": report.IsSpecialEvent,
			"source_service": report.SourceService,
		},
	}
	if err := app.publishEvent(ctx, app.config.Kafka.Topics.DemandSignals, report.ZoneID, event); err != nil {
		app.logger.Warnw(op+": failed to publish demand signal event",
			"error", err,
			"zone_id", report.ZoneID,
		)
	}

	app.logger.Infow("demand report recorded",
		"report_id", report.ID,
		"zone_id", report.ZoneID,
		"demand_ratio", fmt.Sprintf("%.3f", report.DemandRatio),
		"source", report.SourceService,
	)

	c.JSON(http.StatusCreated, DemandReportResponse{
		Data: report,
		Meta: ResponseMeta{
			RequestID: c.GetString("request_id"),
			Timestamp: now,
		},
	})
}

// getDemandForZone handles GET /api/v1/pricing/demand/:zone_id
// Internal endpoint: returns demand history and current state for a zone
func (app *Application) getDemandForZone(c *gin.Context) {
	const op = "Application.getDemandForZone"

	zoneID := c.Param("zone_id")
	if zoneID == "" || !isValidUUID(zoneID) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_ZONE_ID",
			Message: "valid zone ID is required",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Parse time range
	now := time.Now().UTC()
	fromStr := c.DefaultQuery("from", now.Add(-1*time.Hour).Format(time.RFC3339))
	toStr := c.DefaultQuery("to", now.Format(time.RFC3339))
	granularity := c.DefaultQuery("granularity", "5m") // 1m, 5m, 15m, 1h
	includeCurrentStr := c.DefaultQuery("include_current", "true")

	fromTime, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_FROM_TIME",
			Message: "from must be a valid RFC3339 timestamp",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	toTime, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_TO_TIME",
			Message: "to must be a valid RFC3339 timestamp",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	if toTime.Before(fromTime) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_TIME_RANGE",
			Message: "to must be after from",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Enforce max range of 24 hours for performance
	if toTime.Sub(fromTime) > 24*time.Hour {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "TIME_RANGE_TOO_LARGE",
			Message: "time range cannot exceed 24 hours",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Map granularity to truncation interval
	granularityMap := map[string]string{
		"1m": "minute",
		"5m": "5 minutes",
		"15m": "15 minutes",
		"1h": "hour",
	}
	dbInterval, ok := granularityMap[granularity]
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code: "INVALID_GRANULARITY",
			Message: "granularity must be one of: 1m, 5m, 15m, 1h",
			RequestID: c.GetString("request_id"),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	// Verify zone exists
	var zoneExists bool
	if err := app.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM surge_zones WHERE id = $1 AND deleted_at IS NULL)",
		zoneID,
	).Scan(&zoneExists); err != nil {
		app.logger.Errorw(op+": failed to check zone existence", "error", err, "zone_id", zoneID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to retrieve demand data",
			RequestID: c.GetString("request_id"),
		})
		return
	}
	if !zoneExists {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code: "ZONE_NOT_FOUND",
			Message: fmt.Sprintf("surge zone %s not found", zoneID),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Query aggregated demand history using time-bucketing
	// Using PostgreSQL date_trunc for bucketing
	historyQuery := fmt.Sprintf(`
		SELECT
			date_trunc('%s', reported_at) AS bucket,
			AVG(active_riders)::int AS avg_active_riders,
			AVG(available_drivers)::int AS avg_available_drivers,
			AVG(pending_requests)::int AS avg_pending_requests,
			AVG(demand_ratio)::numeric(10,4) AS avg_demand_ratio,
			AVG(average_wait_seconds)::int AS avg_wait_seconds,
			COUNT(*) AS sample_count
		FROM demand_reports
		WHERE zone_id = $1
			AND reported_at >= $2
			AND reported_at <= $3
		GROUP BY bucket
		ORDER BY bucket ASC`,
		dbInterval,
	)

	rows, err := app.db.QueryContext(ctx, historyQuery, zoneID, fromTime, toTime)
	if err != nil {
		app.logger.Errorw(op+": failed to query demand history",
			"error", err,
			"zone_id", zoneID,
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code: "DB_ERROR",
			Message: "failed to retrieve demand history",
			RequestID: c.GetString("request_id"),
		})
		return
	}
	defer rows.Close()

	var history []DemandDataPoint
	for rows.Next() {
		var dp DemandDataPoint
		if err := rows.Scan(
			&dp.Bucket,
			&dp.AvgActiveRiders,
			&dp.AvgAvailableDrivers,
			&dp.AvgPendingRequests,
			&dp.AvgDemandRatio,
			&dp.AvgWaitSeconds,
			&dp.SampleCount,
		); err != nil {
			app.logger.Errorw(op+": failed to scan demand data point", "error", err)
			continue
		}
		history = append(history, dp)
	}
	if err := rows.Err(); err != nil {
		app.logger.Errorw(op+": row iteration error on demand history", "error", err)
	}

	if history == nil {
		history = []DemandDataPoint{}
	}

	// Optionally include current demand state from cache
	var currentState *DemandCurrentState
	includeCurrent, _ := strconv.ParseBool(includeCurrentStr)
	if includeCurrent {
		demandKey := fmt.Sprintf("demand:zone:%s:current", zoneID)
		var stateMap map[string]interface{}
		if err := app.cache.Get(ctx, demandKey, &stateMap); err == nil {
			currentState = &DemandCurrentState{
				ZoneID: zoneID,
				ActiveRiders: toInt(stateMap["active_riders"]),
				AvailableDrivers: toInt(stateMap["available_drivers"]),
				PendingRequests: toInt(stateMap["pending_requests"]),
				DemandRatio: toFloat64(stateMap["demand_ratio"]),
				AverageWaitSeconds: toInt(stateMap["average_wait_seconds"]),
				WeatherCode: toString(stateMap["weather_code"]),
				IsSpecialEvent: toBool(stateMap["is_special_event"]),
				EventName: toString(stateMap["event_name"]),
				UpdatedAt: time.Unix(toInt64(stateMap["updated_at"]), 0).UTC(),
			}
		}
	}

	// Compute summary statistics
	var peakDemandRatio float64
	var avgDemandRatio float64
	var totalSamples int
	for _, dp := range history {
		if dp.AvgDemandRatio > peakDemandRatio {
			peakDemandRatio = dp.AvgDemandRatio
		}
		avgDemandRatio += dp.AvgDemandRatio
		totalSamples += dp.SampleCount
	}
	if len(history) > 0 {
		avgDemandRatio /= float64(len(history))
	}

	app.logger.Debugw("demand data retrieved",
		"zone_id", zoneID,
		"data_points", len(history),
		"total_samples", totalSamples,
		"from", fromTime,
		"to", toTime,
	)

	c.JSON(http.StatusOK, DemandZoneResponse{
		ZoneID: zoneID,
		From: fromTime,
		To: toTime,
		Granularity: granularity,
		History: history,
		CurrentState: currentState,
		Summary: DemandSummary{
			TotalSamples: totalSamples,
			DataPoints: len(history),
			PeakDemandRatio: peakDemandRatio,
			AvgDemandRatio: avgDemandRatio,
		},
		Meta: ResponseMeta{
			RequestID: c.GetString("request_id"),
			Timestamp: now,
		},
	})
}

// âââ Helper Functions âââââââââââââââââââââââââââââââââââââââââââââââââââââââââ

// isValidServiceToken verifies the provided token against configured service tokens
func (app *Application) isValidServiceToken(token string) bool {
	for _, validToken := range app.config.Security.ServiceTokens {
		if subtle.ConstantTimeCompare([]byte(token), []byte(validToken)) == 1 {
			return true
		}
	}
	return false
}

// invalidateCachePattern deletes all cache keys matching a pattern
// Uses Redis SCAN to avoid blocking operations on large keyspaces
func (app *Application) invalidateCachePattern(ctx context.Context, pattern string) {
	if err := app.cache.DeletePattern(ctx, pattern); err != nil {
		app.logger.Warnw("failed to invalidate cache pattern",
			"pattern", pattern,
			"error", err,
		)
	}
}

// validateGeoJSON performs basic structural validation of a GeoJSON geometry object
func validateGeoJSON(geometry interface{}) error {
	if geometry == nil {
		return fmt.Errorf("geometry cannot be nil")
	}
	geomMap, ok := geometry.(map[string]interface{})
	if !ok {
		// Try marshaling and unmarshaling to normalize
		data, err := json.Marshal(geometry)
		if err != nil {
			return fmt.Errorf("geometry must be a valid JSON object")
		}
		if err := json.Unmarshal(data, &geomMap); err != nil {
			return fmt.Errorf("geometry must be a valid JSON object")
		}
	}

	geoType, ok := geomMap["type"].(string)
	if !ok || geoType == "" {
		return fmt.Errorf("geometry must have a 'type' field")
	}

	validTypes := map[string]bool{
		"Point": true, "MultiPoint": true,
		"LineString": true, "MultiLineString": true,
		"Polygon": true, "MultiPolygon": true,
		"GeometryCollection": true,
	}
	if !validTypes[geoType] {
		return fmt.Errorf("invalid geometry type: %s", geoType)
	}

	// For non-collection types, coordinates are required
	if geoType != "GeometryCollection" {
		if _, hasCoords := geomMap["coordinates"]; !hasCoords {
			return fmt.Errorf("geometry of type %s must have 'coordinates'", geoType)
		}
	}

	return nil
}

// isValidUUID validates that a string is a well-formed UUID
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// isUniqueViolation checks if a database error is a unique constraint violation
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL unique violation error code: 23505
	return strings.Contains(err.Error(), "23505") ||
		strings.Contains(err.Error(), "unique_violation") ||
		strings.Contains(err.Error(), "duplicate key")
}

// formatValidationErrors converts validator errors to a human-readable map
func formatValidationErrors(err error) map[string]string {
	errMap := make(map[string]string)
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		for _, ve := range validationErrs {
			field := toSnakeCase(ve.Field())
			switch ve.Tag() {
			case "required":
				errMap[field] = fmt.Sprintf("%s is required", field)
			case "min":
				errMap[field] = fmt.Sprintf("%s must be at least %s", field, ve.Param())
			case "max":
				errMap[field] = fmt.Sprintf("%s must be at most %s", field, ve.Param())
			case "gt":
				errMap[field] = fmt.Sprintf("%s must be greater than %s", field, ve.Param())
			case "gte":
				errMap[field] = fmt.Sprintf("%s must be greater than or equal to %s", field, ve.Param())
			case "lt":
				errMap[field] = fmt.Sprintf("%s must be less than %s", field, ve.Param())
			case "lte":
				errMap[field] = fmt.Sprintf("%s must be less than or equal to %s", field, ve.Param())
			case "email":
				errMap[field] = fmt.Sprintf("%s must be a valid email address", field)
			case "oneof":
				errMap[field] = fmt.Sprintf("%s must be one of: %s", field, ve.Param())
			case "uuid4":
				errMap[field] = fmt.Sprintf("%s must be a valid UUID v4", field)
			default:
				errMap[field] = fmt.Sprintf("%s failed validation: %s", field, ve.Tag())
			}
		}
	}
	return errMap
}

// toSnakeCase converts CamelCase to snake_case for field names
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// Type conversion helpers for cache state maps
func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case json.Number:
		i, _ := val.Int64()
		return int(i)
	}
	return 0
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int64:
		return val
	case float64:
		return int64(val)
	case json.Number:
		i, _ := val.Int64()
		return i
	}
	return 0
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case json.Number:
		f, _ := val.Float64()
		return f
	}
	return 0.0
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		b, _ := strconv.ParseBool(val)
		return b
	case float64:
		return val != 0
	}
	return false
}

// ============================================================
// PART 5: Analytics Handlers, Repository, Cache, Metrics, Utils
// ============================================================

// âââ Analytics Handlers âââââââââââââââââââââââââââââââââââââââââââââââââââââ

func (s *Server) handleGetPricingAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	zoneID := r.URL.Query().Get("zone_id")
	periodStr := r.URL.Query().Get("period") // e.g. "1h", "24h", "7d"
	if periodStr == "" {
		periodStr = "24h"
	}

	duration, err := parseDuration(periodStr)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_PERIOD", "Invalid period format. Use e.g. 1h, 24h, 7d", nil)
		return
	}

	since := time.Now().Add(-duration)

	analytics, err := s.repo.GetPricingAnalytics(ctx, zoneID, since)
	if err != nil {
		s.logger.Error("failed to fetch pricing analytics", zap.Error(err))
		writeErrorResponse(w, http.StatusInternalServerError, "ANALYTICS_ERROR", "Failed to retrieve analytics", nil)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"period": periodStr,
		"since": since.UTC().Format(time.RFC3339),
		"analytics": analytics,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleGetPricingHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	zoneID := r.URL.Query().Get("zone_id")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	limit := 100
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var startTime, endTime time.Time
	var err error

	if startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "INVALID_START_TIME", "Invalid start time format, use RFC3339", nil)
			return
		}
	} else {
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "INVALID_END_TIME", "Invalid end time format, use RFC3339", nil)
			return
		}
	} else {
		endTime = time.Now()
	}

	if endTime.Before(startTime) {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_TIME_RANGE", "End time must be after start time", nil)
		return
	}

	history, total, err := s.repo.GetPricingHistory(ctx, zoneID, startTime, endTime, limit, offset)
	if err != nil {
		s.logger.Error("failed to fetch pricing history", zap.Error(err))
		writeErrorResponse(w, http.StatusInternalServerError, "HISTORY_ERROR", "Failed to retrieve pricing history", nil)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"total": total,
		"limit": limit,
		"offset": offset,
		"start": startTime.UTC().Format(time.RFC3339),
		"end": endTime.UTC().Format(time.RFC3339),
		"zone_id": zoneID,
		"history": history,
	})
}

// âââ Repository Functions ââââââââââââââââââââââââââââââââââââââââââââââââââââ

func (r *Repository) GetAllPricingRules(ctx context.Context) ([]*PricingRule, error) {
	query := `
		SELECT id, name, zone_id, vehicle_type, base_fare, base_distance_rate,
		       base_time_rate, minimum_fare, maximum_fare, currency,
		       surge_enabled, max_surge_multiplier, demand_threshold_low,
		       demand_threshold_medium, demand_threshold_high,
		       demand_threshold_very_high, surge_multiplier_low,
		       surge_multiplier_medium, surge_multiplier_high,
		       surge_multiplier_very_high, time_of_day_enabled,
		       weather_enabled, event_enabled, is_active, priority,
		       effective_from, effective_until, created_at, updated_at
		FROM pricing_rules
		WHERE is_active = TRUE
		  AND (effective_until IS NULL OR effective_until > NOW())
		ORDER BY priority DESC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying pricing rules: %w", err)
	}
	defer rows.Close()

	var rules []*PricingRule
	for rows.Next() {
		rule := &PricingRule{}
		err := rows.Scan(
			&rule.ID, &rule.Name, &rule.ZoneID, &rule.VehicleType,
			&rule.BaseFare, &rule.BaseDistanceRate, &rule.BaseTimeRate,
			&rule.MinimumFare, &rule.MaximumFare, &rule.Currency,
			&rule.SurgeEnabled, &rule.MaxSurgeMultiplier,
			&rule.DemandThresholdLow, &rule.DemandThresholdMedium,
			&rule.DemandThresholdHigh, &rule.DemandThresholdVeryHigh,
			&rule.SurgeMultiplierLow, &rule.SurgeMultiplierMedium,
			&rule.SurgeMultiplierHigh, &rule.SurgeMultiplierVeryHigh,
			&rule.TimeOfDayEnabled, &rule.WeatherEnabled, &rule.EventEnabled,
			&rule.IsActive, &rule.Priority, &rule.EffectiveFrom,
			&rule.EffectiveUntil, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning pricing rule: %w", err)
		}
		rules = append(rules, rule)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating pricing rules: %w", err)
	}

	return rules, nil
}

func (r *Repository) GetPricingRuleByID(ctx context.Context, id string) (*PricingRule, error) {
	query := `
		SELECT id, name, zone_id, vehicle_type, base_fare, base_distance_rate,
		       base_time_rate, minimum_fare, maximum_fare, currency,
		       surge_enabled, max_surge_multiplier, demand_threshold_low,
		       demand_threshold_medium, demand_threshold_high,
		       demand_threshold_very_high, surge_multiplier_low,
		       surge_multiplier_medium, surge_multiplier_high,
		       surge_multiplier_very_high, time_of_day_enabled,
		       weather_enabled, event_enabled, is_active, priority,
		       effective_from, effective_until, created_at, updated_at
		FROM pricing_rules
		WHERE id = $1
	`

	rule := &PricingRule{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rule.ID, &rule.Name, &rule.ZoneID, &rule.VehicleType,
		&rule.BaseFare, &rule.BaseDistanceRate, &rule.BaseTimeRate,
		&rule.MinimumFare, &rule.MaximumFare, &rule.Currency,
		&rule.SurgeEnabled, &rule.MaxSurgeMultiplier,
		&rule.DemandThresholdLow, &rule.DemandThresholdMedium,
		&rule.DemandThresholdHigh, &rule.DemandThresholdVeryHigh,
		&rule.SurgeMultiplierLow, &rule.SurgeMultiplierMedium,
		&rule.SurgeMultiplierHigh, &rule.SurgeMultiplierVeryHigh,
		&rule.TimeOfDayEnabled, &rule.WeatherEnabled, &rule.EventEnabled,
		&rule.IsActive, &rule.Priority, &rule.EffectiveFrom,
		&rule.EffectiveUntil, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying pricing rule by id: %w", err)
	}

	return rule, nil
}

func (r *Repository) CreatePricingRule(ctx context.Context, rule *PricingRule) error {
	query := `
		INSERT INTO pricing_rules (
			id, name, zone_id, vehicle_type, base_fare, base_distance_rate,
			base_time_rate, minimum_fare, maximum_fare, currency,
			surge_enabled, max_surge_multiplier, demand_threshold_low,
			demand_threshold_medium, demand_threshold_high,
			demand_threshold_very_high, surge_multiplier_low,
			surge_multiplier_medium, surge_multiplier_high,
			surge_multiplier_very_high, time_of_day_enabled,
			weather_enabled, event_enabled, is_active, priority,
			effective_from, effective_until, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22,$23,$24,$25,$26,$27,$28,$29
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		rule.ID, rule.Name, rule.ZoneID, rule.VehicleType,
		rule.BaseFare, rule.BaseDistanceRate, rule.BaseTimeRate,
		rule.MinimumFare, rule.MaximumFare, rule.Currency,
		rule.SurgeEnabled, rule.MaxSurgeMultiplier,
		rule.DemandThresholdLow, rule.DemandThresholdMedium,
		rule.DemandThresholdHigh, rule.DemandThresholdVeryHigh,
		rule.SurgeMultiplierLow, rule.SurgeMultiplierMedium,
		rule.SurgeMultiplierHigh, rule.SurgeMultiplierVeryHigh,
		rule.TimeOfDayEnabled, rule.WeatherEnabled, rule.EventEnabled,
		rule.IsActive, rule.Priority, rule.EffectiveFrom,
		rule.EffectiveUntil, rule.CreatedAt, rule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting pricing rule: %w", err)
	}
	return nil
}

func (r *Repository) UpdatePricingRule(ctx context.Context, rule *PricingRule) error {
	query := `
		UPDATE pricing_rules SET
			name=$2, zone_id=$3, vehicle_type=$4, base_fare=$5,
			base_distance_rate=$6, base_time_rate=$7, minimum_fare=$8,
			maximum_fare=$9, currency=$10, surge_enabled=$11,
			max_surge_multiplier=$12, demand_threshold_low=$13,
			demand_threshold_medium=$14, demand_threshold_high=$15,
			demand_threshold_very_high=$16, surge_multiplier_low=$17,
			surge_multiplier_medium=$18, surge_multiplier_high=$19,
			surge_multiplier_very_high=$20, time_of_day_enabled=$21,
			weather_enabled=$22, event_enabled=$23, is_active=$24,
			priority=$25, effective_from=$26, effective_until=$27,
			updated_at=$28
		WHERE id=$1
	`
	_, err := r.db.ExecContext(ctx, query,
		rule.ID, rule.Name, rule.ZoneID, rule.VehicleType,
		rule.BaseFare, rule.BaseDistanceRate, rule.BaseTimeRate,
		rule.MinimumFare, rule.MaximumFare, rule.Currency,
		rule.SurgeEnabled, rule.MaxSurgeMultiplier,
		rule.DemandThresholdLow, rule.DemandThresholdMedium,
		rule.DemandThresholdHigh, rule.DemandThresholdVeryHigh,
		rule.SurgeMultiplierLow, rule.SurgeMultiplierMedium,
		rule.SurgeMultiplierHigh, rule.SurgeMultiplierVeryHigh,
		rule.TimeOfDayEnabled, rule.WeatherEnabled, rule.EventEnabled,
		rule.IsActive, rule.Priority, rule.EffectiveFrom,
		rule.EffectiveUntil, rule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("updating pricing rule: %w", err)
	}
	return nil
}

func (r *Repository) DeletePricingRule(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pricing_rules SET is_active=FALSE, updated_at=$2 WHERE id=$1`,
		id, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("soft-deleting pricing rule: %w", err)
	}
	return nil
}

func (r *Repository) GetAllZones(ctx context.Context) ([]*PricingZone, error) {
	query := `
		SELECT id, name, city, country, zone_type, polygon_geojson,
		       center_lat, center_lng, radius_km, surge_multiplier,
		       is_active, created_at, updated_at
		FROM pricing_zones
		WHERE is_active = TRUE
		ORDER BY city, name
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying zones: %w", err)
	}
	defer rows.Close()

	var zones []*PricingZone
	for rows.Next() {
		z := &PricingZone{}
		err := rows.Scan(
			&z.ID, &z.Name, &z.City, &z.Country, &z.ZoneType,
			&z.PolygonGeoJSON, &z.CenterLat, &z.CenterLng, &z.RadiusKm,
			&z.SurgeMultiplier, &z.IsActive, &z.CreatedAt, &z.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning zone: %w", err)
		}
		zones = append(zones, z)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating zones: %w", err)
	}
	return zones, nil
}

func (r *Repository) GetZoneByID(ctx context.Context, id string) (*PricingZone, error) {
	query := `
		SELECT id, name, city, country, zone_type, polygon_geojson,
		       center_lat, center_lng, radius_km, surge_multiplier,
		       is_active, created_at, updated_at
		FROM pricing_zones WHERE id=$1
	`
	z := &PricingZone{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&z.ID, &z.Name, &z.City, &z.Country, &z.ZoneType,
		&z.PolygonGeoJSON, &z.CenterLat, &z.CenterLng, &z.RadiusKm,
		&z.SurgeMultiplier, &z.IsActive, &z.CreatedAt, &z.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying zone by id: %w", err)
	}
	return z, nil
}

func (r *Repository) CreateZone(ctx context.Context, z *PricingZone) error {
	query := `
		INSERT INTO pricing_zones
		(id,name,city,country,zone_type,polygon_geojson,center_lat,center_lng,
		 radius_km,surge_multiplier,is_active,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`
	_, err := r.db.ExecContext(ctx, query,
		z.ID, z.Name, z.City, z.Country, z.ZoneType, z.PolygonGeoJSON,
		z.CenterLat, z.CenterLng, z.RadiusKm, z.SurgeMultiplier,
		z.IsActive, z.CreatedAt, z.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting zone: %w", err)
	}
	return nil
}

func (r *Repository) UpdateZone(ctx context.Context, z *PricingZone) error {
	query := `
		UPDATE pricing_zones SET
			name=$2,city=$3,country=$4,zone_type=$5,polygon_geojson=$6,
			center_lat=$7,center_lng=$8,radius_km=$9,surge_multiplier=$10,
			is_active=$11,updated_at=$12
		WHERE id=$1
	`
	_, err := r.db.ExecContext(ctx, query,
		z.ID, z.Name, z.City, z.Country, z.ZoneType, z.PolygonGeoJSON,
		z.CenterLat, z.CenterLng, z.RadiusKm, z.SurgeMultiplier,
		z.IsActive, z.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("updating zone: %w", err)
	}
	return nil
}

func (r *Repository) DeleteZone(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE pricing_zones SET is_active=FALSE, updated_at=$2 WHERE id=$1`,
		id, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("soft-deleting zone: %w", err)
	}
	return nil
}

func (r *Repository) SavePricingCalculation(ctx context.Context, calc *PricingCalculation) error {
	query := `
		INSERT INTO pricing_history (
			id, zone_id, rule_id, vehicle_type, pickup_lat, pickup_lng,
			dropoff_lat, dropoff_lng, distance_km, duration_minutes,
			base_fare, surge_multiplier, surge_amount, time_adjustment,
			weather_adjustment, event_adjustment, total_fare, currency,
			demand_level, demand_score, passengers, request_id,
			calculated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22,$23
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		calc.ID, calc.ZoneID, calc.RuleID, calc.VehicleType,
		calc.PickupLat, calc.PickupLng, calc.DropoffLat, calc.DropoffLng,
		calc.DistanceKm, calc.DurationMinutes,
		calc.BaseFare, calc.SurgeMultiplier, calc.SurgeAmount,
		calc.TimeAdjustment, calc.WeatherAdjustment, calc.EventAdjustment,
		calc.TotalFare, calc.Currency, calc.DemandLevel, calc.DemandScore,
		calc.Passengers, calc.RequestID, calc.CalculatedAt,
	)
	if err != nil {
		return fmt.Errorf("saving pricing calculation: %w", err)
	}
	return nil
}

func (r *Repository) GetPricingHistory(
	ctx context.Context,
	zoneID string,
	start, end time.Time,
	limit, offset int,
) ([]*PricingCalculation, int, error) {

	countQuery := `SELECT COUNT(*) FROM pricing_history WHERE calculated_at BETWEEN $1 AND $2`
	dataQuery := `
		SELECT id, zone_id, rule_id, vehicle_type, pickup_lat, pickup_lng,
		       dropoff_lat, dropoff_lng, distance_km, duration_minutes,
		       base_fare, surge_multiplier, surge_amount, time_adjustment,
		       weather_adjustment, event_adjustment, total_fare, currency,
		       demand_level, demand_score, passengers, request_id, calculated_at
		FROM pricing_history
		WHERE calculated_at BETWEEN $1 AND $2
	`

	args := []interface{}{start, end}
	if zoneID != "" {
		countQuery += ` AND zone_id=$3`
		dataQuery += ` AND zone_id=$3`
		args = append(args, zoneID)
	}

	nextArgIdx := len(args) + 1
	dataQuery += fmt.Sprintf(` ORDER BY calculated_at DESC LIMIT $%d OFFSET $%d`, nextArgIdx, nextArgIdx+1)

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("counting pricing history: %w", err)
	}

	paginatedArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, dataQuery, paginatedArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying pricing history: %w", err)
	}
	defer rows.Close()

	var history []*PricingCalculation
	for rows.Next() {
		c := &PricingCalculation{}
		if err := rows.Scan(
			&c.ID, &c.ZoneID, &c.RuleID, &c.VehicleType,
			&c.PickupLat, &c.PickupLng, &c.DropoffLat, &c.DropoffLng,
			&c.DistanceKm, &c.DurationMinutes,
			&c.BaseFare, &c.SurgeMultiplier, &c.SurgeAmount,
			&c.TimeAdjustment, &c.WeatherAdjustment, &c.EventAdjustment,
			&c.TotalFare, &c.Currency, &c.DemandLevel, &c.DemandScore,
			&c.Passengers, &c.RequestID, &c.CalculatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning history record: %w", err)
		}
		history = append(history, c)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating pricing history: %w", err)
	}

	return history, total, nil
}

func (r *Repository) GetPricingAnalytics(
	ctx context.Context,
	zoneID string,
	since time.Time,
) (map[string]interface{}, error) {

	baseWhere := `WHERE ph.calculated_at >= $1`
	args := []interface{}{since}
	if zoneID != "" {
		baseWhere += ` AND ph.zone_id = $2`
		args = append(args, zoneID)
	}

	summaryQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) AS total_calculations,
			AVG(ph.total_fare) AS avg_fare,
			MIN(ph.total_fare) AS min_fare,
			MAX(ph.total_fare) AS max_fare,
			AVG(ph.surge_multiplier) AS avg_surge,
			MAX(ph.surge_multiplier) AS max_surge,
			AVG(ph.distance_km) AS avg_distance,
			SUM(ph.total_fare) AS total_revenue,
			COUNT(DISTINCT ph.zone_id) AS active_zones
		FROM pricing_history ph
		%s
	`, baseWhere)

	row := r.db.QueryRowContext(ctx, summaryQuery, args...)

	var (
		totalCalcs int64
		avgFare, minFare, maxFare float64
		avgSurge, maxSurge float64
		avgDistance, totalRevenue float64
		activeZones int
	)
	if err := row.Scan(
		&totalCalcs, &avgFare, &minFare, &maxFare,
		&avgSurge, &maxSurge, &avgDistance, &totalRevenue, &activeZones,
	); err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("scanning analytics summary: %w", err)
	}

	// Demand level distribution
	distQuery := fmt.Sprintf(`
		SELECT demand_level, COUNT(*) as cnt
		FROM pricing_history ph
		%s
		GROUP BY demand_level
		ORDER BY cnt DESC
	`, baseWhere)

	distRows, err := r.db.QueryContext(ctx, distQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("querying demand distribution: %w", err)
	}
	defer distRows.Close()

	demandDist := make(map[string]int64)
	for distRows.Next() {
		var level string
		var count int64
		if err := distRows.Scan(&level, &count); err != nil {
			return nil, fmt.Errorf("scanning demand distribution: %w", err)
		}
		demandDist[level] = count
	}

	// Hourly trend for last N hours
	trendQuery := fmt.Sprintf(`
		SELECT
			date_trunc('hour', ph.calculated_at) AS hour,
			COUNT(*) AS requests,
			AVG(ph.surge_multiplier) AS avg_surge,
			AVG(ph.total_fare) AS avg_fare
		FROM pricing_history ph
		%s
		GROUP BY hour
		ORDER BY hour ASC
		LIMIT 168
	`, baseWhere)

	trendRows, err := r.db.QueryContext(ctx, trendQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("querying hourly trend: %w", err)
	}
	defer trendRows.Close()

	type HourlyBucket struct {
		Hour string  `json:"hour"`
		Requests int64 `json:"requests"`
		AvgSurge float64 `json:"avg_surge"`
		AvgFare  float64 `json:"avg_fare"`
	}
	var trend []HourlyBucket
	for trendRows.Next() {
		var b HourlyBucket
		var t time.Time
		if err := trendRows.Scan(&t, &b.Requests, &b.AvgSurge, &b.AvgFare); err != nil {
			return nil, fmt.Errorf("scanning trend bucket: %w", err)
		}
		b.Hour = t.UTC().Format(time.RFC3339)
		trend = append(trend, b)
	}

	// Top zones by request volume
	topZonesQuery := fmt.Sprintf(`
		SELECT ph.zone_id, pz.name, COUNT(*) AS requests,
		       AVG(ph.surge_multiplier) AS avg_surge,
		       SUM(ph.total_fare) AS revenue
		FROM pricing_history ph
		LEFT JOIN pricing_zones pz ON pz.id = ph.zone_id
		%s
		GROUP BY ph.zone_id, pz.name
		ORDER BY requests DESC
		LIMIT 10
	`, baseWhere)

	zoneRows, err := r.db.QueryContext(ctx, topZonesQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("querying top zones: %w", err)
	}
	defer zoneRows.Close()

	type ZoneStat struct {
		ZoneID   string  `json:"zone_id"`
		ZoneName string  `json:"zone_name"`
		Requests int64   `json:"requests"`
		AvgSurge float64 `json:"avg_surge"`
		Revenue  float64 `json:"revenue"`
	}
	var topZones []ZoneStat
	for zoneRows.Next() {
		var zs ZoneStat
		if err := zoneRows.Scan(&zs.ZoneID, &zs.ZoneName, &zs.Requests, &zs.AvgSurge, &zs.Revenue); err != nil {
			return nil, fmt.Errorf("scanning zone stat: %w", err)
		}
		topZones = append(topZones, zs)
	}

	return map[string]interface{}{
		"summary": map[string]interface{}{
			"total_calculations": totalCalcs,
			"avg_fare_eur":       roundFloat(avgFare, 2),
			"min_fare_eur":       roundFloat(minFare, 2),
			"max_fare_eur":       roundFloat(maxFare, 2),
			"avg_surge_multiplier": roundFloat(avgSurge, 4),
			"max_surge_multiplier": roundFloat(maxSurge, 4),
			"avg_distance_km":    roundFloat(avgDistance, 2),
			"total_revenue_eur":  roundFloat(totalRevenue, 2),
			"active_zones":       activeZones,
		},
		"demand_distribution": demandDist,
		"hourly_trend":         trend,
		"top_zones":            topZones,
	}, nil
}

func (r *Repository) SaveDemandSnapshot(ctx context.Context, snap *DemandSnapshot) error {
	query := `
		INSERT INTO demand_snapshots
		(id, zone_id, active_requests, available_drivers, demand_score,
		 demand_level, surge_multiplier, captured_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (zone_id, captured_at) DO UPDATE SET
			active_requests  = EXCLUDED.active_requests,
			available_drivers = EXCLUDED.available_drivers,
			demand_score     = EXCLUDED.demand_score,
			demand_level     = EXCLUDED.demand_level,
			surge_multiplier = EXCLUDED.surge_multiplier
	`
	_, err := r.db.ExecContext(ctx, query,
		snap.ID, snap.ZoneID, snap.ActiveRequests, snap.AvailableDrivers,
		snap.DemandScore, snap.DemandLevel, snap.SurgeMultiplier, snap.CapturedAt,
	)
	if err != nil {
		return fmt.Errorf("saving demand snapshot: %w", err)
	}
	return nil
}

func (r *Repository) GetLatestDemandByZone(ctx context.Context, zoneID string) (*DemandSnapshot, error) {
	query := `
		SELECT id, zone_id, active_requests, available_drivers, demand_score,
		       demand_level, surge_multiplier, captured_at
		FROM demand_snapshots
		WHERE zone_id = $1
		ORDER BY captured_at DESC
		LIMIT 1
	`
	snap := &DemandSnapshot{}
	err := r.db.QueryRowContext(ctx, query, zoneID).Scan(
		&snap.ID, &snap.ZoneID, &snap.ActiveRequests, &snap.AvailableDrivers,
		&snap.DemandScore, &snap.DemandLevel, &snap.SurgeMultiplier, &snap.CapturedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying latest demand: %w", err)
	}
	return snap, nil
}

func (r *Repository) PruneOldDemandSnapshots(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM demand_snapshots WHERE captured_at < $1`, cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("pruning demand snapshots: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (r *Repository) PruneOldPricingHistory(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM pricing_history WHERE calculated_at < $1`, cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("pruning pricing history: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// âââ Redis Cache Helpers âââââââââââââââââââââââââââââââââââââââââââââââââââââ

func (c *CacheClient) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshalling cache value: %w", err)
	}
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis SET %s: %w", key, err)
	}
	return nil
}

func (c *CacheClient) GetJSON(ctx context.Context, key string, dest interface{}) (bool, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis GET %s: %w", key, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("unmarshalling cache value for %s: %w", key, err)
	}
	return true, nil
}

func (c *CacheClient) Delete(ctx context.Context, keys ...string) error {
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("redis DEL: %w", err)
	}
	return nil
}

func (c *CacheClient) DeletePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("redis SCAN pattern %s: %w", pattern, err)
		}
		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("redis DEL keys: %w", err)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

func (c *CacheClient) IncrBy(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	pipe := c.client.TxPipeline()
	incrCmd := pipe.IncrBy(ctx, key, delta)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("redis IncrBy pipeline %s: %w", key, err)
	}
	return incrCmd.Val(), nil
}

func (c *CacheClient) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis EXISTS %s: %w", key, err)
	}
	return n > 0, nil
}

func (c *CacheClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := c.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis TTL %s: %w", key, err)
	}
	return ttl, nil
}

func (c *CacheClient) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *CacheClient) Close() error {
	return c.client.Close()
}

// Cache key constructors
func cacheKeyPricingRules() string {
	return "pricing:rules:all"
}

func cacheKeyZones() string {
	return "pricing:zones:all"
}

func cacheKeyZone(zoneID string) string {
	return fmt.Sprintf("pricing:zone:%s", zoneID)
}

func cacheKeyDemand(zoneID string) string {
	return fmt.Sprintf("pricing:demand:%s", zoneID)
}

func cacheKeySurgeMultiplier(zoneID, vehicleType string) string {
	return fmt.Sprintf("pricing:surge:%s:%s", zoneID, vehicleType)
}

func cacheKeyRateLimitCalc(ip string) string {
	return fmt.Sprintf("ratelimit:calc:%s", ip)
}

func cacheKeyRateLimitAPI(ip string) string {
	return fmt.Sprintf("ratelimit:api:%s", ip)
}

// âââ Prometheus Metrics Registration ââââââââââââââââââââââââââââââââââââââââ

func registerMetrics() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(pricingCalculationsTotal)
	prometheus.MustRegister(pricingCalculationDuration)
	prometheus.MustRegister(surgePricingActivations)
	prometheus.MustRegister(surgeMultiplierGauge)
	prometheus.MustRegister(demandScoreGauge)
	prometheus.MustRegister(cacheHitsTotal)
	prometheus.MustRegister(cacheMissesTotal)
	prometheus.MustRegister(dbQueryDuration)
	prometheus.MustRegister(activeRulesGauge)
	prometheus.MustRegister(activeZonesGauge)
	prometheus.MustRegister(rateLimitedRequestsTotal)
	prometheus.MustRegister(fareAmountHistogram)
	prometheus.MustRegister(websocketConnectionsGauge)
}

// Metric variables (defined here so they can be referenced from Part 1's init)
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "surge_http_requests_total",
			Help: "Total number of HTTP requests received.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "surge_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	pricingCalculationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "surge_pricing_calculations_total",
			Help: "Total number of pricing calculations performed.",
		},
		[]string{"zone_id", "vehicle_type", "demand_level"},
	)

	pricingCalculationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "surge_pricing_calculation_duration_seconds",
			Help:    "Time spent computing a price.",
			Buckets: []float64{0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25},
		},
		[]string{"zone_id", "vehicle_type"},
	)

	surgePricingActivations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "surge_activations_total",
			Help: "Total number of times surge pricing was activated (multiplier > 1).",
		},
		[]string{"zone_id", "vehicle_type"},
	)

	surgeMultiplierGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "surge_current_multiplier",
			Help: "Current surge multiplier per zone and vehicle type.",
		},
		[]string{"zone_id", "vehicle_type"},
	)

	demandScoreGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "surge_demand_score",
			Help: "Current demand score per zone (ratio of requests to drivers).",
		},
		[]string{"zone_id", "demand_level"},
	)

	cacheHitsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "surge_cache_hits_total",
			Help: "Total number of cache hits.",
		},
		[]string{"cache_key_type"},
	)

	cacheMissesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "surge_cache_misses_total",
			Help: "Total number of cache misses.",
		},
		[]string{"cache_key_type"},
	)

	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "surge_db_query_duration_seconds",
			Help:    "Database query latency in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		},
		[]string{"operation"},
	)

	activeRulesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "surge_active_pricing_rules",
			Help: "Number of currently active pricing rules.",
		},
	)

	activeZonesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "surge_active_pricing_zones",
			Help: "Number of currently active pricing zones.",
		},
	)

	rateLimitedRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "surge_rate_limited_requests_total",
			Help: "Total number of rate-limited requests.",
		},
		[]string{"endpoint"},
	)

	fareAmountHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "surge_fare_amount_eur",
			Help:    "Distribution of calculated fare amounts in EUR.",
			Buckets: []float64{1, 2, 5, 10, 15, 20, 30, 50, 75, 100, 150, 200},
		},
		[]string{"vehicle_type", "demand_level"},
	)

	websocketConnectionsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "surge_websocket_connections_active",
			Help: "Number of currently active WebSocket connections.",
		},
	)
)

// âââ Utility / Helper Functions ââââââââââââââââââââââââââââââââââââââââââââââ

func writeJSONResponse(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// At this point headers are already sent; best effort log only.
		_ = err
	}
}

func writeErrorResponse(
	w http.ResponseWriter,
	status int,
	code, message string,
	details interface{},
) {
	body := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"details": details,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	writeJSONResponse(w, status, body)
}

func roundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func parseDuration(s string) (time.Duration, error) {
	// Support shorthand like "7d" = 7 * 24h
	if len(s) > 1 && s[len(s)-1] == 'd' {
		days, err := strconv.Atoi(s[:len(s)-1])
		if err != nil {
			return 0, fmt.Errorf("invalid day value in duration %q", s)
		}
		if days < 1 || days > 365 {
			return 0, fmt.Errorf("day value %d out of range [1,365]", days)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", s, err)
	}
	if d <= 0 || d > 365*24*time.Hour {
		return 0, fmt.Errorf("duration %q out of acceptable range", s)
	}
	return d, nil
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180.0
}

func isValidLatitude(lat float64) bool {
	return lat >= -90.0 && lat <= 90.0
}

func isValidLongitude(lng float64) bool {
	return lng >= -180.0 && lng <= 180.0
}

func isValidCoordinates(lat, lng float64) bool {
	return isValidLatitude(lat) && isValidLongitude(lng)
}

// isPointInGermanBounds performs a rough bounding-box check for Germany.
func isPointInGermanBounds(lat, lng float64) bool {
	// Approximate bounding box: lat [47.27, 55.06], lng [5.87, 15.04]
	return lat >= 47.27 && lat <= 55.06 && lng >= 5.87 && lng <= 15.04
}

func generateID() string {
	return uuid.New().String()
}

func clampFloat64(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultValue
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func sanitizeZoneID(id string) string {
	// Allow only alphanumeric, hyphens, underscores (UUID-safe)
	var out []rune
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			out = append(out, r)
		}
	}
	return string(out)
}

func ptrFloat64(f float64) *float64 { return &f }
func ptrString(s string) *string   { return &s }
func ptrBool(b bool) *bool         { return &b }

func formatEuro(amount float64) string {
	return fmt.Sprintf("â¬%.2f", amount)
}

// weightedAverage computes a weighted mean of values[i] * weights[i].
func weightedAverage(values, weights []float64) float64 {
	if len(values) != len(weights) || len(values) == 0 {
		return 0
	}
	var sumWV, sumW float64
	for i := range values {
		sumWV += values[i] * weights[i]
		sumW += weights[i]
	}
	if sumW == 0 {
		return 0
	}
	return sumWV / sumW
}

// exponentialMovingAverage updates EMA with the latest sample.
func exponentialMovingAverage(prevEMA, newSample, alpha float64) float64 {
	return alpha*newSample + (1-alpha)*prevEMA
}

// linearInterpolate returns the interpolated value between a and b at position t â [0,1].
func linearInterpolate(a, b, t float64) float64 {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	return a + (b-a)*t
}

// demandScoreToLevel converts a raw demand score into a named DemandLevel.
func demandScoreToLevel(score float64, rule *PricingRule) DemandLevel {
	switch {
	case score >= rule.DemandThresholdVeryHigh:
		return DemandVeryHigh
	case score >= rule.DemandThresholdHigh:
		return DemandHigh
	case score >= rule.DemandThresholdMedium:
		return DemandMedium
	default:
		return DemandLow
	}
}

// surgeMultiplierForLevel maps a DemandLevel to the configured surge multiplier.
func surgeMultiplierForLevel(level DemandLevel, rule *PricingRule) float64 {
	switch level {
	case DemandVeryHigh:
		return rule.SurgeMultiplierVeryHigh
	case DemandHigh:
		return rule.SurgeMultiplierHigh
	case DemandMedium:
		return rule.SurgeMultiplierMedium
	default:
		return rule.SurgeMultiplierLow
	}
}

// buildDemandSnapshot constructs a ready-to-persist snapshot.
func buildDemandSnapshot(zoneID string, activeReqs, availDrivers int, rule *PricingRule) *DemandSnapshot {
	var score float64
	if availDrivers > 0 {
		score = float64(activeReqs) / float64(availDrivers)
	} else {
		score = float64(activeReqs) * 2.0 // treat zero-driver as very high demand
	}
	level := demandScoreToLevel(score, rule)
	multiplier := surgeMultiplierForLevel(level, rule)
	if multiplier > rule.MaxSurgeMultiplier {
		multiplier = rule.MaxSurgeMultiplier
	}
	return &DemandSnapshot{
		ID:               generateID(),
		ZoneID:           zoneID,
		ActiveRequests:   activeReqs,
		AvailableDrivers: availDrivers,
		DemandScore:      roundFloat(score, 4),
		DemandLevel:      string(level),
		SurgeMultiplier:  roundFloat(multiplier, 4),
		CapturedAt:       time.Now().UTC(),
	}
}

// clientIP extracts the real client IP from a request, honouring X-Forwarded-For.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// requestID returns the X-Request-ID header or generates a new UUID.
func requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return truncateString(id, 64)
	}
	return generateID()
}

// safeDiv divides a by b, returning 0 if b == 0.
func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// jsonMustMarshal marshals v to JSON, panicking only in tests; returns empty braces on error.
func jsonMustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}

// isValidVehicleType checks membership against known types.
func isValidVehicleType(vt string) bool {
	switch VehicleType(vt) {
	case VehicleTypeEconomy, VehicleTypeComfort, VehicleTypeXL,
		VehicleTypeElectric, VehicleTypePremium, VehicleTypePool:
		return true
	}
	return false
}

// isValidZoneType checks membership against known zone types.
func isValidZoneType(zt string) bool {
	switch ZoneType(zt) {
	case ZoneTypeCity, ZoneTypeSuburban, ZoneTypeAirport,
		ZoneTypeStation, ZoneTypeEventVenue, ZoneTypeCustom:
		return true
	}
	return false
}

// contextWithTimeout wraps a context with a database timeout from config.
func contextWithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}

// buildVersionInfo returns a map describing the service version for the /health endpoint.
func buildVersionInfo() map[string]string {
	return map[string]string{
		"service": "surge-pricing-service",
		"version": "1.0.0",
		"go":      runtime.Version(),
		"arch":    runtime.GOARCH,
		"os":      runtime.GOOS,
	}
}

// âââ End of File âââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ

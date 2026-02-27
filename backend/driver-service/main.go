package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// =============================================================================
// GERMAN COMPLIANCE NOTICE
// =============================================================================
// Dieses Modul verarbeitet personenbezogene Daten gemaess DSGVO (EU 2016/679)
// sowie den Anforderungen des Personenbefoerderungsgesetzes (PBefG).
// Datenschutzbeauftragter muss bei der Verarbeitung konsultiert werden.
// Datenspeicherung erfolgt nur fuer den gesetzlich vorgeschriebenen Zeitraum.
// Fahrerdaten unterliegen dem Beschaeftigtendatenschutz (BDSG §26).
// Standortdaten werden nur fuer betriebliche Zwecke verarbeitet (DSGVO Art. 6 Abs. 1 lit. b).
// Alle Datenexporte muessen protokolliert werden (DSGVO Art. 30).
// =============================================================================

// =============================================================================
// CONSTANTS AND ENUMERATIONS
// =============================================================================

const (
	// ServiceName identifies this microservice
	ServiceName = "driver-service"

	// ServiceVersion follows semantic versioning
	ServiceVersion = "1.0.0"

	// DefaultPort is the default HTTP listening port
	DefaultPort = "8080"

	// DefaultReadTimeout is the HTTP server read timeout
	DefaultReadTimeout = 15 * time.Second

	// DefaultWriteTimeout is the HTTP server write timeout
	DefaultWriteTimeout = 15 * time.Second

	// DefaultIdleTimeout is the HTTP server idle timeout
	DefaultIdleTimeout = 60 * time.Second

	// DefaultShutdownTimeout is the graceful shutdown timeout
	DefaultShutdownTimeout = 30 * time.Second

	// MaxLocationHistoryEntries limits stored GPS points per driver (DSGVO Datensparsamkeit)
	MaxLocationHistoryEntries = 500

	// MaxTripHistoryEntries limits stored trips per driver
	MaxTripHistoryEntries = 1000

	// LocationRetentionDays defines how long location data is retained (DSGVO Art. 5 Abs. 1 lit. e)
	LocationRetentionDays = 30

	// EarningsRetentionDays defines how long earnings data is retained for tax purposes (AO §147)
	EarningsRetentionDays = 3650 // 10 Jahre gemaess Abgabenordnung

	// RateLimitRequestsPerMinute defines maximum requests per minute per client
	RateLimitRequestsPerMinute = 120

	// MinLatitude and MaxLatitude define valid GPS latitude bounds
	MinLatitude = -90.0
	MaxLatitude = 90.0

	// MinLongitude and MaxLongitude define valid GPS longitude bounds
	MinLongitude = -180.0
	MaxLongitude = 180.0

	// MaxDriverNameLength limits name field length
	MaxDriverNameLength = 100

	// MaxPhoneLength limits phone number length
	MaxPhoneLength = 20

	// MaxLicensePlateLength limits license plate length (PBefG)
	MaxLicensePlateLength = 15

	// EarthRadiusKm is used for Haversine distance calculation
	EarthRadiusKm = 6371.0
)

// DriverStatus represents the operational status of a driver
// Gemaess PBefG §47 muss der Verfuegbarkeitsstatus protokolliert werden
type DriverStatus string

const (
	// DriverStatusOffline means the driver is not available
	DriverStatusOffline DriverStatus = "offline"

	// DriverStatusOnline means the driver is available for trips
	DriverStatusOnline DriverStatus = "online"

	// DriverStatusBusy means the driver is currently on a trip
	DriverStatusBusy DriverStatus = "busy"

	// DriverStatusOnBreak means the driver is on a legally required break (ArbZG)
	DriverStatusOnBreak DriverStatus = "on_break"

	// DriverStatusSuspended means the driver account is suspended
	DriverStatusSuspended DriverStatus = "suspended"
)

// VehicleType represents the category of the vehicle
// Klassifizierung gemaess PBefG und EU-Verordnung 2018/858
type VehicleType string

const (
	VehicleTypeSedan      VehicleType = "sedan"
	VehicleTypeCombi      VehicleType = "combi"
	VehicleTypeSUV        VehicleType = "suv"
	VehicleTypeVan        VehicleType = "van"
	VehicleTypeMinibus    VehicleType = "minibus"
	VehicleTypeElectric   VehicleType = "electric"
	VehicleTypeHybrid     VehicleType = "hybrid"
	VehicleTypeAccessible VehicleType = "accessible" // Barrierefreiheit gemaess BGG
)

// LicenseClass represents the German driving license class
// Gemaess Fahrerlaubnis-Verordnung (FeV)
type LicenseClass string

const (
	LicenseClassB   LicenseClass = "B"
	LicenseClassBE  LicenseClass = "BE"
	LicenseClassC1  LicenseClass = "C1"
	LicenseClassC1E LicenseClass = "C1E"
	LicenseClassD1  LicenseClass = "D1" // Personenbefoerderung bis 16 Personen
	LicenseClassD   LicenseClass = "D"  // Personenbefoerderung unbegrenzt
)

// TripStatus represents the state of a completed or ongoing trip
type TripStatus string

const (
	TripStatusCompleted  TripStatus = "completed"
	TripStatusCancelled  TripStatus = "cancelled"
	TripStatusInProgress TripStatus = "in_progress"
	TripStatusDisputed   TripStatus = "disputed"
)

// PaymentMethod represents how a trip was paid
type PaymentMethod string

const (
	PaymentMethodCash   PaymentMethod = "cash"
	PaymentMethodCard   PaymentMethod = "card"
	PaymentMethodApp    PaymentMethod = "app"
	PaymentMethodInvoice PaymentMethod = "invoice"
)

// =============================================================================
// DATA MODELS
// =============================================================================

// Address represents a physical address (DSGVO-relevant)
type Address struct {
	Street     string `json:"street"`
	HouseNumber string `json:"house_number"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	State      string `json:"state"` // Bundesland
}

// Vehicle represents a driver's registered vehicle
// Fahrzeugdaten gemaess PBefG §47 Abs. 1
type Vehicle struct {
	ID               string      `json:"id"`
	LicensePlate     string      `json:"license_plate"`      // Kennzeichen
	Make             string      `json:"make"`               // Hersteller
	Model            string      `json:"model"`              // Modell
	Year             int         `json:"year"`               // Baujahr
	Color            string      `json:"color"`              // Farbe
	Type             VehicleType `json:"type"`
	Seats            int         `json:"seats"`              // Sitzplaetze
	HasChildSeat     bool        `json:"has_child_seat"`     // Kindersitz vorhanden
	IsAccessible     bool        `json:"is_accessible"`      // Rollstuhlgerecht
	InsuranceExpiry  time.Time   `json:"insurance_expiry"`   // Versicherungsablauf
	InspectionExpiry time.Time   `json:"inspection_expiry"`  // Hauptuntersuchung (HU)
	TUVExpiry        time.Time   `json:"tuv_expiry"`         // TUeV-Termin
}

// DriverDocument represents compliance documents
// Pflichtdokumente gemaess PBefG und FeV
type DriverDocument struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`         // Dokumenttyp
	Number      string    `json:"number"`       // Dokumentnummer
	IssuedBy    string    `json:"issued_by"`    // Ausstellende Behoerde
	IssuedAt    time.Time `json:"issued_at"`    // Ausstellungsdatum
	ExpiresAt   time.Time `json:"expires_at"`   // Ablaufdatum
	Verified    bool      `json:"verified"`     // Verifiziert
	VerifiedAt  *time.Time `json:"verified_at"` // Verifizierungsdatum
}

// Driver represents a registered driver in the system
// Fahrerdaten gemaess DSGVO Art. 4 Nr. 1 (personenbezogene Daten)
// Speicherung gemaess DSGVO Art. 6 Abs. 1 lit. b (Vertragserfuellung)
type Driver struct {
	ID                string          `json:"id"`
	FirstName         string          `json:"first_name"`
	LastName          string          `json:"last_name"`
	Email             string          `json:"email"`             // DSGVO: verschluesselt speichern
	Phone             string          `json:"phone"`             // DSGVO: verschluesselt speichern
	DateOfBirth       time.Time       `json:"date_of_birth"`     // DSGVO Art. 9 sensible Daten
	Address           Address         `json:"address"`
	LicenseClass      LicenseClass    `json:"license_class"`
	LicenseNumber     string          `json:"license_number"`    // Fuehrerscheinnummer
	LicenseExpiry     time.Time       `json:"license_expiry"`
	PBefGLicense      string          `json:"pbefg_license"`     // P-Schein gemaess PBefG §47
	PBefGExpiry       time.Time       `json:"pbefg_expiry"`      // P-Schein Ablaufdatum
	Status            DriverStatus    `json:"status"`
	Vehicle           *Vehicle        `json:"vehicle,omitempty"`
	Documents         []DriverDocument `json:"documents,omitempty"`
	Rating            float64         `json:"rating"`            // Bewertung (0-5)
	TotalTrips        int             `json:"total_trips"`
	TotalEarnings     float64         `json:"total_earnings"`    // Gesamtverdienst in EUR
	IsActive          bool            `json:"is_active"`
	IsVerified        bool            `json:"is_verified"`       // KYC verifiziert
	ConsentGiven      bool            `json:"consent_given"`     // DSGVO Einwilligung Art. 7
	ConsentTimestamp  *time.Time      `json:"consent_timestamp"` // Zeitstempel der Einwilligung
	DataRetentionDate *time.Time      `json:"data_retention_date,omitempty"` // DSGVO Art. 5 Abs. 1 lit. e
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	LastActiveAt      *time.Time      `json:"last_active_at,omitempty"`
	DeletedAt         *time.Time      `json:"deleted_at,omitempty"` // Soft-Delete fuer DSGVO Loeschpflicht
}

// LocationUpdate represents a real-time GPS position update
// Standortdaten sind besonders schutzwuerdig gemaess DSGVO ErwGr. 75
// Verarbeitung gemaess DSGVO Art. 6 Abs. 1 lit. b (Vertragserfuellung)
type LocationUpdate struct {
	DriverID    string    `json:"driver_id"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Accuracy    float64   `json:"accuracy"`    // GPS-Genauigkeit in Metern
	Speed       float64   `json:"speed"`       // Geschwindigkeit in km/h
	Heading     float64   `json:"heading"`     // Fahrtrichtung in Grad
	Altitude    float64   `json:"altitude"`    // Hoehe ueber NN in Metern
	Timestamp   time.Time `json:"timestamp"`
	IsAnonymized bool     `json:"is_anonymized"` // DSGVO Anonymisierung
}

// CurrentLocation represents the latest known driver position
type CurrentLocation struct {
	DriverID   string    `json:"driver_id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Accuracy   float64   `json:"accuracy"`
	Speed      float64   `json:"speed"`
	Heading    float64   `json:"heading"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// EarningsEntry represents a single earning transaction
// Verdienstdaten gemaess Steuerrecht aufzubewahren (AO §147, 10 Jahre)
type EarningsEntry struct {
	ID            string        `json:"id"`
	DriverID      string        `json:"driver_id"`
	TripID        string        `json:"trip_id"`
	GrossAmount   float64       `json:"gross_amount"`    // Bruttobetrag in EUR
	NetAmount     float64       `json:"net_amount"`      // Nettobetrag nach Provisionsabzug
	Commission    float64       `json:"commission"`      // Plattformprovision
	TaxAmount     float64       `json:"tax_amount"`      // Umsatzsteuer (USt)
	TipAmount     float64       `json:"tip_amount"`      // Trinkgeld
	PaymentMethod PaymentMethod `json:"payment_method"`
	Currency      string        `json:"currency"`        // ISO 4217 (EUR)
	EarnedAt      time.Time     `json:"earned_at"`
	PaidAt        *time.Time    `json:"paid_at,omitempty"`
	InvoiceNumber string        `json:"invoice_number"`  // Rechnungsnummer gemaess UStG
}

// EarningsSummary aggregates earnings over a period
type EarningsSummary struct {
	DriverID      string    `json:"driver_id"`
	Period        string    `json:"period"`         // daily, weekly, monthly
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	TotalGross    float64   `json:"total_gross"`
	TotalNet      float64   `json:"total_net"`
	TotalTips     float64   `json:"total_tips"`
	TotalTax      float64   `json:"total_tax"`
	TotalTrips    int       `json:"total_trips"`
	AveragePerTrip float64  `json:"average_per_trip"`
	Currency      string    `json:"currency"`
}

// TripHistory represents a completed trip in the driver's history
// Fahrtdaten gemaess PBefG §47 aufzubewahren
type TripHistory struct {
	ID              string        `json:"id"`
	DriverID        string        `json:"driver_id"`
	PassengerID     string        `json:"passenger_id"`    // DSGVO: pseudonymisiert
	PickupAddress   string        `json:"pickup_address"`
	DropoffAddress  string        `json:"dropoff_address"`
	PickupLat       float64       `json:"pickup_lat"`
	PickupLng       float64       `json:"pickup_lng"`
	DropoffLat      float64       `json:"dropoff_lat"`
	DropoffLng      float64       `json:"dropoff_lng"`
	DistanceKm      float64       `json:"distance_km"`
	DurationMinutes int           `json:"duration_minutes"`
	Status          TripStatus    `json:"status"`
	PaymentMethod   PaymentMethod `json:"payment_method"`
	FareAmount      float64       `json:"fare_amount"`
	TipAmount       float64       `json:"tip_amount"`
	PassengerRating float64       `json:"passenger_rating"` // Bewertung des Fahrers durch Fahrgast
	DriverRating    float64       `json:"driver_rating"`   // Bewertung des Fahrgasts durch Fahrer
	StartedAt       time.Time     `json:"started_at"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`
	CancelledAt     *time.Time    `json:"cancelled_at,omitempty"`
	CancelReason    string        `json:"cancel_reason,omitempty"`
	Route           []LocationUpdate `json:"route,omitempty"` // GPS-Spur der Fahrt
	CreatedAt       time.Time     `json:"created_at"`
}

// AvailabilityLog tracks driver status changes for PBefG compliance
// Gemaess PBefG §47 sind Verfuegbarkeitszeiten zu protokollieren
type AvailabilityLog struct {
	ID        string       `json:"id"`
	DriverID  string       `json:"driver_id"`
	Status    DriverStatus `json:"status"`
	ChangedAt time.Time    `json:"changed_at"`
	Latitude  float64      `json:"latitude,omitempty"`
	Longitude float64      `json:"longitude,omitempty"`
	Note      string       `json:"note,omitempty"`
}

// =============================================================================
// REQUEST AND RESPONSE TYPES
// =============================================================================

// CreateDriverRequest is the payload for creating a new driver
type CreateDriverRequest struct {
	FirstName     string       `json:"first_name"`
	LastName      string       `json:"last_name"`
	Email         string       `json:"email"`
	Phone         string       `json:"phone"`
	DateOfBirth   time.Time    `json:"date_of_birth"`
	Address       Address      `json:"address"`
	LicenseClass  LicenseClass `json:"license_class"`
	LicenseNumber string       `json:"license_number"`
	LicenseExpiry time.Time    `json:"license_expiry"`
	PBefGLicense  string       `json:"pbefg_license"`
	PBefGExpiry   time.Time    `json:"pbefg_expiry"`
	ConsentGiven  bool         `json:"consent_given"`
}

// UpdateDriverRequest is the payload for updating driver information
type UpdateDriverRequest struct {
	FirstName  *string  `json:"first_name,omitempty"`
	LastName   *string  `json:"last_name,omitempty"`
	Phone      *string  `json:"phone,omitempty"`
	Address    *Address `json:"address,omitempty"`
}

// UpdateStatusRequest is the payload for changing driver availability
type UpdateStatusRequest struct {
	Status    DriverStatus `json:"status"`
	Latitude  float64      `json:"latitude"`
	Longitude float64      `json:"longitude"`
	Note      string       `json:"note,omitempty"`
}

// LocationUpdateRequest is the payload for GPS position updates
type LocationUpdateRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  float64 `json:"accuracy"`
	Speed     float64 `json:"speed"`
	Heading   float64 `json:"heading"`
	Altitude  float64 `json:"altitude"`
}

// AddVehicleRequest is the payload for registering a vehicle
type AddVehicleRequest struct {
	LicensePlate     string      `json:"license_plate"`
	Make             string      `json:"make"`
	Model            string      `json:"model"`
	Year             int         `json:"year"`
	Color            string      `json:"color"`
	Type             VehicleType `json:"type"`
	Seats            int         `json:"seats"`
	HasChildSeat     bool        `json:"has_child_seat"`
	IsAccessible     bool        `json:"is_accessible"`
	InsuranceExpiry  time.Time   `json:"insurance_expiry"`
	InspectionExpiry time.Time   `json:"inspection_expiry"`
	TUVExpiry        time.Time   `json:"tuv_expiry"`
}

// AddTripRequest is the payload for recording a trip
type AddTripRequest struct {
	PassengerID     string        `json:"passenger_id"`
	PickupAddress   string        `json:"pickup_address"`
	DropoffAddress  string        `json:"dropoff_address"`
	PickupLat       float64       `json:"pickup_lat"`
	PickupLng       float64       `json:"pickup_lng"`
	DropoffLat      float64       `json:"dropoff_lat"`
	DropoffLng      float64       `json:"dropoff_lng"`
	DistanceKm      float64       `json:"distance_km"`
	DurationMinutes int           `json:"duration_minutes"`
	Status          TripStatus    `json:"status"`
	PaymentMethod   PaymentMethod `json:"payment_method"`
	FareAmount      float64       `json:"fare_amount"`
	TipAmount       float64       `json:"tip_amount"`
	PassengerRating float64       `json:"passenger_rating"`
	StartedAt       time.Time     `json:"started_at"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`
}

// AddEarningsRequest is the payload for recording an earnings entry
type AddEarningsRequest struct {
	TripID        string        `json:"trip_id"`
	GrossAmount   float64       `json:"gross_amount"`
	Commission    float64       `json:"commission"`
	TaxAmount     float64       `json:"tax_amount"`
	TipAmount     float64       `json:"tip_amount"`
	PaymentMethod PaymentMethod `json:"payment_method"`
}

// APIResponse is the standard response envelope
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id"`
}

// APIError represents a structured error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta contains pagination and response metadata
type Meta struct {
	Total  int `json:"total"`
	Page   int `json:"page"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// HealthResponse is the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
	Uptime    string            `json:"uptime"`
}

// =============================================================================
// IN-MEMORY STORE
// =============================================================================

// DriverStore is the thread-safe in-memory data store
// In production: replace with PostgreSQL + Redis for persistence and caching
type DriverStore struct {
	mu               sync.RWMutex
	drivers          map[string]*Driver
	locations        map[string]*CurrentLocation
	locationHistory  map[string][]LocationUpdate
	earnings         map[string][]EarningsEntry
	tripHistory      map[string][]TripHistory
	availabilityLogs map[string][]AvailabilityLog
}

// NewDriverStore creates and initializes a new DriverStore
func NewDriverStore() *DriverStore {
	return &DriverStore{
		drivers:          make(map[string]*Driver),
		locations:        make(map[string]*CurrentLocation),
		locationHistory:  make(map[string][]LocationUpdate),
		earnings:         make(map[string][]EarningsEntry),
		tripHistory:      make(map[string][]TripHistory),
		availabilityLogs: make(map[string][]AvailabilityLog),
	}
}

// CreateDriver stores a new driver record
func (s *DriverStore) CreateDriver(d *Driver) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.drivers {
		if existing.Email == d.Email && existing.DeletedAt == nil {
			return fmt.Errorf("driver with email %s already exists", d.Email)
		}
	}
	s.drivers[d.ID] = d
	s.locationHistory[d.ID] = []LocationUpdate{}
	s.earnings[d.ID] = []EarningsEntry{}
	s.tripHistory[d.ID] = []TripHistory{}
	s.availabilityLogs[d.ID] = []AvailabilityLog{}
	return nil
}

// GetDriver retrieves a driver by ID
func (s *DriverStore) GetDriver(id string) (*Driver, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.drivers[id]
	if !ok || d.DeletedAt != nil {
		return nil, fmt.Errorf("driver not found: %s", id)
	}
	copy := *d
	return &copy, nil
}

// UpdateDriver updates an existing driver record
func (s *DriverStore) UpdateDriver(d *Driver) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.drivers[d.ID]
	if !ok || existing.DeletedAt != nil {
		return fmt.Errorf("driver not found: %s", d.ID)
	}
	s.drivers[d.ID] = d
	return nil
}

// DeleteDriver performs a soft delete (DSGVO Recht auf Loeschung Art. 17)
// Daten werden pseudonymisiert und nach Aufbewahrungsfrist endgueltig geloescht
func (s *DriverStore) DeleteDriver(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.drivers[id]
	if !ok || d.DeletedAt != nil {
		return fmt.Errorf("driver not found: %s", id)
	}
	now := time.Now().UTC()
	d.DeletedAt = &now
	d.IsActive = false
	// DSGVO: Personenbezogene Daten pseudonymisieren
	d.FirstName = "[GELOESCHT]"
	d.LastName = "[GELOESCHT]"
	d.Email = fmt.Sprintf("deleted_%s@pseudonymized.invalid", id)
	d.Phone = "[GELOESCHT]"
	d.LicenseNumber = "[PSEUDONYMISIERT]"
	d.PBefGLicense = "[PSEUDONYMISIERT]"
	s.drivers[id] = d
	// Standortdaten sofort loeschen (DSGVO Art. 17 Abs. 1)
	delete(s.locations, id)
	s.locationHistory[id] = []LocationUpdate{}
	return nil
}

// ListDrivers returns all active (non-deleted) drivers
func (s *DriverStore) ListDrivers(limit, offset int) ([]*Driver, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var active []*Driver
	for _, d := range s.drivers {
		if d.DeletedAt == nil {
			copy := *d
			active = append(active, &copy)
		}
	}
	total := len(active)
	if offset >= total {
		return []*Driver{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return active[offset:end], total
}

// UpdateLocation stores the latest driver position
// Standortdaten: Verarbeitung nur fuer betriebliche Zwecke (DSGVO Art. 6 Abs. 1 lit. b)
func (s *DriverStore) UpdateLocation(update LocationUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.drivers[update.DriverID]
	if !ok {
		return fmt.Errorf("driver not found: %s", update.DriverID)
	}
	s.locations[update.DriverID] = &CurrentLocation{
		DriverID:  update.DriverID,
		Latitude:  update.Latitude,
		Longitude: update.Longitude,
		Accuracy:  update.Accuracy,
		Speed:     update.Speed,
		Heading:   update.Heading,
		UpdatedAt: update.Timestamp,
	}
	history := s.locationHistory[update.DriverID]
	history = append(history, update)
	// Datensparsamkeit: Nur die letzten N Eintraege behalten (DSGVO Art. 5 Abs. 1 lit. c)
	if len(history) > MaxLocationHistoryEntries {
		history = history[len(history)-MaxLocationHistoryEntries:]
	}
	s.locationHistory[update.DriverID] = history
	return nil
}

// GetLocation retrieves the current location for a driver
func (s *DriverStore) GetLocation(driverID string) (*CurrentLocation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	loc, ok := s.locations[driverID]
	if !ok {
		return nil, fmt.Errorf("location not found for driver: %s", driverID)
	}
	copy := *loc
	return &copy, nil
}

// GetNearbyDrivers returns online drivers within a radius (km)
func (s *DriverStore) GetNearbyDrivers(lat, lng, radiusKm float64) []*CurrentLocation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var nearby []*CurrentLocation
	for driverID, loc := range s.locations {
		d, ok := s.drivers[driverID]
		if !ok || d.DeletedAt != nil || d.Status != DriverStatusOnline {
			continue
		}
		dist := haversineDistance(lat, lng, loc.Latitude, loc.Longitude)
		if dist <= radiusKm {
			copy := *loc
			nearby = append(nearby, &copy)
		}
	}
	return nearby
}

// AddEarnings records an earnings entry for a driver
func (s *DriverStore) AddEarnings(entry EarningsEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.drivers[entry.DriverID]
	if !ok {
		return fmt.Errorf("driver not found: %s", entry.DriverID)
	}
	s.earnings[entry.DriverID] = append(s.earnings[entry.DriverID], entry)
	d := s.drivers[entry.DriverID]
	d.TotalEarnings += entry.NetAmount
	s.drivers[entry.DriverID] = d
	return nil
}

// GetEarnings retrieves earnings entries for a driver within a time range
func (s *DriverStore) GetEarnings(driverID string, from, to time.Time) ([]EarningsEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.drivers[driverID]
	if !ok {
		return nil, fmt.Errorf("driver not found: %s", driverID)
	}
	var filtered []EarningsEntry
	for _, e := range s.earnings[driverID] {
		if (e.EarnedAt.Equal(from) || e.EarnedAt.After(from)) &&
			(e.EarnedAt.Equal(to) || e.EarnedAt.Before(to)) {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
}

// AddTripHistory records a trip in the driver's history
func (s *DriverStore) AddTripHistory(trip TripHistory) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.drivers[trip.DriverID]
	if !ok {
		return fmt.Errorf("driver not found: %s", trip.DriverID)
	}
	trips := s.tripHistory[trip.DriverID]
	trips = append(trips, trip)
	if len(trips) > MaxTripHistoryEntries {
		trips = trips[len(trips)-MaxTripHistoryEntries:]
	}
	s.tripHistory[trip.DriverID] = trips
	d := s.drivers[trip.DriverID]
	d.TotalTrips++
	if trip.PassengerRating > 0 {
		if d.TotalTrips > 1 {
			d.Rating = (d.Rating*float64(d.TotalTrips-1) + trip.PassengerRating) / float64(d.TotalTrips)
		} else {
			d.Rating = trip.PassengerRating
		}
	}
	s.drivers[trip.DriverID] = d
	return nil
}

// GetTripHistory retrieves trips for a driver with pagination
func (s *DriverStore) GetTripHistory(driverID string, limit, offset int) ([]TripHistory, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.drivers[driverID]
	if !ok {
		return nil, 0, fmt.Errorf("driver not found: %s", driverID)
	}
	trips := s.tripHistory[driverID]
	total := len(trips)
	if offset >= total {
		return []TripHistory{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return trips[offset:end], total, nil
}

// UpdateDriverStatus changes the driver's availability status
func (s *DriverStore) UpdateDriverStatus(driverID string, status DriverStatus, lat, lng float64, note string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.drivers[driverID]
	if !ok || d.DeletedAt != nil {
		return fmt.Errorf("driver not found: %s", driverID)
	}
	now := time.Now().UTC()
	d.Status = status
	d.UpdatedAt = now
	d.LastActiveAt = &now
	s.drivers[driverID] = d
	// Protokollierung gemaess PBefG
	logEntry := AvailabilityLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Status:    status,
		ChangedAt: now,
		Latitude:  lat,
		Longitude: lng,
		Note:      note,
	}
	s.availabilityLogs[driverID] = append(s.availabilityLogs[driverID], logEntry)
	return nil
}

// AddVehicle assigns a vehicle to a driver
func (s *DriverStore) AddVehicle(driverID string, v *Vehicle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.drivers[driverID]
	if !ok || d.DeletedAt != nil {
		return fmt.Errorf("driver not found: %s", driverID)
	}
	d.Vehicle = v
	d.UpdatedAt = time.Now().UTC()
	s.drivers[driverID] = d
	return nil
}

// DriverExists checks if a driver exists and is active
func (s *DriverStore) DriverExists(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.drivers[id]
	return ok && d.DeletedAt == nil
}

// =============================================================================
// SERVICE LAYER
// =============================================================================

// DriverService contains the business logic
type DriverService struct {
	store  *DriverStore
	logger *Logger
}

// NewDriverService creates a new DriverService
func NewDriverService(store *DriverStore, logger *Logger) *DriverService {
	return &DriverService{store: store, logger: logger}
}

// CreateDriver validates and creates a new driver
func (svc *DriverService) CreateDriver(req CreateDriverRequest) (*Driver, error) {
	if err := validateCreateDriverRequest(req); err != nil {
		return nil, err
	}
	// DSGVO Art. 7: Einwilligung muss explizit gegeben werden
	if !req.ConsentGiven {
		return nil, fmt.Errorf("DSGVO-Einwilligung ist erforderlich (Art. 7 DSGVO)")
	}
	now := time.Now().UTC()
	consentTime := now
	retentionDate := now.AddDate(0, 0, EarningsRetentionDays)
	d := &Driver{
		ID:                uuid.New().String(),
		FirstName:         strings.TrimSpace(req.FirstName),
		LastName:          strings.TrimSpace(req.LastName),
		Email:             strings.ToLower(strings.TrimSpace(req.Email)),
		Phone:             strings.TrimSpace(req.Phone),
		DateOfBirth:       req.DateOfBirth,
		Address:           req.Address,
		LicenseClass:      req.LicenseClass,
		LicenseNumber:     req.LicenseNumber,
		LicenseExpiry:     req.LicenseExpiry,
		PBefGLicense:      req.PBefGLicense,
		PBefGExpiry:       req.PBefGExpiry,
		Status:            DriverStatusOffline,
		Rating:            0,
		TotalTrips:        0,
		TotalEarnings:     0,
		IsActive:          true,
		IsVerified:        false,
		ConsentGiven:      req.ConsentGiven,
		ConsentTimestamp:  &consentTime,
		DataRetentionDate: &retentionDate,
		Documents:         []DriverDocument{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := svc.store.CreateDriver(d); err != nil {
		return nil, err
	}
	svc.logger.Info("driver_created", map[string]interface{}{
		"driver_id": d.ID,
		"email":     maskEmail(d.Email),
	})
	return d, nil
}

// GetDriver retrieves a driver by ID
func (svc *DriverService) GetDriver(id string) (*Driver, error) {
	if id == "" {
		return nil, fmt.Errorf("driver ID is required")
	}
	return svc.store.GetDriver(id)
}

// UpdateDriver applies partial updates to a driver record
func (svc *DriverService) UpdateDriver(id string, req UpdateDriverRequest) (*Driver, error) {
	d, err := svc.store.GetDriver(id)
	if err != nil {
		return nil, err
	}
	if req.FirstName != nil {
		if len(*req.FirstName) == 0 || len(*req.FirstName) > MaxDriverNameLength {
			return nil, fmt.Errorf("invalid first_name length")
		}
		d.FirstName = strings.TrimSpace(*req.FirstName)
	}
	if req.LastName != nil {
		if len(*req.LastName) == 0 || len(*req.LastName) > MaxDriverNameLength {
			return nil, fmt.Errorf("invalid last_name length")
		}
		d.LastName = strings.TrimSpace(*req.LastName)
	}
	if req.Phone != nil {
		if len(*req.Phone) > MaxPhoneLength {
			return nil, fmt.Errorf("invalid phone length")
		}
		d.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.Address != nil {
		d.Address = *req.Address
	}
	d.UpdatedAt = time.Now().UTC()
	if err := svc.store.UpdateDriver(d); err != nil {
		return nil, err
	}
	svc.logger.Info("driver_updated", map[string]interface{}{"driver_id": id})
	return d, nil
}

// DeleteDriver soft-deletes a driver (DSGVO Recht auf Loeschung)
func (svc *DriverService) DeleteDriver(id string) error {
	if err := svc.store.DeleteDriver(id); err != nil {
		return err
	}
	svc.logger.Info("driver_deleted", map[string]interface{}{
		"driver_id": id,
		"gdpr_note": "Personenbezogene Daten pseudonymisiert gemaess DSGVO Art. 17",
	})
	return nil
}

// ListDrivers retrieves all active drivers
func (svc *DriverService) ListDrivers(limit, offset int) ([]*Driver, int) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return svc.store.ListDrivers(limit, offset)
}

// UpdateStatus changes driver availability
func (svc *DriverService) UpdateStatus(driverID string, req UpdateStatusRequest) (*Driver, error) {
	if !isValidStatus(req.Status) {
		return nil, fmt.Errorf("invalid status: %s", req.Status)
	}
	// PBefG: Validierung des Fahrerausweises vor Statuswechsel zu Online
	d, err := svc.store.GetDriver(driverID)
	if err != nil {
		return nil, err
	}
	if req.Status == DriverStatusOnline {
		if err := svc.validateDriverCompliance(d); err != nil {
			return nil, fmt.Errorf("compliance check failed: %w", err)
		}
	}
	if err := svc.store.UpdateDriverStatus(driverID, req.Status, req.Latitude, req.Longitude, req.Note); err != nil {
		return nil, err
	}
	svc.logger.Info("driver_status_updated", map[string]interface{}{
		"driver_id": driverID,
		"status":    req.Status,
	})
	return svc.store.GetDriver(driverID)
}

// validateDriverCompliance checks PBefG requirements before going online
func (svc *DriverService) validateDriverCompliance(d *Driver) error {
	now := time.Now().UTC()
	if d.LicenseExpiry.Before(now) {
		return fmt.Errorf("Fuehrerschein abgelaufen (PBefG §47)")
	}
	if d.PBefGExpiry.Before(now) {
		return fmt.Errorf("P-Schein (PBefG-Genehmigung) abgelaufen")
	}
	if d.Vehicle != nil {
		if d.Vehicle.InsuranceExpiry.Before(now) {
			return fmt.Errorf("Fahrzeugversicherung abgelaufen")
		}
		if d.Vehicle.InspectionExpiry.Before(now) {
			return fmt.Errorf("Hauptuntersuchung (HU) abgelaufen")
		}
	}
	return nil
}

// UpdateLocation processes a GPS position update
func (svc *DriverService) UpdateLocation(driverID string, req LocationUpdateRequest) (*CurrentLocation, error) {
	if !isValidLatitude(req.Latitude) || !isValidLongitude(req.Longitude) {
		return nil, fmt.Errorf("invalid GPS coordinates: lat=%f lng=%f", req.Latitude, req.Longitude)
	}
	if !svc.store.DriverExists(driverID) {
		return nil, fmt.Errorf("driver not found: %s", driverID)
	}
	update := LocationUpdate{
		DriverID:  driverID,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Accuracy:  req.Accuracy,
		Speed:     req.Speed,
		Heading:   req.Heading,
		Altitude:  req.Altitude,
		Timestamp: time.Now().UTC(),
	}
	if err := svc.store.UpdateLocation(update); err != nil {
		return nil, err
	}
	return svc.store.GetLocation(driverID)
}

// GetLocation retrieves the current location of a driver
func (svc *DriverService) GetLocation(driverID string) (*CurrentLocation, error) {
	if !svc.store.DriverExists(driverID) {
		return nil, fmt.Errorf("driver not found: %s", driverID)
	}
	return svc.store.GetLocation(driverID)
}

// GetNearbyDrivers finds available drivers within a radius
func (svc *DriverService) GetNearbyDrivers(lat, lng, radiusKm float64) ([]*CurrentLocation, error) {
	if !isValidLatitude(lat) || !isValidLongitude(lng) {
		return nil, fmt.Errorf("invalid GPS coordinates")
	}
	if radiusKm <= 0 || radiusKm > 50 {
		radiusKm = 5.0
	}
	return svc.store.GetNearbyDrivers(lat, lng, radiusKm), nil
}

// AddEarnings records an earnings entry
func (svc *DriverService) AddEarnings(driverID string, req AddEarningsRequest) (*EarningsEntry, error) {
	if !svc.store.DriverExists(driverID) {
		return nil, fmt.Errorf("driver not found: %s", driverID)
	}
	if req.GrossAmount < 0 {
		return nil, fmt.Errorf("gross amount cannot be negative")
	}
	netAmount := req.GrossAmount - req.Commission
	invoiceNum := fmt.Sprintf("RE-%s-%d", driverID[:8], time.Now().UnixNano())
	entry := EarningsEntry{
		ID:            uuid.New().String(),
		DriverID:      driverID,
		TripID:        req.TripID,
		GrossAmount:   req.GrossAmount,
		NetAmount:     netAmount,
		Commission:    req.Commission,
		TaxAmount:     req.TaxAmount,
		TipAmount:     req.TipAmount,
		PaymentMethod: req.PaymentMethod,
		Currency:      "EUR",
		EarnedAt:      time.Now().UTC(),
		InvoiceNumber: invoiceNum,
	}
	if err := svc.store.AddEarnings(entry); err != nil {
		return nil, err
	}
	svc.logger.Info("earnings_recorded", map[string]interface{}{
		"driver_id":  driverID,
		"trip_id":    req.TripID,
		"net_amount": netAmount,
		"invoice":    invoiceNum,
	})
	return &entry, nil
}

// GetEarningsSummary calculates earnings for a period
func (svc *DriverService) GetEarningsSummary(driverID, period string) (*EarningsSummary, error) {
	if !svc.store.DriverExists(driverID) {
		return nil, fmt.Errorf("driver not found: %s", driverID)
	}
	now := time.Now().UTC()
	var from, to time.Time
	switch period {
	case "daily":
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		to = from.Add(24 * time.Hour)
	case "weekly":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		from = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.UTC)
		to = from.Add(7 * 24 * time.Hour)
	case "monthly":
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to = from.AddDate(0, 1, 0)
	default:
		return nil, fmt.Errorf("invalid period: %s (use daily, weekly, monthly)", period)
	}
	entries, err := svc.store.GetEarnings(driverID, from, to)
	if err != nil {
		return nil, err
	}
	summary := &EarningsSummary{
		DriverID:  driverID,
		Period:    period,
		StartDate: from,
		EndDate:   to,
		Currency:  "EUR",
	}
	for _, e := range entries {
		summary.TotalGross += e.GrossAmount
		summary.TotalNet += e.NetAmount
		summary.TotalTips += e.TipAmount
		summary.TotalTax += e.TaxAmount
		summary.TotalTrips++
	}
	if summary.TotalTrips > 0 {
		summary.AveragePerTrip = summary.TotalNet / float64(summary.TotalTrips)
	}
	return summary, nil
}

// AddTrip records a trip in the driver's history
func (svc *DriverService) AddTrip(driverID string, req AddTripRequest) (*TripHistory, error) {
	if !svc.store.DriverExists(driverID) {
		return nil, fmt.Errorf("driver not found: %s", driverID)
	}
	if req.FareAmount < 0 {
		return nil, fmt.Errorf("fare amount cannot be negative")
	}
	if req.PassengerRating < 0 || req.PassengerRating > 5 {
		return nil, fmt.Errorf("passenger rating must be between 0 and 5")
	}
	now := time.Now().UTC()
	trip := TripHistory{
		ID:              uuid.New().String(),
		DriverID:        driverID,
		PassengerID:     req.PassengerID,
		PickupAddress:   req.PickupAddress,
		DropoffAddress:  req.DropoffAddress,
		PickupLat:       req.PickupLat,
		PickupLng:       req.PickupLng,
		DropoffLat:      req.DropoffLat,
		DropoffLng:      req.DropoffLng,
		DistanceKm:      req.DistanceKm,
		DurationMinutes: req.DurationMinutes,
		Status:          req.Status,
		PaymentMethod:   req.PaymentMethod,
		FareAmount:      req.FareAmount,
		TipAmount:       req.TipAmount,
		PassengerRating: req.PassengerRating,
		StartedAt:       req.StartedAt,
		CompletedAt:     req.CompletedAt,
		CreatedAt:       now,
	}
	if err := svc.store.AddTripHistory(trip); err != nil {
		return nil, err
	}
	svc.logger.Info("trip_recorded", map[string]interface{}{
		"driver_id": driverID,
		"trip_id":   trip.ID,
		"status":    trip.Status,
		"distance":  trip.DistanceKm,
	})
	return &trip, nil
}

// GetTripHistory retrieves paginated trip history
func (svc *DriverService) GetTripHistory(driverID string, limit, offset int) ([]TripHistory, int, error) {
	if !svc.store.DriverExists(driverID) {
		return nil, 0, fmt.Errorf("driver not found: %s", driverID)
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return svc.store.GetTripHistory(driverID, limit, offset)
}

// AddVehicle registers a vehicle for a driver
func (svc *DriverService) AddVehicle(driverID string, req AddVehicleRequest) (*Vehicle, error) {
	if !svc.store.DriverExists(driverID) {
		return nil, fmt.Errorf("driver not found: %s", driverID)
	}
	if len(req.LicensePlate) == 0 || len(req.LicensePlate) > MaxLicensePlateLength {
		return nil, fmt.Errorf("invalid license plate")
	}
	if req.Seats < 1 || req.Seats > 50 {
		return nil, fmt.Errorf("seats must be between 1 and 50")
	}
	v := &Vehicle{
		ID:               uuid.New().String(),
		LicensePlate:     strings.ToUpper(strings.TrimSpace(req.LicensePlate)),
		Make:             strings.TrimSpace(req.Make),
		Model:            strings.TrimSpace(req.Model),
		Year:             req.Year,
		Color:            req.Color,
		Type:             req.Type,
		Seats:            req.Seats,
		HasChildSeat:     req.HasChildSeat,
		IsAccessible:     req.IsAccessible,
		InsuranceExpiry:  req.InsuranceExpiry,
		InspectionExpiry: req.InspectionExpiry,
		TUVExpiry:        req.TUVExpiry,
	}
	if err := svc.store.AddVehicle(driverID, v); err != nil {
		return nil, err
	}
	svc.logger.Info("vehicle_added", map[string]interface{}{
		"driver_id":     driverID,
		"vehicle_id":    v.ID,
		"license_plate": v.LicensePlate,
	})
	return v, nil
}

// =============================================================================
// STRUCTURED LOGGER
// =============================================================================

// Logger provides structured JSON logging
type Logger struct {
	logger *log.Logger
	service string
}

// NewLogger creates a new structured logger
func NewLogger(service string) *Logger {
	return &Logger{
		logger:  log.New(os.Stdout, "", 0),
		service: service,
	}
}

// logEntry is the JSON log structure
type logEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

func (l *Logger) log(level, message string, fields map[string]interface{}) {
	entry := logEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Service:   l.service,
		Message:   message,
		Fields:    fields,
	}
	b, err := json.Marshal(entry)
	if err != nil {
		l.logger.Printf(`{"level":"error","message":"failed to marshal log entry"}`)
		return
	}
	l.logger.Println(string(b))
}

// Info logs an informational message
func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.log("INFO", message, fields)
}

// Error logs an error message
func (l *Logger) Error(message string, fields map[string]interface{}) {
	l.log("ERROR", message, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields map[string]interface{}) {
	l.log("WARN", message, fields)
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields map[string]interface{}) {
	l.log("DEBUG", message, fields)
}

// =============================================================================
// RATE LIMITER
// =============================================================================

// rateLimiterEntry tracks request counts per client
type rateLimiterEntry struct {
	count    int
	resetAt  time.Time
}

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*rateLimiterEntry
	limit   int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*rateLimiterEntry),
		limit:   requestsPerMinute,
	}
	go rl.cleanup()
	return rl
}

// Allow checks if a client is within the rate limit
func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	entry, ok := rl.clients[clientID]
	if !ok || now.After(entry.resetAt) {
		rl.clients[clientID] = &rateLimiterEntry{
			count:   1,
			resetAt: now.Add(time.Minute),
		}
		return true
	}
	if entry.count >= rl.limit {
		return false
	}
	entry.count++
	return true
}

// cleanup removes expired rate limit entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for k, v := range rl.clients {
			if now.After(v.resetAt) {
				delete(rl.clients, k)
			}
		}
		rl.mu.Unlock()
	}
}

// =============================================================================
// MIDDLEWARE
// =============================================================================

// requestIDKey is the context key for request IDs
type contextKey string

const requestIDKey contextKey = "request_id"

// RequestIDMiddleware injects a unique request ID into each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SecurityHeadersMiddleware sets security-relevant HTTP headers
// DSGVO: Technische Massnahmen gemaess Art. 25 (Privacy by Design)
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs each HTTP request with structured logging
func LoggingMiddleware(logger *Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			duration := time.Since(start)
			requestID, _ := r.Context().Value(requestIDKey).(string)
			logger.Info("http_request", map[string]interface{}{
				"method":     r.Method,
				"path":       r.URL.Path,
				"status":     rw.statusCode,
				"duration_ms": duration.Milliseconds(),
				"request_id": requestID,
				"ip":         getClientIP(r),
				"user_agent": r.UserAgent(),
			})
		})
	}
}

// RateLimitMiddleware enforces rate limiting per client IP
func RateLimitMiddleware(limiter *RateLimiter) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)
			if !limiter.Allow(clientIP) {
				w.Header().Set("Retry-After", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(APIResponse{
					Success:   false,
					Error:     &APIError{Code: "RATE_LIMIT_EXCEEDED", Message: "Zu viele Anfragen. Bitte warten."},
					Timestamp: time.Now().UTC(),
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware validates bearer tokens
// In production: validate JWT tokens with proper signing key
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health and metrics endpoints are exempt from auth
		if r.URL.Path == "/health" || r.URL.Path == "/metrics" || r.URL.Path == "/" {
			next.ServeHTTP(w, r)
			return
		}
		authorization := r.Header.Get("Authorization")
		if authorization == "" {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentifizierung erforderlich", "")
			return
		}
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, r, http.StatusUnauthorized, "INVALID_TOKEN_FORMAT", "Ungültiges Token-Format", "")
			return
		}
		token := parts[1]
		// TODO: In production, validate JWT signature, expiry, and claims
		// For demonstration, accept any non-empty token
		if token == "" {
			writeError(w, r, http.StatusUnauthorized, "INVALID_TOKEN", "Ungültiges Token", "")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ContentTypeMiddleware enforces JSON content type for POST/PUT/PATCH
func ContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			ct := r.Header.Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				writeError(w, r, http.StatusUnsupportedMediaType, "INVALID_CONTENT_TYPE", "Content-Type muss application/json sein", "")
				return
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// =============================================================================
// HTTP HANDLERS
// =============================================================================

// Handler holds dependencies for HTTP handlers
type Handler struct {
	service *DriverService
	logger  *Logger
	startTime time.Time
}

// NewHandler creates a new Handler
func NewHandler(service *DriverService, logger *Logger) *Handler {
	return &Handler{
		service:   service,
		logger:    logger,
		startTime: time.Now(),
	}
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime).String()
	resp := HealthResponse{
		Status:    "healthy",
		Service:   ServiceName,
		Version:   ServiceVersion,
		Timestamp: time.Now().UTC(),
		Uptime:    uptime,
		Checks: map[string]string{
			"store":   "ok",
			"memory":  "ok",
			"service": "ok",
		},
	}
	writeJSON(w, r, http.StatusOK, resp)
}

// Root handles GET /
func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusOK, map[string]string{
		"service": ServiceName,
		"version": ServiceVersion,
		"status":  "running",
	})
}

// CreateDriver handles POST /api/v1/drivers
func (h *Handler) CreateDriver(w http.ResponseWriter, r *http.Request) {
	var req CreateDriverRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Ungültiger Request-Body", err.Error())
		return
	}
	driver, err := h.service.CreateDriver(req)
	if err != nil {
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "already exists") {
			code = http.StatusConflict
		}
		writeError(w, r, code, "CREATE_DRIVER_FAILED", err.Error(), "")
		return
	}
	writeSuccess(w, r, http.StatusCreated, driver, nil)
}

// GetDriver handles GET /api/v1/drivers/{id}
func (h *Handler) GetDriver(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	driver, err := h.service.GetDriver(id)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "DRIVER_NOT_FOUND", "Fahrer nicht gefunden", err.Error())
		return
	}
	writeSuccess(w, r, http.StatusOK, driver, nil)
}

// UpdateDriver handles PUT /api/v1/drivers/{id}
func (h *Handler) UpdateDriver(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req UpdateDriverRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Ungültiger Request-Body", err.Error())
		return
	}
	driver, err := h.service.UpdateDriver(id, req)
	if err != nil {
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		}
		writeError(w, r, code, "UPDATE_DRIVER_FAILED", err.Error(), "")
		return
	}
	writeSuccess(w, r, http.StatusOK, driver, nil)
}

// DeleteDriver handles DELETE /api/v1/drivers/{id}
// DSGVO Art. 17: Recht auf Loeschung ("Recht auf Vergessenwerden")
func (h *Handler) DeleteDriver(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.service.DeleteDriver(id); err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		}
		writeError(w, r, code, "DELETE_DRIVER_FAILED", err.Error(), "")
		return
	}
	writeSuccess(w, r, http.StatusOK, map[string]string{
		"message": "Fahrer wurde gemaess DSGVO Art. 17 geloescht und pseudonymisiert",
		"driver_id": id,
	}, nil)
}

// ListDrivers handles GET /api/v1/drivers
func (h *Handler) ListDrivers(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	drivers, total := h.service.ListDrivers(limit, offset)
	if drivers == nil {
		drivers = []*Driver{}
	}
	writeSuccess(w, r, http.StatusOK, drivers, &Meta{
		Total:  total,
		Page:   (offset / limit) + 1,
		Limit:  limit,
		Offset: offset,
	})
}

// UpdateStatus handles PUT /api/v1/drivers/{id}/status
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req UpdateStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Ungültiger Request-Body", err.Error())
		return
	}
	driver, err := h.service.UpdateStatus(id, req)
	if err != nil {
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		}
		writeError(w, r, code, "UPDATE_STATUS_FAILED", err.Error(), "")
		return
	}
	writeSuccess(w, r, http.StatusOK, driver, nil)
}

// UpdateLocation handles POST /api/v1/drivers/{id}/location
// Standortdaten: Zweckbindung gemaess DSGVO Art. 5 Abs. 1 lit. b
func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req LocationUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Ungültiger Request-Body", err.Error())
		return
	}
	loc, err := h.service.UpdateLocation(id, req)
	if err != nil {
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		}
		writeError(w, r, code, "UPDATE_LOCATION_FAILED", err.Error(), "")
		return
	}
	writeSuccess(w, r, http.StatusOK, loc, nil)
}

// GetLocation handles GET /api/v1/drivers/{id}/location
func (h *Handler) GetLocation(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	loc, err := h.service.GetLocation(id)
	if err != nil {
		code := http.StatusNotFound
		writeError(w, r, code, "LOCATION_NOT_FOUND", err.Error(), "")
		return
	}
	writeSuccess(w, r, http.StatusOK, loc, nil)
}

// GetNearbyDrivers handles GET /api/v1/drivers/nearby
func (h *Handler) GetNearbyDrivers(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")
	radiusStr := r.URL.Query().Get("radius")
	if latStr == "" || lngStr == "" {
		writeError(w, r, http.StatusBadRequest, "MISSING_COORDINATES", "lat und lng sind erforderlich", "")
		return
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_LATITUDE", "Ungültige Latitude", "")
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_LONGITUDE", "Ungültige Longitude", "")
		return
	}
	radius := 5.0
	if radiusStr != "" {
		radius, err = strconv.ParseFloat(radiusStr, 64)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_RADIUS", "Ungültiger Radius", "")
			return
		}
	}
	drivers, err := h.service.GetNearbyDrivers(lat, lng, radius)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "NEARBY_DRIVERS_FAILED", err.Error(), "")
		return
	}
	if drivers == nil {
		drivers = []*CurrentLocation{}
	}
	writeSuccess(w, r, http.StatusOK, drivers, &Meta{Total: len(drivers)})
}

// AddEarnings handles POST /api/v1/drivers/{id}/earnings
func (h *Handler) AddEarnings(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req AddEarningsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Ungültiger Request-Body", err.Error())
		return
	}
	entry, err := h.service.AddEarnings(id, req)
	if err != nil {
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		}
		writeError(w, r, code, "ADD_EARNINGS_FAILED", err.Error(), "")
		return
	}
	writeSuccess(w, r, http.StatusCreated, entry, nil)
}

// GetEarningsSummary handles GET /api/v1/drivers/{id}/earnings/{period}
func (h *Handler) GetEarningsSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	period := vars["period"]
	summary, err := h.service.GetEarningsSummary(id, period)
	if err != nil {
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		}
		writeError(w, r, code, "GET_EARNINGS_FAILED", err.Error(), "")
		return
	}
	writeSuccess(w, r, http.StatusOK, summary, nil)
}

// AddTrip handles POST /api/v1/drivers/{id}/trips
func (h *Handler) AddTrip(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req AddTripRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Ungültiger Request-Body", err.Error())
		return
	}
	trip, err := h.service.AddTrip(id, req)
	if err != nil {
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		}
		writeError(w, r, code, "ADD_TRIP_FAILED", err.Error(), "")
		return
	}
	writeSuccess(w, r, http.StatusCreated, trip, nil)
}

// GetTripHistory handles GET /api/v1/drivers/{id}/trips
func (h *Handler) GetTripHistory(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	limit, offset := parsePagination(r)
	trips, total, err := h.service.GetTripHistory(id, limit, offset)
	if err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		}
		writeError(w, r, code, "GET_TRIPS_FAILED", err.Error(), "")
		return
	}
	if trips == nil {
		trips = []TripHistory{}
	}
	writeSuccess(w, r, http.StatusOK, trips, &Meta{
		Total:  total,
		Page:   (offset / limit) + 1,
		Limit:  limit,
		Offset: offset,
	})
}

// AddVehicle handles POST /api/v1/drivers/{id}/vehicle
func (h *Handler) AddVehicle(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req AddVehicleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Ungültiger Request-Body", err.Error())
		return
	}
	vehicle, err := h.service.AddVehicle(id, req)
	if err != nil {
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		}
		writeError(w, r, code, "ADD_VEHICLE_FAILED", err.Error(), "")
		return
	}
	writeSuccess(w, r, http.StatusCreated, vehicle, nil)
}

// NotFound handles 404 responses
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotFound, "ENDPOINT_NOT_FOUND", fmt.Sprintf("Endpunkt %s %s nicht gefunden", r.Method, r.URL.Path), "")
}

// MethodNotAllowed handles 405 responses
func (h *Handler) MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", fmt.Sprintf("Methode %s nicht erlaubt", r.Method), "")
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// writeJSON encodes and writes a JSON response
func writeJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

// writeSuccess writes a successful API response
func writeSuccess(w http.ResponseWriter, r *http.Request, status int, data interface{}, meta *Meta) {
	requestID, _ := r.Context().Value(requestIDKey).(string)
	writeJSON(w, r, status, APIResponse{
		Success:   true,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now().UTC(),
		RequestID: requestID,
	})
}

// writeError writes an error API response
func writeError(w http.ResponseWriter, r *http.Request, status int, code, message, details string) {
	requestID, _ := r.Context().Value(requestIDKey).(string)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC(),
		RequestID: requestID,
	})
}

// decodeJSON decodes a JSON request body
func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("JSON decode error: %w", err)
	}
	return nil
}

// parsePagination extracts limit and offset from query params
func parsePagination(r *http.Request) (int, int) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	limit := 20
	offset := 0
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}

// getClientIP extracts the real client IP address
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

// maskEmail returns a privacy-safe version of an email for logging
// DSGVO: Datensparsamkeit bei der Protokollierung
func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***@***.***"
	}
	local := parts[0]
	if len(local) > 2 {
		local = local[:2] + strings.Repeat("*", len(local)-2)
	}
	return local + "@" + parts[1]
}

// isValidLatitude checks if a latitude value is within valid bounds
func isValidLatitude(lat float64) bool {
	return lat >= MinLatitude && lat <= MaxLatitude
}

// isValidLongitude checks if a longitude value is within valid bounds
func isValidLongitude(lng float64) bool {
	return lng >= MinLongitude && lng <= MaxLongitude
}

// isValidStatus checks if a DriverStatus value is one of the defined constants
func isValidStatus(s DriverStatus) bool {
	switch s {
	case DriverStatusOffline, DriverStatusOnline, DriverStatusBusy, DriverStatusOnBreak, DriverStatusSuspended:
		return true
	}
	return false
}

// haversineDistance calculates the distance between two GPS points in kilometers
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return EarthRadiusKm * c
}

// degreesToRadians converts degrees to radians
func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180
}

// validateCreateDriverRequest validates the create driver request
func validateCreateDriverRequest(req CreateDriverRequest) error {
	if strings.TrimSpace(req.FirstName) == "" || len(req.FirstName) > MaxDriverNameLength {
		return fmt.Errorf("first_name ist erforderlich und darf maximal %d Zeichen lang sein", MaxDriverNameLength)
	}
	if strings.TrimSpace(req.LastName) == "" || len(req.LastName) > MaxDriverNameLength {
		return fmt.Errorf("last_name ist erforderlich und darf maximal %d Zeichen lang sein", MaxDriverNameLength)
	}
	if !isValidEmail(req.Email) {
		return fmt.Errorf("ungültige E-Mail-Adresse")
	}
	if strings.TrimSpace(req.Phone) == "" || len(req.Phone) > MaxPhoneLength {
		return fmt.Errorf("phone ist erforderlich")
	}
	if req.DateOfBirth.IsZero() {
		return fmt.Errorf("date_of_birth ist erforderlich")
	}
	age := time.Since(req.DateOfBirth).Hours() / 24 / 365
	if age < 21 {
		return fmt.Errorf("Fahrer muss mindestens 21 Jahre alt sein (PBefG)")
	}
	if req.LicenseNumber == "" {
		return fmt.Errorf("license_number ist erforderlich")
	}
	if req.LicenseExpiry.Before(time.Now()) {
		return fmt.Errorf("Fuehrerschein ist abgelaufen")
	}
	if req.PBefGLicense == "" {
		return fmt.Errorf("pbefg_license ist erforderlich (PBefG §47)")
	}
	if req.PBefGExpiry.Before(time.Now()) {
		return fmt.Errorf("PBefG-Genehmigung ist abgelaufen")
	}
	if req.Address.City == "" || req.Address.PostalCode == "" {
		return fmt.Errorf("Adresse (Stadt und Postleitzahl) ist erforderlich")
	}
	return nil
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	if len(email) < 5 || len(email) > 254 {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	return len(parts[0]) > 0 && strings.Contains(parts[1], ".")
}

// getEnv retrieves an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// =============================================================================
// ROUTER SETUP
// =============================================================================

// NewRouter creates and configures the application router
func NewRouter(handler *Handler, logger *Logger, rateLimiter *RateLimiter) *mux.Router {
	r := mux.NewRouter()

	// Global middleware
	r.Use(RequestIDMiddleware)
	r.Use(SecurityHeadersMiddleware)
	r.Use(LoggingMiddleware(logger))
	r.Use(RateLimitMiddleware(rateLimiter))
	r.Use(AuthMiddleware)
	r.Use(ContentTypeMiddleware)

	// Custom error handlers
	r.NotFoundHandler = http.HandlerFunc(handler.NotFound)
	r.MethodNotAllowedHandler = http.HandlerFunc(handler.MethodNotAllowed)

	// Infrastructure endpoints (no auth required)
	r.HandleFunc("/", handler.Root).Methods(http.MethodGet)
	r.HandleFunc("/health", handler.HealthCheck).Methods(http.MethodGet)

	// API v1 routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Driver CRUD
	api.HandleFunc("/drivers", handler.ListDrivers).Methods(http.MethodGet)
	api.HandleFunc("/drivers", handler.CreateDriver).Methods(http.MethodPost)
	api.HandleFunc("/drivers/nearby", handler.GetNearbyDrivers).Methods(http.MethodGet)
	api.HandleFunc("/drivers/{id}", handler.GetDriver).Methods(http.MethodGet)
	api.HandleFunc("/drivers/{id}", handler.UpdateDriver).Methods(http.MethodPut)
	api.HandleFunc("/drivers/{id}", handler.DeleteDriver).Methods(http.MethodDelete)

	// Driver status
	api.HandleFunc("/drivers/{id}/status", handler.UpdateStatus).Methods(http.MethodPut)

	// Driver location
	api.HandleFunc("/drivers/{id}/location", handler.GetLocation).Methods(http.MethodGet)
	api.HandleFunc("/drivers/{id}/location", handler.UpdateLocation).Methods(http.MethodPost)

	// Driver earnings
	api.HandleFunc("/drivers/{id}/earnings", handler.AddEarnings).Methods(http.MethodPost)
	api.HandleFunc("/drivers/{id}/earnings/{period}", handler.GetEarningsSummary).Methods(http.MethodGet)

	// Driver trips
	api.HandleFunc("/drivers/{id}/trips", handler.GetTripHistory).Methods(http.MethodGet)
	api.HandleFunc("/drivers/{id}/trips", handler.AddTrip).Methods(http.MethodPost)

	// Driver vehicle
	api.HandleFunc("/drivers/{id}/vehicle", handler.AddVehicle).Methods(http.MethodPost)

	return r
}

// =============================================================================
// MAIN ENTRY POINT
// =============================================================================

func main() {
	// =========================================================================
	// INITIALIZATION
	// =========================================================================

	logger := NewLogger(ServiceName)

	logger.Info("service_starting", map[string]interface{}{
		"service": ServiceName,
		"version": ServiceVersion,
		"compliance": "DSGVO/PBefG/BDSG",
	})

	// Configuration from environment variables
	port := getEnv("PORT", DefaultPort)
	readTimeout := DefaultReadTimeout
	writeTimeout := DefaultWriteTimeout
	idleTimeout := DefaultIdleTimeout

	if v := getEnv("READ_TIMEOUT_SECONDS", ""); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			readTimeout = time.Duration(secs) * time.Second
		}
	}
	if v := getEnv("WRITE_TIMEOUT_SECONDS", ""); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			writeTimeout = time.Duration(secs) * time.Second
		}
	}

	// =========================================================================
	// DEPENDENCY WIRING
	// =========================================================================

	store := NewDriverStore()
	service := NewDriverService(store, logger)
	handler := NewHandler(service, logger)
	rateLimiter := NewRateLimiter(RateLimitRequestsPerMinute)
	router := NewRouter(handler, logger, rateLimiter)

	// =========================================================================
	// SEED DATA (development only)
	// =========================================================================

	if getEnv("SEED_DATA", "false") == "true" {
		seedDemoData(service, logger)
	}

	// =========================================================================
	// HTTP SERVER
	// =========================================================================

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		ErrorLog:     log.New(os.Stderr, "", 0),
	}

	// Run server in a goroutine so it does not block graceful shutdown
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server_listening", map[string]interface{}{
			"addr":          server.Addr,
			"read_timeout":  readTimeout.String(),
			"write_timeout": writeTimeout.String(),
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// =========================================================================
	// GRACEFUL SHUTDOWN
	// =========================================================================

	// Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case err := <-serverErrors:
		logger.Error("server_error", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	case sig := <-quit:
		logger.Info("shutdown_signal_received", map[string]interface{}{"signal": sig.String()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer cancel()

	logger.Info("server_shutting_down", map[string]interface{}{
		"timeout": DefaultShutdownTimeout.String(),
	})

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server_shutdown_error", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}

	logger.Info("server_stopped", map[string]interface{}{"service": ServiceName})
}

// =============================================================================
// SEED DATA
// =============================================================================

// seedDemoData populates the store with example data for development/testing
// HINWEIS: Nur fuer Entwicklungs- und Testzwecke. Niemals in Produktion verwenden.
func seedDemoData(svc *DriverService, logger *Logger) {
	logger.Warn("seeding_demo_data", map[string]interface{}{
		"warning": "Demo-Daten werden geladen. Nur fuer Entwicklung!",
	})

	consentTime := time.Now().UTC()

	// Seed driver 1
	req1 := CreateDriverRequest{
		FirstName:     "Hans",
		LastName:      "Mueller",
		Email:         "hans.mueller@example.de",
		Phone:         "+49 151 12345678",
		DateOfBirth:   time.Date(1985, 3, 15, 0, 0, 0, 0, time.UTC),
		LicenseClass:  LicenseClassB,
		LicenseNumber: "B072RRF32001",
		LicenseExpiry: time.Now().AddDate(5, 0, 0),
		PBefGLicense:  "P-2024-BER-001234",
		PBefGExpiry:   time.Now().AddDate(2, 0, 0),
		ConsentGiven:  true,
		Address: Address{
			Street:      "Hauptstrasse",
			HouseNumber: "42",
			City:        "Berlin",
			PostalCode:  "10115",
			Country:     "DE",
			State:       "Berlin",
		},
	}
	req1.DateOfBirth = time.Date(1985, 3, 15, 0, 0, 0, 0, time.UTC)
	_ = consentTime

	driver1, err := svc.CreateDriver(req1)
	if err != nil {
		logger.Error("seed_driver_1_failed", map[string]interface{}{"error": err.Error()})
	} else {
		// Add vehicle for driver 1
		_, _ = svc.AddVehicle(driver1.ID, AddVehicleRequest{
			LicensePlate:     "B-MU-1234",
			Make:             "Volkswagen",
			Model:            "Passat",
			Year:             2021,
			Color:            "Silber",
			Type:             VehicleTypeSedan,
			Seats:            4,
			HasChildSeat:     true,
			IsAccessible:     false,
			InsuranceExpiry:  time.Now().AddDate(1, 0, 0),
			InspectionExpiry: time.Now().AddDate(2, 0, 0),
			TUVExpiry:        time.Now().AddDate(2, 0, 0),
		})
		// Set driver online with location
		_, _ = svc.UpdateStatus(driver1.ID, UpdateStatusRequest{
			Status:    DriverStatusOnline,
			Latitude:  52.5200,
			Longitude: 13.4050,
		})
		// Add location
		_, _ = svc.UpdateLocation(driver1.ID, LocationUpdateRequest{
			Latitude:  52.5200,
			Longitude: 13.4050,
			Accuracy:  10,
			Speed:     0,
			Heading:   0,
			Altitude:  35,
		})
		// Add sample trips
		completedAt := time.Now().Add(-2 * time.Hour)
		_, _ = svc.AddTrip(driver1.ID, AddTripRequest{
			PassengerID:     "pax-" + uuid.New().String()[:8],
			PickupAddress:   "Alexanderplatz, 10178 Berlin",
			DropoffAddress:  "Kurfuerstendamm 123, 10719 Berlin",
			PickupLat:       52.5219,
			PickupLng:       13.4132,
			DropoffLat:      52.5048,
			DropoffLng:      13.3343,
			DistanceKm:      8.5,
			DurationMinutes: 22,
			Status:          TripStatusCompleted,
			PaymentMethod:   PaymentMethodCard,
			FareAmount:      18.50,
			TipAmount:       2.00,
			PassengerRating: 4.8,
			StartedAt:       time.Now().Add(-3 * time.Hour),
			CompletedAt:     &completedAt,
		})
		// Add earnings
		_, _ = svc.AddEarnings(driver1.ID, AddEarningsRequest{
			TripID:        uuid.New().String(),
			GrossAmount:   18.50,
			Commission:    3.33, // ~18% Plattformprovision
			TaxAmount:     2.43, // 19% USt auf Nettoanteil
			TipAmount:     2.00,
			PaymentMethod: PaymentMethodCard,
		})
		logger.Info("seed_driver_1_created", map[string]interface{}{"driver_id": driver1.ID})
	}

	// Seed driver 2
	req2 := CreateDriverRequest{
		FirstName:     "Fatima",
		LastName:      "Yilmaz",
		Email:         "fatima.yilmaz@example.de",
		Phone:         "+49 171 98765432",
		DateOfBirth:   time.Date(1990, 7, 22, 0, 0, 0, 0, time.UTC),
		LicenseClass:  LicenseClassB,
		LicenseNumber: "M084TTG45002",
		LicenseExpiry: time.Now().AddDate(3, 0, 0),
		PBefGLicense:  "P-2024-MUC-005678",
		PBefGExpiry:   time.Now().AddDate(1, 6, 0),
		ConsentGiven:  true,
		Address: Address{
			Street:      "Marienplatz",
			HouseNumber: "7",
			City:        "Muenchen",
			PostalCode:  "80331",
			Country:     "DE",
			State:       "Bayern",
		},
	}
	driver2, err := svc.CreateDriver(req2)
	if err != nil {
		logger.Error("seed_driver_2_failed", map[string]interface{}{"error": err.Error()})
	} else {
		_, _ = svc.AddVehicle(driver2.ID, AddVehicleRequest{
			LicensePlate:     "M-FY-5678",
			Make:             "BMW",
			Model:            "5 Series",
			Year:             2022,
			Color:            "Schwarz",
			Type:             VehicleTypeElectric,
			Seats:            4,
			HasChildSeat:     false,
			IsAccessible:     false,
			InsuranceExpiry:  time.Now().AddDate(1, 0, 0),
			InspectionExpiry: time.Now().AddDate(1, 6, 0),
			TUVExpiry:        time.Now().AddDate(1, 6, 0),
		})
		_, _ = svc.UpdateStatus(driver2.ID, UpdateStatusRequest{
			Status:    DriverStatusBusy,
			Latitude:  48.1351,
			Longitude: 11.5820,
		})
		_, _ = svc.UpdateLocation(driver2.ID, LocationUpdateRequest{
			Latitude:  48.1351,
			Longitude: 11.5820,
			Accuracy:  8,
			Speed:     45,
			Heading:   270,
			Altitude:  519,
		})
		logger.Info("seed_driver_2_created", map[string]interface{}{"driver_id": driver2.ID})
	}

	logger.Info("seed_data_complete", map[string]interface{}{"drivers_seeded": 2})
}
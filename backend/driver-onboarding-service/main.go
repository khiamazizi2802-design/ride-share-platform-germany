package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// ============================================================
// Constants and Enumerations
// ============================================================

const (
	ServiceName    = "driver-onboarding-service"
	ServiceVersion = "1.0.0"
	DefaultPort    = "8080"

	// GDPR retention periods
	GDPRRetentionApproved = 10 * 365 * 24 * time.Hour // 10 years for approved drivers
	GDPRRetentionRejected = 6 * 30 * 24 * time.Hour   // 6 months for rejected
	GDPRRetentionPending  = 2 * 365 * 24 * time.Hour   // 2 years for pending
)

type OnboardingStatus string

const (
	StatusRegistration       OnboardingStatus = "REGISTRATION"
	StatusEmailVerification  OnboardingStatus = "EMAIL_VERIFICATION"
	StatusDocumentUpload     OnboardingStatus = "DOCUMENT_UPLOAD"
	StatusKYCVerification    OnboardingStatus = "KYC_VERIFICATION"
	StatusPScheinVerification OnboardingStatus = "P_SCHEIN_VERIFICATION"
	StatusVehicleRegistration OnboardingStatus = "VEHICLE_REGISTRATION"
	StatusBackgroundCheck    OnboardingStatus = "BACKGROUND_CHECK"
	StatusAdminReview        OnboardingStatus = "ADMIN_REVIEW"
	StatusApproved           OnboardingStatus = "APPROVED"
	StatusRejected           OnboardingStatus = "REJECTED"
)

type DocumentType string

const (
	DocumentTypeNationalID       DocumentType = "NATIONAL_ID"
	DocumentTypePassport         DocumentType = "PASSPORT"
	DocumentTypePSchein          DocumentType = "P_SCHEIN"
	DocumentTypeInsurance        DocumentType = "INSURANCE"
	DocumentTypeBackgroundCheck  DocumentType = "BACKGROUND_CHECK"
	DocumentTypeVehicleRegistration DocumentType = "VEHICLE_REGISTRATION"
	DocumentTypeDrivingLicense   DocumentType = "DRIVING_LICENSE"
)

type DocumentStatus string

const (
	DocumentStatusPending  DocumentStatus = "PENDING"
	DocumentStatusVerified DocumentStatus = "VERIFIED"
	DocumentStatusRejected DocumentStatus = "REJECTED"
	DocumentStatusExpired  DocumentStatus = "EXPIRED"
)

type VehicleCategory string

const (
	VehicleCategoryStandard  VehicleCategory = "STANDARD"
	VehicleCategoryComfort   VehicleCategory = "COMFORT"
	VehicleCategoryXL        VehicleCategory = "XL"
	VehicleCategoryEco       VehicleCategory = "ECO"
	VehicleCategoryBusiness  VehicleCategory = "BUSINESS"
)

type AuditAction string

const (
	AuditActionCreate          AuditAction = "CREATE"
	AuditActionUpdate          AuditAction = "UPDATE"
	AuditActionDocumentUpload  AuditAction = "DOCUMENT_UPLOAD"
	AuditActionVerify          AuditAction = "VERIFY"
	AuditActionApprove         AuditAction = "APPROVE"
	AuditActionReject          AuditAction = "REJECT"
	AuditActionSubmit          AuditAction = "SUBMIT"
	AuditActionWebhook         AuditAction = "WEBHOOK"
	AuditActionDataAccess      AuditAction = "DATA_ACCESS"
	AuditActionVehicleRegister AuditAction = "VEHICLE_REGISTER"
)

// ============================================================
// Core Data Models
// ============================================================

// PersonalInfo holds GDPR-sensitive personal data
type PersonalInfo struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"` // YYYY-MM-DD
	Nationality string `json:"nationality"`
	Address     Address `json:"address"`
}

// Address represents a physical address
type Address struct {
	Street     string `json:"street"`
	HouseNumber string `json:"house_number"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	State      string `json:"state"`  // Bundesland
	Country    string `json:"country"` // ISO 3166-1 alpha-2
}

// ContactInfo holds contact details
type ContactInfo struct {
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	PhoneVerified bool  `json:"phone_verified"`
	EmailVerified bool  `json:"email_verified"`
}

// GDPRConsent tracks consent for GDPR compliance
type GDPRConsent struct {
	DataProcessing    bool      `json:"data_processing"`
	MarketingEmails   bool      `json:"marketing_emails"`
	ThirdPartySharing bool      `json:"third_party_sharing"`
	ConsentTimestamp  time.Time `json:"consent_timestamp"`
	ConsentIP         string    `json:"consent_ip"`
	ConsentVersion    string    `json:"consent_version"`
}

// DataRetentionPolicy defines GDPR data retention rules
type DataRetentionPolicy struct {
	RetentionUntil  time.Time `json:"retention_until"`
	RetentionReason string    `json:"retention_reason"`
	AutoDeleteAt    time.Time `json:"auto_delete_at"`
	LegalBasis      string    `json:"legal_basis"`
}

// Document represents an uploaded verification document
type Document struct {
	ID           string         `json:"id"`
	DriverID     string         `json:"driver_id"`
	Type         DocumentType   `json:"type"`
	Status       DocumentStatus `json:"status"`
	FileName     string         `json:"file_name"`
	FileSize     int64          `json:"file_size"`
	MimeType     string         `json:"mime_type"`
	StorageURL   string         `json:"storage_url,omitempty"` // omitted in responses for security
	ExpiryDate   *time.Time     `json:"expiry_date,omitempty"`
	VerifiedAt   *time.Time     `json:"verified_at,omitempty"`
	VerifiedBy   string         `json:"verified_by,omitempty"`
	RejectedAt   *time.Time     `json:"rejected_at,omitempty"`
	RejectReason string         `json:"reject_reason,omitempty"`
	UploadedAt   time.Time      `json:"uploaded_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// PScheinInfo holds P-Schein (PersonenbefÃ¶rderungsschein) specific data
type PScheinInfo struct {
	LicenseNumber  string     `json:"license_number"`
	IssuingAuthority string   `json:"issuing_authority"`
	IssuedAt       time.Time  `json:"issued_at"`
	ExpiryDate     time.Time  `json:"expiry_date"`
	VehicleClasses []string   `json:"vehicle_classes"`
	IsValid        bool       `json:"is_valid"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
}

// Vehicle represents a registered vehicle
type Vehicle struct {
	ID               string          `json:"id"`
	DriverID         string          `json:"driver_id"`
	Make             string          `json:"make"`
	Model            string          `json:"model"`
	Year             int             `json:"year"`
	Color            string          `json:"color"`
	LicensePlate     string          `json:"license_plate"`
	VIN              string          `json:"vin"`
	Category         VehicleCategory `json:"category"`
	Seats            int             `json:"seats"`
	InsurancePolicyNo string         `json:"insurance_policy_no"`
	InsuranceExpiry  time.Time       `json:"insurance_expiry"`
	TUVExpiry        time.Time       `json:"tuv_expiry"` // German technical inspection
	IsAccessible     bool            `json:"is_accessible"` // wheelchair accessible
	RegisteredAt     time.Time       `json:"registered_at"`
	Approved         bool            `json:"approved"`
}

// BackgroundCheck holds background check status
type BackgroundCheck struct {
	ID            string     `json:"id"`
	DriverID      string     `json:"driver_id"`
	Provider      string     `json:"provider"`
	Status        string     `json:"status"`
	RequestedAt   time.Time  `json:"requested_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Result        string     `json:"result,omitempty"`
	ExternalRefID string     `json:"external_ref_id"`
	ValidUntil    *time.Time `json:"valid_until,omitempty"`
}

// OnboardingStep tracks individual steps in the onboarding process
type OnboardingStep struct {
	Step        OnboardingStatus `json:"step"`
	Status      string           `json:"status"` // PENDING, IN_PROGRESS, COMPLETED, FAILED
	StartedAt   *time.Time       `json:"started_at,omitempty"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
	Notes       string           `json:"notes,omitempty"`
	RetryCount  int              `json:"retry_count"`
}

// AuditLog tracks all actions for compliance
type AuditLog struct {
	ID         string      `json:"id"`
	DriverID   string      `json:"driver_id"`
	Action     AuditAction `json:"action"`
	ActorID    string      `json:"actor_id"`
	ActorType  string      `json:"actor_type"` // SYSTEM, ADMIN, DRIVER, WEBHOOK
	Details    string      `json:"details"`
	IPAddress  string      `json:"ip_address"`
	UserAgent  string      `json:"user_agent"`
	Timestamp  time.Time   `json:"timestamp"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// DriverProfile is the central model for driver onboarding
type DriverProfile struct {
	ID                  string               `json:"id"`
	Personal            PersonalInfo         `json:"personal"`
	Contact             ContactInfo          `json:"contact"`
	GDPRConsent         GDPRConsent          `json:"gdpr_consent"`
	DataRetention       DataRetentionPolicy  `json:"data_retention"`
	CurrentStatus       OnboardingStatus     `json:"current_status"`
	OnboardingSteps     []OnboardingStep     `json:"onboarding_steps"`
	Documents           []Document           `json:"documents"`
	PSchein             *PScheinInfo         `json:"p_schein,omitempty"`
	Vehicle             *Vehicle             `json:"vehicle,omitempty"`
	BackgroundCheck     *BackgroundCheck     `json:"background_check,omitempty"`
	AdminNotes          string               `json:"admin_notes,omitempty"`
	RejectionReason     string               `json:"rejection_reason,omitempty"`
	SubmittedAt         *time.Time           `json:"submitted_at,omitempty"`
	ApprovedAt          *time.Time           `json:"approved_at,omitempty"`
	RejectedAt          *time.Time           `json:"rejected_at,omitempty"`
	ApprovedBy          string               `json:"approved_by,omitempty"`
	RejectedBy          string               `json:"rejected_by,omitempty"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
	DeletedAt           *time.Time           `json:"deleted_at,omitempty"`
	IsAnonymized        bool                 `json:"is_anonymized"`
}

// ============================================================
// Request / Response Types
// ============================================================

// StartOnboardingRequest is the payload for POST /onboarding/start
type StartOnboardingRequest struct {
	FirstName         string      `json:"first_name"`
	LastName          string      `json:"last_name"`
	DateOfBirth       string      `json:"date_of_birth"`
	Nationality       string      `json:"nationality"`
	Email             string      `json:"email"`
	Phone             string      `json:"phone"`
	Street            string      `json:"street"`
	HouseNumber       string      `json:"house_number"`
	City              string      `json:"city"`
	PostalCode        string      `json:"postal_code"`
	State             string      `json:"state"`
	Country           string      `json:"country"`
	DataProcessing    bool        `json:"data_processing"`
	MarketingEmails   bool        `json:"marketing_emails"`
	ThirdPartySharing bool        `json:"third_party_sharing"`
	ConsentVersion    string      `json:"consent_version"`
}

// UploadDocumentRequest is the payload for POST /onboarding/:id/documents
type UploadDocumentRequest struct {
	Type       DocumentType      `json:"type"`
	FileName   string            `json:"file_name"`
	FileSize   int64             `json:"file_size"`
	MimeType   string            `json:"mime_type"`
	StorageURL string            `json:"storage_url"`
	ExpiryDate *time.Time        `json:"expiry_date,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// RegisterVehicleRequest is the payload for POST /onboarding/:id/vehicle
type RegisterVehicleRequest struct {
	Make              string          `json:"make"`
	Model             string          `json:"model"`
	Year              int             `json:"year"`
	Color             string          `json:"color"`
	LicensePlate      string          `json:"license_plate"`
	VIN               string          `json:"vin"`
	Category          VehicleCategory `json:"category"`
	Seats             int             `json:"seats"`
	InsurancePolicyNo string          `json:"insurance_policy_no"`
	InsuranceExpiry   time.Time       `json:"insurance_expiry"`
	TUVExpiry         time.Time       `json:"tuv_expiry"`
	IsAccessible      bool            `json:"is_accessible"`
}

// AdminRejectRequest is the payload for POST /admin/onboarding/:id/reject
type AdminRejectRequest struct {
	Reason  string `json:"reason"`
	Notes   string `json:"notes"`
	AdminID string `json:"admin_id"`
}

// AdminApproveRequest is the payload for POST /admin/onboarding/:id/approve
type AdminApproveRequest struct {
	AdminID string `json:"admin_id"`
	Notes   string `json:"notes"`
}

// WebhookVerificationPayload is the payload for POST /webhooks/verification
type WebhookVerificationPayload struct {
	EventType     string            `json:"event_type"`
	ExternalRefID string            `json:"external_ref_id"`
	DriverID      string            `json:"driver_id"`
	Status        string            `json:"status"`
	Result        string            `json:"result"`
	Details       map[string]string `json:"details"`
	Timestamp     time.Time         `json:"timestamp"`
	Signature     string            `json:"signature"`
}

// OnboardingStatusResponse is the response for GET /onboarding/:id/status
type OnboardingStatusResponse struct {
	ID              string           `json:"id"`
	CurrentStatus   OnboardingStatus `json:"current_status"`
	OnboardingSteps []OnboardingStep `json:"onboarding_steps"`
	SubmittedAt     *time.Time       `json:"submitted_at,omitempty"`
	ApprovedAt      *time.Time       `json:"approved_at,omitempty"`
	RejectedAt      *time.Time       `json:"rejected_at,omitempty"`
	RejectionReason string           `json:"rejection_reason,omitempty"`
	CompletedSteps  int              `json:"completed_steps"`
	TotalSteps      int              `json:"total_steps"`
	ProgressPercent float64          `json:"progress_percent"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// APIResponse is a generic API response wrapper
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	RequestID string      `json:"request_id"`
	Timestamp time.Time   `json:"timestamp"`
}

// HealthResponse is the response for GET /health
type HealthResponse struct {
	Status      string            `json:"status"`
	Service     string            `json:"service"`
	Version     string            `json:"version"`
	Timestamp   time.Time         `json:"timestamp"`
	Uptime      string            `json:"uptime"`
	Checks      map[string]string `json:"checks"`
	Stats       map[string]int    `json:"stats"`
}

// PendingApplicationSummary is used in admin listing
type PendingApplicationSummary struct {
	ID            string           `json:"id"`
	FullName      string           `json:"full_name"`
	Email         string           `json:"email"`
	CurrentStatus OnboardingStatus `json:"current_status"`
	SubmittedAt   *time.Time       `json:"submitted_at,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	DocumentCount int              `json:"document_count"`
	HasVehicle    bool             `json:"has_vehicle"`
	HasPSchein    bool             `json:"has_p_schein"`
}

// ============================================================
// In-Memory Storage
// ============================================================

// Store is the thread-safe in-memory data store
type Store struct {
	mu          sync.RWMutex
	drivers     map[string]*DriverProfile
	auditLogs   map[string][]AuditLog
	emailIndex  map[string]string // email -> driverID
}

// NewStore creates and initializes a new Store
func NewStore() *Store {
	return &Store{
		drivers:    make(map[string]*DriverProfile),
		auditLogs:  make(map[string][]AuditLog),
		emailIndex: make(map[string]string),
	}
}

// CreateDriver creates a new driver profile
func (s *Store) CreateDriver(driver *DriverProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.emailIndex[driver.Contact.Email]; exists {
		return fmt.Errorf("driver with email %s already exists", driver.Contact.Email)
	}
	s.drivers[driver.ID] = driver
	s.emailIndex[driver.Contact.Email] = driver.ID
	s.auditLogs[driver.ID] = []AuditLog{}
	return nil
}

// GetDriver retrieves a driver profile by ID
func (s *Store) GetDriver(id string) (*DriverProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	driver, exists := s.drivers[id]
	if !exists {
		return nil, fmt.Errorf("driver %s not found", id)
	}
	return driver, nil
}

// UpdateDriver updates an existing driver profile
func (s *Store) UpdateDriver(driver *DriverProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.drivers[driver.ID]; !exists {
		return fmt.Errorf("driver %s not found", driver.ID)
	}
	driver.UpdatedAt = time.Now().UTC()
	s.drivers[driver.ID] = driver
	return nil
}

// GetPendingDrivers returns all drivers in ADMIN_REVIEW status
func (s *Store) GetPendingDrivers() []*DriverProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*DriverProfile
	for _, d := range s.drivers {
		if d.CurrentStatus == StatusAdminReview {
			result = append(result, d)
		}
	}
	return result
}

// GetDriverByExternalRef finds a driver by background check external ref
func (s *Store) GetDriverByExternalRef(refID string) (*DriverProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.drivers {
		if d.BackgroundCheck != nil && d.BackgroundCheck.ExternalRefID == refID {
			return d, nil
		}
	}
	return nil, fmt.Errorf("driver with external ref %s not found", refID)
}

// AddAuditLog appends an audit log entry
func (s *Store) AddAuditLog(entry AuditLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auditLogs[entry.DriverID] = append(s.auditLogs[entry.DriverID], entry)
}

// GetAuditLogs returns all audit logs for a driver
func (s *Store) GetAuditLogs(driverID string) []AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auditLogs[driverID]
}

// TotalDrivers returns the count of all drivers
func (s *Store) TotalDrivers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.drivers)
}

// CountByStatus returns a map of status counts
func (s *Store) CountByStatus() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	counts := make(map[string]int)
	for _, d := range s.drivers {
		counts[string(d.CurrentStatus)]++
	}
	return counts
}

// ============================================================
// Service Layer
// ============================================================

// OnboardingService handles all business logic
type OnboardingService struct {
	store     *Store
	logger    *log.Logger
	startTime time.Time
}

// NewOnboardingService creates a new OnboardingService
func NewOnboardingService(store *Store, logger *log.Logger) *OnboardingService {
	return &OnboardingService{
		store:     store,
		logger:    logger,
		startTime: time.Now().UTC(),
	}
}

// buildInitialSteps creates the ordered list of onboarding steps
func buildInitialSteps() []OnboardingStep {
	statuses := []OnboardingStatus{
		StatusRegistration,
		StatusEmailVerification,
		StatusDocumentUpload,
		StatusKYCVerification,
		StatusPScheinVerification,
		StatusVehicleRegistration,
		StatusBackgroundCheck,
		StatusAdminReview,
	}
	steps := make([]OnboardingStep, 0, len(statuses))
	for i, s := range statuses {
		step := OnboardingStep{
			Step:   s,
			Status: "PENDING",
		}
		if i == 0 {
			now := time.Now().UTC()
			step.Status = "IN_PROGRESS"
			step.StartedAt = &now
		}
		steps = append(steps, step)
	}
	return steps
}

// computeRetentionPolicy sets GDPR data retention based on status
func computeRetentionPolicy(status OnboardingStatus) DataRetentionPolicy {
	now := time.Now().UTC()
	switch status {
	case StatusApproved:
		return DataRetentionPolicy{
			RetentionUntil:  now.Add(GDPRRetentionApproved),
			RetentionReason: "Active driver data - legal obligation",
			AutoDeleteAt:    now.Add(GDPRRetentionApproved),
			LegalBasis:      "Art. 6(1)(b) DSGVO - contractual necessity",
		}
	case StatusRejected:
		return DataRetentionPolicy{
			RetentionUntil:  now.Add(GDPRRetentionRejected),
			RetentionReason: "Rejected application - fraud prevention",
			AutoDeleteAt:    now.Add(GDPRRetentionRejected),
			LegalBasis:      "Art. 6(1)(f) DSGVO - legitimate interest",
		}
	default:
		return DataRetentionPolicy{
			RetentionUntil:  now.Add(GDPRRetentionPending),
			RetentionReason: "Pending application",
			AutoDeleteAt:    now.Add(GDPRRetentionPending),
			LegalBasis:      "Art. 6(1)(a) DSGVO - consent",
		}
	}
}

// validatePScheinNumber validates a German P-Schein license number format
// German P-Schein numbers follow regional authority patterns
func validatePScheinNumber(licenseNumber string) bool {
	if licenseNumber == "" {
		return false
	}
	// P-Schein numbers vary by Bundesland authority
	// General pattern: 2-3 letter authority code + digits
	pattern := regexp.MustCompile(`^[A-Z]{1,4}[-/]?\d{4,10}[-/]?[A-Z0-9]{0,4}$`)
	return pattern.MatchString(strings.ToUpper(licenseNumber))
}

// validateGermanLicensePlate validates a German vehicle license plate
func validateGermanLicensePlate(plate string) bool {
	if plate == "" {
		return false
	}
	// German license plates: 1-3 letter district + 1-2 letter + 1-4 digit
	pattern := regexp.MustCompile(`^[A-ZÃÃÃ]{1,3}[-\s]?[A-Z]{1,2}[-\s]?\d{1,4}[EH]?$`)
	return pattern.MatchString(strings.ToUpper(strings.ReplaceAll(plate, " ", "")))
}

// validateGermanPostalCode validates a German postal code (PLZ)
func validateGermanPostalCode(plz string) bool {
	pattern := regexp.MustCompile(`^\d{5}$`)
	return pattern.MatchString(plz)
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	pattern := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return pattern.MatchString(email)
}

// isValidPhone validates a phone number (German and international)
func isValidPhone(phone string) bool {
	// Allow +49... or 0... format
	pattern := regexp.MustCompile(`^(\+49|0049|0)[1-9]\d{6,14}$`)
	clean := strings.ReplaceAll(strings.ReplaceAll(phone, " ", ""), "-", "")
	return pattern.MatchString(clean)
}

// advanceStep marks the current step as completed and moves to the next
func advanceStep(driver *DriverProfile, nextStatus OnboardingStatus) {
	now := time.Now().UTC()
	for i := range driver.OnboardingSteps {
		if driver.OnboardingSteps[i].Step == driver.CurrentStatus {
			driver.OnboardingSteps[i].Status = "COMPLETED"
			driver.OnboardingSteps[i].CompletedAt = &now
		}
		if driver.OnboardingSteps[i].Step == nextStatus {
			driver.OnboardingSteps[i].Status = "IN_PROGRESS"
			driver.OnboardingSteps[i].StartedAt = &now
		}
	}
	driver.CurrentStatus = nextStatus
	driver.UpdatedAt = now
}

// calculateProgress computes the onboarding progress percentage
func calculateProgress(steps []OnboardingStep) (int, float64) {
	total := len(steps)
	completed := 0
	for _, s := range steps {
		if s.Status == "COMPLETED" {
			completed++
		}
	}
	if total == 0 {
		return 0, 0.0
	}
	return completed, float64(completed) / float64(total) * 100.0
}

// StartOnboarding creates a new driver onboarding profile
func (svc *OnboardingService) StartOnboarding(req StartOnboardingRequest, clientIP string) (*DriverProfile, error) {
	// Validate required fields
	if req.FirstName == "" || req.LastName == "" {
		return nil, fmt.Errorf("first_name and last_name are required")
	}
	if !isValidEmail(req.Email) {
		return nil, fmt.Errorf("invalid email address")
	}
	if req.Phone != "" && !isValidPhone(req.Phone) {
		return nil, fmt.Errorf("invalid phone number format; expected German format (+49...)")
	}
	if req.Country == "DE" && req.PostalCode != "" && !validateGermanPostalCode(req.PostalCode) {
		return nil, fmt.Errorf("invalid German postal code (PLZ)")
	}
	if !req.DataProcessing {
		return nil, fmt.Errorf("data_processing consent is required under DSGVO")
	}
	if req.DateOfBirth == "" {
		return nil, fmt.Errorf("date_of_birth is required")
	}
	// Validate age (must be >= 21 for German P-Schein holders)
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return nil, fmt.Errorf("date_of_birth must be in YYYY-MM-DD format")
	}
	age := int(time.Since(dob).Hours() / 24 / 365)
	if age < 21 {
		return nil, fmt.Errorf("driver must be at least 21 years old (Mindestalter fÃ¼r P-Schein)")
	}
	if req.ConsentVersion == "" {
		req.ConsentVersion = "1.0"
	}
	if req.Country == "" {
		req.Country = "DE"
	}

	now := time.Now().UTC()
	driverID := uuid.New().String()

	driver := &DriverProfile{
		ID: driverID,
		Personal: PersonalInfo{
			FirstName:   req.FirstName,
			LastName:    req.LastName,
			DateOfBirth: req.DateOfBirth,
			Nationality: req.Nationality,
			Address: Address{
				Street:      req.Street,
				HouseNumber: req.HouseNumber,
				City:        req.City,
				PostalCode:  req.PostalCode,
				State:       req.State,
				Country:     req.Country,
			},
		},
		Contact: ContactInfo{
			Email:         req.Email,
			Phone:         req.Phone,
			EmailVerified: false,
			PhoneVerified: false,
		},
		GDPRConsent: GDPRConsent{
			DataProcessing:    req.DataProcessing,
			MarketingEmails:   req.MarketingEmails,
			ThirdPartySharing: req.ThirdPartySharing,
			ConsentTimestamp:  now,
			ConsentIP:         clientIP,
			ConsentVersion:    req.ConsentVersion,
		},
		DataRetention:   computeRetentionPolicy(StatusRegistration),
		CurrentStatus:   StatusRegistration,
		OnboardingSteps: buildInitialSteps(),
		Documents:       []Document{},
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := svc.store.CreateDriver(driver); err != nil {
		return nil, err
	}

	// Simulate email verification trigger
	svc.simulateEmailVerification(driver)

	svc.store.AddAuditLog(AuditLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Action:    AuditActionCreate,
		ActorID:   driverID,
		ActorType: "DRIVER",
		Details:   fmt.Sprintf("Onboarding started for %s %s", req.FirstName, req.LastName),
		IPAddress: clientIP,
		Timestamp: now,
		Metadata: map[string]string{
			"consent_version": req.ConsentVersion,
			"country":         req.Country,
		},
	})

	svc.logger.Printf("[INFO] New driver onboarding started: id=%s email=%s", driverID, req.Email)
	return driver, nil
}

// simulateEmailVerification simulates sending a verification email and auto-verifying
func (svc *OnboardingService) simulateEmailVerification(driver *DriverProfile) {
	go func() {
		// Simulate async email sending delay
		time.Sleep(time.Duration(rand.Intn(500)+100) * time.Millisecond)
		driver2, err := svc.store.GetDriver(driver.ID)
		if err != nil {
			return
		}
		now := time.Now().UTC()
		driver2.Contact.EmailVerified = true
		advanceStep(driver2, StatusDocumentUpload)
		// Mark email verification as completed too
		for i := range driver2.OnboardingSteps {
			if driver2.OnboardingSteps[i].Step == StatusEmailVerification {
				driver2.OnboardingSteps[i].Status = "COMPLETED"
				driver2.OnboardingSteps[i].CompletedAt = &now
				driver2.OnboardingSteps[i].Notes = "Email verified via simulation"
			}
		}
		_ = svc.store.UpdateDriver(driver2)
		svc.store.AddAuditLog(AuditLog{
			ID:        uuid.New().String(),
			DriverID:  driver.ID,
			Action:    AuditActionVerify,
			ActorID:   "SYSTEM",
			ActorType: "SYSTEM",
			Details:   "Email address verified",
			Timestamp: now,
		})
	}()
}

// GetOnboardingStatus returns the current onboarding status
func (svc *OnboardingService) GetOnboardingStatus(driverID string) (*OnboardingStatusResponse, error) {
	driver, err := svc.store.GetDriver(driverID)
	if err != nil {
		return nil, err
	}
	svc.store.AddAuditLog(AuditLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Action:    AuditActionDataAccess,
		ActorID:   driverID,
		ActorType: "DRIVER",
		Details:   "Status checked by driver",
		Timestamp: time.Now().UTC(),
	})
	completed, progress := calculateProgress(driver.OnboardingSteps)
	return &OnboardingStatusResponse{
		ID:              driver.ID,
		CurrentStatus:   driver.CurrentStatus,
		OnboardingSteps: driver.OnboardingSteps,
		SubmittedAt:     driver.SubmittedAt,
		ApprovedAt:      driver.ApprovedAt,
		RejectedAt:      driver.RejectedAt,
		RejectionReason: driver.RejectionReason,
		CompletedSteps:  completed,
		TotalSteps:      len(driver.OnboardingSteps),
		ProgressPercent: progress,
		CreatedAt:       driver.CreatedAt,
		UpdatedAt:       driver.UpdatedAt,
	}, nil
}

// UploadDocument adds a document to the driver's profile
func (svc *OnboardingService) UploadDocument(driverID string, req UploadDocumentRequest, clientIP string) (*Document, error) {
	driver, err := svc.store.GetDriver(driverID)
	if err != nil {
		return nil, err
	}

	allowedStatuses := map[OnboardingStatus]bool{
		StatusDocumentUpload:     true,
		StatusKYCVerification:    true,
		StatusPScheinVerification: true,
		StatusVehicleRegistration: true,
		StatusBackgroundCheck:    true,
		StatusAdminReview:        true,
	}
	if !allowedStatuses[driver.CurrentStatus] {
		return nil, fmt.Errorf("document upload not allowed in status %s", driver.CurrentStatus)
	}

	if req.Type == "" {
		return nil, fmt.Errorf("document type is required")
	}
	if req.FileName == "" {
		return nil, fmt.Errorf("file_name is required")
	}
	allowedMimes := map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"application/pdf": true,
		"image/webp":      true,
	}
	if req.MimeType != "" && !allowedMimes[req.MimeType] {
		return nil, fmt.Errorf("mime type %s not allowed; accepted: jpeg, png, pdf, webp", req.MimeType)
	}
	const maxFileSize = 20 * 1024 * 1024 // 20 MB
	if req.FileSize > maxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum of 20MB")
	}

	now := time.Now().UTC()
	doc := Document{
		ID:         uuid.New().String(),
		DriverID:   driverID,
		Type:       req.Type,
		Status:     DocumentStatusPending,
		FileName:   req.FileName,
		FileSize:   req.FileSize,
		MimeType:   req.MimeType,
		StorageURL: req.StorageURL,
		ExpiryDate: req.ExpiryDate,
		UploadedAt: now,
		Metadata:   req.Metadata,
	}

	driver.Documents = append(driver.Documents, doc)

	// Trigger next step based on document type
	svc.handleDocumentUploadTransition(driver, req.Type)

	if err := svc.store.UpdateDriver(driver); err != nil {
		return nil, err
	}

	svc.store.AddAuditLog(AuditLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Action:    AuditActionDocumentUpload,
		ActorID:   driverID,
		ActorType: "DRIVER",
		Details:   fmt.Sprintf("Document uploaded: type=%s file=%s", req.Type, req.FileName),
		IPAddress: clientIP,
		Timestamp: now,
		Metadata: map[string]string{
			"document_id":   doc.ID,
			"document_type": string(req.Type),
		},
	})

	svc.logger.Printf("[INFO] Document uploaded: driver=%s type=%s doc_id=%s", driverID, req.Type, doc.ID)
	return &doc, nil
}

// handleDocumentUploadTransition moves the workflow based on what was uploaded
func (svc *OnboardingService) handleDocumentUploadTransition(driver *DriverProfile, docType DocumentType) {
	switch docType {
	case DocumentTypeNationalID, DocumentTypePassport:
		if driver.CurrentStatus == StatusDocumentUpload {
			advanceStep(driver, StatusKYCVerification)
			// Simulate async KYC check
			go svc.simulateKYCVerification(driver.ID)
		}
	case DocumentTypePSchein:
		if driver.CurrentStatus == StatusKYCVerification || driver.CurrentStatus == StatusPScheinVerification {
			advanceStep(driver, StatusPScheinVerification)
			go svc.simulatePScheinVerification(driver.ID)
		}
	case DocumentTypeInsurance:
		// Insurance doc just updated, no transition
	case DocumentTypeBackgroundCheck:
		if driver.CurrentStatus == StatusBackgroundCheck {
			advanceStep(driver, StatusAdminReview)
		}
	}
}

// simulateKYCVerification simulates an external KYC check
func (svc *OnboardingService) simulateKYCVerification(driverID string) {
	time.Sleep(time.Duration(rand.Intn(2000)+500) * time.Millisecond)
	driver, err := svc.store.GetDriver(driverID)
	if err != nil {
		return
	}
	if driver.CurrentStatus != StatusKYCVerification {
		return
	}
	now := time.Now().UTC()
	// Mark KYC documents as verified
	for i := range driver.Documents {
		if driver.Documents[i].Type == DocumentTypeNationalID || driver.Documents[i].Type == DocumentTypePassport {
			driver.Documents[i].Status = DocumentStatusVerified
			driver.Documents[i].VerifiedAt = &now
			driver.Documents[i].VerifiedBy = "KYC_SYSTEM"
		}
	}
	advanceStep(driver, StatusPScheinVerification)
	_ = svc.store.UpdateDriver(driver)
	svc.store.AddAuditLog(AuditLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Action:    AuditActionVerify,
		ActorID:   "KYC_SYSTEM",
		ActorType: "SYSTEM",
		Details:   "KYC verification completed successfully",
		Timestamp: now,
	})
	svc.logger.Printf("[INFO] KYC verified: driver=%s", driverID)
}

// simulatePScheinVerification simulates P-Schein validation
func (svc *OnboardingService) simulatePScheinVerification(driverID string) {
	time.Sleep(time.Duration(rand.Intn(3000)+1000) * time.Millisecond)
	driver, err := svc.store.GetDriver(driverID)
	if err != nil {
		return
	}
	if driver.CurrentStatus != StatusPScheinVerification {
		return
	}
	now := time.Now().UTC()
	for i := range driver.Documents {
		if driver.Documents[i].Type == DocumentTypePSchein {
			driver.Documents[i].Status = DocumentStatusVerified
			driver.Documents[i].VerifiedAt = &now
			driver.Documents[i].VerifiedBy = "P_SCHEIN_AUTHORITY"
		}
	}
	if driver.PSchein == nil {
		expiry := now.Add(5 * 365 * 24 * time.Hour)
		driver.PSchein = &PScheinInfo{
			LicenseNumber:    fmt.Sprintf("B-%d", rand.Intn(900000)+100000),
			IssuingAuthority: "Landratsamt MÃ¼nchen",
			IssuedAt:         now.Add(-2 * 365 * 24 * time.Hour),
			ExpiryDate:       expiry,
			VehicleClasses:   []string{"B", "B96"},
			IsValid:          true,
			VerifiedAt:       &now,
		}
	} else {
		driver.PSchein.IsValid = true
		driver.PSchein.VerifiedAt = &now
	}
	advanceStep(driver, StatusVehicleRegistration)
	_ = svc.store.UpdateDriver(driver)
	svc.store.AddAuditLog(AuditLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Action:    AuditActionVerify,
		ActorID:   "P_SCHEIN_SYSTEM",
		ActorType: "SYSTEM",
		Details:   "P-Schein (PersonenbefÃ¶rderungsschein) verified",
		Timestamp: now,
	})
	svc.logger.Printf("[INFO] P-Schein verified: driver=%s", driverID)
}

// ListDocuments returns all documents for a driver
func (svc *OnboardingService) ListDocuments(driverID string) ([]Document, error) {
	driver, err := svc.store.GetDriver(driverID)
	if err != nil {
		return nil, err
	}
	// Scrub storage URLs for security
	docs := make([]Document, len(driver.Documents))
	copy(docs, driver.Documents)
	for i := range docs {
		docs[i].StorageURL = ""
	}
	return docs, nil
}

// RegisterVehicle registers a vehicle for a driver
func (svc *OnboardingService) RegisterVehicle(driverID string, req RegisterVehicleRequest, clientIP string) (*Vehicle, error) {
	driver, err := svc.store.GetDriver(driverID)
	if err != nil {
		return nil, err
	}

	allowedStatuses := map[OnboardingStatus]bool{
		StatusVehicleRegistration: true,
		StatusBackgroundCheck:     true,
		StatusAdminReview:         true,
	}
	if !allowedStatuses[driver.CurrentStatus] {
		return nil, fmt.Errorf("vehicle registration not allowed in status %s; current status: %s",
			StatusVehicleRegistration, driver.CurrentStatus)
	}

	if req.Make == "" || req.Model == "" {
		return nil, fmt.Errorf("vehicle make and model are required")
	}
	if req.LicensePlate == "" {
		return nil, fmt.Errorf("license plate is required")
	}
	if !validateGermanLicensePlate(req.LicensePlate) {
		return nil, fmt.Errorf("invalid German license plate format")
	}
	if req.Year < 2000 || req.Year > time.Now().Year()+1 {
		return nil, fmt.Errorf("vehicle year must be between 2000 and %d", time.Now().Year()+1)
	}
	if req.Seats < 2 || req.Seats > 9 {
		return nil, fmt.Errorf("vehicle must have between 2 and 9 seats")
	}
	if req.TUVExpiry.Before(time.Now()) {
		return nil, fmt.Errorf("TÃV certificate is expired")
	}
	if req.InsuranceExpiry.Before(time.Now()) {
		return nil, fmt.Errorf("insurance is expired")
	}
	if req.Category == "" {
		req.Category = VehicleCategoryStandard
	}

	now := time.Now().UTC()
	vehicle := &Vehicle{
		ID:                uuid.New().String(),
		DriverID:          driverID,
		Make:              req.Make,
		Model:             req.Model,
		Year:              req.Year,
		Color:             req.Color,
		LicensePlate:      strings.ToUpper(req.LicensePlate),
		VIN:               req.VIN,
		Category:          req.Category,
		Seats:             req.Seats,
		InsurancePolicyNo: req.InsurancePolicyNo,
		InsuranceExpiry:   req.InsuranceExpiry,
		TUVExpiry:         req.TUVExpiry,
		IsAccessible:      req.IsAccessible,
		RegisteredAt:      now,
		Approved:          false,
	}

	driver.Vehicle = vehicle

	if driver.CurrentStatus == StatusVehicleRegistration {
		advanceStep(driver, StatusBackgroundCheck)
		go svc.initiateBackgroundCheck(driver.ID)
	}

	if err := svc.store.UpdateDriver(driver); err != nil {
		return nil, err
	}

	svc.store.AddAuditLog(AuditLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Action:    AuditActionVehicleRegister,
		ActorID:   driverID,
		ActorType: "DRIVER",
		Details:   fmt.Sprintf("Vehicle registered: %s %s %d (%s)", req.Make, req.Model, req.Year, req.LicensePlate),
		IPAddress: clientIP,
		Timestamp: now,
		Metadata: map[string]string{
			"vehicle_id":    vehicle.ID,
			"license_plate": vehicle.LicensePlate,
			"category":      string(vehicle.Category),
		},
	})

	svc.logger.Printf("[INFO] Vehicle registered: driver=%s vehicle=%s plate=%s", driverID, vehicle.ID, vehicle.LicensePlate)
	return vehicle, nil
}

// initiateBackgroundCheck starts an async background check
func (svc *OnboardingService) initiateBackgroundCheck(driverID string) {
	time.Sleep(time.Duration(rand.Intn(1000)+200) * time.Millisecond)
	driver, err := svc.store.GetDriver(driverID)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	extRefID := fmt.Sprintf("BGC-%s-%d", driverID[:8], rand.Intn(999999))
	check := &BackgroundCheck{
		ID:            uuid.New().String(),
		DriverID:      driverID,
		Provider:      "Bundeszentralregister-Integration",
		Status:        "PENDING",
		RequestedAt:   now,
		ExternalRefID: extRefID,
	}
	driver.BackgroundCheck = check
	_ = svc.store.UpdateDriver(driver)
	svc.store.AddAuditLog(AuditLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Action:    AuditActionVerify,
		ActorID:   "BGC_SYSTEM",
		ActorType: "SYSTEM",
		Details:   fmt.Sprintf("Background check initiated: ref=%s", extRefID),
		Timestamp: now,
	})
	svc.logger.Printf("[INFO] Background check initiated: driver=%s ref=%s", driverID, extRefID)

	// Simulate background check completing after a delay
	go func() {
		time.Sleep(time.Duration(rand.Intn(5000)+2000) * time.Millisecond)
		svc.completeBackgroundCheck(driverID, extRefID, "CLEAR")
	}()
}

// completeBackgroundCheck marks the background check as done
func (svc *OnboardingService) completeBackgroundCheck(driverID, extRefID, result string) {
	driver, err := svc.store.GetDriver(driverID)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	if driver.BackgroundCheck != nil && driver.BackgroundCheck.ExternalRefID == extRefID {
		driver.BackgroundCheck.Status = "COMPLETED"
		driver.BackgroundCheck.CompletedAt = &now
		driver.BackgroundCheck.Result = result
		validUntil := now.Add(365 * 24 * time.Hour)
		driver.BackgroundCheck.ValidUntil = &validUntil
	}
	if result == "CLEAR" && driver.CurrentStatus == StatusBackgroundCheck {
		advanceStep(driver, StatusAdminReview)
	}
	_ = svc.store.UpdateDriver(driver)
	svc.store.AddAuditLog(AuditLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Action:    AuditActionVerify,
		ActorID:   "BGC_SYSTEM",
		ActorType: "SYSTEM",
		Details:   fmt.Sprintf("Background check completed: result=%s ref=%s", result, extRefID),
		Timestamp: now,
	})
	svc.logger.Printf("[INFO] Background check completed: driver=%s result=%s", driverID, result)
}

// SubmitForReview submits the application for admin review
func (svc *OnboardingService) SubmitForReview(driverID string, clientIP string) (*DriverProfile, error) {
	driver, err := svc.store.GetDriver(driverID)
	if err != nil {
		return nil, err
	}

	// Validate all required steps are complete enough to submit
	allowedStatuses := map[OnboardingStatus]bool{
		StatusDocumentUpload:     true,
		StatusKYCVerification:    true,
		StatusPScheinVerification: true,
		StatusVehicleRegistration: true,
		StatusBackgroundCheck:    true,
		StatusAdminReview:        true,
	}
	if !allowedStatuses[driver.CurrentStatus] {
		return nil, fmt.Errorf("cannot submit from status %s", driver.CurrentStatus)
	}
	if len(driver.Documents) == 0 {
		return nil, fmt.Errorf("at least one document must be uploaded before submitting")
	}

	now := time.Now().UTC()
	driver.SubmittedAt = &now

	if driver.CurrentStatus != StatusAdminReview {
		advanceStep(driver, StatusAdminReview)
	}

	if err := svc.store.UpdateDriver(driver); err != nil {
		return nil, err
	}

	svc.store.AddAuditLog(AuditLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Action:    AuditActionSubmit,
		ActorID:   driverID,
		ActorType: "DRIVER",
		Details:   "Application submitted for admin review",
		IPAddress: clientIP,
		Timestamp: now,
	})

	svc.logger.Printf("[INFO] Application submitted for review: driver=%s", driverID)
	return driver, nil
}

// AdminApprove approves a driver application
func (svc *OnboardingService) AdminApprove(driverID string, req AdminApproveRequest, clientIP string) (*DriverProfile, error) {
	driver, err := svc.store.GetDriver(driverID)
	if err != nil {
		return nil, err
	}
	if driver.CurrentStatus != StatusAdminReview {
		return nil, fmt.Errorf("driver is not in ADMIN_REVIEW status (current: %s)", driver.CurrentStatus)
	}
	if req.AdminID == "" {
		return nil, fmt.Errorf("admin_id is required")
	}

	now := time.Now().UTC()
	driver.CurrentStatus = StatusApproved
	driver.ApprovedAt = &now
	driver.ApprovedBy = req.AdminID
	driver.AdminNotes = req.Notes
	driver.UpdatedAt = now
	// Mark all remaining steps as completed
	for i := range driver.OnboardingSteps {
		if driver.OnboardingSteps[i].Status != "COMPLETED" {
			driver.OnboardingSteps[i].Status = "COMPLETED"
			driver.OnboardingSteps[i].CompletedAt = &now
		}
	}
	// Update GDPR retention for approved driver
	driver.DataRetention = computeRetentionPolicy(StatusApproved)
	// Approve vehicle
	if driver.Vehicle != nil {
		driver.Vehicle.Approved = true
	}

	if err := svc.store.UpdateDriver(driver); err != nil {
		return nil, err
	}

	svc.store.AddAuditLog(AuditLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Action:    AuditActionApprove,
		ActorID:   req.AdminID,
		ActorType: "ADMIN",
		Details:   fmt.Sprintf("Driver application approved by admin %s", req.AdminID),
		IPAddress: clientIP,
		Timestamp: now,
		Metadata: map[string]string{
			"admin_notes": req.Notes,
		},
	})

	svc.logger.Printf("[INFO] Driver approved: driver=%s admin=%s", driverID, req.AdminID)
	return driver, nil
}

// AdminReject rejects a driver application
func (svc *OnboardingService) AdminReject(driverID string, req AdminRejectRequest, clientIP string) (*DriverProfile, error) {
	driver, err := svc.store.GetDriver(driverID)
	if err != nil {
		return nil, err
	}
	if driver.CurrentStatus != StatusAdminReview {
		return nil, fmt.Errorf("driver is not in ADMIN_REVIEW status (current: %s)", driver.CurrentStatus)
	}
	if req.Reason == "" {
		return nil, fmt.Errorf("rejection reason is required")
	}
	if req.AdminID == "" {
		return nil, fmt.Errorf("admin_id is required")
	}

	now := time.Now().UTC()
	driver.CurrentStatus = StatusRejected
	driver.RejectedAt = &now
	driver.RejectedBy = req.AdminID
	driver.RejectionReason = req.Reason
	driver.AdminNotes = req.Notes
	driver.UpdatedAt = now
	// Mark current step as failed
	for i := range driver.OnboardingSteps {
		if driver.OnboardingSteps[i].Step == StatusAdminReview {
			driver.OnboardingSteps[i].Status = "FAILED"
			driver.OnboardingSteps[i].CompletedAt = &now
			driver.OnboardingSteps[i].Notes = req.Reason
		}
	}
	driver.DataRetention = computeRetentionPolicy(StatusRejected)

	if err := svc.store.UpdateDriver(driver); err != nil {
		return nil, err
	}

	svc.store.AddAuditLog(AuditLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Action:    AuditActionReject,
		ActorID:   req.AdminID,
		ActorType: "ADMIN",
		Details:   fmt.Sprintf("Driver application rejected by admin %s: %s", req.AdminID, req.Reason),
		IPAddress: clientIP,
		Timestamp: now,
		Metadata: map[string]string{
			"rejection_reason": req.Reason,
			"admin_notes":      req.Notes,
		},
	})

	svc.logger.Printf("[INFO] Driver rejected: driver=%s admin=%s reason=%s", driverID, req.AdminID, req.Reason)
	return driver, nil
}

// GetPendingApplications returns all applications awaiting admin review
func (svc *OnboardingService) GetPendingApplications() []PendingApplicationSummary {
	drivers := svc.store.GetPendingDrivers()
	summaries := make([]PendingApplicationSummary, 0, len(drivers))
	for _, d := range drivers {
		summaries = append(summaries, PendingApplicationSummary{
			ID:            d.ID,
			FullName:      d.Personal.FirstName + " " + d.Personal.LastName,
			Email:         d.Contact.Email,
			CurrentStatus: d.CurrentStatus,
			SubmittedAt:   d.SubmittedAt,
			CreatedAt:     d.CreatedAt,
			DocumentCount: len(d.Documents),
			HasVehicle:    d.Vehicle != nil,
			HasPSchein:    d.PSchein != nil && d.PSchein.IsValid,
		})
	}
	return summaries
}

// ProcessWebhook handles external verification webhook events
func (svc *OnboardingService) ProcessWebhook(payload WebhookVerificationPayload, clientIP string) error {
	svc.logger.Printf("[INFO] Webhook received: event=%s ref=%s driver=%s status=%s",
		payload.EventType, payload.ExternalRefID, payload.DriverID, payload.Status)

	var driverID string
	if payload.DriverID != "" {
		driverID = payload.DriverID
	} else if payload.ExternalRefID != "" {
		driver, err := svc.store.GetDriverByExternalRef(payload.ExternalRefID)
		if err != nil {
			return fmt.Errorf("driver not found for external ref %s", payload.ExternalRefID)
		}
		driverID = driver.ID
	} else {
		return fmt.Errorf("either driver_id or external_ref_id must be provided")
	}

	driver, err := svc.store.GetDriver(driverID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	switch payload.EventType {
	case "background_check.completed":
		if driver.BackgroundCheck != nil {
			driver.BackgroundCheck.Status = "COMPLETED"
			driver.BackgroundCheck.CompletedAt = &now
			driver.BackgroundCheck.Result = payload.Result
			validUntil := now.Add(365 * 24 * time.Hour)
			driver.BackgroundCheck.ValidUntil = &validUntil
			if payload.Result == "CLEAR" && driver.CurrentStatus == StatusBackgroundCheck {
				advanceStep(driver, StatusAdminReview)
			}
		}
	case "kyc.verified":
		for i := range driver.Documents {
			if driver.Documents[i].Type == DocumentTypeNationalID || driver.Documents[i].Type == DocumentTypePassport {
				driver.Documents[i].Status = DocumentStatusVerified
				driver.Documents[i].VerifiedAt = &now
				driver.Documents[i].VerifiedBy = "WEBHOOK_KYC"
			}
		}
		if driver.CurrentStatus == StatusKYCVerification {
			advanceStep(driver, StatusPScheinVerification)
		}
	case "p_schein.verified":
		for i := range driver.Documents {
			if driver.Documents[i].Type == DocumentTypePSchein {
				driver.Documents[i].Status = DocumentStatusVerified
				driver.Documents[i].VerifiedAt = &now
				driver.Documents[i].VerifiedBy = "WEBHOOK_P_SCHEIN"
			}
		}
		if driver.PSchein != nil {
			driver.PSchein.IsValid = true
			driver.PSchein.VerifiedAt = &now
		}
		if driver.CurrentStatus == StatusPScheinVerification {
			advanceStep(driver, StatusVehicleRegistration)
		}
	case "document.rejected":
		for i := range driver.Documents {
			if driver.Documents[i].ID == payload.Details["document_id"] {
				driver.Documents[i].Status = DocumentStatusRejected
				driver.Documents[i].RejectedAt = &now
				driver.Documents[i].RejectReason = payload.Details["reason"]
			}
		}
	default:
		svc.logger.Printf("[WARN] Unknown webhook event type: %s", payload.EventType)
	}

	driver.UpdatedAt = now
	if err := svc.store.UpdateDriver(driver); err != nil {
		return err
	}

	svc.store.AddAuditLog(AuditLog{
		ID:        uuid.New().String(),
		DriverID:  driverID,
		Action:    AuditActionWebhook,
		ActorID:   "WEBHOOK",
		ActorType: "WEBHOOK",
		Details:   fmt.Sprintf("Webhook processed: event=%s status=%s", payload.EventType, payload.Status),
		IPAddress: clientIP,
		Timestamp: now,
		Metadata: payload.Details,
	})

	return nil
}

// ============================================================
// HTTP Handler Layer
// ============================================================

// Handler wraps the service and provides HTTP handlers
type Handler struct {
	service *OnboardingService
	logger  *log.Logger
}

// NewHandler creates a new Handler
func NewHandler(service *OnboardingService, logger *log.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// respondJSON writes a JSON response
func (h *Handler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Printf("[ERROR] Failed to encode response: %v", err)
	}
}

// respondSuccess sends a successful API response
func (h *Handler) respondSuccess(w http.ResponseWriter, statusCode int, message string, data interface{}, requestID string) {
	h.respondJSON(w, statusCode, APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	})
}

// respondError sends an error API response
func (h *Handler) respondError(w http.ResponseWriter, statusCode int, errMsg string, requestID string) {
	h.respondJSON(w, statusCode, APIResponse{
		Success:   false,
		Error:     errMsg,
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	})
}

// getRequestID extracts or generates a request ID
func getRequestID(r *http.Request) string {
	if rid := r.Header.Get("X-Request-ID"); rid != "" {
		return rid
	}
	return uuid.New().String()
}

// getClientIP extracts the client IP address
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	if r.RemoteAddr != "" {
		if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
			return r.RemoteAddr[:idx]
		}
		return r.RemoteAddr
	}
	return "unknown"
}

// HandleStartOnboarding handles POST /onboarding/start
func (h *Handler) HandleStartOnboarding(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	clientIP := getClientIP(r)
	var req StartOnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error(), requestID)
		return
	}
	driver, err := h.service.StartOnboarding(req, clientIP)
	if err != nil {
		statusCode := http.StatusBadRequest
		if strings.Contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
		}
		h.respondError(w, statusCode, err.Error(), requestID)
		return
	}
	h.respondSuccess(w, http.StatusCreated, "Onboarding started successfully", driver, requestID)
}

// HandleGetOnboardingStatus handles GET /onboarding/:id/status
func (h *Handler) HandleGetOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	vars := mux.Vars(r)
	driverID := vars["id"]
	status, err := h.service.GetOnboardingStatus(driverID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, err.Error(), requestID)
		return
	}
	h.respondSuccess(w, http.StatusOK, "Onboarding status retrieved", status, requestID)
}

// HandleUploadDocument handles POST /onboarding/:id/documents
func (h *Handler) HandleUploadDocument(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	clientIP := getClientIP(r)
	vars := mux.Vars(r)
	driverID := vars["id"]
	var req UploadDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error(), requestID)
		return
	}
	doc, err := h.service.UploadDocument(driverID, req, clientIP)
	if err != nil {
		statusCode := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		h.respondError(w, statusCode, err.Error(), requestID)
		return
	}
	h.respondSuccess(w, http.StatusCreated, "Document uploaded successfully", doc, requestID)
}

// HandleListDocuments handles GET /onboarding/:id/documents
func (h *Handler) HandleListDocuments(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	vars := mux.Vars(r)
	driverID := vars["id"]
	docs, err := h.service.ListDocuments(driverID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, err.Error(), requestID)
		return
	}
	h.respondSuccess(w, http.StatusOK, fmt.Sprintf("%d document(s) found", len(docs)), docs, requestID)
}

// HandleRegisterVehicle handles POST /onboarding/:id/vehicle
func (h *Handler) HandleRegisterVehicle(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	clientIP := getClientIP(r)
	vars := mux.Vars(r)
	driverID := vars["id"]
	var req RegisterVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error(), requestID)
		return
	}
	vehicle, err := h.service.RegisterVehicle(driverID, req, clientIP)
	if err != nil {
		statusCode := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		h.respondError(w, statusCode, err.Error(), requestID)
		return
	}
	h.respondSuccess(w, http.StatusCreated, "Vehicle registered successfully", vehicle, requestID)
}

// HandleGetVehicle handles GET /onboarding/:id/vehicle
func (h *Handler) HandleGetVehicle(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	vars := mux.Vars(r)
	driverID := vars["id"]
	driver, err := h.service.store.GetDriver(driverID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, err.Error(), requestID)
		return
	}
	if driver.Vehicle == nil {
		h.respondError(w, http.StatusNotFound, "no vehicle registered for this driver", requestID)
		return
	}
	h.respondSuccess(w, http.StatusOK, "Vehicle details retrieved", driver.Vehicle, requestID)
}

// HandleSubmitForReview handles POST /onboarding/:id/submit
func (h *Handler) HandleSubmitForReview(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	clientIP := getClientIP(r)
	vars := mux.Vars(r)
	driverID := vars["id"]
	driver, err := h.service.SubmitForReview(driverID, clientIP)
	if err != nil {
		statusCode := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		h.respondError(w, statusCode, err.Error(), requestID)
		return
	}
	h.respondSuccess(w, http.StatusOK, "Application submitted for review", driver, requestID)
}

// HandleAdminApprove handles POST /admin/onboarding/:id/approve
func (h *Handler) HandleAdminApprove(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	clientIP := getClientIP(r)
	vars := mux.Vars(r)
	driverID := vars["id"]
	var req AdminApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error(), requestID)
		return
	}
	driver, err := h.service.AdminApprove(driverID, req, clientIP)
	if err != nil {
		statusCode := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		h.respondError(w, statusCode, err.Error(), requestID)
		return
	}
	h.respondSuccess(w, http.StatusOK, "Driver application approved", driver, requestID)
}

// HandleAdminReject handles POST /admin/onboarding/:id/reject
func (h *Handler) HandleAdminReject(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	clientIP := getClientIP(r)
	vars := mux.Vars(r)
	driverID := vars["id"]
	var req AdminRejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error(), requestID)
		return
	}
	driver, err := h.service.AdminReject(driverID, req, clientIP)
	if err != nil {
		statusCode := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		h.respondError(w, statusCode, err.Error(), requestID)
		return
	}
	h.respondSuccess(w, http.StatusOK, "Driver application rejected", driver, requestID)
}

// HandleGetPendingApplications handles GET /admin/onboarding/pending
func (h *Handler) HandleGetPendingApplications(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	summaries := h.service.GetPendingApplications()
	h.respondSuccess(w, http.StatusOK, fmt.Sprintf("%d pending application(s)", len(summaries)), summaries, requestID)
}

// HandleWebhook handles POST /webhooks/verification
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	clientIP := getClientIP(r)
	// Basic webhook signature check placeholder
	wWebhookSecret := os.Getenv("WEBHOOK_SECRET")
	if wWebhookSecret != "" {
		sig := r.Header.Get("X-Webhook-Signature")
		if sig == "" {
			h.respondError(w, http.StatusUnauthorized, "missing webhook signature", requestID)
			return
		}
		// In production, verify HMAC-SHA256 of body against signature
	}
	var payload WebhookVerificationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid webhook payload: "+err.Error(), requestID)
		return
	}
	if err := h.service.ProcessWebhook(payload, clientIP); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error(), requestID)
		return
	}
	h.respondSuccess(w, http.StatusOK, "Webhook processed", nil, requestID)
}

// HandleHealth handles GET /health
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.service.startTime).Round(time.Second).String()
	counts := h.service.store.CountByStatus()
	total := h.service.store.TotalDrivers()
	checkResults := map[string]string{
		"store":   "ok",
		"memory":  "ok",
		"service": "ok",
	}
	stats := map[string]int{
		"total_drivers": total,
	}
	for k, v := range counts {
		stats["status_"+strings.ToLower(k)] = v
	}
	h.respondJSON(w, http.StatusOK, HealthResponse{
		Status:    "healthy",
		Service:   ServiceName,
		Version:   ServiceVersion,
		Timestamp: time.Now().UTC(),
		Uptime:    uptime,
		Checks:    checkResults,
		Stats:     stats,
	})
}

// ============================================================
// Middleware
// ============================================================

// loggingMiddleware logs all incoming HTTP requests
func loggingMiddleware(logger *log.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			duration := time.Since(start)
			logger.Printf("[HTTP] method=%s path=%s status=%d duration=%s ip=%s ua=%s",
				r.Method,
				r.URL.Path,
				rw.statusCode,
				duration.Round(time.Millisecond),
				getClientIP(r),
				r.UserAgent(),
			)
		})
	}
}

// securityHeadersMiddleware adds common security headers
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Service", ServiceName)
		w.Header().Set("X-Service-Version", ServiceVersion)
		next.ServeHTTP(w, r)
	})
}

// requestIDMiddleware ensures every request has an ID
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = uuid.New().String()
			r.Header.Set("X-Request-ID", rid)
		}
		w.Header().Set("X-Request-ID", rid)
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

// ============================================================
// Router Setup
// ============================================================

// NewRouter creates and configures the mux router
func NewRouter(handler *Handler, logger *log.Logger) *mux.Router {
	r := mux.NewRouter()

	// Apply middleware
	r.Use(requestIDMiddleware)
	r.Use(securityHeadersMiddleware)
	r.Use(loggingMiddleware(logger))

	// Health check
	r.HandleFunc("/health", handler.HandleHealth).Methods(http.MethodGet)

	// Driver onboarding routes
	r.HandleFunc("/onboarding/start", handler.HandleStartOnboarding).Methods(http.MethodPost)
	r.HandleFunc("/onboarding/{id}/status", handler.HandleGetOnboardingStatus).Methods(http.MethodGet)
	r.HandleFunc("/onboarding/{id}/documents", handler.HandleUploadDocument).Methods(http.MethodPost)
	r.HandleFunc("/onboarding/{id}/documents", handler.HandleListDocuments).Methods(http.MethodGet)
	r.HandleFunc("/onboarding/{id}/vehicle", handler.HandleRegisterVehicle).Methods(http.MethodPost)
	r.HandleFunc("/onboarding/{id}/vehicle", handler.HandleGetVehicle).Methods(http.MethodGet)
	r.HandleFunc("/onboarding/{id}/submit", handler.HandleSubmitForReview).Methods(http.MethodPost)

	// Admin routes
	r.HandleFunc("/admin/onboarding/pending", handler.HandleGetPendingApplications).Methods(http.MethodGet)
	r.HandleFunc("/admin/onboarding/{id}/approve", handler.HandleAdminApprove).Methods(http.MethodPost)
	r.HandleFunc("/admin/onboarding/{id}/reject", handler.HandleAdminReject).Methods(http.MethodPost)

	// Webhook route
	r.HandleFunc("/webhooks/verification", handler.HandleWebhook).Methods(http.MethodPost)

	// 404 handler
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(APIResponse{
			Success:   false,
			Error:     fmt.Sprintf("route not found: %s %s", r.Method, r.URL.Path),
			RequestID: getRequestID(r),
			Timestamp: time.Now().UTC(),
		})
	})

	// 405 handler
	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(APIResponse{
			Success:   false,
			Error:     fmt.Sprintf("method %s not allowed for %s", r.Method, r.URL.Path),
			RequestID: getRequestID(r),
			Timestamp: time.Now().UTC(),
		})
	})

	return r
}

// ============================================================
// Main Entry Point
// ============================================================

func main() {
	// Setup structured logger
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
	logger.Printf("[INFO] Starting %s v%s", ServiceName, ServiceVersion)

	// Determine port
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	// Initialize store
	store := NewStore()
	logger.Printf("[INFO] In-memory store initialized")

	// Initialize service
	service := NewOnboardingService(store, logger)
	logger.Printf("[INFO] Onboarding service initialized")

	// Initialize handler
	handler := NewHandler(service, logger)

	// Setup router
	router := NewRouter(handler, logger)
	logger.Printf("[INFO] Router configured with %d routes", 12)

	// Print route summary
	logger.Printf("[INFO] Available routes:")
	logger.Printf("[INFO]   GET  /health")
	logger.Printf("[INFO]   POST /onboarding/start")
	logger.Printf("[INFO]   GET  /onboarding/{id}/status")
	logger.Printf("[INFO]   POST /onboarding/{id}/documents")
	logger.Printf("[INFO]   GET  /onboarding/{id}/documents")
	logger.Printf("[INFO]   POST /onboarding/{id}/vehicle")
	logger.Printf("[INFO]   GET  /onboarding/{id}/vehicle")
	logger.Printf("[INFO]   POST /onboarding/{id}/submit")
	logger.Printf("[INFO]   GET  /admin/onboarding/pending")
	logger.Printf("[INFO]   POST /admin/onboarding/{id}/approve")
	logger.Printf("[INFO]   POST /admin/onboarding/{id}/reject")
	logger.Printf("[INFO]   POST /webhooks/verification")

	// Configure HTTP server with production-appropriate timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	logger.Printf("[INFO] Server listening on port %s", port)
	logger.Printf("[INFO] GDPR compliance mode: DSGVO (Datenschutz-Grundverordnung)")
	logger.Printf("[INFO] P-Schein validation: PersonenbefÃ¶rderungsschein (PBefG Â§47)")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("[FATAL] Server failed to start: %v", err)
	}
}

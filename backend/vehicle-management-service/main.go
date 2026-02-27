package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// ============================================================
// SECTION 1: Constants and Enums
// ============================================================

const (
	ServiceName    = "vehicle-management-service"
	ServiceVersion = "1.0.0"
	DefaultPort    = "8080"
	APIPrefix      = "/api/v1"

	MaxVehicleAge     = 50
	MinVehicleYear    = 1970
	MaxLicensePlateLen = 15
	MinVINLength      = 17
	MaxVINLength      = 17

	MaintenanceWarningDays = 30
	InsuranceWarningDays   = 30
	DocumentExpiryWarning  = 30
)

type VehicleStatus string

const (
	VehicleStatusActive    VehicleStatus = "active"
	VehicleStatusInactive  VehicleStatus = "inactive"
	VehicleStatusPending   VehicleStatus = "pending"
	VehicleStatusSuspended VehicleStatus = "suspended"
	VehicleStatusArchived  VehicleStatus = "archived"
)

type DocumentType string

const (
	DocumentTypeRegistration DocumentType = "registration_certificate"
	DocumentTypeInsurance    DocumentType = "insurance"
	DocumentTypeTUV          DocumentType = "tuv_inspection"
	DocumentTypeEmissions    DocumentType = "emissions_test"
	DocumentTypeCustom       DocumentType = "custom"
)

type DocumentStatus string

const (
	DocumentStatusPending  DocumentStatus = "pending"
	DocumentStatusVerified DocumentStatus = "verified"
	DocumentStatusRejected DocumentStatus = "rejected"
	DocumentStatusExpired  DocumentStatus = "expired"
)

type MaintenanceType string

const (
	MaintenanceTypeOilChange    MaintenanceType = "oil_change"
	MaintenanceTypeTireRotation MaintenanceType = "tire_rotation"
	MaintenanceTypeBrakeService MaintenanceType = "brake_service"
	MaintenanceTypeInspection   MaintenanceType = "inspection"
	MaintenanceTypeRepair       MaintenanceType = "repair"
	MaintenanceTypeOther        MaintenanceType = "other"
)

type AuditAction string

const (
	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
	AuditActionView   AuditAction = "view"
	AuditActionVerify AuditAction = "verify"
	AuditActionReject AuditAction = "reject"
)

// ============================================================
// SECTION 2: Data Models
// ============================================================

type Vehicle struct {
	ID              string            `json:"id"`
	OwnerID         string            `json:"owner_id"`
	Make            string            `json:"make"`
	Model           string            `json:"model"`
	Year            int               `json:"year"`
	Color           string            `json:"color"`
	LicensePlate    string            `json:"license_plate"`
	VIN             string            `json:"vin"`
	Status          VehicleStatus     `json:"status"`
	Mileage         int               `json:"mileage"`
	FuelType        string            `json:"fuel_type"`
	Transmission    string            `json:"transmission"`
	EngineSize      float64           `json:"engine_size"`
	SeatingCapacity int               `json:"seating_capacity"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	InsuranceValid  bool              `json:"insurance_valid"`
	InsuranceExpiry *time.Time        `json:"insurance_expiry,omitempty"`
	NextServiceDue  *time.Time        `json:"next_service_due,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	CreatedBy       string            `json:"created_by"`
	UpdatedBy       string            `json:"updated_by"`
}

type VehicleDocument struct {
	ID           string         `json:"id"`
	VehicleID    string         `json:"vehicle_id"`
	Type         DocumentType   `json:"type"`
	Status       DocumentStatus `json:"status"`
	Title        string         `json:"title"`
	Description  string         `json:"description,omitempty"`
	DocumentURL  string         `json:"document_url,omitempty"`
	DocumentHash string         `json:"document_hash,omitempty"`
	Issuer       string         `json:"issuer,omitempty"`
	IssuedAt     *time.Time     `json:"issued_at,omitempty"`
	ExpiresAt    *time.Time     `json:"expires_at,omitempty"`
	VerifiedAt   *time.Time     `json:"verified_at,omitempty"`
	VerifiedBy   string         `json:"verified_by,omitempty"`
	RejectedAt   *time.Time     `json:"rejected_at,omitempty"`
	RejectedBy   string         `json:"rejected_by,omitempty"`
	RejectionReason string      `json:"rejection_reason,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	CreatedBy    string         `json:"created_by"`
}

type MaintenanceRecord struct {
	ID              string          `json:"id"`
	VehicleID       string          `json:"vehicle_id"`
	Type            MaintenanceType `json:"type"`
	Title           string          `json:"title"`
	Description     string          `json:"description,omitempty"`
	MileageAtService int            `json:"mileage_at_service"`
	Cost            float64         `json:"cost"`
	Currency        string          `json:"currency"`
	ServiceProvider string          `json:"service_provider,omitempty"`
	TechnicianName  string          `json:"technician_name,omitempty"`
	PartsReplaced   []string        `json:"parts_replaced,omitempty"`
	NextServiceMileage int          `json:"next_service_mileage,omitempty"`
	NextServiceDate *time.Time      `json:"next_service_date,omitempty"`
	ServicedAt      time.Time       `json:"serviced_at"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	CreatedBy       string          `json:"created_by"`
}

type AuditLog struct {
	ID         string      `json:"id"`
	EntityType string      `json:"entity_type"`
	EntityID   string      `json:"entity_id"`
	Action     AuditAction `json:"action"`
	ActorID    string      `json:"actor_id"`
	ActorRole  string      `json:"actor_role"`
	IPAddress  string      `json:"ip_address"`
	UserAgent  string      `json:"user_agent"`
	OldData    interface{} `json:"old_data,omitempty"`
	NewData    interface{} `json:"new_data,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
}

type InsuranceInfo struct {
	VehicleID      string     `json:"vehicle_id"`
	PolicyNumber   string     `json:"policy_number"`
	Provider       string     `json:"provider"`
	CoverageType   string     `json:"coverage_type"`
	PremiumAmount  float64    `json:"premium_amount"`
	Currency       string     `json:"currency"`
	StartDate      time.Time  `json:"start_date"`
	EndDate        time.Time  `json:"end_date"`
	IsValid        bool       `json:"is_valid"`
	DaysUntilExpiry int       `json:"days_until_expiry"`
	WarningActive  bool       `json:"warning_active"`
}

// ============================================================
// SECTION 3: Request/Response Models
// ============================================================

type CreateVehicleRequest struct {
	OwnerID         string            `json:"owner_id"`
	Make            string            `json:"make"`
	Model           string            `json:"model"`
	Year            int               `json:"year"`
	Color           string            `json:"color"`
	LicensePlate    string            `json:"license_plate"`
	VIN             string            `json:"vin"`
	Mileage         int               `json:"mileage"`
	FuelType        string            `json:"fuel_type"`
	Transmission    string            `json:"transmission"`
	EngineSize      float64           `json:"engine_size"`
	SeatingCapacity int               `json:"seating_capacity"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
}

type UpdateVehicleRequest struct {
	Color           *string           `json:"color,omitempty"`
	LicensePlate    *string           `json:"license_plate,omitempty"`
	Mileage         *int              `json:"mileage,omitempty"`
	FuelType        *string           `json:"fuel_type,omitempty"`
	Transmission    *string           `json:"transmission,omitempty"`
	EngineSize      *float64          `json:"engine_size,omitempty"`
	SeatingCapacity *int              `json:"seating_capacity,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
}

type CreateDocumentRequest struct {
	Type        DocumentType      `json:"type"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	DocumentURL string            `json:"document_url,omitempty"`
	Issuer      string            `json:"issuer,omitempty"`
	IssuedAt    *time.Time        `json:"issued_at,omitempty"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type VerifyDocumentRequest struct {
	Status          DocumentStatus `json:"status"`
	RejectionReason string         `json:"rejection_reason,omitempty"`
}

type CreateMaintenanceRequest struct {
	Type               MaintenanceType `json:"type"`
	Title              string          `json:"title"`
	Description        string          `json:"description,omitempty"`
	MileageAtService   int             `json:"mileage_at_service"`
	Cost               float64         `json:"cost"`
	Currency           string          `json:"currency"`
	ServiceProvider    string          `json:"service_provider,omitempty"`
	TechnicianName     string          `json:"technician_name,omitempty"`
	PartsReplaced      []string        `json:"parts_replaced,omitempty"`
	NextServiceMileage int             `json:"next_service_mileage,omitempty"`
	NextServiceDate    *time.Time      `json:"next_service_date,omitempty"`
	ServicedAt         time.Time       `json:"serviced_at"`
}

type UpdateStatusRequest struct {
	Status VehicleStatus `json:"status"`
	Reason string        `json:"reason,omitempty"`
}

type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	RequestID string      `json:"request_id"`
	Timestamp time.Time   `json:"timestamp"`
}

type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

type VehicleStats struct {
	TotalVehicles      int            `json:"total_vehicles"`
	ByStatus           map[string]int `json:"by_status"`
	ByFuelType         map[string]int `json:"by_fuel_type"`
	InsuranceExpiring  int            `json:"insurance_expiring_soon"`
	MaintenanceDue     int            `json:"maintenance_due_soon"`
	PendingDocuments   int            `json:"pending_documents"`
	ExpiredDocuments   int            `json:"expired_documents"`
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// ============================================================
// SECTION 4: In-Memory Store
// ============================================================

type Store struct {
	mu           sync.RWMutex
	vehicles     map[string]*Vehicle
	documents    map[string]*VehicleDocument
	maintenance  map[string]*MaintenanceRecord
	auditLogs    []*AuditLog
	vinIndex     map[string]string
	plateIndex   map[string]string
}

func NewStore() *Store {
	return &Store{
		vehicles:    make(map[string]*Vehicle),
		documents:   make(map[string]*VehicleDocument),
		maintenance: make(map[string]*MaintenanceRecord),
		auditLogs:   make([]*AuditLog, 0),
		vinIndex:    make(map[string]string),
		plateIndex:  make(map[string]string),
	}
}

func (s *Store) CreateVehicle(v *Vehicle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.vinIndex[v.VIN]; exists {
		return fmt.Errorf("vehicle with VIN %s already exists", v.VIN)
	}
	if _, exists := s.plateIndex[v.LicensePlate]; exists {
		return fmt.Errorf("vehicle with license plate %s already exists", v.LicensePlate)
	}
	s.vehicles[v.ID] = v
	s.vinIndex[v.VIN] = v.ID
	s.plateIndex[v.LicensePlate] = v.ID
	return nil
}

func (s *Store) GetVehicle(id string) (*Vehicle, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, exists := s.vehicles[id]
	if !exists {
		return nil, fmt.Errorf("vehicle not found: %s", id)
	}
	copy := *v
	return &copy, nil
}

func (s *Store) UpdateVehicle(v *Vehicle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, exists := s.vehicles[v.ID]
	if !exists {
		return fmt.Errorf("vehicle not found: %s", v.ID)
	}
	if existing.LicensePlate != v.LicensePlate {
		if _, taken := s.plateIndex[v.LicensePlate]; taken {
			return fmt.Errorf("license plate %s already in use", v.LicensePlate)
		}
		delete(s.plateIndex, existing.LicensePlate)
		s.plateIndex[v.LicensePlate] = v.ID
	}
	s.vehicles[v.ID] = v
	return nil
}

func (s *Store) DeleteVehicle(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, exists := s.vehicles[id]
	if !exists {
		return fmt.Errorf("vehicle not found: %s", id)
	}
	delete(s.vinIndex, v.VIN)
	delete(s.plateIndex, v.LicensePlate)
	delete(s.vehicles, id)
	return nil
}

func (s *Store) ListVehicles(ownerID string, status VehicleStatus, page, pageSize int) ([]*Vehicle, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Vehicle
	for _, v := range s.vehicles {
		if ownerID != "" && v.OwnerID != ownerID {
			continue
		}
		if status != "" && v.Status != status {
			continue
		}
		copy := *v
		result = append(result, &copy)
	}
	total := len(result)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []*Vehicle{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return result[start:end], total
}

func (s *Store) CreateDocument(d *VehicleDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents[d.ID] = d
	return nil
}

func (s *Store) GetDocument(id string) (*VehicleDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, exists := s.documents[id]
	if !exists {
		return nil, fmt.Errorf("document not found: %s", id)
	}
	copy := *d
	return &copy, nil
}

func (s *Store) UpdateDocument(d *VehicleDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.documents[d.ID]; !exists {
		return fmt.Errorf("document not found: %s", d.ID)
	}
	s.documents[d.ID] = d
	return nil
}

func (s *Store) ListDocuments(vehicleID string) ([]*VehicleDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*VehicleDocument
	for _, d := range s.documents {
		if d.VehicleID == vehicleID {
			copy := *d
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (s *Store) DeleteDocument(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.documents[id]; !exists {
		return fmt.Errorf("document not found: %s", id)
	}
	delete(s.documents, id)
	return nil
}

func (s *Store) CreateMaintenanceRecord(m *MaintenanceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maintenance[m.ID] = m
	return nil
}

func (s *Store) GetMaintenanceRecord(id string) (*MaintenanceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, exists := s.maintenance[id]
	if !exists {
		return nil, fmt.Errorf("maintenance record not found: %s", id)
	}
	copy := *m
	return &copy, nil
}

func (s *Store) ListMaintenanceRecords(vehicleID string) ([]*MaintenanceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*MaintenanceRecord
	for _, m := range s.maintenance {
		if m.VehicleID == vehicleID {
			copy := *m
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (s *Store) UpdateMaintenanceRecord(m *MaintenanceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.maintenance[m.ID]; !exists {
		return fmt.Errorf("maintenance record not found: %s", m.ID)
	}
	s.maintenance[m.ID] = m
	return nil
}

func (s *Store) AddAuditLog(log *AuditLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auditLogs = append(s.auditLogs, log)
	if len(s.auditLogs) > 10000 {
		s.auditLogs = s.auditLogs[1000:]
	}
}

func (s *Store) GetAuditLogs(entityID string, limit int) []*AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*AuditLog
	for _, l := range s.auditLogs {
		if entityID == "" || l.EntityID == entityID {
			copy := *l
			result = append(result, &copy)
		}
	}
	if limit > 0 && len(result) > limit {
		return result[len(result)-limit:]
	}
	return result
}

func (s *Store) GetStats() VehicleStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := VehicleStats{
		ByStatus:   make(map[string]int),
		ByFuelType: make(map[string]int),
	}
	now := time.Now()
	warning := now.AddDate(0, 0, InsuranceWarningDays)
	for _, v := range s.vehicles {
		stats.TotalVehicles++
		stats.ByStatus[string(v.Status)]++
		if v.FuelType != "" {
			stats.ByFuelType[v.FuelType]++
		}
		if v.InsuranceExpiry != nil && v.InsuranceExpiry.Before(warning) && v.InsuranceExpiry.After(now) {
			stats.InsuranceExpiring++
		}
		if v.NextServiceDue != nil && v.NextServiceDue.Before(warning) && v.NextServiceDue.After(now) {
			stats.MaintenanceDue++
		}
	}
	for _, d := range s.documents {
		if d.Status == DocumentStatusPending {
			stats.PendingDocuments++
		}
		if d.ExpiresAt != nil && d.ExpiresAt.Before(now) {
			stats.ExpiredDocuments++
		}
	}
	return stats
}

// ============================================================
// SECTION 5: Service Layer
// ============================================================

type VehicleService struct {
	store  *Store
	logger *log.Logger
}

func NewVehicleService(store *Store, logger *log.Logger) *VehicleService {
	return &VehicleService{store: store, logger: logger}
}

func (svc *VehicleService) RegisterVehicle(req CreateVehicleRequest, actorID string) (*Vehicle, error) {
	if err := validateCreateVehicleRequest(req); err != nil {
		return nil, err
	}
	now := time.Now()
	v := &Vehicle{
		ID:              generateID(),
		OwnerID:         req.OwnerID,
		Make:            strings.TrimSpace(req.Make),
		Model:           strings.TrimSpace(req.Model),
		Year:            req.Year,
		Color:           strings.TrimSpace(req.Color),
		LicensePlate:    strings.ToUpper(strings.TrimSpace(req.LicensePlate)),
		VIN:             strings.ToUpper(strings.TrimSpace(req.VIN)),
		Status:          VehicleStatusPending,
		Mileage:         req.Mileage,
		FuelType:        strings.ToLower(strings.TrimSpace(req.FuelType)),
		Transmission:    strings.ToLower(strings.TrimSpace(req.Transmission)),
		EngineSize:      req.EngineSize,
		SeatingCapacity: req.SeatingCapacity,
		Metadata:        req.Metadata,
		Tags:            req.Tags,
		InsuranceValid:  false,
		CreatedAt:       now,
		UpdatedAt:       now,
		CreatedBy:       actorID,
		UpdatedBy:       actorID,
	}
	if v.SeatingCapacity == 0 {
		v.SeatingCapacity = 5
	}
	if v.FuelType == "" {
		v.FuelType = "gasoline"
	}
	if v.Transmission == "" {
		v.Transmission = "automatic"
	}
	if err := svc.store.CreateVehicle(v); err != nil {
		return nil, err
	}
	svc.addAuditLog("vehicle", v.ID, AuditActionCreate, actorID, "user", "", "", nil, v)
	svc.logger.Printf("[INFO] Vehicle registered: id=%s vin=%s plate=%s owner=%s", v.ID, v.VIN, v.LicensePlate, v.OwnerID)
	return v, nil
}

func (svc *VehicleService) GetVehicle(id string, actorID string) (*Vehicle, error) {
	v, err := svc.store.GetVehicle(id)
	if err != nil {
		return nil, err
	}
	svc.addAuditLog("vehicle", id, AuditActionView, actorID, "user", "", "", nil, nil)
	return v, nil
}

func (svc *VehicleService) UpdateVehicle(id string, req UpdateVehicleRequest, actorID string) (*Vehicle, error) {
	v, err := svc.store.GetVehicle(id)
	if err != nil {
		return nil, err
	}
	old := *v
	if req.Color != nil {
		v.Color = strings.TrimSpace(*req.Color)
	}
	if req.LicensePlate != nil {
		plate := strings.ToUpper(strings.TrimSpace(*req.LicensePlate))
		if len(plate) > MaxLicensePlateLen {
			return nil, fmt.Errorf("license plate too long")
		}
		v.LicensePlate = plate
	}
	if req.Mileage != nil {
		if *req.Mileage < v.Mileage {
			return nil, fmt.Errorf("mileage cannot decrease")
		}
		v.Mileage = *req.Mileage
	}
	if req.FuelType != nil {
		v.FuelType = strings.ToLower(strings.TrimSpace(*req.FuelType))
	}
	if req.Transmission != nil {
		v.Transmission = strings.ToLower(strings.TrimSpace(*req.Transmission))
	}
	if req.EngineSize != nil {
		v.EngineSize = *req.EngineSize
	}
	if req.SeatingCapacity != nil {
		v.SeatingCapacity = *req.SeatingCapacity
	}
	if req.Metadata != nil {
		if v.Metadata == nil {
			v.Metadata = make(map[string]string)
		}
		for k, val := range req.Metadata {
			v.Metadata[k] = val
		}
	}
	if req.Tags != nil {
		v.Tags = req.Tags
	}
	v.UpdatedAt = time.Now()
	v.UpdatedBy = actorID
	if err := svc.store.UpdateVehicle(v); err != nil {
		return nil, err
	}
	svc.addAuditLog("vehicle", id, AuditActionUpdate, actorID, "user", "", "", old, v)
	svc.logger.Printf("[INFO] Vehicle updated: id=%s actor=%s", id, actorID)
	return v, nil
}

func (svc *VehicleService) UpdateVehicleStatus(id string, req UpdateStatusRequest, actorID string) (*Vehicle, error) {
	if !isValidVehicleStatus(req.Status) {
		return nil, fmt.Errorf("invalid status: %s", req.Status)
	}
	v, err := svc.store.GetVehicle(id)
	if err != nil {
		return nil, err
	}
	old := *v
	v.Status = req.Status
	v.UpdatedAt = time.Now()
	v.UpdatedBy = actorID
	if err := svc.store.UpdateVehicle(v); err != nil {
		return nil, err
	}
	meta := map[string]string{"reason": req.Reason, "old_status": string(old.Status)}
	svc.addAuditLogWithMeta("vehicle", id, AuditActionUpdate, actorID, "user", "", "", old, v, meta)
	svc.logger.Printf("[INFO] Vehicle status updated: id=%s status=%s actor=%s", id, req.Status, actorID)
	return v, nil
}

func (svc *VehicleService) DeleteVehicle(id string, actorID string) error {
	v, err := svc.store.GetVehicle(id)
	if err != nil {
		return err
	}
	if v.Status == VehicleStatusActive {
		return fmt.Errorf("cannot delete an active vehicle; deactivate it first")
	}
	if err := svc.store.DeleteVehicle(id); err != nil {
		return err
	}
	svc.addAuditLog("vehicle", id, AuditActionDelete, actorID, "user", "", "", v, nil)
	svc.logger.Printf("[INFO] Vehicle deleted: id=%s actor=%s", id, actorID)
	return nil
}

func (svc *VehicleService) ListVehicles(ownerID string, status VehicleStatus, page, pageSize int) ([]*Vehicle, int) {
	return svc.store.ListVehicles(ownerID, status, page, pageSize)
}

func (svc *VehicleService) AddDocument(vehicleID string, req CreateDocumentRequest, actorID string) (*VehicleDocument, error) {
	if _, err := svc.store.GetVehicle(vehicleID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("document title is required")
	}
	if !isValidDocumentType(req.Type) {
		return nil, fmt.Errorf("invalid document type: %s", req.Type)
	}
	if req.ExpiresAt != nil && req.IssuedAt != nil && req.ExpiresAt.Before(*req.IssuedAt) {
		return nil, fmt.Errorf("expiry date cannot be before issue date")
	}
	now := time.Now()
	d := &VehicleDocument{
		ID:          generateID(),
		VehicleID:   vehicleID,
		Type:        req.Type,
		Status:      DocumentStatusPending,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		DocumentURL: req.DocumentURL,
		Issuer:      req.Issuer,
		IssuedAt:    req.IssuedAt,
		ExpiresAt:   req.ExpiresAt,
		Metadata:    req.Metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   actorID,
	}
	if err := svc.store.CreateDocument(d); err != nil {
		return nil, err
	}
	svc.addAuditLog("document", d.ID, AuditActionCreate, actorID, "user", "", "", nil, d)
	svc.logger.Printf("[INFO] Document added: id=%s vehicle=%s type=%s actor=%s", d.ID, vehicleID, req.Type, actorID)
	return d, nil
}

func (svc *VehicleService) VerifyDocument(documentID string, req VerifyDocumentRequest, actorID string) (*VehicleDocument, error) {
	d, err := svc.store.GetDocument(documentID)
	if err != nil {
		return nil, err
	}
	if d.Status != DocumentStatusPending {
		return nil, fmt.Errorf("document is not in pending status, current status: %s", d.Status)
	}
	old := *d
	now := time.Now()
	switch req.Status {
	case DocumentStatusVerified:
		d.Status = DocumentStatusVerified
		d.VerifiedAt = &now
		d.VerifiedBy = actorID
		if d.Type == DocumentTypeInsurance {
			svc.updateVehicleInsurance(d.VehicleID, d.ExpiresAt)
		}
		if err := svc.checkAndActivateVehicle(d.VehicleID); err != nil {
			svc.logger.Printf("[WARN] Could not auto-activate vehicle %s: %v", d.VehicleID, err)
		}
	case DocumentStatusRejected:
		if strings.TrimSpace(req.RejectionReason) == "" {
			return nil, fmt.Errorf("rejection reason is required when rejecting a document")
		}
		d.Status = DocumentStatusRejected
		d.RejectedAt = &now
		d.RejectedBy = actorID
		d.RejectionReason = req.RejectionReason
	default:
		return nil, fmt.Errorf("invalid verification status: %s", req.Status)
	}
	d.UpdatedAt = now
	if err := svc.store.UpdateDocument(d); err != nil {
		return nil, err
	}
	action := AuditActionVerify
	if req.Status == DocumentStatusRejected {
		action = AuditActionReject
	}
	svc.addAuditLog("document", documentID, action, actorID, "admin", "", "", old, d)
	svc.logger.Printf("[INFO] Document verified: id=%s status=%s actor=%s", documentID, req.Status, actorID)
	return d, nil
}

func (svc *VehicleService) GetDocument(id, vehicleID string) (*VehicleDocument, error) {
	d, err := svc.store.GetDocument(id)
	if err != nil {
		return nil, err
	}
	if vehicleID != "" && d.VehicleID != vehicleID {
		return nil, fmt.Errorf("document does not belong to vehicle")
	}
	return d, nil
}

func (svc *VehicleService) ListDocuments(vehicleID string) ([]*VehicleDocument, error) {
	if _, err := svc.store.GetVehicle(vehicleID); err != nil {
		return nil, err
	}
	return svc.store.ListDocuments(vehicleID)
}

func (svc *VehicleService) DeleteDocument(id, vehicleID, actorID string) error {
	d, err := svc.store.GetDocument(id)
	if err != nil {
		return err
	}
	if vehicleID != "" && d.VehicleID != vehicleID {
		return fmt.Errorf("document does not belong to vehicle")
	}
	if d.Status == DocumentStatusVerified {
		return fmt.Errorf("cannot delete a verified document")
	}
	if err := svc.store.DeleteDocument(id); err != nil {
		return err
	}
	svc.addAuditLog("document", id, AuditActionDelete, actorID, "user", "", "", d, nil)
	return nil
}

func (svc *VehicleService) AddMaintenanceRecord(vehicleID string, req CreateMaintenanceRequest, actorID string) (*MaintenanceRecord, error) {
	v, err := svc.store.GetVehicle(vehicleID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("maintenance title is required")
	}
	if !isValidMaintenanceType(req.Type) {
		return nil, fmt.Errorf("invalid maintenance type: %s", req.Type)
	}
	if req.MileageAtService < 0 {
		return nil, fmt.Errorf("mileage at service cannot be negative")
	}
	now := time.Now()
	m := &MaintenanceRecord{
		ID:               generateID(),
		VehicleID:        vehicleID,
		Type:             req.Type,
		Title:            strings.TrimSpace(req.Title),
		Description:      req.Description,
		MileageAtService: req.MileageAtService,
		Cost:             req.Cost,
		Currency:         req.Currency,
		ServiceProvider:  req.ServiceProvider,
		TechnicianName:   req.TechnicianName,
		PartsReplaced:    req.PartsReplaced,
		NextServiceMileage: req.NextServiceMileage,
		NextServiceDate:  req.NextServiceDate,
		ServicedAt:       req.ServicedAt,
		CreatedAt:        now,
		UpdatedAt:        now,
		CreatedBy:        actorID,
	}
	if m.Currency == "" {
		m.Currency = "EUR"
	}
	if req.ServicedAt.IsZero() {
		m.ServicedAt = now
	}
	if err := svc.store.CreateMaintenanceRecord(m); err != nil {
		return nil, err
	}
	if req.MileageAtService > v.Mileage {
		v.Mileage = req.MileageAtService
		v.UpdatedAt = now
		v.UpdatedBy = actorID
		_ = svc.store.UpdateVehicle(v)
	}
	if req.NextServiceDate != nil {
		v.NextServiceDue = req.NextServiceDate
		v.UpdatedAt = now
		v.UpdatedBy = actorID
		_ = svc.store.UpdateVehicle(v)
	}
	svc.addAuditLog("maintenance", m.ID, AuditActionCreate, actorID, "user", "", "", nil, m)
	svc.logger.Printf("[INFO] Maintenance record added: id=%s vehicle=%s type=%s", m.ID, vehicleID, req.Type)
	return m, nil
}

func (svc *VehicleService) GetMaintenanceRecord(id, vehicleID string) (*MaintenanceRecord, error) {
	m, err := svc.store.GetMaintenanceRecord(id)
	if err != nil {
		return nil, err
	}
	if vehicleID != "" && m.VehicleID != vehicleID {
		return nil, fmt.Errorf("maintenance record does not belong to vehicle")
	}
	return m, nil
}

func (svc *VehicleService) ListMaintenanceRecords(vehicleID string) ([]*MaintenanceRecord, error) {
	if _, err := svc.store.GetVehicle(vehicleID); err != nil {
		return nil, err
	}
	return svc.store.ListMaintenanceRecords(vehicleID)
}

func (svc *VehicleService) ValidateInsurance(vehicleID string) (*InsuranceInfo, error) {
	v, err := svc.store.GetVehicle(vehicleID)
	if err != nil {
		return nil, err
	}
	docs, err := svc.store.ListDocuments(vehicleID)
	if err != nil {
		return nil, err
	}
	info := &InsuranceInfo{
		VehicleID: vehicleID,
		IsValid:   false,
	}
	now := time.Now()
	for _, d := range docs {
		if d.Type == DocumentTypeInsurance && d.Status == DocumentStatusVerified {
			if d.ExpiresAt != nil && d.ExpiresAt.After(now) {
				info.IsValid = true
				info.Provider = d.Issuer
				days := int(d.ExpiresAt.Sub(now).Hours() / 24)
				info.DaysUntilExpiry = days
				info.WarningActive = days <= InsuranceWarningDays
				if d.IssuedAt != nil {
					info.StartDate = *d.IssuedAt
				}
				info.EndDate = *d.ExpiresAt
				if meta := d.Metadata; meta != nil {
					info.PolicyNumber = meta["policy_number"]
					info.CoverageType = meta["coverage_type"]
				}
				break
			}
		}
	}
	if info.IsValid != v.InsuranceValid {
		v.InsuranceValid = info.IsValid
		v.UpdatedAt = now
		_ = svc.store.UpdateVehicle(v)
	}
	return info, nil
}

func (svc *VehicleService) GetStats() VehicleStats {
	return svc.store.GetStats()
}

func (svc *VehicleService) GetAuditLogs(entityID string, limit int) []*AuditLog {
	return svc.store.GetAuditLogs(entityID, limit)
}

func (svc *VehicleService) updateVehicleInsurance(vehicleID string, expiry *time.Time) {
	v, err := svc.store.GetVehicle(vehicleID)
	if err != nil {
		return
	}
	v.InsuranceValid = true
	v.InsuranceExpiry = expiry
	v.UpdatedAt = time.Now()
	_ = svc.store.UpdateVehicle(v)
}

func (svc *VehicleService) checkAndActivateVehicle(vehicleID string) error {
	v, err := svc.store.GetVehicle(vehicleID)
	if err != nil {
		return err
	}
	if v.Status != VehicleStatusPending {
		return nil
	}
	docs, err := svc.store.ListDocuments(vehicleID)
	if err != nil {
		return err
	}
	hasRegistration := false
	hasInsurance := false
	for _, d := range docs {
		if d.Status != DocumentStatusVerified {
			continue
		}
		if d.Type == DocumentTypeRegistration {
			hasRegistration = true
		}
		if d.Type == DocumentTypeInsurance {
			hasInsurance = true
		}
	}
	if hasRegistration && hasInsurance {
		v.Status = VehicleStatusActive
		v.UpdatedAt = time.Now()
		v.UpdatedBy = "system"
		_ = svc.store.UpdateVehicle(v)
		svc.logger.Printf("[INFO] Vehicle auto-activated: id=%s", vehicleID)
	}
	return nil
}

func (svc *VehicleService) addAuditLog(entityType, entityID string, action AuditAction, actorID, actorRole, ip, ua string, oldData, newData interface{}) {
	svc.addAuditLogWithMeta(entityType, entityID, action, actorID, actorRole, ip, ua, oldData, newData, nil)
}

func (svc *VehicleService) addAuditLogWithMeta(entityType, entityID string, action AuditAction, actorID, actorRole, ip, ua string, oldData, newData interface{}, meta map[string]string) {
	al := &AuditLog{
		ID:         generateID(),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		ActorID:    actorID,
		ActorRole:  actorRole,
		IPAddress:  ip,
		UserAgent:  ua,
		OldData:    oldData,
		NewData:    newData,
		Metadata:   meta,
		CreatedAt:  time.Now(),
	}
	svc.store.AddAuditLog(al)
}

// ============================================================
// SECTION 6: Validation Helpers
// ============================================================

func validateCreateVehicleRequest(req CreateVehicleRequest) error {
	if strings.TrimSpace(req.OwnerID) == "" {
		return fmt.Errorf("owner_id is required")
	}
	if strings.TrimSpace(req.Make) == "" {
		return fmt.Errorf("make is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return fmt.Errorf("model is required")
	}
	currentYear := time.Now().Year()
	if req.Year < MinVehicleYear || req.Year > currentYear+1 {
		return fmt.Errorf("year must be between %d and %d", MinVehicleYear, currentYear+1)
	}
	if strings.TrimSpace(req.LicensePlate) == "" {
		return fmt.Errorf("license_plate is required")
	}
	if len(req.LicensePlate) > MaxLicensePlateLen {
		return fmt.Errorf("license plate too long (max %d chars)", MaxLicensePlateLen)
	}
	vin := strings.TrimSpace(req.VIN)
	if len(vin) != MinVINLength {
		return fmt.Errorf("VIN must be exactly %d characters", MinVINLength)
	}
	vinRe := regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)
	if !vinRe.MatchString(strings.ToUpper(vin)) {
		return fmt.Errorf("VIN contains invalid characters")
	}
	if req.Mileage < 0 {
		return fmt.Errorf("mileage cannot be negative")
	}
	if req.EngineSize < 0 {
		return fmt.Errorf("engine size cannot be negative")
	}
	return nil
}

func isValidVehicleStatus(s VehicleStatus) bool {
	switch s {
	case VehicleStatusActive, VehicleStatusInactive, VehicleStatusPending, VehicleStatusSuspended, VehicleStatusArchived:
		return true
	}
	return false
}

func isValidDocumentType(t DocumentType) bool {
	switch t {
	case DocumentTypeRegistration, DocumentTypeInsurance, DocumentTypeTUV, DocumentTypeEmissions, DocumentTypeCustom:
		return true
	}
	return false
}

func isValidMaintenanceType(t MaintenanceType) bool {
	switch t {
	case MaintenanceTypeOilChange, MaintenanceTypeTireRotation, MaintenanceTypeBrakeService,
		MaintenanceTypeInspection, MaintenanceTypeRepair, MaintenanceTypeOther:
		return true
	}
	return false
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// ============================================================
// SECTION 7: HTTP Handlers
// ============================================================

type Handler struct {
	svc    *VehicleService
	logger *log.Logger
}

func NewHandler(svc *VehicleService, logger *log.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Printf("[ERROR] Failed to write JSON response: %v", err)
	}
}

func (h *Handler) successResponse(w http.ResponseWriter, r *http.Request, status int, data interface{}, message string) {
	h.writeJSON(w, status, APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		RequestID: getRequestID(r),
		Timestamp: time.Now(),
	})
}

func (h *Handler) errorResponse(w http.ResponseWriter, r *http.Request, status int, errMsg string) {
	h.writeJSON(w, status, APIResponse{
		Success:   false,
		Error:     errMsg,
		RequestID: getRequestID(r),
		Timestamp: time.Now(),
	})
}

func (h *Handler) decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func getRequestID(r *http.Request) string {
	if id := r.Context().Value(contextKeyRequestID); id != nil {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}

func getActorID(r *http.Request) string {
	if id := r.Context().Value(contextKeyActorID); id != nil {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return "anonymous"
}

// Vehicle Handlers

func (h *Handler) CreateVehicle(w http.ResponseWriter, r *http.Request) {
	var req CreateVehicleRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.errorResponse(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	v, err := h.svc.RegisterVehicle(req, getActorID(r))
	if err != nil {
		h.logger.Printf("[ERROR] CreateVehicle: %v", err)
		if strings.Contains(err.Error(), "already exists") {
			h.errorResponse(w, r, http.StatusConflict, err.Error())
			return
		}
		h.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.successResponse(w, r, http.StatusCreated, v, "Vehicle registered successfully")
}

func (h *Handler) GetVehicle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	v, err := h.svc.GetVehicle(id, getActorID(r))
	if err != nil {
		h.errorResponse(w, r, http.StatusNotFound, err.Error())
		return
	}
	h.successResponse(w, r, http.StatusOK, v, "")
}

func (h *Handler) UpdateVehicle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var req UpdateVehicleRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.errorResponse(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	v, err := h.svc.UpdateVehicle(id, req, getActorID(r))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, r, http.StatusNotFound, err.Error())
			return
		}
		h.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.successResponse(w, r, http.StatusOK, v, "Vehicle updated successfully")
}

func (h *Handler) UpdateVehicleStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var req UpdateStatusRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.errorResponse(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	v, err := h.svc.UpdateVehicleStatus(id, req, getActorID(r))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, r, http.StatusNotFound, err.Error())
			return
		}
		h.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.successResponse(w, r, http.StatusOK, v, "Vehicle status updated")
}

func (h *Handler) DeleteVehicle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := h.svc.DeleteVehicle(id, getActorID(r)); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, r, http.StatusNotFound, err.Error())
			return
		}
		h.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.successResponse(w, r, http.StatusOK, nil, "Vehicle deleted successfully")
}

func (h *Handler) ListVehicles(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ownerID := query.Get("owner_id")
	statusStr := query.Get("status")
	var status VehicleStatus
	if statusStr != "" {
		status = VehicleStatus(statusStr)
		if !isValidVehicleStatus(status) {
			h.errorResponse(w, r, http.StatusBadRequest, "invalid status filter")
			return
		}
	}
	page := 1
	pageSize := 20
	if p := query.Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := query.Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}
	vehicles, total := h.svc.ListVehicles(ownerID, status, page, pageSize)
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	h.successResponse(w, r, http.StatusOK, PaginatedResponse{
		Items:      vehicles,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, "")
}

// Document Handlers

func (h *Handler) AddDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vehicleID := vars["id"]
	var req CreateDocumentRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.errorResponse(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	d, err := h.svc.AddDocument(vehicleID, req, getActorID(r))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, r, http.StatusNotFound, err.Error())
			return
		}
		h.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.successResponse(w, r, http.StatusCreated, d, "Document added successfully")
}

func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vehicleID := vars["id"]
	docs, err := h.svc.ListDocuments(vehicleID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, r, http.StatusNotFound, err.Error())
			return
		}
		h.errorResponse(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if docs == nil {
		docs = []*VehicleDocument{}
	}
	h.successResponse(w, r, http.StatusOK, docs, "")
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vehicleID := vars["id"]
	docID := vars["docId"]
	d, err := h.svc.GetDocument(docID, vehicleID)
	if err != nil {
		h.errorResponse(w, r, http.StatusNotFound, err.Error())
		return
	}
	h.successResponse(w, r, http.StatusOK, d, "")
}

func (h *Handler) VerifyDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	docID := vars["docId"]
	var req VerifyDocumentRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.errorResponse(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	d, err := h.svc.VerifyDocument(docID, req, getActorID(r))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, r, http.StatusNotFound, err.Error())
			return
		}
		h.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.successResponse(w, r, http.StatusOK, d, "Document verification updated")
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vehicleID := vars["id"]
	docID := vars["docId"]
	if err := h.svc.DeleteDocument(docID, vehicleID, getActorID(r)); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, r, http.StatusNotFound, err.Error())
			return
		}
		h.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.successResponse(w, r, http.StatusOK, nil, "Document deleted successfully")
}

// Maintenance Handlers

func (h *Handler) AddMaintenanceRecord(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vehicleID := vars["id"]
	var req CreateMaintenanceRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.errorResponse(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	m, err := h.svc.AddMaintenanceRecord(vehicleID, req, getActorID(r))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, r, http.StatusNotFound, err.Error())
			return
		}
		h.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.successResponse(w, r, http.StatusCreated, m, "Maintenance record added")
}

func (h *Handler) ListMaintenanceRecords(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vehicleID := vars["id"]
	records, err := h.svc.ListMaintenanceRecords(vehicleID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, r, http.StatusNotFound, err.Error())
			return
		}
		h.errorResponse(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if records == nil {
		records = []*MaintenanceRecord{}
	}
	h.successResponse(w, r, http.StatusOK, records, "")
}

func (h *Handler) GetMaintenanceRecord(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vehicleID := vars["id"]
	recordID := vars["recordId"]
	m, err := h.svc.GetMaintenanceRecord(recordID, vehicleID)
	if err != nil {
		h.errorResponse(w, r, http.StatusNotFound, err.Error())
		return
	}
	h.successResponse(w, r, http.StatusOK, m, "")
}

// Insurance Handler

func (h *Handler) ValidateInsurance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vehicleID := vars["id"]
	info, err := h.svc.ValidateInsurance(vehicleID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.errorResponse(w, r, http.StatusNotFound, err.Error())
			return
		}
		h.errorResponse(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	h.successResponse(w, r, http.StatusOK, info, "")
}

// Stats and Audit Handlers

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.svc.GetStats()
	h.successResponse(w, r, http.StatusOK, stats, "")
}

func (h *Handler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	entityID := query.Get("entity_id")
	limit := 100
	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}
	logs := h.svc.GetAuditLogs(entityID, limit)
	if logs == nil {
		logs = []*AuditLog{}
	}
	h.successResponse(w, r, http.StatusOK, logs, "")
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, HealthResponse{
		Status:    "healthy",
		Service:   ServiceName,
		Version:   ServiceVersion,
		Timestamp: time.Now(),
		Checks: map[string]string{
			"store": "ok",
			"api":   "ok",
		},
	})
}

// ============================================================
// SECTION 8: Middleware
// ============================================================

type contextKey string

const (
	contextKeyRequestID contextKey = "request_id"
	contextKeyActorID   contextKey = "actor_id"
	contextKeyActorRole contextKey = "actor_role"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateID()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), contextKeyRequestID, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

func LoggingMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			duration := time.Since(start)
			logger.Printf("[HTTP] method=%s path=%s status=%d duration=%s ip=%s ua=%s",
				r.Method, r.URL.Path, rw.status, duration,
				r.RemoteAddr, r.UserAgent())
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func JWTAuthMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			actorID := "anonymous"
			actorRole := "guest"
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					token := parts[1]
					claims, err := parseSimulatedJWT(token)
					if err != nil {
						logger.Printf("[WARN] Invalid JWT token: %v", err)
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusUnauthorized)
						json.NewEncoder(w).Encode(APIResponse{
							Success:   false,
							Error:     "invalid or expired token",
							Timestamp: time.Now(),
						})
						return
					}
					actorID = claims["sub"]
					actorRole = claims["role"]
				}
			}
			ctx := context.WithValue(r.Context(), contextKeyActorID, actorID)
			ctx = context.WithValue(ctx, contextKeyActorRole, actorRole)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseSimulatedJWT(token string) (map[string]string, error) {
	if token == "" {
		return nil, fmt.Errorf("empty token")
	}
	if strings.HasPrefix(token, "invalid") {
		return nil, fmt.Errorf("invalid token")
	}
	if token == "admin-token" {
		return map[string]string{"sub": "admin-001", "role": "admin"}, nil
	}
	if strings.HasPrefix(token, "user-") {
		return map[string]string{"sub": token, "role": "user"}, nil
	}
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		return map[string]string{"sub": "user-" + parts[1], "role": "user"}, nil
	}
	return map[string]string{"sub": "user-" + token, "role": "user"}, nil
}

func RateLimitMiddleware(maxRequests int, window time.Duration, logger *log.Logger) func(http.Handler) http.Handler {
	type client struct {
		count    int
		resetAt  time.Time
	}
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for ip, c := range clients {
				if now.After(c.resetAt) {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := strings.Split(r.RemoteAddr, ":")[0]
			mu.Lock()
			c, exists := clients[ip]
			if !exists || time.Now().After(c.resetAt) {
				c = &client{count: 0, resetAt: time.Now().Add(window)}
				clients[ip] = c
			}
			c.count++
			count := c.count
			mu.Unlock()
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(maxRequests-count))
			if count > maxRequests {
				logger.Printf("[WARN] Rate limit exceeded for IP: %s", ip)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(APIResponse{
					Success:   false,
					Error:     "rate limit exceeded",
					Timestamp: time.Now(),
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowed := false
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ============================================================
// SECTION 9: Router Setup
// ============================================================

func SetupRouter(h *Handler, logger *log.Logger) *mux.Router {
	r := mux.NewRouter()
	r.Use(RequestIDMiddleware)
	r.Use(SecurityHeadersMiddleware)
	r.Use(LoggingMiddleware(logger))
	r.Use(CORSMiddleware([]string{"*"}))
	r.Use(RateLimitMiddleware(500, time.Minute, logger))
	r.Use(JWTAuthMiddleware(logger))

	r.HandleFunc("/health", h.HealthCheck).Methods(http.MethodGet)
	r.HandleFunc("/ready", h.HealthCheck).Methods(http.MethodGet)

	api := r.PathPrefix(APIPrefix).Subrouter()

	// Vehicle routes
	vehicles := api.PathPrefix("/vehicles").Subrouter()
	vehicles.HandleFunc("", h.CreateVehicle).Methods(http.MethodPost)
	vehicles.HandleFunc("", h.ListVehicles).Methods(http.MethodGet)
	vehicles.HandleFunc("/{id}", h.GetVehicle).Methods(http.MethodGet)
	vehicles.HandleFunc("/{id}", h.UpdateVehicle).Methods(http.MethodPut, http.MethodPatch)
	vehicles.HandleFunc("/{id}", h.DeleteVehicle).Methods(http.MethodDelete)
	vehicles.HandleFunc("/{id}/status", h.UpdateVehicleStatus).Methods(http.MethodPut, http.MethodPatch)

	// Document routes
	vehicles.HandleFunc("/{id}/documents", h.AddDocument).Methods(http.MethodPost)
	vehicles.HandleFunc("/{id}/documents", h.ListDocuments).Methods(http.MethodGet)
	vehicles.HandleFunc("/{id}/documents/{docId}", h.GetDocument).Methods(http.MethodGet)
	vehicles.HandleFunc("/{id}/documents/{docId}", h.DeleteDocument).Methods(http.MethodDelete)

	// Document verification (admin route)
	api.HandleFunc("/documents/{docId}/verify", h.VerifyDocument).Methods(http.MethodPost)

	// Maintenance routes
	vehicles.HandleFunc("/{id}/maintenance", h.AddMaintenanceRecord).Methods(http.MethodPost)
	vehicles.HandleFunc("/{id}/maintenance", h.ListMaintenanceRecords).Methods(http.MethodGet)
	vehicles.HandleFunc("/{id}/maintenance/{recordId}", h.GetMaintenanceRecord).Methods(http.MethodGet)

	// Insurance validation
	vehicles.HandleFunc("/{id}/insurance/validate", h.ValidateInsurance).Methods(http.MethodGet)

	// Stats and audit
	api.HandleFunc("/stats", h.GetStats).Methods(http.MethodGet)
	api.HandleFunc("/audit-logs", h.GetAuditLogs).Methods(http.MethodGet)

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponse{
			Success:   false,
			Error:     fmt.Sprintf("route not found: %s %s", r.Method, r.URL.Path),
			Timestamp: time.Now(),
		})
	})

	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(APIResponse{
			Success:   false,
			Error:     fmt.Sprintf("method not allowed: %s", r.Method),
			Timestamp: time.Now(),
		})
	})

	return r
}

// ============================================================
// SECTION 10: Main Entry Point
// ============================================================

func main() {
	logger := log.New(os.Stdout, fmt.Sprintf("[%s] ", ServiceName), log.LstdFlags|log.Lmicroseconds|log.LUTC)

	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	store := NewStore()
	svc := NewVehicleService(store, logger)
	h := NewHandler(svc, logger)
	router := SetupRouter(h, logger)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Printf("[INFO] %s v%s starting on port %s", ServiceName, ServiceVersion, port)
		logger.Printf("[INFO] API available at http://localhost:%s%s", port, APIPrefix)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("[FATAL] Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Printf("[INFO] Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("[FATAL] Server forced to shutdown: %v", err)
	}
	logger.Printf("[INFO] Server stopped")
}

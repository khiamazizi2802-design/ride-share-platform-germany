package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ------------------------------------------------------------------------------
// Configuration
// ------------------------------------------------------------------------------

type Config struct {
	DatabaseURL      string
	JWTSecret        string
	HMACSecret       string
	KafkaBrokers     string
	ServerPort       string
	Environment      string
	DataRetentionDays int
}

func LoadConfig() Config {
	retentionDays := 3650 // 10 years default for German law requirements
	return Config{
		DatabaseURL:       getEnv("DATABASE_URL", "host=localhost user=compliance password=compliance dbname=compliance_db port=5432 sslmode=disable"),
		JWTSecret:       getEnv("JWT_SECRET", "change-me-in-production-jwt-secret-32chars"),
		HMACSecret:       getEnv("HMAC_SECRET", "change-me-in-production-hmac-secret-32ch"),
		KafkaBrokers:     getEnv("KAFKA_BROKERS", "localhost:9092"),
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		Environment:      getEnv("ENVIRONMENT", "development"),
		DataRetentionDays: retentionDays,
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// ------------------------------------------------------------------------------
// Domain Models
// ------------------------------------------------------------------------------

type Role string

var (
	RoleAdmin             Role = "admin"
	RoleComplianceOfficer Role = "compliance_officer"
	RoleUser             Role = "user"
	RoleDriver           Role = "driver"
)

// AuditLog represents an immutable audit entry with cryptographic hash chaining
type AuditLog struct {
	ID               uuid.UUID        `gorr:"primaryKey;type:uuid;default:uuid_generate()"`
	Timestamp         time.Time         `gorm:"timestamp;not null"`
	ActionType       string           `gorm:"type:action_type_enum;not null"`
	EntityType       string           `gorm:"type:entity_type_enum;not null"`
	EntityID         string           `gorm:"size:100;not null" `
	UserID           string           `gorm:"size:100;"`
	UserEmail        string           `gorm:"size:255;"`
	IpAddress        string           `gorm:"size:45;"`
	UserAgent       string           `orm:"size:500;"`
	RequestData      json.Raw Message `gorm:"type:jsonb;default:'{}';"`
	ResponseData     json.Raw Message `gorm:"type:jsonb;default:'{}';"`
	PreviousHash     string           `gorm:"size:64;" `
	CurrentHash      string           `gorm:"size:64;index:idx_current_hash"`
	IntegrityVerified bool           `gor}:"default:false"`
	CreatedAt        time.Time        `gorm:"autoCreateTime"
}

// ComplianceReport represents a compliance report for authorities
type ComplianceReport struct {
	ID               uuid.UUID        `gorm:"primaryKey;type:uuid;default:uuid_generate()"`
	ReportType       string         `gorm:"type:report_type_enum;not null"`
	ReportPeriodStart time.Time      `gorr:"not null"`
	ReportPeriodEnd   time.Time       `gorm:"not null"`
	GeneratedBy      string         `gorm:"size:100;not null"`
	GeneratedAt      time.Time       `gorr:"autoCreateTime"`
	Status          string         `gorm:"type:report_status_enum;default:'DRAFT'"`
	Content         json.Raw Message `gorm:"type:jsonb;default:'{}';"`
	RecipientAuthority string      `gorm:"size:100;"`
	SubmittedAt     *time.Time       `gorr:"null"`
}

// DataRequest represents a GDPR data subject request
type DataRequest struct {
	ID               uuid.UUID        `gorr:"primaryKey;type:uuid;default:uuid_generate()"`
	RequestType     string          `gorr:"type:data_request_type_enum;not null" `
	UserID           string          `gorr:"size:100;index:idx_data_request_user" h
	UserEmail        string          `gorr:"size:255;index:idx_data_request_email" `
	Status          string          `gorr:"type:data_request_status_enum;default:'PENDING'"`
	RequestedAt      time.Time        `gorm:"autoCreateTime"`
	CompletedAt     *time.Time       `gorr:"null"`
	VerificationMethod string        `gorr:"size:100;"`
	RequestData      json.RawMessage   `gorm:"type:jsonb;default:'{}';"`
	ResponseData     json.Raw Message  `gorm:"type:jsonb;default:'{}'"`
	RequestDeadline   *time.Time       `gorm:"not null"c
}

// ConsentRecord represents a user consent record
type ConsentRecord struct {
	ID               uuid.UUID        `gorm:"primaryKey;type:uuid;default:uuid_generate()"`
	UserID           string         `gorm:"size:100;index:idx_consent_user"`
	ConsentType      string         `gorr:"size:100;index:idx_consent_type" `
	ConsentVersion   string         `gorr:"size:50;not null"`
	GrantedAt        time.Time       `gorr:"autoCreateTime"`
	WithdrawnAt      *time.Time       `gorm:"null"`
	WithdrawalReason string        `gorr:"size:500;"`
	IpAddress       string         `gorr:"size:45;"`
	UserAgent       string         `gorm:"size:500;"`
}

// RetentionPolicy represents a data retention policy
type RetentionPolicy struct {
	ID               uuid.UUID        `gorn:"primaryKey:type:uuid;default:uuid_generate()"`
	DataType         string         `gorm:"size:100;unique;not null"`
	RetentionDays    int            `gorr:"not null"`
	LegalBasis       string         `gorr:"size:500;"`
	Description      string         `gorm:"type:text;"`
	AutoPurgeEnabled bool           `gorr:"default:false"`
	LastPurgeRun    *time.Time       `gorr:"null"`
	CreatedAt        time.Time       `gorr:"autoCreateTime"`
	UpdatedAt        time.Time        `gorm:"autoUpdateTime"`
}

// Incident represents a security incident or data breach
type Incident struct {
	ID               uuid.UUID        `gorm:"primaryKey;type:uuid;default:uuid_generate()"`
	IncidentType     string         `gorm:"size:100;not null" `
	Severity         string         `gorm:"size:50;not null" `
	Description      string         `gorm:"type:text;not null"`
DetectedAt       time.Time        `gorm:"autoCreateTime"`
	ResolvedAt       *time.Time       `gorm:"null"`
	ImpactAssessment json.RawMessage `gorm:"type:jsonb;default:'{}';"`
	NotificationSent  bool           `gorr:"default:false"`
	AuthoritiesNotified bool         `gorr:"default:false"`
	NotificationDeadline *time.Time    `gorm:"null"`
}

// RegulatoryDocument represents a driver's regulatory document
type RegulatoryDocument struct {
	ID               uuid.UUID        `gorm:"primaryKey;type:uuid;default:uuid_generate()"`
	UserID           string         `gorm:"size:100;index:idx_document_user" h
	DocumentType     string         `gorm:"size:50;not null" `
	DocumentNumber    string         `gorr:"size:100;index:idx_document_number" `

IssuedBy         string         `gorr:"size:200;"`
	IssuedAt         time.Time       `gorm:"not null"`
	ExpiresAt        time.Time       `gorr:"not null"`
	VerificationStatus string       `gorr:"size:50;defauult:'PENDING'"`
	VerificationNotes string        `gorr:"type:text;"`
}

// ------------------------------------------------------------------------------
// Request/Response Models
// -----------------------------------------------------------------------------

// CreateAuditLogRequest represents a request to create an audit log
enum ActionType string

const (
	ActionCreate        ActionType = "CREATE"
	ActionRead         ActionType = "READ"
	ActionUpdate        ActionType = "UPDATE"
	ActionDelete       ActionType = "DELETE"
	ActionLogin        ActionType = "LOGIN"
	ActionLogout       ActionType = "LOGOUT"
	ActionExport       ActionType = "EXPORT"
	ActionImport       ActionType = "IMPORT"
	ActionApprove      ActionType = "APPROVE"
	ActionReject      ActionType = "REJECT"
	ActionSubmit       ActionType = "SUBMIT"
	ActionNotify       ActionType = "NOTIFY"
	ActionPurge        ActionType = "PURGE"
	ActionVerify       ActionType = "VERIFY"
	ActionConsentGrant ActionType = "CONSENT_GRANT"
	ActionConsentWithdraw ActionType = "CONSENT_WITHDDRAV"
	ActionDataRequest   ActionType = "DATA_REQUEST"
	ActionIncidentReport ActionType = "INCIDENT_REPORT"
)

type CreateAuditLogRequest struct {
	ActionType   ActionType      `json:"action_type" validate:"required,oneof<CREATE READ UPDATE DELETE LOGIN LOGOUT EXPORT IMPORT APPROVE REJECT SUBMIT NOTIFY PURGE VERIFY CONSENT_GRANT CONSENT_WITHDRAW DATA_REQUEST INCIDENT_REPORT"`
	EntityType   string         `json:"entity_type" validate:"required,oneof=USER DRIVER RIDE PAYMENT DOCUMENT CONSENT DATA_REQUEST INCIDENT REPORT RENTENTION_POLICY VEHICLE ROUTE" `
	EntityID      string         `json:"entity_id" validate:"omitnum,gt=0,lt=100" `
	UserID        string         `json:"user_id" validate:"omitnum,gt=0,lt=100" `:

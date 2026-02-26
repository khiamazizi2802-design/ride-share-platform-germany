# Driver Onboarding Service

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)](Dockerfile)

---

## Overview

The **Driver Onboarding Service** is a dedicated microservice responsible for managing the end-to-end onboarding lifecycle of drivers within the platform. It orchestrates the multi-step registration process, including identity verification (KYC), professional license validation (P-Schein), secure document management, and compliance checks.

Built with Go, this service provides a robust, scalable, and GDPR-compliant foundation for driver registration workflows. It integrates seamlessly with the User Service for account management and the Notification Service for real-time status updates throughout the onboarding journey.

---

## Features

### Identity & Verification
- **KYC (Know Your Customer)** â Automated identity verification including government-issued ID scanning, liveness checks, and fraud prevention screening.
- **P-Schein Validation** â Professional driving license (PersonenbefÃ¶rderungsschein) verification and validation against official registries.
- **Background Check Integration** â Automated background screening with configurable third-party providers.

### Document Management
- **Secure Document Upload** â Encrypted storage for all submitted driver documents (ID cards, licenses, insurance certificates, vehicle registration).
- **Document Expiry Tracking** â Automated monitoring and alerting for expiring documents with configurable lead times.
- **Document Versioning** â Full audit trail with version history for all submitted documents.
- **Multi-Format Support** â Accepts PDF, JPEG, PNG, and HEIC file formats with automatic validation and virus scanning.

### Onboarding Workflow
- **Multi-Step Workflow Engine** â Configurable step-by-step onboarding process with state management and resume capability.
- **Progress Tracking** â Real-time onboarding progress visibility for both drivers and administrators.
- **Rejection & Resubmission Handling** â Structured rejection workflows with reason codes and guided resubmission flows.
- **Admin Review Dashboard Support** â Endpoints for manual review queues and approval workflows.

### Compliance & Security
- **GDPR Compliance** â Full compliance with EU General Data Protection Regulation including data minimization, right to erasure, and consent management.
- **Audit Logging** â Immutable audit trails for all onboarding events and data access.
- **Data Encryption** â AES-256 encryption at rest for all sensitive personal data.
- **Role-Based Access Control (RBAC)** â Fine-grained access control for service-to-service and admin operations.

### Notifications
- **Real-Time Status Updates** â Instant notifications at each onboarding milestone via the Notification Service.
- **Reminder System** â Automated reminders for incomplete onboarding steps with configurable intervals.
- **Multi-Channel Delivery** â Email, SMS, and push notification support through the Notification Service integration.

---

## API Endpoints

All endpoints are prefixed with `/api/v1` unless otherwise noted. Authentication is required via Bearer token for all endpoints except where noted.

### Health & Status

| Method | Endpoint | Auth Required | Description |
|--------|----------|---------------|-------------|
| `GET` | `/health` | No | Service health check. Returns service status, version, and dependency connectivity. |
| `GET` | `/ready` | No | Readiness probe for Kubernetes/container orchestration. |
| `GET` | `/metrics` | Internal Only | Prometheus metrics endpoint. |

---

### Onboarding Applications

| Method | Endpoint | Auth Required | Description |
|--------|----------|---------------|-------------|
| `POST` | `/api/v1/onboarding/applications` | Yes | Initiate a new driver onboarding application. Creates a new application record and returns the application ID and initial step details. |
| `GET` | `/api/v1/onboarding/applications/:id` | Yes | Retrieve the full details of a specific onboarding application including current status, completed steps, and pending requirements. |
| `GET` | `/api/v1/onboarding/applications` | Yes (Admin) | List all onboarding applications with optional filters for status, date range, and region. Supports pagination. |
| `PATCH` | `/api/v1/onboarding/applications/:id` | Yes | Update an existing onboarding application. Used to progress through workflow steps. |
| `DELETE` | `/api/v1/onboarding/applications/:id` | Yes (Admin) | Soft-delete an onboarding application. Triggers GDPR data retention evaluation. |

**Request Body â POST `/api/v1/onboarding/applications`:**
```json
{
  "driver_id": "string (required)",
  "email": "string (required)",
  "phone_number": "string (required)",
  "first_name": "string (required)",
  "last_name": "string (required)",
  "date_of_birth": "string (ISO 8601, required)",
  "nationality": "string (ISO 3166-1 alpha-2, required)",
  "region": "string (required)",
  "consent": {
    "gdpr_processing": "boolean (required)",
    "background_check": "boolean (required)",
    "terms_of_service": "boolean (required)"
  }
}
```

**Response â 201 Created:**
```json
{
  "application_id": "uuid",
  "status": "INITIATED",
  "current_step": "PERSONAL_INFORMATION",
  "steps": [
    { "step": "PERSONAL_INFORMATION", "status": "IN_PROGRESS" },
    { "step": "KYC_VERIFICATION", "status": "PENDING" },
    { "step": "P_SCHEIN_UPLOAD", "status": "PENDING" },
    { "step": "DOCUMENT_SUBMISSION", "status": "PENDING" },
    { "step": "BACKGROUND_CHECK", "status": "PENDING" },
    { "step": "FINAL_REVIEW", "status": "PENDING" }
  ],
  "created_at": "2024-01-15T10:30:00Z",
  "expires_at": "2024-02-15T10:30:00Z"
}
```

---

### KYC Verification

| Method | Endpoint | Auth Required | Description |
|--------|----------|---------------|-------------|
| `POST` | `/api/v1/onboarding/applications/:id/kyc/initiate` | Yes | Initiate the KYC verification process for an application. Returns a session token for the identity verification provider SDK. |
| `GET` | `/api/v1/onboarding/applications/:id/kyc/status` | Yes | Retrieve current KYC verification status and any failure reasons. |
| `POST` | `/api/v1/onboarding/applications/:id/kyc/webhook` | Internal | Webhook endpoint for KYC provider callbacks. Processes identity verification results. |
| `POST` | `/api/v1/onboarding/applications/:id/kyc/retry` | Yes | Initiate a KYC retry for a failed verification attempt (subject to maximum retry limits). |

**KYC Status Values:**
```
PENDING | IN_PROGRESS | AWAITING_REVIEW | APPROVED | REJECTED | RETRY_REQUIRED
```

---

### P-Schein (Professional License)

| Method | Endpoint | Auth Required | Description |
|--------|----------|---------------|-------------|
| `POST` | `/api/v1/onboarding/applications/:id/pschein` | Yes | Submit P-Schein (PersonenbefÃ¶rderungsschein) details and documentation. Triggers automated validation. |
| `GET` | `/api/v1/onboarding/applications/:id/pschein` | Yes | Retrieve submitted P-Schein details and validation status. |
| `PUT` | `/api/v1/onboarding/applications/:id/pschein` | Yes | Update or resubmit P-Schein information following a rejection. |

**Request Body â POST `/api/v1/onboarding/applications/:id/pschein`:**
```json
{
  "license_number": "string (required)",
  "issuing_authority": "string (required)",
  "issue_date": "string (ISO 8601, required)",
  "expiry_date": "string (ISO 8601, required)",
  "license_class": "string (required)",
  "document_front_id": "uuid (required â from Document Upload)",
  "document_back_id": "uuid (optional â from Document Upload)"
}
```

---

### Document Management

| Method | Endpoint | Auth Required | Description |
|--------|----------|---------------|-------------|
| `POST` | `/api/v1/onboarding/applications/:id/documents` | Yes | Upload a document for the onboarding application. Accepts multipart/form-data. Returns a document ID for reference in other endpoints. |
| `GET` | `/api/v1/onboarding/applications/:id/documents` | Yes | List all documents associated with an onboarding application including type, status, and expiry information. |
| `GET` | `/api/v1/onboarding/applications/:id/documents/:doc_id` | Yes | Retrieve metadata for a specific document. |
| `GET` | `/api/v1/onboarding/applications/:id/documents/:doc_id/download` | Yes (Admin) | Generate a signed, time-limited download URL for a specific document. Admin access only. |
| `DELETE` | `/api/v1/onboarding/applications/:id/documents/:doc_id` | Yes | Remove a document from an application (only permitted during active onboarding steps). |

**Supported Document Types:**
```
NATIONAL_ID | PASSPORT | DRIVING_LICENSE | P_SCHEIN | VEHICLE_REGISTRATION
INSURANCE_CERTIFICATE | PROFILE_PHOTO | POLICE_CLEARANCE | OTHER
```

**Request â POST `/api/v1/onboarding/applications/:id/documents` (multipart/form-data):**
```
file:          binary (required, max 25MB)
document_type: string (required, see supported types)
expiry_date:   string (ISO 8601, optional)
description:   string (optional)
```

---

### Background Checks

| Method | Endpoint | Auth Required | Description |
|--------|----------|---------------|-------------|
| `POST` | `/api/v1/onboarding/applications/:id/background-check/initiate` | Yes (Admin) | Manually trigger a background check for an application. |
| `GET` | `/api/v1/onboarding/applications/:id/background-check/status` | Yes | Retrieve the current background check status and results summary. |
| `POST` | `/api/v1/onboarding/applications/:id/background-check/webhook` | Internal | Webhook for background check provider callbacks. |

---

### Admin & Review

| Method | Endpoint | Auth Required | Description |
|--------|----------|---------------|-------------|
| `GET` | `/api/v1/admin/onboarding/queue` | Yes (Admin) | Retrieve the manual review queue with configurable filters and pagination. |
| `POST` | `/api/v1/admin/onboarding/applications/:id/approve` | Yes (Admin) | Approve a completed onboarding application, activating the driver account. |
| `POST` | `/api/v1/admin/onboarding/applications/:id/reject` | Yes (Admin) | Reject an onboarding application with required reason codes and optional notes. |
| `POST` | `/api/v1/admin/onboarding/applications/:id/request-resubmission` | Yes (Admin) | Request resubmission of specific documents or information. |
| `GET` | `/api/v1/admin/onboarding/statistics` | Yes (Admin) | Retrieve onboarding funnel statistics, conversion rates, and average completion times. |

---

### GDPR & Data Subject Rights

| Method | Endpoint | Auth Required | Description |
|--------|----------|---------------|-------------|
| `GET` | `/api/v1/gdpr/export/:driver_id` | Yes | Generate a complete data export (SAR â Subject Access Request) for a driver in JSON format. |
| `DELETE` | `/api/v1/gdpr/erase/:driver_id` | Yes (Admin) | Process a Right to Erasure request. Schedules or immediately executes data deletion in compliance with retention policies. |
| `GET` | `/api/v1/gdpr/consent/:driver_id` | Yes | Retrieve all recorded consent records for a driver. |
| `PUT` | `/api/v1/gdpr/consent/:driver_id` | Yes | Update consent preferences for a driver. |

---

## Onboarding Workflow

The Driver Onboarding Service implements a configurable, state-machine-based multi-step workflow. Each step must be completed and validated before the driver can progress to the next stage. The workflow supports pausing and resuming, allowing drivers to complete onboarding over multiple sessions.

### Workflow Steps

```
âââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ
â                    DRIVER ONBOARDING WORKFLOW                   â
âââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ

  [1] APPLICATION INITIATED
       â
       â¼
  [2] PERSONAL INFORMATION
       â  â¢ Full legal name, date of birth, nationality
       â  â¢ Contact information (email, phone)
       â  â¢ Address details
       â  â¢ GDPR consent capture
       â¼
  [3] KYC VERIFICATION
       â  â¢ Government ID upload (front + back)
       â  â¢ Liveness / selfie check
       â  â¢ Identity verification via third-party provider
       â  â¢ Fraud screening
       â¼
  [4] P-SCHEIN SUBMISSION
       â  â¢ Professional license number entry
       â  â¢ Document upload (front + optional back)
       â  â¢ Automated validity and expiry check
       â  â¢ Registry lookup (where available)
       â¼
  [5] SUPPLEMENTARY DOCUMENT SUBMISSION
       â  â¢ Vehicle registration certificate
       â  â¢ Vehicle insurance certificate
       â  â¢ Profile photograph
       â  â¢ Any region-specific required documents
       â¼
  [6] BACKGROUND CHECK
       â  â¢ Automated submission to background check provider
       â  â¢ Criminal record check
       â  â¢ Driving record verification
       â  â¢ Configurable adjudication rules
       â¼
  [7] FINAL REVIEW
       â  â¢ Automated compliance validation
       â  â¢ Manual review queue (if required)
       â  â¢ Admin approval / rejection
       â¼
  [8] APPROVED / REJECTED
       â
       âââ APPROVED â User Service notified â Driver account activated
       â             â Notification Service â Welcome communication sent
       â
       âââ REJECTED â Reason codes recorded
                    â Notification Service â Rejection communication with reasons
                    â Resubmission window opened (if applicable)
```

### Application Status Values

| Status | Description |
|--------|-------------|
| `INITIATED` | Application created, awaiting driver input. |
| `IN_PROGRESS` | Driver is actively completing onboarding steps. |
| `PENDING_KYC` | Awaiting KYC verification result from provider. |
| `PENDING_BACKGROUND_CHECK` | Awaiting background check result. |
| `PENDING_REVIEW` | All steps complete, awaiting admin review. |
| `RESUBMISSION_REQUIRED` | Admin has requested additional information or documents. |
| `APPROVED` | Onboarding complete, driver account activated. |
| `REJECTED` | Application rejected. Reason codes available. |
| `EXPIRED` | Application not completed within the allowed time window. |
| `WITHDRAWN` | Driver withdrew their application. |

### Step Validation Rules

- Steps must be completed **in order**. A driver cannot skip to document submission without completing KYC.
- Each step has a configurable **maximum retry limit** (default: 3 for KYC, configurable per region).
- Applications expire if not completed within the configured window (default: **30 days** from initiation).
- Approved drivers are subject to **annual re-verification** for expiring documents.

---

## GDPR Compliance

The Driver Onboarding Service is designed with privacy by design and GDPR compliance as core principles.

### Data Processing Basis

All personal data collected during onboarding is processed under one or more of the following lawful bases:

| Data Category | Legal Basis | Retention Period |
|---------------|-------------|------------------|
| Identity documents (KYC) | Legal obligation + Contractual necessity | 7 years from account closure |
| P-Schein / Professional license | Legal obligation | Duration of active driver relationship + 3 years |
| Background check results | Legitimate interest + Consent | 3 years from collection |
| Onboarding audit logs | Legal obligation | 10 years |
| Consent records | Legal obligation | 5 years from consent withdrawal |
| Rejected application data | Legitimate interest (fraud prevention) | 6 months from rejection |

### Data Minimization

- Only data strictly necessary for the onboarding purpose is collected.
- Document images are stored encrypted with access limited to authorized personnel and systems.
- KYC document images may be deleted after verification is complete, subject to regulatory requirements in the operating region.

### Right to Erasure (Article 17)

The service supports Right to Erasure requests via the GDPR API endpoints. Erasure requests are evaluated against retention obligations:

1. **Immediate deletion** â Data held solely on consent basis with no overriding legal obligation.
2. **Scheduled deletion** â Data subject to a legal retention period is flagged for deletion at the end of the retention window.
3. **Partial erasure** â Where full erasure is not permissible (e.g., fraud prevention logs), identifiable fields are pseudonymized.

All erasure actions are logged in the immutable audit trail.

### Data Subject Access Requests (Article 15)

The `/api/v1/gdpr/export/:driver_id` endpoint generates a machine-readable JSON export of all personal data held for a driver, including:
- Personal information
- Document metadata (not binary content)
- Onboarding history and step completion records
- Consent history
- Audit log entries

### Data Transfers

- KYC verification data may be transferred to third-party identity verification providers. These transfers are governed by Data Processing Agreements (DPAs) and, where applicable, Standard Contractual Clauses (SCCs).
- Background check data transfers are subject to equivalent protections.
- All third-party providers are listed in the platform's Records of Processing Activities (RoPA).

### Security Measures

- All personal data is encrypted at rest using **AES-256**.
- All API communications use **TLS 1.2+**.
- Document storage uses server-side encryption with customer-managed keys (CMK) where configured.
- Access to personal data is logged and subject to RBAC controls.

---

## Environment Variables

The following environment variables must be configured before running the service. Variables marked as **Required** have no default value and must be explicitly set.

### Service Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | The port the HTTP server listens on. |
| `ENV` | No | `development` | Runtime environment. Values: `development`, `staging`, `production`. Controls logging verbosity and error detail. |
| `LOG_LEVEL` | No | `info` | Log level. Values: `debug`, `info`, `warn`, `error`. |
| `SERVICE_NAME` | No | `driver-onboarding-service` | Service name used in logs and distributed tracing. |
| `API_VERSION` | No | `v1` | API version prefix. |

### Database

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | **Yes** | â | PostgreSQL connection string. Format: `postgres://user:password@host:port/dbname?sslmode=require` |
| `DATABASE_MAX_OPEN_CONNS` | No | `25` | Maximum number of open database connections. |
| `DATABASE_MAX_IDLE_CONNS` | No | `10` | Maximum number of idle database connections. |
| `DATABASE_CONN_MAX_LIFETIME` | No | `5m` | Maximum lifetime of a database connection. |
| `DATABASE_MIGRATION_AUTO_RUN` | No | `false` | Automatically run pending database migrations on startup. Set to `true` with caution in production. |

### Authentication & Security

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | **Yes** | â | Secret key for JWT token verification. Must be at least 32 characters. |
| `JWT_ISSUER` | No | `auth-service` | Expected JWT issuer claim value. |
| `INTERNAL_API_KEY` | **Yes** | â | API key for authenticating internal service-to-service calls (webhooks, etc.). |
| `ENCRYPTION_KEY` | **Yes** | â | AES-256 encryption key for at-rest data encryption. Must be exactly 32 bytes (base64-encoded). |

### Document Storage

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `STORAGE_PROVIDER` | No | `s3` | Storage backend. Values: `s3`, `gcs`, `azure`. |
| `STORAGE_BUCKET_NAME` | **Yes** | â | Name of the storage bucket for document uploads. |
| `STORAGE_REGION` | **Yes** (for S3/GCS) | â | Cloud region for the storage bucket. |
| `AWS_ACCESS_KEY_ID` | **Yes** (if S3) | â | AWS access key ID for S3 access. |
| `AWS_SECRET_ACCESS_KEY` | **Yes** (if S3) | â | AWS secret access key for S3 access. |
| `STORAGE_MAX_FILE_SIZE_MB` | No | `25` | Maximum allowed file upload size in megabytes. |
| `SIGNED_URL_EXPIRY_MINUTES` | No | `15` | Validity duration of signed download URLs in minutes. |

### KYC Provider

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `KYC_PROVIDER` | No | `onfido` | KYC provider to use. Values: `onfido`, `jumio`, `trulioo`. |
| `KYC_API_KEY` | **Yes** | â | API key for the configured KYC provider. |
| `KYC_WEBHOOK_SECRET` | **Yes** | â | Secret for validating KYC provider webhook payloads. |
| `KYC_MAX_RETRY_ATTEMPTS` | No | `3` | Maximum number of KYC retry attempts per application. |
| `KYC_SANDBOX_MODE` | No | `false` | Enable KYC provider sandbox/test mode. Set to `true` in non-production environments. |

### Background Check Provider

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `BACKGROUND_CHECK_PROVIDER` | No | `checkr` | Background check provider. Values: `checkr`, `sterling`, `certn`. |
| `BACKGROUND_CHECK_API_KEY` | **Yes** | â | API key for the configured background check provider. |
| `BACKGROUND_CHECK_WEBHOOK_SECRET` | **Yes** | â | Secret for validating background check webhook payloads. |
| `BACKGROUND_CHECK_PACKAGE` | No | `driver_pro` | Default screening package identifier (provider-specific). |

### Integration Services

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `USER_SERVICE_URL` | **Yes** | â | Base URL of the User Service. Example: `http://user-service:8081` |
| `USER_SERVICE_TIMEOUT` | No | `10s` | HTTP timeout for User Service requests. |
| `NOTIFICATION_SERVICE_URL` | **Yes** | â | Base URL of the Notification Service. Example: `http://notification-service:8082` |
| `NOTIFICATION_SERVICE_TIMEOUT` | No | `5s` | HTTP timeout for Notification Service requests. |
| `NOTIFICATION_SERVICE_RETRY_ATTEMPTS` | No | `3` | Number of retry attempts for failed notification deliveries. |

### Workflow Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ONBOARDING_EXPIRY_DAYS` | No | `30` | Number of days before an incomplete application expires. |
| `DOCUMENT_EXPIRY_WARNING_DAYS` | No | `30` | Days before document expiry to send a warning notification. |
| `ADMIN_REVIEW_REQUIRED` | No | `true` | Whether all applications require manual admin approval before activation. |

### Observability

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `METRICS_ENABLED` | No | `true` | Enable Prometheus metrics endpoint. |
| `TRACING_ENABLED` | No | `false` | Enable distributed tracing (OpenTelemetry). |
| `TRACING_ENDPOINT` | No | â | OpenTelemetry collector endpoint. Required if `TRACING_ENABLED=true`. |
| `SENTRY_DSN` | No | â | Sentry DSN for error tracking. Leave empty to disable. |

---

## Running Locally

### Prerequisites

- [Go 1.21+](https://golang.org/dl/)
- [PostgreSQL 14+](https://www.postgresql.org/)
- Access to required third-party services (KYC provider, Background Check provider) or their sandbox equivalents.
- Configured cloud storage bucket (or use LocalStack for local S3 emulation).

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/driver-onboarding-service.git
cd driver-onboarding-service
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Set Up Environment Variables

Copy the example environment file and populate the required values:

```bash
cp .env.example .env
```

Edit `.env` with your local configuration:

```dotenv
# Service
PORT=8080
ENV=development
LOG_LEVEL=debug

# Database
DATABASE_URL=postgres://postgres:password@localhost:5432/driver_onboarding_dev?sslmode=disable
DATABASE_MIGRATION_AUTO_RUN=true

# Security
JWT_SECRET=your-local-development-jwt-secret-minimum-32-chars
INTERNAL_API_KEY=local-internal-api-key
ENCRYPTION_KEY=base64-encoded-32-byte-encryption-key==

# Storage (using LocalStack)
STORAGE_PROVIDER=s3
STORAGE_BUCKET_NAME=driver-documents-local
STORAGE_REGION=us-east-1
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test

# KYC (Sandbox)
KYC_PROVIDER=onfido
KYC_API_KEY=your-onfido-sandbox-api-key
KYC_WEBHOOK_SECRET=local-webhook-secret
KYC_SANDBOX_MODE=true

# Background Check (Sandbox)
BACKGROUND_CHECK_PROVIDER=checkr
BACKGROUND_CHECK_API_KEY=your-checkr-sandbox-api-key
BACKGROUND_CHECK_WEBHOOK_SECRET=local-webhook-secret

# Integration Services (local)
USER_SERVICE_URL=http://localhost:8081
NOTIFICATION_SERVICE_URL=http://localhost:8082
```

### 4. Set Up the Database

```bash
# Create the database
createdb driver_onboarding_dev

# Run migrations manually (or set DATABASE_MIGRATION_AUTO_RUN=true)
go run ./cmd/migrate/main.go up
```

### 5. Run the Service

```bash
go run ./cmd/server/main.go
```

The service will start and listen on the configured `PORT` (default: `8080`).

```
2024/01/15 10:30:00 INFO  Driver Onboarding Service starting
2024/01/15 10:30:00 INFO  Database connection established
2024/01/15 10:30:00 INFO  Running pending migrations: 3 applied
2024/01/15 10:30:00 INFO  HTTP server listening on :8080
2024/01/15 10:30:00 INFO  Environment: development
```

### 6. Verify the Service

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "environment": "development",
  "timestamp": "2024-01-15T10:30:00Z",
  "dependencies": {
    "database": "healthy",
    "storage": "healthy",
    "kyc_provider": "healthy",
    "user_service": "healthy",
    "notification_service": "healthy"
  }
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run only unit tests (exclude integration tests)
go test -short ./...

# Run integration tests (requires live dependencies)
go test -tags integration ./...
```

### Running with Live Reload (Development)

Install [Air](https://github.com/cosmtrek/air) for live reloading:

```bash
go install github.com/cosmtrek/air@latest
air
```

---

## Docker

### Build the Docker Image

```bash
# Build with default tag
docker build -t driver-onboarding-service:latest .

# Build with a specific version tag
docker build -t driver-onboarding-service:1.0.0 .

# Build with build arguments (for CI/CD pipelines)
docker build \
  --build-arg BUILD_VERSION=1.0.0 \
  --build-arg BUILD_COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t driver-onboarding-service:1.0.0 .
```

### Run the Docker Container

```bash
docker run \
  --name driver-onboarding-service \
  -p 8080:8080 \
  -e PORT=8080 \
  -e ENV=production \
  -e DATABASE_URL="postgres://user:password@db-host:5432/driver_onboarding?sslmode=require" \
  -e JWT_SECRET="your-production-jwt-secret" \
  -e INTERNAL_API_KEY="your-internal-api-key" \
  -e ENCRYPTION_KEY="your-base64-encoded-encryption-key" \
  -e STORAGE_BUCKET_NAME="your-production-bucket" \
  -e KYC_API_KEY="your-kyc-api-key" \
  -e BACKGROUND_CHECK_API_KEY="your-bgc-api-key" \
  -e USER_SERVICE_URL="http://user-service:8081" \
  -e NOTIFICATION_SERVICE_URL="http://notification-service:8082" \
  --restart unless-stopped \
  driver-onboarding-service:latest
```

### Docker Compose (Full Local Stack)

A `docker-compose.yml` is provided for running the full local development stack:

```yaml
version: '3.9'

services:
  driver-onboarding-service:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - ENV=development
      - DATABASE_URL=postgres://postgres:password@postgres:5432/driver_onboarding?sslmode=disable
      - JWT_SECRET=local-development-secret-minimum-32-chars
      - INTERNAL_API_KEY=local-internal-api-key
      - ENCRYPTION_KEY=bG9jYWwtZW5jcnlwdGlvbi1rZXktMzJieXRlcw==
      - STORAGE_PROVIDER=s3
      - STORAGE_BUCKET_NAME=driver-documents-local
      - STORAGE_REGION=us-east-1
      - AWS_ACCESS_KEY_ID=test
      - AWS_SECRET_ACCESS_KEY=test
      - KYC_SANDBOX_MODE=true
      - KYC_API_KEY=sandbox-kyc-key
      - KYC_WEBHOOK_SECRET=local-webhook-secret
      - BACKGROUND_CHECK_API_KEY=sandbox-bgc-key
      - BACKGROUND_CHECK_WEBHOOK_SECRET=local-webhook-secret
      - USER_SERVICE_URL=http://user-service:8081
      - NOTIFICATION_SERVICE_URL=http://notification-service:8082
      - DATABASE_MIGRATION_AUTO_RUN=true
    depends_on:
      postgres:
        condition: service_healthy
      localstack:
        condition: service_healthy
    restart: unless-stopped
    networks:
      - platform-network

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: driver_onboarding
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - platform-network

  localstack:
    image: localstack/localstack:latest
    ports:
      - "4566:4566"
    environment:
      - SERVICES=s3
      - AWS_DEFAULT_REGION=us-east-1
    volumes:
      - localstack-data:/tmp/localstack
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:4566/_localstack/health"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - platform-network

volumes:
  postgres-data:
  localstack-data:

networks:
  platform-network:
    driver: bridge
```

```bash
# Start the full stack
docker-compose up -d

# View logs
docker-compose logs -f driver-onboarding-service

# Stop the stack
docker-compose down

# Stop and remove volumes (clean slate)
docker-compose down -v
```

### Multi-Stage Dockerfile

The provided `Dockerfile` uses a multi-stage build for minimal production image size:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s' \
    -o driver-onboarding-service \
    ./cmd/server/main.go

# Final stage
FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/driver-onboarding-service /driver-onboarding-service
EXPOSE 8080
ENTRYPOINT ["/driver-onboarding-service"]
```

---

## Integration Points

### User Service

The Driver Onboarding Service communicates with the **User Service** for account lifecycle management.

**Communication Pattern:** Synchronous HTTP REST

**Integration Scenarios:**

| Event | Action | Endpoint Called |
|-------|--------|-----------------|
| Onboarding application initiated | Verify driver account exists and is in `PENDING_ONBOARDING` state | `GET /api/v1/users/:id` |
| KYC approved | Update user KYC status | `PATCH /api/v1/users/:id/kyc-status` |
| Onboarding fully approved | Activate driver account and assign `DRIVER` role | `POST /api/v1/users/:id/activate` |
| Onboarding rejected | Update user account status to `ONBOARDING_REJECTED` | `PATCH /api/v1/users/:id/status` |
| Right to Erasure processed | Trigger user account deletion workflow | `DELETE /api/v1/users/:id` |

**Configuration:**
```dotenv
USER_SERVICE_URL=http://user-service:8081
USER_SERVICE_TIMEOUT=10s
```

**Error Handling:**
- If the User Service is unavailable during onboarding initiation, the request is rejected with a `503 Service Unavailable` response.
- For non-critical status updates (e.g., intermediate step notifications), failures are queued for retry using an internal retry mechanism with exponential backoff.
- Circuit breaker pattern is implemented to prevent cascade failures during User Service outages.

---

### Notification Service

The Driver Onboarding Service communicates with the **Notification Service** to deliver timely, contextual communications to drivers and administrators.

**Communication Pattern:** Asynchronous HTTP REST with retry logic

**Notification Events:**

| Trigger | Recipient | Notification Type | Channel |
|---------|-----------|-------------------|---------|
| Application initiated | Driver | Welcome / next steps | Email, Push |
| KYC session ready | Driver | KYC action required | Email, SMS, Push |
| KYC approved | Driver | KYC success confirmation | Email, Push |
| KYC rejected | Driver | KYC failure with reason | Email, SMS |
| Document rejected | Driver | Resubmission required | Email, Push |
| Resubmission requested | Driver | Action required | Email, SMS, Push |
| Background check initiated | Driver | Process update | Email |
| Application approved | Driver | Welcome â activation confirmed | Email, SMS, Push |
| Application rejected | Driver | Rejection with reason codes | Email |
| Application expired | Driver | Expiry warning (D-7, D-1) | Email, SMS |
| Document expiring | Driver | Document renewal reminder | Email, Push |
| Application pending review | Admin | Review queue notification | Email, Push |

**Request Format to Notification Service:**
```json
{
  "recipient_id": "uuid",
  "recipient_email": "string",
  "recipient_phone": "string",
  "notification_type": "ONBOARDING_APPLICATION_APPROVED",
  "channels": ["email", "push"],
  "template_id": "driver-onboarding-approved-v2",
  "template_variables": {
    "driver_first_name": "string",
    "application_id": "uuid",
    "activation_date": "ISO 8601 date string"
  },
  "priority": "high",
  "metadata": {
    "source_service": "driver-onboarding-service",
    "correlation_id": "uuid"
  }
}
```

**Configuration:**
```dotenv
NOTIFICATION_SERVICE_URL=http://notification-service:8082
NOTIFICATION_SERVICE_TIMEOUT=5s
NOTIFICATION_SERVICE_RETRY_ATTEMPTS=3
```

**Error Handling:**
- Notification delivery is treated as **non-blocking** â failures do not halt the onboarding workflow.
- Failed notifications are retried up to `NOTIFICATION_SERVICE_RETRY_ATTEMPTS` times with exponential backoff.
- Persistent failures are logged and surfaced via the metrics endpoint for alerting.
- Critical notifications (e.g., approval, rejection) are flagged as high-priority and dead-lettered for manual intervention if all retries are exhausted.

---

## Architecture Notes

```
âââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ
â                    DRIVER ONBOARDING SERVICE                        â
â                                                                     â
â  ââââââââââââââââ  ââââââââââââââââ  ââââââââââââââââââââââââââ   â
â  â  HTTP Server  â  â  Workflow    â  â   Document Manager     â   â
â  â  (REST API)   â  â  Engine      â  â   (Upload/Storage)     â   â
â  ââââââââ¬ââââââââ  ââââââââ¬ââââââââ  âââââââââââââ¬âââââââââââââ   â
â         â                 â                       â                 â
â  ââââââââ¼ââââââââââââââââââ¼ââââââââââââââââââââââââ¼âââââââââââââ   â
â  â                     Service Layer                           â   â
â  ââââââââ¬ââââââââââââââââââââââââââââââââââââââââââââââââââ¬âââââ   â
â         â                                                 â        â
â  ââââââââ¼âââââââââââ                         âââââââââââââ¼âââââââ â
â  â   PostgreSQL    â                         â   Cloud Storage  â â
â  â   (State +      â                         â   (Documents)    â â
â  â    Audit Log)   â                         ââââââââââââââââââââ â
â  âââââââââââââââââââ                                               â
ââââââââââââââââââââââââââ¬âââââââââââââââââââ¬âââââââââââââââââââââââââ
                         â                  â
              ââââââââââââ¼âââ        ââââââââ¼âââââââââââ
              â  User       â        â  Notification   â
              â  Service    â        â  Service        â
              âââââââââââââââ        âââââââââââââââââââ
                    â
         ââââââââââââ´âââââââââââ
         â                     â
  ââââââââ¼âââââââ       ââââââââ¼âââââââââââ
  â  KYC        â       â  Background     â
  â  Provider   â       â  Check Provider â
  âââââââââââââââ       âââââââââââââââââââ
```

---

## Contributing

Please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) guide for coding standards, branch naming conventions, and the pull request process.

## License

This project is licensed under the MIT License â see the [LICENSE](LICENSE) file for details.

---

*For internal platform documentation, architecture decision records (ADRs), and runbook procedures, refer to the internal engineering wiki.*
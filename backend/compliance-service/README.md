# Compliance Service

## Overview

The **Compliance Service** is a critical microservice within the mobility platform responsible for ensuring adherence to German regulatory requirements and European Union data protection laws. It provides automated reporting, audit logging, data governance, and regulatory compliance monitoring across the entire platform ecosystem.

This service acts as the central authority for:

- **PBefG (Personenbeförderungsgesetz)** — German Passenger Transport Act compliance and regulatory reporting to *Kraftfahrt-Bundesamt (KBA)* and regional transport authorities (*Verkehrsbehörden*)
- **DSGVO/GDPR** — EU General Data Protection Regulation and its German implementation (*Bundesdatenschutzgesetz — BDSG*) enforcement
- **Audit Logging** — Immutable, tamper-evident audit trails for all data access and business operations
- **Data Retention** — Automated lifecycle management of personal and operational data

---

## Table of Contents

1. [Features](#features)
2. [Architecture](#architecture)
3. [Getting Started](#getting-started)
4. [Environment Variables](#environment-variables)
5. [API Endpoints](#api-endpoints)
6. [Database Schema](#database-schema)
7. [Kafka Events](#kafka-events)
8. [Deployment](#deployment)
9. [German Regulatory Compliance Notes](#german-regulatory-compliance-notes)
10. [Security](#security)
11. [Testing](#testing)
12. [Contributing](#contributing)

---

## Features

### Regulatory Reporting
- Automated **PBefG §§ 45, 46, 49** compliant reporting for ride-hailing and taxi operations
- Scheduled report generation for *Kraftfahrt-Bundesamt (KBA)*, *Bundesnetzagentur*, and regional *Verkehrsbehörden*
- XML/JSON report export in formats specified by German authorities
- Vehicle utilization and driver working time reports per *Arbeitszeitgesetz (ArbZG)*
- Quarterly and annual regulatory submission tracking

### GDPR / DSGVO Compliance
- Data subject rights management: right of access (*Auskunftsrecht*), rectification, erasure (*Recht auf Vergessenwerden*), portability, and restriction
- Automated Data Subject Access Request (DSAR) / *Auskunftsersuchen* workflow
- Consent management and consent withdrawal tracking
- Privacy Impact Assessment (PIA) / *Datenschutz-Folgenabschätzung (DSFA)* registry
- Lawful basis tracking for all data processing activities
- Cross-border data transfer records and Standard Contractual Clauses (SCC) registry

### Audit Logging
- Cryptographically signed, append-only audit log using HMAC-SHA256 chain
- Structured logging of all PII access events with actor, purpose, and legal basis
- Integration with all platform services via Kafka event stream
- Log integrity verification endpoints for internal audits
- Long-term archive with configurable retention per data category

### Data Retention & Lifecycle Management
- Configurable retention policies per data class (*Datenkategorie*) and legal basis
- Automated anonymization and pseudonymization pipelines
- Deletion verification with cryptographic receipts
- Legal hold management for ongoing litigation or investigations

### Compliance Monitoring & Alerting
- Real-time compliance posture dashboard
- SLA tracking for DSAR response deadlines (30-day statutory limit)
- Automated alerting for approaching deadlines and policy violations
- Integration with Prometheus and Grafana for compliance metrics

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    API Gateway / Kong                   │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────┐
│                  Compliance Service                     │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │  PBefG      │  │  GDPR/DSGVO  │  │  Audit Log    │  │
│  │  Reporter   │  │  Manager     │  │  Engine       │  │
│  └─────────────┘  └──────────────┘  └───────────────┘  │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │  Retention  │  │  Consent     │  │  Reporting    │  │
│  │  Manager    │  │  Tracker     │  │  Scheduler    │  │
│  └─────────────┘  └──────────────┘  └───────────────┘  │
└───────┬─────────────────────┬───────────────────────────┘
        │                     │
┌───────▼──────┐    ┌─────────▼────────────────────────┐
│  PostgreSQL  │    │           Apache Kafka            │
│  (Primary)   │    │  compliance.audit / dsgvo.events  │
└───────┬──────┘    └──────────────────────────────────┘
        │
┌───────▼──────┐
│    Redis     │
│  (Cache /    │
│   Sessions)  │
└──────────────┘
```

---

## Getting Started

### Prerequisites

- Node.js >= 18.x LTS
- PostgreSQL >= 14.x
- Redis >= 7.x
- Apache Kafka >= 3.x
- Docker & Docker Compose (for local development)

### Local Development Setup

```bash
# Clone the repository
git clone https://github.com/your-org/compliance-service.git
cd compliance-service

# Install dependencies
npm install

# Copy environment configuration
cp .env.example .env

# Start infrastructure dependencies
docker-compose up -d postgres redis kafka zookeeper

# Run database migrations
npm run db:migrate

# Seed reference data (retention policies, regulatory templates)
npm run db:seed

# Start the service in development mode
npm run dev
```

The service will be available at `http://localhost:3005`.

### Health Check

```bash
curl http://localhost:3005/health
```

```json
{
  "status": "healthy",
  "version": "2.4.1",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "kafka": "ok",
    "auditLogChain": "ok"
  }
}
```

---

## Environment Variables

### Application

| Variable | Required | Default | Description |
|---|---|---|---|
| `NODE_ENV` | Yes | `development` | Runtime environment (`development`, `staging`, `production`) |
| `PORT` | No | `3005` | HTTP server port |
| `SERVICE_NAME` | Yes | — | Service identifier for audit logs (`compliance-service`) |
| `LOG_LEVEL` | No | `info` | Logging verbosity (`error`, `warn`, `info`, `debug`) |
| `API_KEY_SECRET` | Yes | — | Secret for internal service-to-service API key validation |
| `JWT_PUBLIC_KEY` | Yes | — | RSA public key for JWT validation (PEM format) |

### Database

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string (`postgresql://user:pass@host:5432/compliance_db`) |
| `DATABASE_SSL` | No | `true` | Enforce SSL for database connections |
| `DATABASE_POOL_MIN` | No | `2` | Minimum database connection pool size |
| `DATABASE_POOL_MAX` | No | `10` | Maximum database connection pool size |
| `DATABASE_REPLICA_URL` | No | — | Read replica connection string for reporting queries |

### Redis

| Variable | Required | Default | Description |
|---|---|---|---|
| `REDIS_URL` | Yes | — | Redis connection URL (`redis://localhost:6379`) |
| `REDIS_TLS` | No | `true` | Enable TLS for Redis connections in production |
| `CACHE_TTL_SECONDS` | No | `300` | Default cache time-to-live in seconds |

### Kafka

| Variable | Required | Default | Description |
|---|---|---|---|
| `KAFKA_BROKERS` | Yes | — | Comma-separated list of Kafka broker addresses |
| `KAFKA_CLIENT_ID` | Yes | — | Kafka client identifier (`compliance-service`) |
| `KAFKA_GROUP_ID` | Yes | — | Consumer group ID (`compliance-service-group`) |
| `KAFKA_SSL` | No | `true` | Enable SSL for Kafka connections |
| `KAFKA_SASL_USERNAME` | No | — | SASL username for Kafka authentication |
| `KAFKA_SASL_PASSWORD` | No | — | SASL password for Kafka authentication |
| `KAFKA_SASL_MECHANISM` | No | `scram-sha-256` | SASL mechanism (`plain`, `scram-sha-256`, `scram-sha-512`) |

### Regulatory Reporting

| Variable | Required | Default | Description |
|---|---|---|---|
| `KBA_API_ENDPOINT` | Yes | — | Kraftfahrt-Bundesamt API endpoint for electronic submissions |
| `KBA_API_KEY` | Yes | — | API key for KBA electronic reporting portal |
| `KBA_OPERATOR_ID` | Yes | — | Assigned operator ID from KBA registration |
| `VERKEHRSBEHOERDE_REGION` | Yes | — | Regional transport authority code (e.g., `DE-BY` for Bavaria) |
| `PBEFG_REPORT_SCHEDULE` | No | `0 2 1 * *` | Cron schedule for PBefG monthly reports |
| `ANNUAL_REPORT_SCHEDULE` | No | `0 3 1 1 *` | Cron schedule for annual regulatory submissions |

### GDPR / DSGVO

| Variable | Required | Default | Description |
|---|---|---|---|
| `DSAR_RESPONSE_DEADLINE_DAYS` | No | `30` | Statutory deadline for DSAR responses in days |
| `DSAR_ALERT_THRESHOLD_DAYS` | No | `7` | Days before deadline to trigger escalation alerts |
| `DPO_EMAIL` | Yes | — | Data Protection Officer (*Datenschutzbeauftragter*) email address |
| `DPO_NAME` | Yes | — | Name of the appointed Data Protection Officer |
| `COMPANY_LEGAL_NAME` | Yes | — | Full legal name of the data controller (*Verantwortlicher*) |
| `COMPANY_REGISTERED_ADDRESS` | Yes | — | Registered address of the data controller |
| `SUPERVISORY_AUTHORITY` | Yes | — | Competent supervisory authority (*Aufsichtsbehörde*) code |

### Audit Logging

| Variable | Required | Default | Description |
|---|---|---|---|
| `AUDIT_HMAC_SECRET` | Yes | — | Secret key for HMAC-SHA256 audit log chain signing (min 256 bits) |
| `AUDIT_LOG_RETENTION_DAYS` | No | `3650` | Audit log retention period in days (10 years default per HGB) |
| `AUDIT_ARCHIVE_BUCKET` | No | — | S3-compatible bucket for long-term audit log archival |
| `AUDIT_ARCHIVE_ENDPOINT` | No | — | S3-compatible endpoint (e.g., for MinIO) |

### Encryption

| Variable | Required | Default | Description |
|---|---|---|---|
| `ENCRYPTION_KEY` | Yes | — | AES-256 encryption key for sensitive fields at rest (base64-encoded) |
| `ENCRYPTION_KEY_ID` | Yes | — | Key identifier for key rotation tracking |
| `KMS_ENDPOINT` | No | — | AWS KMS or compatible endpoint for envelope encryption |

### Notifications

| Variable | Required | Default | Description |
|---|---|---|---|
| `SMTP_HOST` | Yes | — | SMTP server hostname for compliance notifications |
| `SMTP_PORT` | No | `587` | SMTP server port |
| `SMTP_USER` | Yes | — | SMTP authentication username |
| `SMTP_PASSWORD` | Yes | — | SMTP authentication password |
| `NOTIFICATION_FROM_EMAIL` | Yes | — | Sender email address for compliance notifications |
| `SLACK_WEBHOOK_URL` | No | — | Slack webhook for compliance alert notifications |

---

## API Endpoints

All endpoints require a valid JWT Bearer token in the `Authorization` header unless otherwise noted. Internal service-to-service calls use an additional `X-API-Key` header.

Base URL: `https://api.yourplatform.de/compliance/v1`

### Health & Status

#### `GET /health`

Returns service health status. No authentication required.

**Response:**
```json
{
  "status": "healthy",
  "version": "2.4.1",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "kafka": "ok",
    "auditLogChain": "ok"
  }
}
```

#### `GET /metrics`

Prometheus metrics endpoint. Restricted to internal network.

---

### PBefG Reporting

#### `POST /reports/pbefg/generate`

Triggers generation of a PBefG compliance report for a specified period.

**Authorization:** `ROLE_COMPLIANCE_ADMIN` or `ROLE_REGULATORY_OFFICER`

**Request Body:**
```json
{
  "reportType": "MONTHLY",
  "period": {
    "year": 2024,
    "month": 1
  },
  "scope": {
    "region": "DE-BY",
    "vehicleClasses": ["TAXI", "MIETWAGEN"],
    "includeDriverHours": true
  },
  "submissionTarget": "KBA",
  "dryRun": false
}
```

**Response `201 Created`:**
```json
{
  "reportId": "rpt_01HQ4K7MNPBEFG2024JAN",
  "status": "GENERATING",
  "reportType": "MONTHLY",
  "period": {
    "year": 2024,
    "month": 1,
    "startDate": "2024-01-01",
    "endDate": "2024-01-31"
  },
  "estimatedCompletionTime": "2024-02-01T02:05:00.000Z",
  "checkStatusUrl": "/reports/pbefg/rpt_01HQ4K7MNPBEFG2024JAN",
  "createdBy": "usr_compliance_officer_01",
  "createdAt": "2024-02-01T02:00:00.000Z"
}
```

#### `GET /reports/pbefg/:reportId`

Retrieves status and details of a specific PBefG report.

**Authorization:** `ROLE_COMPLIANCE_ADMIN` or `ROLE_REGULATORY_OFFICER`

**Response `200 OK`:**
```json
{
  "reportId": "rpt_01HQ4K7MNPBEFG2024JAN",
  "status": "COMPLETED",
  "reportType": "MONTHLY",
  "period": {
    "year": 2024,
    "month": 1
  },
  "summary": {
    "totalTrips": 142567,
    "totalVehicles": 834,
    "totalDrivers": 1203,
    "totalPassengerKilometers": 2847392.5,
    "complianceScore": 98.7,
    "violations": 3
  },
  "submissionStatus": "SUBMITTED",
  "submittedAt": "2024-02-01T03:10:00.000Z",
  "kbaAcknowledgmentId": "KBA-2024-BY-001-20240201",
  "downloadUrl": "/reports/pbefg/rpt_01HQ4K7MNPBEFG2024JAN/download",
  "createdAt": "2024-02-01T02:00:00.000Z",
  "completedAt": "2024-02-01T02:08:43.000Z"
}
```

#### `GET /reports/pbefg`

Lists PBefG reports with filtering and pagination.

**Query Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `status` | `string` | Filter by status: `GENERATING`, `COMPLETED`, `FAILED`, `SUBMITTED` |
| `reportType` | `string` | `MONTHLY`, `QUARTERLY`, `ANNUAL` |
| `year` | `number` | Filter by report year |
| `month` | `number` | Filter by report month |
| `limit` | `number` | Results per page (default: 20, max: 100) |
| `offset` | `number` | Pagination offset |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "reportId": "rpt_01HQ4K7MNPBEFG2024JAN",
      "status": "SUBMITTED",
      "reportType": "MONTHLY",
      "period": { "year": 2024, "month": 1 },
      "submissionStatus": "SUBMITTED",
      "createdAt": "2024-02-01T02:00:00.000Z"
    }
  ],
  "pagination": {
    "total": 24,
    "limit": 20,
    "offset": 0,
    "hasMore": true
  }
}
```

#### `GET /reports/pbefg/:reportId/download`

Downloads the generated report in the specified format.

**Query Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `format` | `string` | `XML` (KBA standard), `JSON`, `PDF` |

**Response:** Binary file download with appropriate `Content-Type` header.

---

### GDPR / DSGVO — Data Subject Rights

#### `POST /dsgvo/dsar`

Creates a new Data Subject Access Request (*Auskunftsersuchen*). Initiates the 30-day statutory response workflow.

**Authorization:** `ROLE_SUPPORT`, `ROLE_COMPLIANCE_ADMIN`, or authenticated data subject (via customer JWT)

**Request Body:**
```json
{
  "subjectType": "CUSTOMER",
  "subjectId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
  "requestType": "ACCESS",
  "requestChannel": "EMAIL",
  "contactEmail": "max.mustermann@example.de",
  "requestDetails": "Ich beantrage gemäß Art. 15 DSGVO Auskunft über alle zu meiner Person gespeicherten Daten.",
  "identityVerified": true,
  "identityVerificationMethod": "EMAIL_OTP",
  "requestedDataCategories": ["ACCOUNT", "TRIPS", "PAYMENT", "LOCATION", "CONSENT"],
  "locale": "de-DE"
}
```

**Response `201 Created`:**
```json
{
  "dsarId": "dsar_01HQ8XMTNZ4VWCFK7PQRB3Y5J",
  "requestType": "ACCESS",
  "status": "RECEIVED",
  "subjectId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
  "legalBasis": "DSGVO Art. 15 Abs. 1",
  "receivedAt": "2024-01-15T10:30:00.000Z",
  "statutoryDeadline": "2024-02-14T23:59:59.000Z",
  "estimatedCompletionDate": "2024-01-29T17:00:00.000Z",
  "referenceNumber": "DSAR-2024-0042",
  "confirmationEmailSent": true,
  "dpoNotified": true,
  "trackingUrl": "/dsgvo/dsar/dsar_01HQ8XMTNZ4VWCFK7PQRB3Y5J/status"
}
```

#### `GET /dsgvo/dsar/:dsarId`

Retrieves the full details and current status of a DSAR.

**Response `200 OK`:**
```json
{
  "dsarId": "dsar_01HQ8XMTNZ4VWCFK7PQRB3Y5J",
  "referenceNumber": "DSAR-2024-0042",
  "requestType": "ACCESS",
  "status": "DATA_COMPILED",
  "subjectId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
  "requestDetails": "Ich beantrage gemäß Art. 15 DSGVO...",
  "legalBasis": "DSGVO Art. 15 Abs. 1",
  "receivedAt": "2024-01-15T10:30:00.000Z",
  "statutoryDeadline": "2024-02-14T23:59:59.000Z",
  "daysRemaining": 21,
  "isAtRisk": false,
  "timeline": [
    {
      "event": "REQUEST_RECEIVED",
      "timestamp": "2024-01-15T10:30:00.000Z",
      "actor": "system"
    },
    {
      "event": "IDENTITY_VERIFIED",
      "timestamp": "2024-01-15T10:31:00.000Z",
      "actor": "system"
    },
    {
      "event": "DATA_COMPILATION_STARTED",
      "timestamp": "2024-01-15T10:32:00.000Z",
      "actor": "system"
    },
    {
      "event": "DATA_COMPILED",
      "timestamp": "2024-01-15T11:45:00.000Z",
      "actor": "system"
    }
  ],
  "dataPackageReady": true,
  "dataPackageUrl": "/dsgvo/dsar/dsar_01HQ8XMTNZ4VWCFK7PQRB3Y5J/data-package",
  "assignedTo": "usr_dpo_officer_01"
}
```

#### `GET /dsgvo/dsar/:dsarId/data-package`

Downloads the compiled personal data package for a completed DSAR. Returns a password-protected ZIP archive.

**Authorization:** `ROLE_COMPLIANCE_ADMIN`, `ROLE_DPO`, or authenticated data subject

**Response:** Binary ZIP file download. Password communicated to data subject via separate secure channel.

#### `POST /dsgvo/erasure`

Initiates a data erasure request (*Recht auf Vergessenwerden*) per DSGVO Art. 17.

**Request Body:**
```json
{
  "subjectId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
  "subjectType": "CUSTOMER",
  "erasureScope": "FULL",
  "legalBasisForRefusal": null,
  "retainForLegalObligation": false,
  "requestedBy": "DATA_SUBJECT",
  "verificationToken": "tok_erasure_confirm_abc123"
}
```

**Response `202 Accepted`:**
```json
{
  "erasureRequestId": "era_01HR2KPNM8VXZQW4TLFCB9D6S",
  "status": "PENDING_REVIEW",
  "subjectId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
  "estimatedCompletionDate": "2024-01-22T17:00:00.000Z",
  "legalHoldsFound": false,
  "retentionObligationsFound": true,
  "retentionDetails": [
    {
      "dataCategory": "PAYMENT_RECORDS",
      "retentionBasis": "§ 257 HGB, § 147 AO",
      "retentionUntil": "2033-12-31",
      "action": "PSEUDONYMIZE_AND_RETAIN"
    }
  ],
  "anonymizationScheduled": true,
  "confirmationEmailSent": true
}
```

#### `GET /dsgvo/dsar`

Lists all DSARs with filtering.

**Query Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `status` | `string` | `RECEIVED`, `IN_PROGRESS`, `DATA_COMPILED`, `SENT`, `CLOSED`, `OVERDUE` |
| `requestType` | `string` | `ACCESS`, `ERASURE`, `RECTIFICATION`, `PORTABILITY`, `RESTRICTION` |
| `isAtRisk` | `boolean` | Filter requests approaching or past their statutory deadline |
| `assignedTo` | `string` | Filter by assigned officer user ID |
| `limit` | `number` | Results per page |
| `offset` | `number` | Pagination offset |

#### `PATCH /dsgvo/dsar/:dsarId`

Updates the status or assignment of a DSAR.

**Request Body:**
```json
{
  "status": "SENT",
  "resolution": "Data package delivered to data subject via secure email on 2024-01-24.",
  "sentAt": "2024-01-24T14:30:00.000Z",
  "deliveryMethod": "SECURE_EMAIL"
}
```

---

### Consent Management

#### `POST /consent`

Records a new consent decision for a data subject.

**Request Body:**
```json
{
  "subjectId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
  "subjectType": "CUSTOMER",
  "consentPurposes": [
    {
      "purposeId": "MARKETING_EMAIL",
      "granted": true,
      "version": "2024-01-v1"
    },
    {
      "purposeId": "LOCATION_ANALYTICS",
      "granted": false,
      "version": "2024-01-v1"
    }
  ],
  "capturedAt": "2024-01-15T10:30:00.000Z",
  "capturedVia": "MOBILE_APP",
  "ipAddress": "192.168.1.1",
  "userAgent": "MobilityApp/3.2.1 iOS/17.2",
  "consentDocumentVersion": "2024-01-v1"
}
```

**Response `201 Created`:**
```json
{
  "consentRecordId": "cns_01HQ9YTMRZ5VXBFL8PMQD4K7H",
  "subjectId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
  "recordedAt": "2024-01-15T10:30:00.000Z",
  "purposes": [
    { "purposeId": "MARKETING_EMAIL", "granted": true, "recordedAt": "2024-01-15T10:30:00.000Z" },
    { "purposeId": "LOCATION_ANALYTICS", "granted": false, "recordedAt": "2024-01-15T10:30:00.000Z" }
  ],
  "checksum": "sha256:a7f3b2c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9"
}
```

#### `GET /consent/:subjectId/current`

Returns the current effective consent state for a data subject.

**Response `200 OK`:**
```json
{
  "subjectId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
  "effectiveConsent": {
    "MARKETING_EMAIL": { "granted": true, "since": "2024-01-15T10:30:00.000Z", "version": "2024-01-v1" },
    "LOCATION_ANALYTICS": { "granted": false, "since": "2024-01-15T10:30:00.000Z", "version": "2024-01-v1" },
    "THIRD_PARTY_SHARING": { "granted": false, "since": "2023-09-01T00:00:00.000Z", "version": "2023-09-v2" }
  },
  "lastUpdated": "2024-01-15T10:30:00.000Z",
  "consentDocumentVersion": "2024-01-v1",
  "requiresRenewal": false
}
```

#### `GET /consent/:subjectId/history`

Returns the full immutable consent history for a data subject.

---

### Audit Log

#### `GET /audit-log`

Queries the audit log with filtering. All queries are themselves logged.

**Authorization:** `ROLE_COMPLIANCE_ADMIN`, `ROLE_DPO`, `ROLE_AUDITOR`

**Query Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `subjectId` | `string` | Filter by data subject ID |
| `actorId` | `string` | Filter by actor (user or service) performing the action |
| `eventType` | `string` | Filter by event type (see Event Types below) |
| `resourceType` | `string` | `CUSTOMER`, `DRIVER`, `TRIP`, `PAYMENT`, `VEHICLE` |
| `startDate` | `ISO8601` | Filter events from this date |
| `endDate` | `ISO8601` | Filter events until this date |
| `limit` | `number` | Results per page (max: 500) |
| `cursor` | `string` | Pagination cursor for keyset pagination |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "auditId": "aud_01HQBXMNPZ4VWCFK7TQRB3Y5J",
      "eventType": "PII_ACCESS",
      "eventTimestamp": "2024-01-15T10:30:00.000Z",
      "actorId": "usr_support_agent_07",
      "actorType": "HUMAN",
      "actorIpAddress": "10.0.1.45",
      "resourceType": "CUSTOMER",
      "resourceId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
      "action": "READ",
      "fieldsAccessed": ["fullName", "email", "phoneNumber"],
      "legalBasis": "CONTRACT_PERFORMANCE",
      "purpose": "CUSTOMER_SUPPORT",
      "serviceOrigin": "customer-service",
      "sessionId": "sess_abc123",
      "chainHash": "sha256:f7e6d5c4b3a2918273645566778899aabbccddeeff",
      "integrityVerified": true
    }
  ],
  "pagination": {
    "total": 15842,
    "limit": 100,
    "nextCursor": "cur_eyJpZCI6ImF1ZF8wMUhRQlhNTlBaNCJ9",
    "hasMore": true
  }
}
```

#### `POST /audit-log/verify`

Verifies the integrity of the audit log chain for a specified range.

**Request Body:**
```json
{
  "startAuditId": "aud_01HQBXMNPZ4VWCFK7TQRB3Y5J",
  "endAuditId": "aud_01HQCXMNPZ9VWZFK7TQRB3Y5K",
  "expectedEntryCount": 1000
}
```

**Response `200 OK`:**
```json
{
  "verificationId": "ver_01HQDXMNPZ4VWCFK7PQRB3Y5L",
  "status": "VERIFIED",
  "entriesVerified": 1000,
  "chainIntact": true,
  "tamperingDetected": false,
  "startHash": "sha256:a1b2c3d4e5f6...",
  "endHash": "sha256:f9e8d7c6b5a4...",
  "verifiedAt": "2024-01-15T12:00:00.000Z"
}
```

#### Audit Log Event Types

| Event Type | Description |
|---|---|
| `PII_ACCESS` | Personal data was read by a human or service |
| `PII_MODIFICATION` | Personal data was created or updated |
| `PII_DELETION` | Personal data was deleted or anonymized |
| `DSAR_CREATED` | A data subject access request was received |
| `DSAR_FULFILLED` | A DSAR response was dispatched to the subject |
| `CONSENT_GRANTED` | Explicit consent was recorded |
| `CONSENT_WITHDRAWN` | Consent was revoked by the data subject |
| `AUTHENTICATION_SUCCESS` | Successful login event |
| `AUTHENTICATION_FAILURE` | Failed authentication attempt |
| `AUTHORIZATION_DENIED` | Access to a resource was denied |
| `DATA_EXPORT` | Data was exported from the system |
| `DATA_BREACH_SUSPECTED` | A potential data breach indicator was detected |
| `REGULATORY_REPORT_GENERATED` | A PBefG or other regulatory report was generated |
| `REGULATORY_REPORT_SUBMITTED` | A regulatory report was submitted to an authority |
| `LEGAL_HOLD_APPLIED` | A legal hold was placed on data |
| `RETENTION_POLICY_APPLIED` | A data retention or deletion policy was executed |

---

### Retention Policies

#### `GET /retention/policies`

Returns all configured data retention policies.

**Response `200 OK`:**
```json
{
  "policies": [
    {
      "policyId": "pol_trip_records",
      "name": "Trip Records Retention",
      "dataCategory": "TRIP_RECORDS",
      "retentionPeriodDays": 3650,
      "legalBasis": "§ 257 HGB — Handelsrechtliche Aufbewahrungspflicht",
      "actionOnExpiry": "ANONYMIZE",
      "applies To": ["trips", "route_data"],
      "lastReviewedAt": "2024-01-01T00:00:00.000Z",
      "reviewDueAt": "2025-01-01T00:00:00.000Z",
      "approvedBy": "DPO"
    },
    {
      "policyId": "pol_payment_records",
      "name": "Payment Records Retention",
      "dataCategory": "PAYMENT_RECORDS",
      "retentionPeriodDays": 3650,
      "legalBasis": "§ 257 HGB, § 147 AO — Steuerliche Aufbewahrungspflicht (10 Jahre)",
      "actionOnExpiry": "DELETE",
      "appliesTo": ["invoices", "payment_transactions"],
      "lastReviewedAt": "2024-01-01T00:00:00.000Z"
    },
    {
      "policyId": "pol_location_data",
      "name": "Location Data Retention",
      "dataCategory": "PRECISE_LOCATION",
      "retentionPeriodDays": 90,
      "legalBasis": "DSGVO Art. 5 Abs. 1 lit. e — Speicherbegrenzung",
      "actionOnExpiry": "DELETE",
      "appliesTo": ["gps_tracks", "pickup_locations"],
      "lastReviewedAt": "2024-01-01T00:00:00.000Z"
    }
  ]
}
```

#### `GET /retention/schedule`

Returns upcoming scheduled data deletion and anonymization jobs.

#### `POST /retention/dry-run`

Simulates a retention policy execution without making changes.

---

### Data Breach Management

#### `POST /breach/report`

Reports a suspected or confirmed personal data breach (*Datenpanne*). Initiates the 72-hour notification workflow to the supervisory authority as required by DSGVO Art. 33.

**Authorization:** `ROLE_COMPLIANCE_ADMIN`, `ROLE_DPO`, `ROLE_SECURITY_OFFICER`

**Request Body:**
```json
{
  "discoveredAt": "2024-01-15T08:00:00.000Z",
  "breachType": "UNAUTHORIZED_ACCESS",
  "severity": "HIGH",
  "affectedDataCategories": ["EMAIL", "PHONE_NUMBER", "TRIP_HISTORY"],
  "estimatedAffectedSubjects": 1500,
  "affectedSubjectTypes": ["CUSTOMER"],
  "description": "Unauthorized API access detected via compromised service account credentials.",
  "immediateActionsToken": [
    "Service account credentials revoked at 08:05",
    "Affected API endpoints taken offline at 08:10",
    "Security team notified at 08:15"
  ],
  "likelyToResultInHighRisk": true,
  "containedAt": "2024-01-15T09:30:00.000Z"
}
```

**Response `201 Created`:**
```json
{
  "breachId": "brc_01HQ3PMNRZ8VWCFK7TQRB3Y5J",
  "status": "UNDER_ASSESSMENT",
  "supervisoryAuthorityNotificationDeadline": "2024-01-18T08:00:00.000Z",
  "hoursRemainingToNotify": 72,
  "subjectNotificationRequired": true,
  "subjectNotificationDeadline": "2024-01-22T08:00:00.000Z",
  "dpoAlerted": true,
  "supervisoryAuthority": "Bayerisches Landesamt für Datenschutzaufsicht (BayLDA)",
  "supervisoryAuthorityPortalUrl": "https://www.lda.bayern.de/meldung",
  "referenceNumber": "BREACH-2024-001",
  "createdAt": "2024-01-15T10:30:00.000Z"
}
```

---

### Processing Register (Verzeichnis der Verarbeitungstätigkeiten)

#### `GET /processing-register`

Returns the *Verzeichnis der Verarbeitungstätigkeiten* (Record of Processing Activities) as required by DSGVO Art. 30.

**Response `200 OK`:**
```json
{
  "controller": {
    "name": "Mobility Platform GmbH",
    "address": "Musterstraße 1, 80331 München, Deutschland",
    "contact": "datenschutz@mobilityplatform.de"
  },
  "dpo": {
    "name": "Dr. Anna Datenschutz",
    "contact": "dpo@mobilityplatform.de"
  },
  "processingActivities": [
    {
      "activityId": "proc_001",
      "name": "Fahrgastbeförderung und Buchungsabwicklung",
      "purpose": "Vertragserfüllung für Buchung und Durchführung von Fahrten",
      "legalBasis": "Art. 6 Abs. 1 lit. b DSGVO (Vertragserfüllung)",
      "dataCategories": ["Name", "Kontaktdaten", "Standortdaten", "Zahlungsdaten"],
      "dataSubjects": ["Fahrgäste", "Fahrer"],
      "recipients": ["Zahlungsdienstleister", "Versicherungen"],
      "thirdCountryTransfers": false,
      "retentionPeriod": "10 Jahre (§ 257 HGB)",
      "technicalMeasures": ["AES-256 Verschlüsselung", "TLS 1.3", "Zugriffskontrolle"]
    }
  ],
  "lastUpdated": "2024-01-01T00:00:00.000Z",
  "version": "2024-1"
}
```

#### `GET /processing-register/export`

Exports the processing register as a PDF document suitable for submission to supervisory authorities.

---

## Database Schema

The Compliance Service uses PostgreSQL with row-level security enabled for sensitive tables.

### Tables Overview

#### `pbefg_reports`
Stores generated PBefG regulatory reports.

```sql
CREATE TABLE pbefg_reports (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  report_id     TEXT UNIQUE NOT NULL,
  report_type   TEXT NOT NULL CHECK (report_type IN ('MONTHLY', 'QUARTERLY', 'ANNUAL')),
  period_year   SMALLINT NOT NULL,
  period_month  SMALLINT,
  period_quarter SMALLINT,
  region        TEXT NOT NULL,
  status        TEXT NOT NULL DEFAULT 'GENERATING',
  summary       JSONB,
  report_data   BYTEA,               -- Encrypted report payload
  submission_status TEXT DEFAULT 'PENDING',
  submission_target TEXT,
  submitted_at  TIMESTAMPTZ,
  kba_acknowledgment_id TEXT,
  file_path     TEXT,
  created_by    TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at  TIMESTAMPTZ,
  CONSTRAINT valid_period CHECK (
    (report_type = 'MONTHLY' AND period_month IS NOT NULL) OR
    (report_type = 'QUARTERLY' AND period_quarter IS NOT NULL) OR
    (report_type = 'ANNUAL')
  )
);

CREATE INDEX idx_pbefg_reports_period ON pbefg_reports (period_year, period_month);
CREATE INDEX idx_pbefg_reports_status ON pbefg_reports (status);
```

#### `dsar_requests`
Manages Data Subject Access Requests and their lifecycle.

```sql
CREATE TABLE dsar_requests (
  id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  dsar_id                     TEXT UNIQUE NOT NULL,
  reference_number            TEXT UNIQUE NOT NULL,
  subject_id                  TEXT NOT NULL,
  subject_type                TEXT NOT NULL,
  request_type                TEXT NOT NULL CHECK (request_type IN (
    'ACCESS', 'ERASURE', 'RECTIFICATION', 'PORTABILITY', 'RESTRICTION', 'OBJECTION'
  )),
  status                      TEXT NOT NULL DEFAULT 'RECEIVED',
  contact_email               TEXT NOT NULL,
  request_details             TEXT,
  legal_basis                 TEXT,
  identity_verified           BOOLEAN NOT NULL DEFAULT FALSE,
  identity_verification_method TEXT,
  requested_data_categories   TEXT[],
  received_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  statutory_deadline          TIMESTAMPTZ NOT NULL,
  estimated_completion_date   TIMESTAMPTZ,
  assigned_to                 TEXT,
  data_package_path           TEXT,
  data_package_ready          BOOLEAN DEFAULT FALSE,
  resolution                  TEXT,
  sent_at                     TIMESTAMPTZ,
  closed_at                   TIMESTAMPTZ,
  request_channel             TEXT,
  locale                      TEXT DEFAULT 'de-DE',
  created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dsar_subject ON dsar_requests (subject_id, subject_type);
CREATE INDEX idx_dsar_status ON dsar_requests (status);
CREATE INDEX idx_dsar_deadline ON dsar_requests (statutory_deadline) WHERE status NOT IN ('CLOSED', 'REJECTED');
```

#### `audit_log`
Immutable append-only audit log with HMAC chain integrity.

```sql
CREATE TABLE audit_log (
  id                  BIGSERIAL PRIMARY KEY,
  audit_id            TEXT UNIQUE NOT NULL,
  event_type          TEXT NOT NULL,
  event_timestamp     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  actor_id            TEXT NOT NULL,
  actor_type          TEXT NOT NULL CHECK (actor_type IN ('HUMAN', 'SERVICE', 'SYSTEM')),
  actor_ip_address    INET,
  actor_user_agent    TEXT,
  resource_type       TEXT,
  resource_id         TEXT,
  action              TEXT NOT NULL,
  fields_accessed     TEXT[],
  legal_basis         TEXT,
  purpose             TEXT,
  service_origin      TEXT NOT NULL,
  session_id          TEXT,
  request_id          TEXT,
  event_data          JSONB,
  previous_chain_hash TEXT,
  chain_hash          TEXT NOT NULL,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (event_timestamp);

-- Monthly partitions for performance
CREATE TABLE audit_log_2024_01 PARTITION OF audit_log
  FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Enforce immutability
CREATE RULE no_update_audit AS ON UPDATE TO audit_log DO INSTEAD NOTHING;
CREATE RULE no_delete_audit AS ON DELETE TO audit_log DO INSTEAD NOTHING;

CREATE INDEX idx_audit_event_type ON audit_log (event_type, event_timestamp DESC);
CREATE INDEX idx_audit_subject ON audit_log (resource_id, resource_type);
CREATE INDEX idx_audit_actor ON audit_log (actor_id, event_timestamp DESC);
```

#### `consent_records`
Immutable consent history per data subject.

```sql
CREATE TABLE consent_records (
  id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  consent_record_id           TEXT UNIQUE NOT NULL,
  subject_id                  TEXT NOT NULL,
  subject_type                TEXT NOT NULL,
  purpose_id                  TEXT NOT NULL,
  granted                     BOOLEAN NOT NULL,
  consent_document_version    TEXT NOT NULL,
  captured_at                 TIMESTAMPTZ NOT NULL,
  captured_via                TEXT NOT NULL,
  ip_address                  INET,
  user_agent                  TEXT,
  withdrawn_at                TIMESTAMPTZ,
  withdrawal_method           TEXT,
  checksum                    TEXT NOT NULL,
  created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_consent_subject ON consent_records (subject_id, purpose_id, captured_at DESC);
```

#### `data_breach_records`

```sql
CREATE TABLE data_breach_records (
  id                                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  breach_id                             TEXT UNIQUE NOT NULL,
  reference_number                      TEXT UNIQUE NOT NULL,
  discovered_at                         TIMESTAMPTZ NOT NULL,
  breach_type                           TEXT NOT NULL,
  severity                              TEXT NOT NULL,
  status                                TEXT NOT NULL DEFAULT 'UNDER_ASSESSMENT',
  affected_data_categories              TEXT[],
  estimated_affected_subjects           INTEGER,
  affected_subject_types                TEXT[],
  description                           TEXT NOT NULL,
  immediate_actions                     TEXT[],
  likely_to_result_in_high_risk         BOOLEAN DEFAULT FALSE,
  contained_at                          TIMESTAMPTZ,
  supervisory_authority_notified_at     TIMESTAMPTZ,
  supervisory_authority_reference       TEXT,
  subject_notification_required         BOOLEAN DEFAULT FALSE,
  subjects_notified_at                  TIMESTAMPTZ,
  root_cause                            TEXT,
  remediation_steps                     TEXT,
  closed_at                             TIMESTAMPTZ,
  created_at                            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### `retention_policies`

```sql
CREATE TABLE retention_policies (
  id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  policy_id               TEXT UNIQUE NOT NULL,
  name                    TEXT NOT NULL,
  data_category           TEXT NOT NULL UNIQUE,
  retention_period_days   INTEGER NOT NULL,
  legal_basis             TEXT NOT NULL,
  action_on_expiry        TEXT NOT NULL CHECK (action_on_expiry IN ('DELETE', 'ANONYMIZE', 'PSEUDONYMIZE', 'ARCHIVE')),
  applies_to_tables       TEXT[],
  is_active               BOOLEAN DEFAULT TRUE,
  approved_by             TEXT,
  last_reviewed_at        TIMESTAMPTZ,
  review_due_at           TIMESTAMPTZ,
  created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### `legal_holds`

```sql
CREATE TABLE legal_holds (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  hold_id         TEXT UNIQUE NOT NULL,
  subject_id      TEXT,
  resource_type   TEXT,
  resource_id     TEXT,
  reason          TEXT NOT NULL,
  requested_by    TEXT NOT NULL,
  approved_by     TEXT,
  active          BOOLEAN DEFAULT TRUE,
  applied_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at      TIMESTAMPTZ,
  released_at     TIMESTAMPTZ,
  released_by     TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## Kafka Events

### Topics Produced

The Compliance Service produces events to the following Kafka topics:

#### `compliance.audit.events`

Published for every auditable action across the platform. All platform services should publish PII access events here.

**Message Schema:**
```json
{
  "eventId": "evt_01HQ4KMNPZ4VWCFK7TQRB3Y5J",
  "eventType": "PII_ACCESS",
  "eventTimestamp": "2024-01-15T10:30:00.000Z",
  "schemaVersion": "1.0",
  "source": "compliance-service",
  "payload": {
    "auditId": "aud_01HQBXMNPZ4VWCFK7TQRB3Y5J",
    "actorId": "usr_support_agent_07",
    "resourceType": "CUSTOMER",
    "resourceId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
    "action": "READ",
    "chainHash": "sha256:f7e6d5c4b3a2..."
  }
}
```

#### `compliance.dsar.created`

Published when a new DSAR is received. Consumed by all services holding personal data to initiate data compilation.

```json
{
  "eventId": "evt_01HQ8XMTNZ4VWCFK7PQRB3Y5J",
  "eventType": "DSAR_CREATED",
  "eventTimestamp": "2024-01-15T10:30:00.000Z",
  "payload": {
    "dsarId": "dsar_01HQ8XMTNZ4VWCFK7PQRB3Y5J",
    "subjectId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
    "subjectType": "CUSTOMER",
    "requestType": "ACCESS",
    "requestedDataCategories": ["ACCOUNT", "TRIPS", "PAYMENT"],
    "statutoryDeadline": "2024-02-14T23:59:59.000Z",
    "callbackTopic": "compliance.dsar.data.response"
  }
}
```

#### `compliance.erasure.initiated`

Published when a data erasure request is approved. All services must delete or anonymize the specified subject's data.

```json
{
  "eventId": "evt_01HR2KPNM8VXZQW4TLFCB9D6S",
  "eventType": "ERASURE_INITIATED",
  "eventTimestamp": "2024-01-15T12:00:00.000Z",
  "payload": {
    "erasureRequestId": "era_01HR2KPNM8VXZQW4TLFCB9D6S",
    "subjectId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
    "subjectType": "CUSTOMER",
    "erasureScope": "FULL",
    "exemptDataCategories": ["PAYMENT_RECORDS"],
    "exemptionReason": "§ 257 HGB retention obligation",
    "completionDeadline": "2024-01-22T17:00:00.000Z",
    "callbackTopic": "compliance.erasure.confirmation"
  }
}
```

#### `compliance.consent.updated`

Published when a data subject updates their consent preferences.

```json
{
  "eventId": "evt_01HQ9YTMRZ5VXBFL8PMQD4K7H",
  "eventType": "CONSENT_UPDATED",
  "eventTimestamp": "2024-01-15T10:30:00.000Z",
  "payload": {
    "subjectId": "cus_01HPZQ4M7NKDVBFX3QGTL9W2K",
    "subjectType": "CUSTOMER",
    "changes": [
      { "purposeId": "MARKETING_EMAIL", "granted": true },
      { "purposeId": "LOCATION_ANALYTICS", "granted": false }
    ],
    "effectiveAt": "2024-01-15T10:30:00.000Z"
  }
}
```

#### `compliance.breach.declared`

Published when a data breach is formally declared. Triggers incident response workflows.

#### `compliance.retention.execution`

Published when a retention policy is about to be executed. Gives downstream services advance notice to prepare.

---

### Topics Consumed

The Compliance Service subscribes to the following Kafka topics:

#### `compliance.dsar.data.response`

Consumed from all platform services providing data in response to a DSAR.

**Message Schema:**
```json
{
  "eventType": "DSAR_DATA_RESPONSE",
  "payload": {
    "dsarId": "dsar_01HQ8XMTNZ4VWCFK7PQRB3Y5J",
    "respondingService": "trip-service",
    "dataCategory": "TRIPS",
    "recordCount": 247,
    "dataPayload": { },
    "respondedAt": "2024-01-15T11:00:00.000Z"
  }
}
```

#### `compliance.erasure.confirmation`

Consumed from all platform services confirming data erasure completion.

```json
{
  "eventType": "ERASURE_CONFIRMED",
  "payload": {
    "erasureRequestId": "era_01HR2KPNM8VXZQW4TLFCB9D6S",
    "confirmingService": "trip-service",
    "recordsDeleted": 247,
    "recordsAnonymized": 0,
    "exemptions": [],
    "confirmedAt": "2024-01-15T13:00:00.000Z",
    "verificationHash": "sha256:abc123..."
  }
}
```

#### `platform.audit.ingest`

All platform services publish PII access and modification events to this topic. The Compliance Service ingests, validates, and stores them in the immutable audit log.

#### `platform.user.events`

Consumed to track authentication events, account creation, and deletions for audit purposes.

#### `platform.trip.completed`

Consumed to extract metrics required for PBefG regulatory reports.

#### `platform.driver.events`

Consumed to track driver working hours for *Arbeitszeitgesetz (ArbZG)* compliance.

---

## Deployment

### Docker

```bash
# Build the image
docker build -t compliance-service:latest .

# Run with environment file
docker run -d \
  --name compliance-service \
  -p 3005:3005 \
  --env-file .env.production \
  --restart unless-stopped \
  compliance-service:latest
```

### Docker Compose (Development)

```yaml
version: '3.8'

services:
  compliance-service:
    build: .
    ports:
      - "3005:3005"
    environment:
      - NODE_ENV=development
      - DATABASE_URL=postgresql://compliance:secret@postgres:5432/compliance_db
      - REDIS_URL=redis://redis:6379
      - KAFKA_BROKERS=kafka:9092
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      kafka:
        condition: service_healthy
    volumes:
      - ./src:/app/src
    command: npm run dev

  postgres:
    image: postgres:14-alpine
    environment:
      POSTGRES_DB: compliance_db
      POSTGRES_USER: compliance
      POSTGRES_PASSWORD: secret
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U compliance"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  kafka:
    image: confluentinc/cp-kafka:7.5.0
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
    depends_on:
      - zookeeper

  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181

volumes:
  postgres_data:
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: compliance-service
  namespace: platform
  labels:
    app: compliance-service
    component: regulatory
spec:
  replicas: 2
  selector:
    matchLabels:
      app: compliance-service
  template:
    metadata:
      labels:
        app: compliance-service
    spec:
      serviceAccountName: compliance-service
      securityContext:
        runAsNonRoot: true
        runAsUser: 1001
        fsGroup: 1001
      containers:
        - name: compliance-service
          image: your-registry/compliance-service:2.4.1
          imagePullPolicy: Always
          ports:
            - containerPort: 3005
              name: http
          env:
            - name: NODE_ENV
              value: production
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: compliance-secrets
                  key: database-url
            - name: AUDIT_HMAC_SECRET
              valueFrom:
                secretKeyRef:
                  name: compliance-secrets
                  key: audit-hmac-secret
            - name: ENCRYPTION_KEY
              valueFrom:
                secretKeyRef:
                  name: compliance-secrets
                  key: encryption-key
          resources:
            requests:
              memory: "256Mi"
              cpu: "100m"
            limits:
              memory: "1Gi"
              cpu: "500m"
          readinessProbe:
            httpGet:
              path: /health
              port: 3005
            initialDelaySeconds: 10
            periodSeconds: 15
          livenessProbe:
            httpGet:
              path: /health
              port: 3005
            initialDelaySeconds: 30
            periodSeconds: 30
          volumeMounts:
            - name: tmp
              mountPath: /tmp
      volumes:
        - name: tmp
          emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: compliance-service
  namespace: platform
spec:
  selector:
    app: compliance-service
  ports:
    - name: http
      port: 80
      targetPort: 3005
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: compliance-service-pdb
  namespace: platform
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: compliance-service
```

### Database Migrations

```bash
# Run pending migrations
npm run db:migrate

# Roll back last migration
npm run db:migrate:rollback

# Check migration status
npm run db:migrate:status

# Generate new migration
npm run db:migrate:make -- --name add_new_compliance_table
```

### CI/CD Pipeline Checklist

Before deploying to production, ensure:

- [ ] All database migrations are idempotent
- [ ] `AUDIT_HMAC_SECRET` is rotated and old key added to key history
- [ ] `ENCRYPTION_KEY_ID` is updated if encryption key was rotated
- [ ] DPO has reviewed any changes to data processing logic
- [ ] Retention policies have been verified after schema changes
- [ ] Audit log chain integrity verified on staging environment
- [ ] PBefG report templates validated against current KBA specifications
- [ ] All Kafka consumer group offsets are healthy

---

## German Regulatory Compliance Notes

### PBefG — Personenbeförderungsgesetz

The **Personenbeförderungsgesetz (PBefG)** governs the commercial transportation of persons in Germany. The 2021 reform (*PBefG-Novelle 2021*) introduced specific regulations for platform-based mobility services.

**Relevant Provisions:**

| Paragraph | Requirement | Implementation |
|---|---|---|
| § 45 PBefG | Mietwagen must return to the operating base after each ride unless a pre-ordered trip is available (*Rückkehrpflicht*) | Trip routing validation in trip-service; exceptions logged |
| § 46 PBefG | Ride-pooling services require special permit (*Genehmigung*) | Permit registry maintained in compliance-service |
| § 49 PBefG | Freigestellter Verkehr (exempt transport) conditions must be met and documented | Automated classification and audit logging |
| § 51 PBefG | Driver qualification and licensing records must be maintained | Driver compliance records with expiry alerts |
| § 54 PBefG | Operating authority (*Betriebspflicht*) and tariff obligations | Service level monitoring and reporting |
| § 57 PBefG | Regular reporting to *Genehmigungsbehörde* (licensing authority) | Automated monthly and annual report generation |

**Monthly Reporting Requirements:**
- Total trips and passenger kilometers by vehicle category
- Vehicle utilization rates and operational hours
- Driver working hours compliance with *Arbeitszeitgesetz (ArbZG)*
- Incidents, accidents, and regulatory violations
- Pool utilization rates for *§ 50 Linienbedarfsverkehr*

**Annual KBA Submission:**
Annual statistics are submitted electronically to the *Kraftfahrt-Bundesamt* via their e-reporting portal by **31 March** each year for the preceding calendar year.

### DSGVO / GDPR

The **Datenschutz-Grundverordnung (DSGVO)** and its German implementation via the **Bundesdatenschutzgesetz (BDSG neue Fassung)** impose strict requirements on personal data processing.

**Data Subject Rights Implementation:**

| Right | Article | Statutory Timeframe | Implementation |
|---|---|---|---|
| Auskunftsrecht (Access) | Art. 15 DSGVO | 1 month (extendable to 3) | Automated DSAR workflow |
| Berichtigung (Rectification) | Art. 16 DSGVO | Without undue delay | Direct update with audit trail |
| Löschung (Erasure) | Art. 17 DSGVO | Without undue delay | Cascading deletion with legal hold checks |
| Einschränkung (Restriction) | Art. 18 DSGVO | Without undue delay | Processing restriction flags |
| Datenübertragbarkeit (Portability) | Art. 20 DSGVO | 1 month | Machine-readable export (JSON/CSV) |
| Widerspruchsrecht (Objection) | Art. 21 DSGVO | Upon receipt | Consent processing pipeline |

**Legal Bases (*Rechtsgrundlagen*) for Processing:**

| Processing Activity | Legal Basis | Relevant Article |
|---|---|---|
| Trip booking and execution | Vertragserfüllung (Contract performance) | Art. 6 Abs. 1 lit. b DSGVO |
| Payment processing | Vertragserfüllung | Art. 6 Abs. 1 lit. b DSGVO |
| Marketing emails | Einwilligung (Consent) | Art. 6 Abs. 1 lit. a DSGVO |
| Tax record retention | Rechtliche Verpflichtung (Legal obligation) | Art. 6 Abs. 1 lit. c DSGVO, § 147 AO |
| Fraud prevention | Berechtigtes Interesse (Legitimate interests) | Art. 6 Abs. 1 lit. f DSGVO |
| Safety monitoring | Öffentliches Interesse (Public interest) | Art. 6 Abs. 1 lit. e DSGVO |

**Mandatory Data Protection Requirements:**

1. **Datenschutzbeauftragter (DPO):** A Data Protection Officer must be appointed per § 38 BDSG when regularly processing personal data. The DPO's contact details are published in the Impressum and privacy policy.

2. **Datenschutz-Folgenabschätzung (DSFA):** Privacy Impact Assessments are required per Art. 35 DSGVO for high-risk processing (e.g., large-scale location tracking). All DSFA records are maintained in the compliance service.

3. **Verzeichnis der Verarbeitungstätigkeiten:** The record of processing activities per Art. 30 DSGVO is maintained and available to supervisory authorities upon request.

4. **Auftragsverarbeitungsverträge (AVV):** Data processing agreements with all data processors are tracked in the processing register.

5. **Datenpannen-Meldung:** Data breaches must be reported to the competent supervisory authority within 72 hours per Art. 33 DSGVO, and to affected data subjects without undue delay when high risk exists per Art. 34 DSGVO.

### Aufsichtsbehörden (Supervisory Authorities)

The competent supervisory authority depends on the company's *Hauptniederlassung* (principal establishment):

| Federal State | Authority | Contact |
|---|---|---|
| Bayern | Bayerisches Landesamt für Datenschutzaufsicht (BayLDA) | https://www.lda.bayern.de |
| Berlin | Berliner Beauftragte für Datenschutz und Informationsfreiheit | https://www.datenschutz-berlin.de |
| Hamburg | Der Hamburgische Beauftragte für Datenschutz und Informationsfreiheit | https://datenschutz.hamburg.de |
| Nordrhein-Westfalen | Landesbeauftragte für Datenschutz und Informationsfreiheit NRW | https://www.ldi.nrw.de |

### Data Retention Legal Obligations (*Aufbewahrungsfristen*)

| Data Category | Retention Period | Legal Basis |
|---|---|---|
| Commercial invoices and receipts | 10 years | § 257 Abs. 1 HGB |
| Tax-relevant records | 10 years | § 147 Abs. 1 AO |
| Business correspondence | 6 years | § 257 Abs. 1 Nr. 2 HGB |
| Driver working time records | 2 years | § 21a ArbZG |
| Accident and incident records | 3 years | § 195 BGB (Regelverjährung) |
| Personal data (no legal obligation) | Duration of contract + reasonable period | Art. 5 Abs. 1 lit. e DSGVO |
| Precise location data | 90 days maximum | DSGVO Datensparsamkeit |
| Audit logs | 10 years recommended | § 257 HGB by analogy |

### Arbeitszeitgesetz (ArbZG) Compliance

Driver working time must comply with the **Arbeitszeitgesetz** and **EU Directive 2002/15/EC** (Road Transport Working Time Directive):

- Maximum 8 hours daily working time (extendable to 10 hours if average over 6 months ≤ 8 hours)
- Mandatory rest periods: 30-minute break after 6 hours, 45 minutes after 9 hours
- Minimum 11 consecutive hours daily rest
- Maximum 48 hours average weekly working time over 4-month reference period
- Violations are automatically flagged and reported in PBefG compliance reports

---

## Security

### Authentication & Authorization

- All API endpoints require JWT authentication issued by the central Auth Service
- Role-Based Access Control (RBAC) with compliance-specific roles:
  - `ROLE_COMPLIANCE_ADMIN` — Full access to all compliance functions
  - `ROLE_DPO` — Data Protection Officer with DSAR and breach management access
  - `ROLE_REGULATORY_OFFICER` — PBefG reporting access
  - `ROLE_AUDITOR` — Read-only access to audit logs and reports
  - `ROLE_SUPPORT` — Limited DSAR initiation and status checking

### Data Protection

- All PII fields encrypted at rest using AES-256-GCM
- Envelope encryption with AWS KMS or compatible key management system
- All connections secured with TLS 1.3 minimum
- Database connections use SSL with certificate verification
- Sensitive fields in audit logs pseudonymized before archival after 2 years

### Audit Log Integrity

The audit log uses an HMAC-SHA256 chain to detect tampering:

```
chain_hash[n] = HMAC-SHA256(
  key = AUDIT_HMAC_SECRET,
  message = chain_hash[n-1] + audit_id[n] + event_timestamp[n] + event_data_hash[n]
)
```

Periodic integrity verification runs nightly and alerts the DPO and Security team if any tampering is detected.

### Secrets Management

- All secrets managed via HashiCorp Vault or AWS Secrets Manager
- Secrets never committed to version control
- Encryption keys rotated annually with overlap period for decryption of existing data
- AUDIT_HMAC_SECRET rotation requires re-signing of all audit log entries in a maintenance window

---

## Testing

```bash
# Run all tests
npm test

# Run unit tests
npm run test:unit

# Run integration tests (requires Docker)
npm run test:integration

# Run compliance-specific scenario tests
npm run test:compliance

# Run audit log integrity tests
npm run test:audit-chain

# Coverage report
npm run test:coverage
```

### Compliance Test Scenarios

The test suite includes specific scenarios for:

- DSAR 30-day deadline enforcement and escalation
- Audit log chain integrity verification
- PBefG report generation correctness
- Data erasure cascade verification across services
- Breach notification 72-hour deadline tracking
- Retention policy dry-run accuracy
- Consent withdrawal propagation

---

## Monitoring & Alerting

### Key Compliance Metrics (Prometheus)

| Metric | Type | Description |
|---|---|---|
| `compliance_dsar_open_total` | Gauge | Total open DSAR requests |
| `compliance_dsar_at_risk_total` | Gauge | DSARs within 7 days of statutory deadline |
| `compliance_dsar_overdue_total` | Gauge | DSARs past statutory deadline |
| `compliance_audit_log_entries_total` | Counter | Total audit log entries processed |
| `compliance_audit_chain_integrity` | Gauge | Audit chain integrity status (1=ok, 0=error) |
| `compliance_pbefg_report_last_submitted` | Gauge | Unix timestamp of last successful PBefG submission |
| `compliance_breach_open_total` | Gauge | Open data breach incidents |
| `compliance_erasure_pending_total` | Gauge | Pending erasure confirmations from downstream services |

### Critical Alerts

- **DSAR_OVERDUE:** Any DSAR past its 30-day statutory deadline — **PagerDuty P1**
- **BREACH_72H_APPROACHING:** Data breach notification deadline within 12 hours — **PagerDuty P1**
- **AUDIT_CHAIN_TAMPERED:** Audit log integrity check failure — **PagerDuty P1, Security team**
- **PBEFG_SUBMISSION_FAILED:** Monthly PBefG submission failure — **PagerDuty P2**
- **DPO_EMAIL_UNREACHABLE:** DPO notification delivery failure — **PagerDuty P2**

---

## Contributing

### Development Guidelines

1. All changes to data processing logic must be reviewed by the Data Protection Officer before merging
2. New personal data fields must be accompanied by a retention policy update and processing register amendment
3. API changes must maintain backward compatibility for a minimum of 2 major versions
4. All regulatory report templates must be validated against the current authority specifications before release
5. Security-sensitive changes (encryption, audit logging, authentication) require two-person review

### Branching Strategy

```
main          — Production-ready code
develop       — Integration branch
feature/*     — Feature development
bugfix/*      — Bug fixes
regulatory/*  — Branches for regulatory requirement updates
hotfix/*      — Emergency production fixes
```

### Commit Message Convention

```
feat(dsar): add automated data package generation
fix(pbefg): correct Mietwagen trip count aggregation for § 57 report
reg(dsgvo): update consent withdrawal to Art. 7 Abs. 3 requirements
sec(audit): strengthen HMAC chain with SHA-512
docs(pbefg): update KBA submission format for 2024 specification
```

---

## License

Proprietary — All rights reserved. This software contains regulatory compliance logic subject to German law. Unauthorized use, reproduction, or distribution is prohibited.

---

## Contact

**Service Owners:** Platform Compliance Team <compliance-team@mobilityplatform.de>

**Data Protection Officer:** <dpo@mobilityplatform.de>

**Security Issues:** <security@mobilityplatform.de> (PGP key available on keyserver)

**Regulatory Queries:** <regulatory@mobilityplatform.de>

---

*Last Updated: January 2024 | Service Version: 2.4.1 | DSGVO Reviewed: 2024-01-01 | PBefG Reviewed: 2024-01-01*
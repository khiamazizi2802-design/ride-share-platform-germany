# Compliance & Audit Service

> **Compliance- und Audit-Dienst** — A production-grade microservice for GDPR compliance, immutable audit logging, and German regulatory adherence for ride-sharing platforms operating under **BZP**, **P-Schein**, **Fahrerlaubnis zur Fahrgastbeförderung**, and **TSE (Technische Sicherheitseinrichtung)** requirements.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Proprietary-red.svg)](LICENSE)
[![GDPR Compliant](https://img.shields.io/badge/GDPR-Compliant-green.svg)](https://gdpr.eu)
[![German Law](https://img.shields.io/badge/PBefG-Compliant-blue.svg)](https://www.gesetze-im-internet.de/pbefg/)

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [API Endpoints](#api-endpoints)
- [Environment Variables](#environment-variables)
- [Database Schema](#database-schema)
- [Kafka Events](#kafka-events)
- [Deployment](#deployment)
- [Compliance Notes](#compliance-notes)
- [Development](#development)
- [License](#license)

---

## Overview

The **Compliance & Audit Service** is the regulatory and data-governance backbone of a German ride-sharing platform. It provides a centralized, tamper-proof record of all system activities, ensures adherence to the **Bundesdatenschutzgesetz (BDSG)**, the **EU General Data Protection Regulation (GDPR / DSGVO)**, the **Personenbeförderungsgesetz (PBefG)**, and the German **Kassensicherungsverordnung (KassenSichV)** for TSE-compliant fiscal logging.

All audit entries are cryptographically chained (SHA-256 hash chain), making retroactive tampering detectable. The service integrates with the broader platform via Apache Kafka for event-driven compliance workflows and exposes a RESTful API for internal services and authorized regulatory interfaces.

**Regulatory Scope:**
- 🇩🇪 BDSG (Bundesdatenschutzgesetz) — Federal Data Protection Act
- 🇪🇺 DSGVO / GDPR — General Data Protection Regulation (EU) 2016/679
- 🚖 PBefG — Personenbeförderungsgesetz (Passenger Transport Act)
- 🧾 KassenSichV — Kassensicherungsverordnung (Cash Register Security Ordinance)
- 📋 DSGVO Art. 30 — Records of Processing Activities (Verarbeitungsverzeichnis)

---

## Features

### 🔒 GDPR / DSGVO Compliance Tracking
- Full **Records of Processing Activities (RoPA)** as required by Art. 30 DSGVO
- Automated **lawful basis** tracking for every data processing operation
- Data subject rights workflow management (access, rectification, erasure, portability)
- **Data Protection Impact Assessment (DPIA)** trigger detection for high-risk processing
- Cross-border data transfer monitoring with Standard Contractual Clauses (SCC) logging

### 🚗 German Regulatory Compliance
- **P-Schein (Personenbeförderungsschein)** validity tracking and expiry alerting
- **Fahrerlaubnis zur Fahrgastbeförderung** license verification audit trail
- **TSE (Technische Sicherheitseinrichtung)** transaction logging per KassenSichV
- **Pflichtversicherungsnachweis** (mandatory insurance) compliance records
- **BOKraft** (Betriebsordnung für den gewerblichen Binnenschifffahrt- und Kraftfahrzeugverkehr) compliance checks

### 🔗 Immutable Audit Logging with Cryptographic Verification
- SHA-256 **hash-chained audit log** — each entry includes the hash of the previous entry
- **HMAC-SHA256** signatures per entry using a rotating service key
- Append-only PostgreSQL partitioned tables with write-protected policies
- **Merkle tree** snapshots for bulk log integrity verification
- Tamper detection API for forensic audit use

### 🗄️ Data Retention Policy Enforcement
- Configurable retention schedules per data category and legal basis
- Automated **purge jobs** for expired personal data (DSGVO Art. 5(1)(e) — storage limitation)
- Legal hold support to suspend purge during investigations or litigation
- Retention certificates issued after successful purge
- German tax retention requirements (10-year fiscal records per §147 AO)

### 📊 Compliance Reporting for Authorities
- Structured report generation for **Landesdatenschutzbehörden** (State DPAs)
- **Bundesbeauftragte für den Datenschutz und die Informationsfreiheit (BfDI)** report templates
- Machine-readable export formats (JSON, XML, CSV) for regulatory submission
- Automated annual processing activity reports (Verarbeitungsverzeichnis)
- Audit trail summaries for **Ordnungsamt** and **Gewerbeaufsicht** inspections

### 🚨 Incident Tracking & Breach Notifications
- DSGVO **72-hour breach notification** countdown timer (Art. 33)
- Automated **supervisory authority notification** workflow (Meldepflicht)
- Risk scoring engine (low / medium / high / critical) for data breaches
- **Art. 34 DSGVO** — automated affected data subject notification queue
- Full incident lifecycle management: detection → containment → eradication → recovery → lessons learned

### ✅ Consent Management
- Granular consent recording per purpose, data category, and legal basis
- Consent versioning — historical record of all consent states
- **Double opt-in** workflow support
- Consent expiry and renewal reminders
- DSGVO Art. 7 compliant withdrawal processing

### 🗑️ Right to Erasure Automation
- Automated **Recht auf Vergessenwerden** (Art. 17 DSGVO) workflow
- Cross-service erasure orchestration via Kafka events
- Erasure verification receipts with cryptographic proof
- Exceptions handling: legal obligation, public interest, legal claims (Art. 17(3))

### 📦 Data Portability Requests
- DSGVO Art. 20 compliant **data portability** request processing
- Export in **machine-readable formats**: JSON, CSV, XML
- Secure, time-limited download links via pre-signed URLs
- Cross-service data aggregation for complete data subject profiles

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Compliance & Audit Service                       │
│                                                                       │
│  ┌─────────────┐   ┌──────────────┐   ┌──────────────────────────┐  │
│  │  REST API   │   │  Kafka       │   │  Scheduler (cron)        │  │
│  │  (Gin/HTTP) │   │  Consumer    │   │  - Retention purge       │  │
│  │             │   │  - audit.*   │   │  - Report generation     │  │
│  └──────┬──────┘   │  - consent.* │   │  - Breach notifications  │  │
│         │          │  - gdpr.*    │   │  - License expiry alerts │  │
│         │          └──────┬───────┘   └──────────┬───────────────┘  │
│         │                 │                      │                   │
│         └─────────────────┴──────────────────────┘                   │
│                                    │                                  │
│         ┌──────────────────────────▼──────────────────────────┐      │
│         │                  Service Layer                        │      │
│         │  AuditService │ ComplianceService │ ConsentService    │      │
│         │  IncidentService │ RetentionService │ GDPRService     │      │
│         └──────────────────────────┬──────────────────────────┘      │
│                                    │                                  │
│         ┌──────────────────────────▼──────────────────────────┐      │
│         │               Repository Layer (GORM)                │      │
│         └──────────┬───────────────────────────────────────────┘      │
│                    │                                                   │
└────────────────────┼───────────────────────────────────────────────-─┘
                     │
     ┌───────────────┼───────────────────┐
     ▼               ▼                   ▼
┌─────────┐   ┌────────────┐   ┌──────────────────┐
│PostgreSQL│   │Apache Kafka│   │  Redis (cache /  │
│(primary) │   │(events)    │   │  rate limiting)  │
└─────────┘   └────────────┘   └──────────────────┘
```

**Technology Stack:**

| Component        | Technology                          |
|-----------------|-------------------------------------|
| Language         | Go 1.22+                            |
| Web Framework    | Gin                                 |
| ORM              | GORM                                |
| Database         | PostgreSQL 15+ (partitioned tables) |
| Message Broker   | Apache Kafka 3.6+                   |
| Cache            | Redis 7+                            |
| Crypto           | Go `crypto/sha256`, `crypto/hmac`   |
| Scheduler        | `robfig/cron` v3                    |
| Observability    | OpenTelemetry + Prometheus          |
| Auth             | JWT (RS256) + mTLS for service mesh |

---

## API Endpoints

All endpoints are prefixed with `/api/v1` unless stated otherwise. Authentication is required via `Authorization: Bearer <JWT>` for all endpoints except `/health`.

---

### 📝 Audit Logs

#### `POST /api/v1/audit/log`

Create a new immutable audit log entry. The service automatically computes and appends the cryptographic hash chain.

**Request Body:**
```json
{
  "actor_id": "usr_01HQ7BNMK2V3W4X5Y6Z7",
  "actor_type": "user",
  "action": "driver.license.verified",
  "resource_type": "driver_profile",
  "resource_id": "drv_01HQ7BNMK2V3W4X5Y6Z7",
  "outcome": "success",
  "metadata": {
    "license_type": "Fahrerlaubnis_Klasse_B",
    "p_schein_valid_until": "2026-12-31",
    "verification_authority": "Kraftfahrtbundesamt_Flensburg"
  },
  "ip_address": "192.168.1.100",
  "service_name": "driver-service",
  "correlation_id": "corr_01HQ7BNMK2V3W4X5Y6Z7",
  "legal_basis": "Art6_1b_DSGVO",
  "data_categories": ["identity", "license_data", "biometric_reference"]
}
```

**Response `201 Created`:**
```json
{
  "id": "aud_01HQ7BNMK2V3W4X5Y6Z7",
  "sequence_number": 487291,
  "entry_hash": "sha256:a3f2c1d4e5b6a7f8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
  "previous_hash": "sha256:b4e3d2c1f0a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3",
  "timestamp": "2024-03-15T14:23:45.123456Z",
  "created_at": "2024-03-15T14:23:45.123456Z"
}
```

**Error Responses:**
- `400 Bad Request` — Invalid payload or missing required fields
- `401 Unauthorized` — Missing or invalid JWT
- `422 Unprocessable Entity` — Unknown actor type or action category

---

#### `GET /api/v1/audit/logs`

Query audit logs with filtering, pagination, and optional integrity verification.

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `actor_id` | string | Filter by actor UUID |
| `resource_type` | string | Filter by resource type |
| `resource_id` | string | Filter by specific resource |
| `action` | string | Filter by action (supports `*` wildcard) |
| `legal_basis` | string | Filter by DSGVO legal basis |
| `from` | RFC3339 | Start timestamp (inclusive) |
| `to` | RFC3339 | End timestamp (inclusive) |
| `outcome` | string | `success`, `failure`, `error` |
| `service_name` | string | Originating service name |
| `page` | int | Page number (default: 1) |
| `page_size` | int | Results per page (default: 50, max: 500) |
| `verify_integrity` | bool | Run hash-chain verification on results (expensive) |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "aud_01HQ7BNMK2V3W4X5Y6Z7",
      "sequence_number": 487291,
      "actor_id": "usr_01HQ7BNMK2V3W4X5Y6Z7",
      "actor_type": "user",
      "action": "driver.license.verified",
      "resource_type": "driver_profile",
      "resource_id": "drv_01HQ7BNMK2V3W4X5Y6Z7",
      "outcome": "success",
      "legal_basis": "Art6_1b_DSGVO",
      "data_categories": ["identity", "license_data"],
      "entry_hash": "sha256:a3f2c1d4...",
      "integrity_verified": true,
      "timestamp": "2024-03-15T14:23:45.123456Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 50,
    "total": 12834,
    "total_pages": 257
  },
  "integrity_check": {
    "verified": true,
    "chain_intact": true,
    "checked_at": "2024-03-15T14:24:00.000000Z"
  }
}
```

---

### 📋 Compliance Reports

#### `POST /api/v1/compliance/report`

Generate a compliance report. Report generation is asynchronous for large datasets; the response returns a report ID that can be polled.

**Request Body:**
```json
{
  "report_type": "verarbeitungsverzeichnis",
  "period_start": "2024-01-01T00:00:00Z",
  "period_end": "2024-12-31T23:59:59Z",
  "scope": ["data_processing_activities", "consent_records", "dsar_summary"],
  "requested_by": "dpo_01HQ7BNMK2V3W4X5Y6Z7",
  "authority": "BfDI",
  "format": "pdf",
  "language": "de",
  "include_tse_log": true,
  "filters": {
    "legal_bases": ["Art6_1b_DSGVO", "Art6_1c_DSGVO"],
    "data_categories": ["identity", "location", "financial"]
  }
}
```

**Supported `report_type` values:**

| Value | Description |
|-------|-------------|
| `verarbeitungsverzeichnis` | Art. 30 DSGVO Records of Processing Activities |
| `datenpanne_meldung` | Art. 33 DSGVO breach notification report |
| `einwilligungsnachweis` | Consent evidence report |
| `loeschkonzept` | Deletion concept per DSGVO Art. 5(1)(e) |
| `tse_tagesabschluss` | KassenSichV TSE daily closing report |
| `p_schein_pruefung` | P-Schein validity audit report |
| `dsar_summary` | Data Subject Access Request summary |
| `retention_audit` | Data retention compliance audit |

**Response `202 Accepted`:**
```json
{
  "report_id": "rpt_01HQ7BNMK2V3W4X5Y6Z7",
  "status": "generating",
  "estimated_completion": "2024-03-15T14:35:00Z",
  "created_at": "2024-03-15T14:25:00Z"
}
```

---

#### `GET /api/v1/compliance/reports`

List generated compliance reports with optional filters.

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `report_type` | string | Filter by report type |
| `status` | string | `generating`, `completed`, `failed` |
| `authority` | string | Filter by target authority |
| `from` | RFC3339 | Created after timestamp |
| `to` | RFC3339 | Created before timestamp |
| `requested_by` | string | Filter by requester ID |
| `page` | int | Page number |
| `page_size` | int | Results per page (max: 100) |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "rpt_01HQ7BNMK2V3W4X5Y6Z7",
      "report_type": "verarbeitungsverzeichnis",
      "status": "completed",
      "period_start": "2024-01-01T00:00:00Z",
      "period_end": "2024-12-31T23:59:59Z",
      "authority": "BfDI",
      "format": "pdf",
      "language": "de",
      "download_url": "https://secure.internal/reports/rpt_01HQ7BNMK2V3W4X5Y6Z7.pdf",
      "download_expires_at": "2024-03-16T14:25:00Z",
      "file_size_bytes": 524288,
      "checksum_sha256": "d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9",
      "requested_by": "dpo_01HQ7BNMK2V3W4X5Y6Z7",
      "created_at": "2024-03-15T14:25:00Z",
      "completed_at": "2024-03-15T14:32:14Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 47,
    "total_pages": 3
  }
}
```

---

### 🛡️ GDPR Data Requests (DSARs)

#### `POST /api/v1/data-request`

Submit a GDPR Data Subject Access Request (DSAR). Supports all Art. 15–20 DSGVO request types.

**Request Body:**
```json
{
  "subject_id": "usr_01HQ7BNMK2V3W4X5Y6Z7",
  "subject_email": "max.mustermann@example.de",
  "request_type": "access",
  "description": "Ich beantrage gemäß Art. 15 DSGVO Auskunft über alle mich betreffenden personenbezogenen Daten.",
  "identity_verified": true,
  "verification_method": "email_otp",
  "requested_data_categories": ["all"],
  "preferred_format": "json",
  "preferred_language": "de",
  "channel": "app"
}
```

**Supported `request_type` values:**

| Value | DSGVO Article | Description |
|-------|---------------|-------------|
| `access` | Art. 15 | Auskunftsrecht — Right of access |
| `rectification` | Art. 16 | Berichtigungsrecht — Right to rectification |
| `erasure` | Art. 17 | Recht auf Löschung — Right to erasure |
| `restriction` | Art. 18 | Einschränkung der Verarbeitung |
| `portability` | Art. 20 | Datenübertragbarkeit — Data portability |
| `objection` | Art. 21 | Widerspruchsrecht — Right to object |

**Response `201 Created`:**
```json
{
  "id": "dsar_01HQ7BNMK2V3W4X5Y6Z7",
  "status": "received",
  "request_type": "access",
  "subject_id": "usr_01HQ7BNMK2V3W4X5Y6Z7",
  "legal_deadline": "2024-04-14T23:59:59Z",
  "acknowledgement_sent_at": "2024-03-15T14:23:45Z",
  "created_at": "2024-03-15T14:23:45Z"
}
```

> **⚠️ Legal Note:** DSGVO Art. 12(3) mandates a response within **1 month** (extendable by 2 months for complex requests). The service automatically sets `legal_deadline` and escalates overdue DSARs.

---

#### `GET /api/v1/data-requests`

List all GDPR data requests with filtering support.

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `subject_id` | string | Filter by data subject ID |
| `request_type` | string | Filter by request type |
| `status` | string | `received`, `in_progress`, `completed`, `rejected`, `overdue` |
| `overdue_only` | bool | Show only requests past legal deadline |
| `from` | RFC3339 | Created after |
| `to` | RFC3339 | Created before |
| `page` | int | Page number |
| `page_size` | int | Results per page |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "dsar_01HQ7BNMK2V3W4X5Y6Z7",
      "status": "in_progress",
      "request_type": "erasure",
      "subject_id": "usr_01HQ7BNMK2V3W4X5Y6Z7",
      "legal_deadline": "2024-04-14T23:59:59Z",
      "days_remaining": 28,
      "assigned_to": "dpo_01HQ7BNMK2V3W4X5Y6Z7",
      "services_notified": ["user-service", "trip-service", "payment-service"],
      "services_confirmed": ["user-service"],
      "created_at": "2024-03-15T14:23:45Z",
      "updated_at": "2024-03-16T09:15:00Z"
    }
  ],
  "summary": {
    "total": 23,
    "overdue": 1,
    "due_within_7_days": 3
  },
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 23,
    "total_pages": 2
  }
}
```

---

#### `PUT /api/v1/data-request/{id}/process`

Process a pending GDPR data request (mark as in-progress, completed, or rejected).

**Path Parameters:**
- `id` — DSAR UUID (e.g., `dsar_01HQ7BNMK2V3W4X5Y6Z7`)

**Request Body:**
```json
{
  "action": "complete",
  "processed_by": "dpo_01HQ7BNMK2V3W4X5Y6Z7",
  "notes": "All personal data deleted from user-service, trip-service, payment-service. TSE fiscal records retained per §147 AO (10-year requirement).",
  "rejection_reason": null,
  "legal_exception": null,
  "evidence_ids": ["aud_01HQ7B", "aud_01HQ7C"],
  "download_url": null
}
```

**Supported `action` values:**

| Value | Description |
|-------|-------------|
| `start` | Mark request as in-progress |
| `complete` | Mark request as fulfilled |
| `reject` | Reject with reason (e.g., legal exception) |
| `extend` | Extend deadline by up to 2 months (Art. 12(3)) |
| `verify_identity` | Mark identity re-verification as required |

**Response `200 OK`:**
```json
{
  "id": "dsar_01HQ7BNMK2V3W4X5Y6Z7",
  "status": "completed",
  "completed_at": "2024-03-16T10:30:00Z",
  "processed_by": "dpo_01HQ7BNMK2V3W4X5Y6Z7",
  "response_time_hours": 20,
  "audit_entry_id": "aud_01HQ7BNMK2V3W4X5Y6Z7"
}
```

---

### ✅ Consent Management

#### `POST /api/v1/consent`

Record a new consent event for a data subject. Compliant with DSGVO Art. 7 and ePrivacy requirements.

**Request Body:**
```json
{
  "subject_id": "usr_01HQ7BNMK2V3W4X5Y6Z7",
  "purposes": [
    "location_tracking_realtime",
    "marketing_email",
    "analytics_pseudonymized"
  ],
  "legal_basis": "Art6_1a_DSGVO",
  "consent_version": "v3.2.1",
  "privacy_policy_version": "v2024-03",
  "collection_method": "explicit_checkbox",
  "ui_language": "de",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0 (Android 14; ...)",
  "platform": "mobile_app",
  "double_opt_in_required": true,
  "double_opt_in_token": null,
  "metadata": {
    "app_version": "4.2.1",
    "screen_name": "onboarding_consent"
  }
}
```

**Response `201 Created`:**
```json
{
  "id": "cns_01HQ7BNMK2V3W4X5Y6Z7",
  "status": "active",
  "purposes_granted": ["location_tracking_realtime", "analytics_pseudonymized"],
  "purposes_pending_double_opt_in": ["marketing_email"],
  "double_opt_in_email_sent": true,
  "expires_at": null,
  "created_at": "2024-03-15T14:23:45Z"
}
```

---

#### `PUT /api/v1/consent/{id}/withdraw`

Withdraw a previously granted consent. Per DSGVO Art. 7(3), withdrawal must be as easy as giving consent.

**Path Parameters:**
- `id` — Consent record UUID (e.g., `cns_01HQ7BNMK2V3W4X5Y6Z7`)

**Request Body:**
```json
{
  "withdrawn_by": "usr_01HQ7BNMK2V3W4X5Y6Z7",
  "withdrawal_method": "app_settings",
  "purposes_to_withdraw": ["marketing_email"],
  "withdraw_all": false,
  "reason": "no_longer_interested",
  "ip_address": "192.168.1.100"
}
```

**Response `200 OK`:**
```json
{
  "id": "cns_01HQ7BNMK2V3W4X5Y6Z7",
  "status": "partially_withdrawn",
  "withdrawn_purposes": ["marketing_email"],
  "remaining_active_purposes": ["location_tracking_realtime", "analytics_pseudonymized"],
  "downstream_notified": true,
  "services_notified": ["marketing-service", "notification-service"],
  "withdrawn_at": "2024-03-16T09:00:00Z",
  "audit_entry_id": "aud_01HQ7BNMK2V3W4X5Y6Z7"
}
```

---

### 🚨 Incident Management

#### `POST /api/v1/incident`

Report a security incident or personal data breach. Triggers the 72-hour DSGVO Art. 33 notification timer.

**Request Body:**
```json
{
  "incident_type": "personal_data_breach",
  "title": "Unbeabichtigte Offenlegung von Fahreradressen",
  "description": "Ein Konfigurationsfehler im Trip-Service hat zwischen 14:00 und 15:30 Uhr Fahreradressen in API-Antworten an Fahrgäste exponiert.",
  "detected_at": "2024-03-15T15:45:00Z",
  "estimated_breach_start": "2024-03-15T14:00:00Z",
  "estimated_breach_end": "2024-03-15T15:30:00Z",
  "reported_by": "eng_01HQ7BNMK2V3W4X5Y6Z7",
  "affected_data_categories": ["home_address", "phone_number"],
  "affected_subject_types": ["drivers"],
  "estimated_affected_count": 47,
  "data_transferred_outside_eu": false,
  "containment_measures_taken": "API endpoint patched and redeployed at 15:32.",
  "risk_level": "high",
  "notification_required": true,
  "services_affected": ["trip-service", "user-service"]
}
```

**Supported `incident_type` values:**

| Value | Description |
|-------|-------------|
| `personal_data_breach` | DSGVO Art. 4(12) — Personal data breach |
| `unauthorized_access` | Unauthorized system access |
| `data_loss` | Data loss or destruction |
| `system_unavailability` | System downtime affecting compliance |
| `p_schein_violation` | Invalid P-Schein discovered post-trip |
| `tse_integrity_failure` | TSE tamper detection event |
| `insurance_gap` | Insurance coverage gap identified |

**Response `201 Created`:**
```json
{
  "id": "inc_01HQ7BNMK2V3W4X5Y6Z7",
  "status": "open",
  "risk_level": "high",
  "notification_deadline": "2024-03-18T15:45:00Z",
  "hours_until_deadline": 71,
  "supervisory_authority": "LDA Bayern",
  "supervisory_authority_contact": "poststelle@lda.bayern.de",
  "dpo_notified": true,
  "created_at": "2024-03-15T15:50:00Z"
}
```

---

#### `GET /api/v1/incidents`

List incidents with filtering and escalation status.

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `incident_type` | string | Filter by type |
| `status` | string | `open`, `investigating`, `contained`, `resolved`, `closed` |
| `risk_level` | string | `low`, `medium`, `high`, `critical` |
| `notification_overdue` | bool | Show only incidents past notification deadline |
| `from` | RFC3339 | Detected after |
| `to` | RFC3339 | Detected before |
| `page` | int | Page number |
| `page_size` | int | Results per page |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "inc_01HQ7BNMK2V3W4X5Y6Z7",
      "incident_type": "personal_data_breach",
      "title": "Unbeabichtigte Offenlegung von Fahreradressen",
      "status": "investigating",
      "risk_level": "high",
      "affected_subject_count": 47,
      "notification_deadline": "2024-03-18T15:45:00Z",
      "hours_until_deadline": 68,
      "authority_notified": false,
      "subjects_notified": false,
      "detected_at": "2024-03-15T15:45:00Z",
      "created_at": "2024-03-15T15:50:00Z"
    }
  ],
  "summary": {
    "total": 3,
    "open": 1,
    "notification_overdue": 0,
    "critical": 0,
    "high": 1
  },
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 3,
    "total_pages": 1
  }
}
```

---

### 🗄️ Data Retention

#### `GET /api/v1/retention/policies`

Retrieve all configured data retention policies.

**Response `200 OK`:**
```json
{
  "policies": [
    {
      "id": "ret_01HQ7BNMK2V3W4X5Y6Z7",
      "name": "Fahrerdaten_Verarbeitung",
      "data_category": "driver_profile",
      "legal_basis": "Art6_1b_DSGVO",
      "retention_period_days": 3650,
      "legal_reference": "§147_AO_Steuerrecht",
      "description": "Steuerrechtlich relevante Fahrdaten und Abrechnungen werden 10 Jahre gemäß §147 AO aufbewahrt.",
      "deletion_method": "secure_wipe",
      "applies_to_services": ["trip-service", "payment-service"],
      "active": true,
      "last_purge_at": "2024-03-01T02:00:00Z",
      "next_purge_at": "2024-04-01T02:00:00Z"
    },
    {
      "id": "ret_02HQ7BNMK2V3W4X5Y6Z8",
      "name": "Marketingdaten",
      "data_category": "marketing_profile",
      "legal_basis": "Art6_1a_DSGVO",
      "retention_period_days": 730,
      "legal_reference": "Art5_1e_DSGVO",
      "description": "Marketingprofile werden nach Widerruf der Einwilligung oder nach 2 Jahren gelöscht.",
      "deletion_method": "anonymization",
      "applies_to_services": ["marketing-service"],
      "active": true,
      "last_purge_at": "2024-03-14T03:00:00Z",
      "next_purge_at": "2024-03-21T03:00:00Z"
    }
  ]
}
```

---

#### `POST /api/v1/retention/purge`

Manually trigger a data purge job for a specific policy or data category. Requires elevated DPO privileges.

**Request Body:**
```json
{
  "policy_id": "ret_02HQ7BNMK2V3W4X5Y6Z8",
  "dry_run": true,
  "reason": "Manual purge requested by DPO following quarterly review.",
  "requested_by": "dpo_01HQ7BNMK2V3W4X5Y6Z7",
  "override_legal_hold": false
}
```

**Response `202 Accepted`:**
```json
{
  "purge_job_id": "prg_01HQ7BNMK2V3W4X5Y6Z7",
  "status": "scheduled",
  "dry_run": true,
  "estimated_records_affected": 1842,
  "estimated_completion": "2024-03-15T15:00:00Z",
  "audit_entry_id": "aud_01HQ7BNMK2V3W4X5Y6Z7",
  "created_at": "2024-03-15T14:45:00Z"
}
```

---

### 🏥 Health Check

#### `GET /health`

Service health check endpoint. Returns the status of all critical dependencies.

**Response `200 OK`:**
```json
{
  "status": "healthy",
  "version": "1.8.3",
  "build_commit": "a3f2c1d",
  "timestamp": "2024-03-15T14:23:45Z",
  "uptime_seconds": 864231,
  "checks": {
    "database": {
      "status": "healthy",
      "latency_ms": 2,
      "details": "PostgreSQL 15.4 — 487,291 audit entries"
    },
    "kafka": {
      "status": "healthy",
      "latency_ms": 5,
      "details": "Connected to 3 brokers"
    },
    "redis": {
      "status": "healthy",
      "latency_ms": 1
    },
    "hash_chain_integrity": {
      "status": "healthy",
      "last_verified": "2024-03-15T14:00:00Z",
      "chain_intact": true
    },
    "tse_connection": {
      "status": "healthy",
      "tse_serial": "TSE-DE-2024-XXXXXX",
      "tse_signature_algorithm": "ecdsa-plain-SHA384"
    }
  }
}
```

**Response `503 Service Unavailable`** when any critical check fails.

---

## Environment Variables

### Required

| Variable | Description | Example |
|----------|-------------|--------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@localhost:5432/compliance_db?sslmode=require` |
| `KAFKA_BROKERS` | Comma-separated Kafka broker addresses | `kafka-1:9092,kafka-2:9092,kafka-3:9092` |
| `KAFKA_GROUP_ID` | Kafka consumer group ID | `compliance-service-v1` |
| `REDIS_URL` | Redis connection string | `redis://:password@redis:6379/0` |
| `JWT_PUBLIC_KEY_PATH` | Path to RS256 public key for JWT validation | `/secrets/jwt_public.pem` |
| `AUDIT_HMAC_SECRET` | HMAC-SHA256 secret for audit entry signatures | `base64-encoded-256-bit-key` |
| `ENCRYPTION_KEY` | AES-256 key for PII field encryption at rest | `base64-encoded-256-bit-key` |
| `SERVICE_NAME` | Service identifier for audit logs | `compliance-service` |
| `ENVIRONMENT` | Deployment environment | `production` |

### Optional

| Variable | Description | Default |
|----------|-------------|--------|
| `SERVER_PORT` | HTTP server port | `8080` |
| `SERVER_READ_TIMEOUT` | HTTP read timeout | `30s` |
| `SERVER_WRITE_TIMEOUT` | HTTP write timeout | `30s` |
| `LOG_LEVEL` | Logging level (`debug`, `info`, `warn`, `error`) | `info` |
| `LOG_FORMAT` | Log format (`json`, `text`) | `json` |
| `DB_MAX_OPEN_CONNS` | Max open DB connections | `25` |
| `DB_MAX_IDLE_CONNS` | Max idle DB connections | `5` |
| `DB_CONN_MAX_LIFETIME` | DB connection max lifetime | `5m` |
| `KAFKA_CONSUMER_WORKERS` | Number of Kafka consumer goroutines | `4` |
| `RETENTION_PURGE_CRON` | Cron schedule for retention purge | `0 2 * * *` |
| `BREACH_NOTIFICATION_CRON` | Cron for breach deadline check | `*/15 * * * *` |
| `HASH_CHAIN_VERIFY_CRON` | Cron for scheduled hash chain verification | `0 4 * * *` |
| `DSAR_DEADLINE_CRON` | Cron for DSAR deadline escalation | `0 8 * * *` |
| `REPORT_STORAGE_PATH` | Local path for report file storage | `/data/reports` |
| `REPORT_S3_BUCKET` | S3/MinIO bucket for report storage | `compliance-reports` |
| `S3_ENDPOINT` | S3-compatible storage endpoint | `https://s3.eu-central-1.amazonaws.com` |
| `S3_ACCESS_KEY_ID` | S3 access key | *(from AWS credential chain)* |
| `S3_SECRET_ACCESS_KEY` | S3 secret key | *(from AWS credential chain)* |
| `TSE_ENABLED` | Enable TSE (KassenSichV) integration | `true` |
| `TSE_ENDPOINT` | TSE device/service endpoint | `https://tse.internal:8443` |
| `TSE_CLIENT_CERT_PATH` | mTLS client cert for TSE communication | `/secrets/tse_client.pem` |
| `DPO_EMAIL` | Data Protection Officer email for alerts | `dpo@company.de` |
| `SUPERVISORY_AUTHORITY` | Default supervisory DPA | `BfDI` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector endpoint | `http://otel-collector:4317` |
| `METRICS_PATH` | Prometheus metrics path | `/metrics` |
| `CORS_ALLOWED_ORIGINS` | Allowed CORS origins | `https://admin.internal` |
| `RATE_LIMIT_RPS` | Requests per second per IP | `100` |

---

## Database Schema

The service uses PostgreSQL 15+ with the following primary tables. All tables containing personal data use **column-level AES-256 encryption** for PII fields.

### `audit_logs`
Immutable, partitioned (by month) audit log table.
- `id` (UUID PK), `sequence_number` (BIGSERIAL, unique), `actor_id`, `actor_type`, `action`, `resource_type`, `resource_id`, `outcome`, `metadata` (JSONB), `legal_basis`, `data_categories` (TEXT[]), `entry_hash` (SHA-256 of entry content), `previous_hash` (hash of previous entry), `hmac_signature`, `ip_address`, `service_name`, `correlation_id`, `timestamp`
- **Index:** `sequence_number`, `actor_id`, `resource_id`, `timestamp`, `action`
- **Policy:** Row-level security — INSERT only, no UPDATE or DELETE

### `compliance_reports`
Generated compliance report metadata.
- `id`, `report_type`, `status`, `period_start`, `period_end`, `authority`, `format`, `language`, `file_path`, `file_size_bytes`, `checksum_sha256`, `requested_by`, `filters` (JSONB), `created_at`, `completed_at`

### `data_requests` (DSARs)
GDPR data subject request tracking.
- `id`, `subject_id`, `subject_email` *(encrypted)*, `request_type`, `status`, `description`, `identity_verified`, `verification_method`, `legal_deadline`, `assigned_to`, `services_notified` (TEXT[]), `services_confirmed` (TEXT[]), `rejection_reason`, `legal_exception`, `evidence_ids` (TEXT[]), `processed_by`, `download_url` *(encrypted)*, `created_at`, `updated_at`, `completed_at`

### `consents`
Consent records with full versioning history.
- `id`, `subject_id`, `purposes` (TEXT[]), `legal_basis`, `consent_version`, `privacy_policy_version`, `collection_method`, `status`, `ip_address` *(encrypted)*, `user_agent`, `platform`, `double_opt_in_confirmed`, `double_opt_in_confirmed_at`, `withdrawn_at`, `withdrawal_method`, `expires_at`, `metadata` (JSONB), `created_at`

### `incidents`
Security and data breach incident tracking.
- `id`, `incident_type`, `title`, `description`, `status`, `risk_level`, `detected_at`, `estimated_breach_start`, `estimated_breach_end`, `affected_data_categories` (TEXT[]), `affected_subject_types` (TEXT[]), `estimated_affected_count`, `actual_affected_count`, `notification_deadline`, `authority_notified_at`, `subjects_notified_at`, `supervisory_authority`, `containment_measures`, `services_affected` (TEXT[]), `reported_by`, `assigned_to`, `closed_at`, `created_at`, `updated_at`

### `retention_policies`
Configured data retention schedules.
- `id`, `name`, `data_category`, `legal_basis`, `legal_reference`, `retention_period_days`, `deletion_method`, `applies_to_services` (TEXT[]), `active`, `legal_hold`, `legal_hold_reason`, `last_purge_at`, `next_purge_at`, `created_at`, `updated_at`

### `purge_jobs`
Data purge execution history.
- `id`, `policy_id` (FK), `status`, `dry_run`, `records_affected`, `services_notified`, `services_confirmed`, `requested_by`, `started_at`, `completed_at`, `certificate_hash`, `created_at`

### `tse_transactions`
KassenSichV TSE transaction log (fiscal immutable records).
- `id`, `transaction_number` (BIGSERIAL), `process_type`, `process_data` *(encrypted)*, `tse_serial_number`, `tse_signature`, `tse_signature_algorithm`, `tse_transaction_start`, `tse_transaction_end`, `tse_certificate_serial`, `created_at`
- **Policy:** INSERT only — 10-year legal hold per §147 AO

---

## Kafka Events

### Published Events (Producer)

| Topic | Event Key | Description | Schema Version |
|-------|-----------|-------------|----------------|
| `compliance.audit.logged` | `audit_entry_id` | New audit entry created | v1 |
| `compliance.dsar.received` | `dsar_id` | New GDPR data request submitted | v1 |
| `compliance.dsar.erasure_requested` | `subject_id` | Erasure request ready — triggers cross-service deletion | v1 |
| `compliance.dsar.portability_ready` | `dsar_id` | Data export ready for download | v1 |
| `compliance.consent.granted` | `consent_id` | Consent granted for purposes | v1 |
| `compliance.consent.withdrawn` | `consent_id` | Consent withdrawn — downstream services must stop processing | v1 |
| `compliance.incident.opened` | `incident_id` | New incident reported | v1 |
| `compliance.incident.escalated` | `incident_id` | Incident escalated (risk level increased) | v1 |
| `compliance.incident.breach_notification_due` | `incident_id` | 72-hour DPA notification deadline approaching (<4h) | v1 |
| `compliance.retention.purge_completed` | `purge_job_id` | Data purge job completed with certificate | v1 |
| `compliance.tse.transaction_signed` | `tse_transaction_id` | TSE-signed fiscal transaction logged | v1 |

### Consumed Events (Consumer)

| Topic | Event Key | Action Taken | Source Service |
|-------|-----------|-------------|----------------|
| `driver.license.verified` | `driver_id` | Create audit log entry for license verification | driver-service |
| `driver.pschein.updated` | `driver_id` | Update P-Schein validity record and audit log | driver-service |
| `driver.insurance.updated` | `driver_id` | Update insurance compliance record | driver-service |
| `user.registered` | `user_id` | Create initial data processing consent record | user-service |
| `user.deleted` | `user_id` | Trigger DSAR erasure workflow if pending | user-service |
| `trip.completed` | `trip_id` | Log TSE fiscal transaction for completed fare | trip-service |
| `payment.processed` | `payment_id` | Audit financial data processing under §147 AO | payment-service |
| `gdpr.erasure_confirmed` | `subject_id` + `service_name` | Record cross-service erasure confirmation | all services |
| `gdpr.portability_data_ready` | `dsar_id` + `service_name` | Aggregate data for portability export | all services |
| `auth.admin_access` | `user_id` | Audit privileged admin action | auth-service |
| `security.suspicious_activity` | `user_id` | Create incident record for suspicious activity | security-service |

### Kafka Configuration

```yaml
kafka:
  replication_factor: 3
  partitions: 12
  retention_ms: 2592000000  # 30 days for compliance topics
  cleanup_policy: delete
  min_insync_replicas: 2
  acks: all  # Ensure no audit events are lost
  enable_idempotence: true
  max_in_flight_requests_per_connection: 1
```

> **⚠️ Important:** All compliance-related Kafka topics must use `acks=all` and `enable.idempotence=true` to guarantee exactly-once audit event delivery. Topic replication factor must be at least 3 in production.

---

## Deployment

### Prerequisites

- Docker 24+ / Docker Compose v2+
- Kubernetes 1.28+ (for production)
- PostgreSQL 15+ with `uuid-ossp` and `pgcrypto` extensions
- Apache Kafka 3.6+ (3-broker minimum for production)
- Redis 7+

### Docker (Development)

**1. Build the image:**
```bash
docker build \
  --build-arg BUILD_VERSION=1.8.3 \
  --build-arg BUILD_COMMIT=$(git rev-parse --short HEAD) \
  -t compliance-service:1.8.3 \
  -f Dockerfile .
```

**2. Start with Docker Compose:**
```bash
# Copy and configure environment
cp .env.example .env
vim .env  # Set required secrets

# Start all services
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# View logs
docker compose logs -f compliance-service

# Run database migrations
docker compose exec compliance-service ./compliance-service migrate up
```

**`docker-compose.yml` (excerpt):**
```yaml
services:
  compliance-service:
    image: compliance-service:1.8.3
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - KAFKA_BROKERS=${KAFKA_BROKERS}
      - REDIS_URL=${REDIS_URL}
      - AUDIT_HMAC_SECRET=${AUDIT_HMAC_SECRET}
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
    volumes:
      - ./secrets:/secrets:ro
      - report-data:/data/reports
    depends_on:
      postgres:
        condition: service_healthy
      kafka:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 15s
    restart: unless-stopped

volumes:
  report-data:
    driver: local
```

### Kubernetes (Production)

**1. Create namespace and secrets:**
```bash
kubectl create namespace compliance

# Create secrets (use sealed-secrets or Vault in production)
kubectl create secret generic compliance-secrets \
  --namespace compliance \
  --from-literal=database-url="postgres://..." \
  --from-literal=audit-hmac-secret="..." \
  --from-literal=encryption-key="..." \
  --from-file=jwt-public-key=./secrets/jwt_public.pem \
  --from-file=tse-client-cert=./secrets/tse_client.pem
```

**2. Apply Kubernetes manifests:**
```bash
# Apply all manifests
kubectl apply -k kubernetes/overlays/production/

# Or apply individually
kubectl apply -f kubernetes/base/configmap.yaml -n compliance
kubectl apply -f kubernetes/base/deployment.yaml -n compliance
kubectl apply -f kubernetes/base/service.yaml -n compliance
kubectl apply -f kubernetes/base/hpa.yaml -n compliance
kubectl apply -f kubernetes/base/network-policy.yaml -n compliance
kubectl apply -f kubernetes/base/pod-disruption-budget.yaml -n compliance
```

**`kubernetes/base/deployment.yaml` (excerpt):**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: compliance-service
  namespace: compliance
  labels:
    app: compliance-service
    version: "1.8.3"
    compliance.gdpr/enabled: "true"
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  selector:
    matchLabels:
      app: compliance-service
  template:
    metadata:
      labels:
        app: compliance-service
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: compliance-service
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        fsGroup: 65534
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: compliance-service
          image: compliance-service:1.8.3
          ports:
            - containerPort: 8080
              name: http
          env:
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
              cpu: "250m"
              memory: "256Mi"
            limits:
              cpu: "1000m"
              memory: "1Gi"
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop: ["ALL"]
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 15
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          volumeMounts:
            - name: secrets
              mountPath: /secrets
              readOnly: true
            - name: tmp
              mountPath: /tmp
      volumes:
        - name: secrets
          secret:
            secretName: compliance-secrets
        - name: tmp
          emptyDir: {}
```

**3. Run database migrations in Kubernetes:**
```bash
kubectl create job compliance-migrate \
  --namespace compliance \
  --image=compliance-service:1.8.3 \
  -- ./compliance-service migrate up

kubectl wait --for=condition=complete \
  job/compliance-migrate \
  --namespace compliance \
  --timeout=120s
```

**4. Verify deployment:**
```bash
kubectl rollout status deployment/compliance-service -n compliance
kubectl get pods -n compliance -l app=compliance-service
kubectl exec -n compliance deploy/compliance-service -- wget -qO- http://localhost:8080/health | jq .
```

### Horizontal Pod Autoscaling

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: compliance-service-hpa
  namespace: compliance
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: compliance-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

---

## Compliance Notes

### 🇪🇺 DSGVO / GDPR (EU) 2016/679

| Requirement | Implementation |
|-------------|---------------|
| **Art. 5** — Data minimisation & storage limitation | Automated retention policies with configurable purge schedules |
| **Art. 7** — Conditions for consent | Consent versioning, double opt-in support, easy withdrawal |
| **Art. 12-20** — Data subject rights | Full DSAR workflow with 30-day deadline enforcement |
| **Art. 25** — Data protection by design | Minimal data collection, pseudonymization, encryption at rest |
| **Art. 30** — Records of processing | Automated Verarbeitungsverzeichnis report generation |
| **Art. 32** — Security of processing | AES-256 encryption, TLS 1.3, RBAC, audit logging |
| **Art. 33-34** — Breach notification | 72-hour countdown timer, automated DPA/subject notification |
| **Art. 35** — DPIA | High-risk processing detection and DPIA trigger logging |

### 🇩🇪 BDSG (Bundesdatenschutzgesetz)

- **§26 BDSG** — Employee data processing: All processing of driver/employee data is logged with legal basis `Art6_1b_DSGVO` + `§26_BDSG`
- **§64 BDSG** — Technical and organizational measures: Documented TOMs linked to each processing activity
- **§77 BDSG** — Obligation to designate a DPO: DPO contact details stored and used for incident escalation

### 🚖 PBefG — Personenbeförderungsgesetz

- **P-Schein (§47 PBefG)** — Validity tracking: All drivers require a valid *Personenbeförderungsschein*. The service tracks expiry dates, sends 60/30/7-day renewal alerts, and creates audit entries for every verification event.
- **Fahrerlaubnis zur Fahrgastbeförderung (§48 StVZO)** — Special passenger transport license tracking with automated Kraftfahrtbundesamt integration audit trail.
- **Pflichtversicherungsnachweis** — Mandatory vehicle insurance compliance records per **§1 PflVG**.
- **Fahrzeugzulassung** — Vehicle registration compliance for all registered platform vehicles.

### 🧾 KassenSichV — Kassensicherungsverordnung

- **TSE (Technische Sicherheitseinrichtung)** — All fare transactions are signed by a certified TSE device per KassenSichV §2. Signatures use ECDSA with SHA-384 per BSI TR-03151.
- **DSFinV-K** — Digital interface standard for financial transaction exports to Finanzverwaltung.
- **Tagesabschluss** — Daily closing reports generated and stored per KassenSichV §6.
- **Unveränderbarkeit** — TSE transaction records have a 10-year legal hold per **§147 AO** (Abgabenordnung) and cannot be deleted or modified.

### 🏛️ Supervisory Authorities (Aufsichtsbehörden)

The service is pre-configured with contact information for relevant German supervisory authorities:

| Authority | Jurisdiction | Contact |
|-----------|-------------|--------|
| BfDI | Federal (Bundesbehörden) | poststelle@bfdi.bund.de |
| LDA Bayern | Bayern | poststelle@lda.bayern.de |
| BlnBDI | Berlin | mailbox@datenschutz-berlin.de |
| LfDI BW | Baden-Württemberg | poststelle@lfdi.bwl.de |
| HmbBfDI | Hamburg | mailbox@datenschutz.hamburg.de |

The applicable supervisory authority is automatically selected based on the company's registered Hauptsitz (principal place of business) and the location of the data breach.

### 🔐 Data Retention Schedule (Aufbewahrungsfristen)

| Data Category | Retention Period | Legal Basis |
|--------------|-----------------|-------------|
| Fiscal / TSE records | 10 years | §147 AO |
| Commercial correspondence | 6 years | §257 HGB |
| Driver license records | Duration of contract + 3 years | §147 AO |
| Personal data (contract) | Contract duration + 3 years | Art. 6(1)(b) DSGVO |
| Marketing profiles | 2 years or until withdrawal | Art. 6(1)(a) DSGVO |
| Location data (real-time) | 72 hours | Art. 5(1)(e) DSGVO |
| Location data (aggregated) | 6 months | Art. 5(1)(e) DSGVO |
| Audit logs | 3 years | BDSG + internal policy |
| Incident records | 5 years | Art. 5(2) DSGVO (accountability) |

---

## Development

### Running Tests

```bash
# Unit tests
go test ./... -v -count=1

# Unit tests with coverage
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -html=coverage.out -o coverage.html

# Integration tests (requires running infrastructure)
go test ./tests/integration/... -v -tags=integration

# Compliance-specific tests
go test ./internal/compliance/... -v -run TestGDPR
go test ./internal/audit/... -v -run TestHashChain

# Race condition detection
go test ./... -race -count=3
```

### Local Development Setup

```bash
# Start infrastructure only
docker compose up -d postgres kafka redis

# Wait for services to be ready
until docker compose exec postgres pg_isready; do sleep 1; done

# Run migrations
go run cmd/migrate/main.go up

# Seed development data
go run cmd/seed/main.go --env=development

# Start the service with hot reload
air -c .air.toml
```

### Code Structure

```
.
├── cmd/
│   ├── server/         # Main application entrypoint
│   ├── migrate/        # Database migration CLI
│   └── seed/           # Development seed data
├── internal/
│   ├── api/            # HTTP handlers and routing (Gin)
│   ├── audit/          # Audit log service + hash chain logic
│   ├── compliance/     # Compliance report generation
│   ├── consent/        # Consent management
│   ├── gdpr/           # DSAR workflow management
│   ├── incident/       # Incident tracking + breach notifications
│   ├── retention/      # Retention policy engine + purge jobs
│   ├── tse/            # TSE (KassenSichV) integration
│   ├── kafka/          # Kafka producers and consumers
│   ├── repository/     # GORM database repositories
│   ├── scheduler/      # Cron job management
│   └── middleware/     # Auth, logging, rate limiting
├── migrations/         # SQL migration files
├── kubernetes/         # K8s manifests
├── tests/
│   ├── unit/
│   └── integration/
├── .env.example
├── docker-compose.yml
├── Dockerfile
└── README.md
```

---

## License

This software is proprietary and confidential. Unauthorized copying, distribution, or use is strictly prohibited.

**© 2024 [Your Company GmbH], Germany. All rights reserved.**

---

*Für Fragen zum Datenschutz wenden Sie sich bitte an: datenschutz@company.de*

*Datenschutzbeauftragter (DSB) / Data Protection Officer (DPO): dpo@company.de*

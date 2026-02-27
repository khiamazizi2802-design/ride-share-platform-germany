# Notification Service

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()
[![Coverage](https://img.shields.io/badge/coverage-87%25-green)]()
[![License](https://img.shields.io/badge/license-MIT-blue)]()
[![Version](https://img.shields.io/badge/version-2.4.1-informational)]()

A production-grade, multi-channel notification microservice responsible for delivering transactional and marketing notifications via Email, SMS, and Push channels. Built with DSGVO/GDPR compliance at its core, supporting data residency requirements for German and EU deployments.

---

## Table of Contents

- [Service Overview](#service-overview)
- [Architecture](#architecture)
- [API Documentation](#api-documentation)
- [Configuration](#configuration)
- [German Compliance (DSGVO)](#german-compliance-dsgvo)
- [Integration Points](#integration-points)
- [Deployment](#deployment)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [Changelog](#changelog)

---

## Service Overview

### Description

The Notification Service is a stateless, horizontally scalable microservice that serves as the single point of truth for all outbound user communications across the platform. It abstracts provider-specific complexity behind a unified API, enabling other services to send notifications without knowing about the underlying delivery infrastructure.

The service supports templated and raw notifications, delivery receipts, retry logic with exponential backoff, and per-user communication preferences. All personally identifiable information (PII) is handled in accordance with DSGVO (Datenschutz-Grundverordnung) and stored exclusively on infrastructure located within the European Union.

### Key Features

| Feature | Description |
|---|---|
| Multi-Channel Delivery | Email (SendGrid), SMS (Twilio), Push (FCM/APNS) |
| Template Engine | Handlebars-based templates with i18n support (de, en, fr, es) |
| Delivery Tracking | Per-notification status lifecycle with webhook callbacks |
| Retry Logic | Configurable exponential backoff with dead-letter queue |
| Preference Management | Per-user opt-in/opt-out and channel preferences |
| Rate Limiting | Per-user and per-tenant rate limiting to prevent spam |
| DSGVO Compliance | Data minimization, right to erasure, audit logging |
| Idempotency | Idempotency keys to prevent duplicate delivery |
| Batching | Bulk send endpoint for up to 10,000 recipients |
| Observability | Prometheus metrics, structured JSON logging, OpenTelemetry tracing |

### Technology Stack

| Layer | Technology | Version |
|---|---|---|
| Runtime | Node.js | 20.x LTS |
| Framework | Fastify | 4.x |
| Language | TypeScript | 5.x |
| Database | PostgreSQL | 15 |
| Cache / Queue | Redis | 7.x |
| Message Broker | RabbitMQ | 3.12 |
| ORM | Prisma | 5.x |
| Email Provider | SendGrid | API v3 |
| SMS Provider | Twilio | REST API 2010-04-01 |
| Push Provider | Firebase Cloud Messaging (FCM) | HTTP v1 |
| Container | Docker | 24.x |
| Orchestration | Kubernetes | 1.28+ |
| Service Mesh | Istio | 1.19 |
| Observability | Prometheus + Grafana + Jaeger | - |

---

## Architecture

```
âââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ
â                    Notification Service                      â
â                                                             â
â  ââââââââââââ   ââââââââââââââââ   âââââââââââââââââââââ   â
â  â REST API â   â  RabbitMQ    â   â  Scheduler (Cron) â   â
â  â (Fastify)â   â  Consumer    â   â  (Digest / Retry) â   â
â  ââââââ¬ââââââ   ââââââââ¬ââââââââ   âââââââââââ¬ââââââââââ   â
â       â                â                     â             â
â       ââââââââââââââââââ¼ââââââââââââââââââââââ             â
â                        â¼                                   â
â               âââââââââââââââââââ                          â
â               â  Channel Router â                          â
â               âââââââââ¬ââââââââââ                          â
â          ââââââââââââââ¼ââââââââââââââ                      â
â          â¼            â¼             â¼                      â
â   ââââââââââââââ âââââââââââ ââââââââââââ                 â
â   âEmail Workerâ âSMS Work.â âPush Work.â                 â
â   â(SendGrid)  â â(Twilio) â â(FCM/APNS)â                 â
â   ââââââââââââââ âââââââââââ ââââââââââââ                 â
â                                                             â
â  âââââââââââââââââââââââââââââââââââââââââââââââââââââââââ â
â  â             Data Layer                                â â
â  â  PostgreSQL (notifications, templates, preferences)   â â
â  â  Redis (rate limiting, idempotency, caching)          â â
â  âââââââââââââââââââââââââââââââââââââââââââââââââââââââââ â
âââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ
```

### Notification Lifecycle

```
CREATED â QUEUED â PROCESSING â SENT â DELIVERED
                                     â FAILED â RETRYING â SENT
                                                          â DEAD_LETTERED
```

---

## API Documentation

### Base URL

```
Production:  https://notifications.internal.example.com/api/v2
Staging:     https://notifications.staging.internal.example.com/api/v2
Local:       http://localhost:3040/api/v2
```

### Authentication

All endpoints require a valid service-to-service JWT bearer token issued by the platform Identity Service. Tokens must include the `notifications:write` or `notifications:read` scope as appropriate.

```http
Authorization: Bearer <service_jwt_token>
```

Tokens expire after 1 hour. Use the Identity Service's `/oauth/token` endpoint with `grant_type=client_credentials` to obtain a fresh token.

---

### Endpoints

#### 1. Send Notification

Send a single notification to one recipient via one or more channels.

```
POST /notifications
```

**Request Headers**

| Header | Required | Description |
|---|---|---|
| `Authorization` | Yes | Bearer JWT token |
| `Content-Type` | Yes | `application/json` |
| `Idempotency-Key` | Recommended | UUID v4 to prevent duplicate delivery |
| `X-Correlation-ID` | Recommended | Trace ID propagated across services |

**Request Body**

```json
{
  "recipient": {
    "userId": "usr_01H9XKQPZB4FVWGM3TNDRJYCE",
    "email": "max.mustermann@example.de",
    "phone": "+4915123456789",
    "deviceTokens": [
      "fcm_token_abc123"
    ],
    "locale": "de"
  },
  "channels": ["email", "push"],
  "templateId": "order_confirmation_v2",
  "templateData": {
    "orderNumber": "ORD-2024-88412",
    "customerName": "Max Mustermann",
    "totalAmount": "â¬124,99",
    "estimatedDelivery": "2024-11-20"
  },
  "priority": "high",
  "scheduledAt": null,
  "metadata": {
    "sourceService": "order-service",
    "sourceEventId": "evt_01H9XKQPZB4FVWGM3TNDRJYCE",
    "tags": ["transactional", "order"]
  }
}
```

**Request Body Fields**

| Field | Type | Required | Description |
|---|---|---|---|
| `recipient.userId` | string | Yes | Internal user identifier |
| `recipient.email` | string | Conditional | Required if `email` channel is specified |
| `recipient.phone` | string | Conditional | E.164 format. Required if `sms` channel is specified |
| `recipient.deviceTokens` | string[] | Conditional | FCM/APNS tokens. Required if `push` channel is specified |
| `recipient.locale` | string | No | BCP 47 locale code. Defaults to `de` |
| `channels` | string[] | Yes | One or more of: `email`, `sms`, `push` |
| `templateId` | string | Conditional | Template identifier. Required unless `rawContent` is provided |
| `rawContent` | object | Conditional | See Raw Content below. Required unless `templateId` is provided |
| `templateData` | object | No | Key-value pairs injected into the template |
| `priority` | string | No | `low`, `normal`, `high`. Default: `normal` |
| `scheduledAt` | string | No | ISO 8601 datetime for scheduled delivery. Null for immediate |
| `metadata` | object | No | Arbitrary metadata attached to the notification record |

**Success Response â 202 Accepted**

```json
{
  "notificationId": "ntf_01HBKRM5PQWXY2ZVTGF3CNAEJ8",
  "status": "QUEUED",
  "channels": {
    "email": { "status": "QUEUED", "provider": "sendgrid" },
    "push": { "status": "QUEUED", "provider": "fcm" }
  },
  "scheduledAt": null,
  "createdAt": "2024-11-15T10:32:00.000Z"
}
```

**Error Responses**

| Status | Code | Description |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Request body failed schema validation |
| 400 | `INVALID_CHANNEL` | Unknown or unsupported channel specified |
| 400 | `MISSING_RECIPIENT_FIELD` | Channel-required recipient field is absent |
| 400 | `TEMPLATE_NOT_FOUND` | `templateId` does not match any registered template |
| 409 | `IDEMPOTENCY_CONFLICT` | A notification with this `Idempotency-Key` already exists |
| 422 | `USER_OPTED_OUT` | Recipient has opted out of this notification category |
| 429 | `RATE_LIMIT_EXCEEDED` | Per-user or per-tenant rate limit reached |
| 503 | `PROVIDER_UNAVAILABLE` | Upstream provider is unreachable; notification enqueued for retry |

**Error Response Body**

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request body validation failed",
    "details": [
      {
        "field": "recipient.email",
        "message": "must be a valid email address"
      }
    ],
    "correlationId": "corr_01HBKRM5PQWXY2ZVTGF3CNAEJ8",
    "timestamp": "2024-11-15T10:32:00.000Z"
  }
}
```

---

#### 2. Send Batch Notification

Send the same notification to multiple recipients in a single API call. Maximum 10,000 recipients per request.

```
POST /notifications/batch
```

**Request Body**

```json
{
  "recipients": [
    {
      "userId": "usr_01H9XKQPZB4FVWGM3TNDRJYCE",
      "email": "user1@example.de",
      "locale": "de"
    },
    {
      "userId": "usr_02H9XKQPZB4FVWGM3TNDRJYCF",
      "email": "user2@example.de",
      "locale": "en"
    }
  ],
  "channels": ["email"],
  "templateId": "newsletter_november_2024",
  "templateData": {
    "campaignName": "November Newsletter"
  },
  "priority": "low"
}
```

**Success Response â 202 Accepted**

```json
{
  "batchId": "bat_01HBKRM5PQWXY2ZVTGF3CNAEJ9",
  "status": "ACCEPTED",
  "totalRecipients": 2,
  "estimatedCompletionAt": "2024-11-15T11:00:00.000Z",
  "statusWebhookUrl": "https://notifications.internal.example.com/api/v2/batches/bat_01HBKRM5PQWXY2ZVTGF3CNAEJ9"
}
```

---

#### 3. Get Notification Status

Retrieve the current delivery status and lifecycle history of a notification.

```
GET /notifications/{notificationId}
```

**Path Parameters**

| Parameter | Type | Description |
|---|---|---|
| `notificationId` | string | The notification ID returned from the send endpoint |

**Success Response â 200 OK**

```json
{
  "notificationId": "ntf_01HBKRM5PQWXY2ZVTGF3CNAEJ8",
  "status": "DELIVERED",
  "recipient": {
    "userId": "usr_01H9XKQPZB4FVWGM3TNDRJYCE"
  },
  "channels": {
    "email": {
      "status": "DELIVERED",
      "provider": "sendgrid",
      "providerMessageId": "sg_msg_abc123xyz",
      "sentAt": "2024-11-15T10:32:05.000Z",
      "deliveredAt": "2024-11-15T10:32:08.000Z",
      "openedAt": null,
      "clickedAt": null
    }
  },
  "attempts": 1,
  "createdAt": "2024-11-15T10:32:00.000Z",
  "updatedAt": "2024-11-15T10:32:08.000Z",
  "metadata": {
    "sourceService": "order-service",
    "tags": ["transactional", "order"]
  }
}
```

---

#### 4. Cancel Scheduled Notification

Cancel a notification that has not yet been dispatched to a provider. Only applicable to notifications in `QUEUED` or `SCHEDULED` status.

```
DELETE /notifications/{notificationId}
```

**Success Response â 200 OK**

```json
{
  "notificationId": "ntf_01HBKRM5PQWXY2ZVTGF3CNAEJ8",
  "status": "CANCELLED",
  "cancelledAt": "2024-11-15T10:35:00.000Z"
}
```

**Error Response â 409 Conflict**

```json
{
  "error": {
    "code": "CANNOT_CANCEL",
    "message": "Notification is already in SENT status and cannot be cancelled.",
    "currentStatus": "SENT"
  }
}
```

---

#### 5. List Templates

List all available notification templates with optional filtering.

```
GET /templates
```

**Query Parameters**

| Parameter | Type | Description |
|---|---|---|
| `channel` | string | Filter by channel: `email`, `sms`, `push` |
| `locale` | string | Filter by locale: `de`, `en`, `fr`, `es` |
| `page` | integer | Page number (default: 1) |
| `pageSize` | integer | Results per page (default: 20, max: 100) |

**Success Response â 200 OK**

```json
{
  "data": [
    {
      "templateId": "order_confirmation_v2",
      "name": "Order Confirmation",
      "channel": "email",
      "supportedLocales": ["de", "en", "fr"],
      "variables": ["orderNumber", "customerName", "totalAmount", "estimatedDelivery"],
      "category": "transactional",
      "version": 2,
      "createdAt": "2024-01-10T09:00:00.000Z",
      "updatedAt": "2024-09-01T14:22:00.000Z"
    }
  ],
  "pagination": {
    "page": 1,
    "pageSize": 20,
    "total": 48,
    "totalPages": 3
  }
}
```

---

#### 6. Get User Preferences

Retrieve communication preferences for a specific user.

```
GET /preferences/{userId}
```

**Success Response â 200 OK**

```json
{
  "userId": "usr_01H9XKQPZB4FVWGM3TNDRJYCE",
  "globalOptOut": false,
  "channels": {
    "email": { "enabled": true, "optedOutAt": null },
    "sms": { "enabled": false, "optedOutAt": "2024-08-01T12:00:00.000Z" },
    "push": { "enabled": true, "optedOutAt": null }
  },
  "categories": {
    "transactional": { "enabled": true },
    "marketing": { "enabled": false, "optedOutAt": "2024-06-15T08:30:00.000Z" },
    "security": { "enabled": true }
  },
  "updatedAt": "2024-08-01T12:00:00.000Z"
}
```

---

#### 7. Update User Preferences

Update communication preferences for a specific user. This endpoint is DSGVO-critical â all changes are audit-logged.

```
PUT /preferences/{userId}
```

**Request Body**

```json
{
  "globalOptOut": false,
  "channels": {
    "sms": { "enabled": false }
  },
  "categories": {
    "marketing": { "enabled": false }
  }
}
```

**Success Response â 200 OK**

```json
{
  "userId": "usr_01H9XKQPZB4FVWGM3TNDRJYCE",
  "updated": true,
  "updatedAt": "2024-11-15T10:40:00.000Z"
}
```

---

#### 8. Erase User Data (DSGVO Right to Erasure)

Permanently anonymize or delete all PII associated with a user, in compliance with DSGVO Article 17. Notification records are retained for audit purposes but all PII fields are overwritten with anonymized values.

```
DELETE /users/{userId}/data
```

**Request Headers**

| Header | Required | Description |
|---|---|---|
| `X-Erasure-Request-Id` | Yes | Unique ID from the Data Subject Request tracking system |

**Success Response â 200 OK**

```json
{
  "userId": "usr_01H9XKQPZB4FVWGM3TNDRJYCE",
  "erasureStatus": "COMPLETED",
  "erasureRequestId": "dsr_01HBKRM5PQWXY2ZVTGF3CNAEJ0",
  "recordsAnonymized": 142,
  "preferencesDeleted": true,
  "deviceTokensRevoked": true,
  "completedAt": "2024-11-15T10:41:00.000Z"
}
```

---

#### 9. Health Check

```
GET /health
```

**Success Response â 200 OK**

```json
{
  "status": "healthy",
  "version": "2.4.1",
  "uptime": 1209600,
  "checks": {
    "database": { "status": "ok", "responseTimeMs": 2 },
    "redis": { "status": "ok", "responseTimeMs": 1 },
    "rabbitmq": { "status": "ok", "queueDepth": 34 },
    "sendgrid": { "status": "ok" },
    "twilio": { "status": "ok" },
    "fcm": { "status": "ok" }
  }
}
```

**Degraded Response â 207 Multi-Status**

```json
{
  "status": "degraded",
  "checks": {
    "database": { "status": "ok", "responseTimeMs": 3 },
    "sendgrid": { "status": "error", "message": "Connection timeout", "since": "2024-11-15T10:00:00.000Z" }
  }
}
```

---

#### 10. Metrics

Prometheus-compatible metrics endpoint. Accessible only within the cluster network or via authenticated scraping.

```
GET /metrics
```

**Key Metrics Exported**

| Metric | Type | Description |
|---|---|---|
| `notifications_sent_total` | Counter | Total notifications sent, labeled by channel and status |
| `notifications_delivery_duration_seconds` | Histogram | End-to-end delivery latency per channel |
| `notifications_queue_depth` | Gauge | Current RabbitMQ queue depth per channel |
| `notifications_retry_total` | Counter | Total retry attempts by channel and reason |
| `notifications_dead_letter_total` | Counter | Notifications that exhausted all retries |
| `provider_api_request_duration_seconds` | Histogram | Provider API call latency |
| `rate_limit_rejected_total` | Counter | Requests rejected due to rate limiting |

---

## Configuration

### Environment Variables

All configuration is provided through environment variables. Sensitive values must be stored in Kubernetes Secrets or HashiCorp Vault and never committed to source control.

#### Application

| Variable | Required | Default | Description |
|---|---|---|---|
| `NODE_ENV` | Yes | `production` | Runtime environment: `development`, `staging`, `production` |
| `PORT` | No | `3040` | HTTP server port |
| `LOG_LEVEL` | No | `info` | Logging verbosity: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | No | `json` | Log format: `json`, `pretty` |
| `SERVICE_NAME` | No | `notification-service` | Service name used in logs and traces |
| `API_BASE_PATH` | No | `/api/v2` | Base path prefix for all routes |
| `MAX_REQUEST_BODY_SIZE` | No | `5mb` | Maximum incoming request body size |

#### Database

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | â | PostgreSQL connection string (DSN format) |
| `DATABASE_POOL_MIN` | No | `2` | Minimum DB connection pool size |
| `DATABASE_POOL_MAX` | No | `20` | Maximum DB connection pool size |
| `DATABASE_SSL` | No | `true` | Enforce TLS for DB connections |
| `DATABASE_SSL_REJECT_UNAUTHORIZED` | No | `true` | Reject self-signed certificates |
| `DATABASE_SCHEMA` | No | `notifications` | PostgreSQL schema name |

#### Redis

| Variable | Required | Default | Description |
|---|---|---|---|
| `REDIS_URL` | Yes | â | Redis connection URL (`redis://` or `rediss://` for TLS) |
| `REDIS_KEY_PREFIX` | No | `notif:` | Namespace prefix for all Redis keys |
| `REDIS_TLS` | No | `true` | Enable TLS for Redis connection |
| `CACHE_TTL_SECONDS` | No | `300` | Default TTL for cached template lookups |
| `IDEMPOTENCY_KEY_TTL_SECONDS` | No | `86400` | TTL for idempotency keys (24 hours) |

#### RabbitMQ

| Variable | Required | Default | Description |
|---|---|---|---|
| `RABBITMQ_URL` | Yes | â | AMQP connection URL |
| `RABBITMQ_EXCHANGE` | No | `notifications` | Main topic exchange name |
| `RABBITMQ_PREFETCH` | No | `10` | Consumer prefetch count per channel worker |
| `RABBITMQ_MAX_RETRIES` | No | `5` | Max delivery attempts before dead-lettering |
| `RABBITMQ_RETRY_DELAY_MS` | No | `5000` | Base delay for first retry (exponential backoff) |

#### SendGrid (Email)

| Variable | Required | Default | Description |
|---|---|---|---|
| `SENDGRID_API_KEY` | Yes | â | SendGrid API key (starts with `SG.`) |
| `SENDGRID_FROM_EMAIL` | Yes | â | Default sender email address |
| `SENDGRID_FROM_NAME` | No | `Benachrichtigungen` | Default sender display name |
| `SENDGRID_REPLY_TO` | No | â | Default reply-to address |
| `SENDGRID_SANDBOX_MODE` | No | `false` | Enable sandbox mode (emails not actually sent) |
| `SENDGRID_TIMEOUT_MS` | No | `10000` | API request timeout in milliseconds |

#### Twilio (SMS)

| Variable | Required | Default | Description |
|---|---|---|---|
| `TWILIO_ACCOUNT_SID` | Yes | â | Twilio Account SID |
| `TWILIO_AUTH_TOKEN` | Yes | â | Twilio Auth Token |
| `TWILIO_FROM_NUMBER` | Yes | â | E.164-formatted sender number or Messaging Service SID |
| `TWILIO_MESSAGING_SERVICE_SID` | No | â | Twilio Messaging Service SID (overrides `FROM_NUMBER` if set) |
| `TWILIO_TIMEOUT_MS` | No | `10000` | API request timeout in milliseconds |
| `TWILIO_MAX_SMS_LENGTH` | No | `1600` | Maximum SMS content length in characters |

#### Firebase Cloud Messaging (Push)

| Variable | Required | Default | Description |
|---|---|---|---|
| `FCM_PROJECT_ID` | Yes | â | Firebase project ID |
| `FCM_CLIENT_EMAIL` | Yes | â | Firebase service account client email |
| `FCM_PRIVATE_KEY` | Yes | â | Firebase service account private key (PEM format, newlines as `\n`) |
| `FCM_TIMEOUT_MS` | No | `10000` | API request timeout in milliseconds |
| `APNS_KEY_ID` | Conditional | â | APNs key ID (required if iOS push is enabled) |
| `APNS_TEAM_ID` | Conditional | â | Apple Developer Team ID |
| `APNS_PRIVATE_KEY` | Conditional | â | APNs private key in PEM format |
| `APNS_BUNDLE_ID` | Conditional | â | iOS app bundle identifier |

#### Rate Limiting

| Variable | Required | Default | Description |
|---|---|---|---|
| `RATE_LIMIT_ENABLED` | No | `true` | Toggle global rate limiting |
| `RATE_LIMIT_USER_PER_HOUR` | No | `100` | Max notifications per user per hour |
| `RATE_LIMIT_TENANT_PER_MINUTE` | No | `5000` | Max notifications per tenant per minute |
| `RATE_LIMIT_BATCH_PER_HOUR` | No | `10` | Max batch requests per tenant per hour |

#### Observability

| Variable | Required | Default | Description |
|---|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | â | OpenTelemetry collector endpoint |
| `OTEL_SERVICE_NAME` | No | `notification-service` | Service name reported to tracing backend |
| `METRICS_ENABLED` | No | `true` | Enable Prometheus `/metrics` endpoint |
| `METRICS_PORT` | No | `9090` | Separate port for metrics scraping (optional) |

### Example `.env` File (Development)

```env
NODE_ENV=development
PORT=3040
LOG_LEVEL=debug
LOG_FORMAT=pretty

DATABASE_URL=postgresql://notif_user:password@localhost:5432/notifications_dev
DATABASE_SSL=false

REDIS_URL=redis://localhost:6379
REDIS_TLS=false

RABBITMQ_URL=amqp://guest:guest@localhost:5672

SENDGRID_API_KEY=SG.test_key
SENDGRID_FROM_EMAIL=dev-notifications@example.de
SENDGRID_SANDBOX_MODE=true

TWILIO_ACCOUNT_SID=ACtest
TWILIO_AUTH_TOKEN=test_token
TWILIO_FROM_NUMBER=+4915100000000

FCM_PROJECT_ID=example-project-dev
FCM_CLIENT_EMAIL=firebase-adminsdk@example-project-dev.iam.gserviceaccount.com
FCM_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n..."
```

---

## German Compliance (DSGVO)

This service is designed and operated in full compliance with the **Datenschutz-Grundverordnung (DSGVO / GDPR)**, Regulation (EU) 2016/679. The following sections document the specific technical and organisational measures implemented.

### Data Residency

- **All data is stored exclusively within the European Union.** PostgreSQL, Redis, and RabbitMQ are hosted on infrastructure located in **Frankfurt, Germany (eu-central-1 / Germany West Central)**.
- No PII is transmitted to infrastructure outside the EEA without explicit data transfer agreements (Standard Contractual Clauses) where technically required (e.g., SendGrid EU data processing).
- SendGrid is configured to use the **EU regional endpoint** (`https://api.eu.sendgrid.com`) to ensure email data does not transit US infrastructure.
- Twilio is configured under a **DSGVO Data Processing Agreement (DPA)** with EU data storage.
- Firebase Cloud Messaging data is treated as transient metadata only. No message payload PII is stored by FCM beyond delivery.

### Data Minimization

| Data Category | Stored | Retention | Notes |
|---|---|---|---|
| User ID | Yes | Lifetime of record | Pseudonymized identifier only |
| Email address | Yes | 90 days after send | Encrypted at rest (AES-256) |
| Phone number | Yes | 90 days after send | Encrypted at rest (AES-256) |
| Device tokens | Yes | Until revoked or 90 days | Revoked on user erasure |
| Notification content | No | N/A | Template data not stored; only template ID is logged |
| Delivery metadata | Yes | 2 years | Status, timestamps, provider IDs |
| IP addresses | No | N/A | Not logged or stored |
| Open/click tracking | Opt-in only | 90 days | Only collected if user consented |

### Data Retention Policies

| Record Type | Retention Period | Deletion Method |
|---|---|---|
| Notification records (PII fields) | 90 days | Automated nightly job â fields overwritten with anonymized values |
| Anonymized notification metadata | 2 years | Hard delete after 2 years |
| User preferences | Until erasure request | Hard delete on Right to Erasure |
| Audit logs | 3 years | Immutable, then hard delete |
| Delivery attempt logs | 90 days | Hard delete |
| Dead-letter queue messages | 30 days | Hard delete |

Retention enforcement is handled by the `RetentionJobService` which runs at `02:00 CET` daily. Job execution is logged in the audit table with record counts.

### GDPR Features

| GDPR Article | Feature |
|---|---|
| Art. 7 â Consent | Per-category opt-in/out preference API. Consent timestamps recorded. |
| Art. 17 â Right to Erasure | `DELETE /users/{userId}/data` endpoint anonymizes all PII within 72 hours |
| Art. 20 â Data Portability | `GET /users/{userId}/data/export` returns all stored data as JSON (not listed above; internal use) |
| Art. 25 â Privacy by Design | PII encrypted at rest, no unnecessary data collected, data minimization enforced |
| Art. 30 â Records of Processing | Processing activity register maintained in Data Governance service |
| Art. 32 â Security of Processing | TLS 1.3 in transit, AES-256 at rest, access via mTLS only within the cluster |
| Art. 33 â Breach Notification | Automated alerting pipeline to DPO on anomalous data access patterns |

### Encryption

- **In transit:** TLS 1.3 enforced for all internal and external communication. Mutual TLS (mTLS) enforced within the Kubernetes service mesh via Istio.
- **At rest:** All PII fields (`email`, `phone`, `deviceTokens`) are encrypted at the application layer using AES-256-GCM before writing to PostgreSQL. Encryption keys are managed by HashiCorp Vault with automatic key rotation every 90 days.
- **Database:** PostgreSQL volume encryption enabled (LUKS) at the infrastructure layer.

### Audit Logging

All DSGVO-sensitive operations generate structured audit log entries persisted to a dedicated, tamper-evident `audit_events` table:

- Preference changes (who changed, old value, new value)
- Data erasure requests and completion
- Data export requests
- Any administrative access to raw PII

Audit logs are exported to the centralised SIEM system in real-time via a dedicated RabbitMQ exchange (`audit.events`).

### Data Processing Agreement

A signed Data Processing Agreement (DPA) must be in place with all upstream caller services before this service processes their users' PII. Contact the Data Protection Officer (DPO) at `dpo@example.de` to initiate the DPA process.

---

## Integration Points

### How Other Services Integrate

Other platform services can send notifications via two integration patterns:

#### Pattern A: Direct REST API Call (Synchronous)

Best for transactional notifications where the calling service needs immediate confirmation that the notification was accepted.

```typescript
// Example: Order Service calling Notification Service
const response = await fetch('https://notifications.internal.example.com/api/v2/notifications', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${serviceToken}`,
    'Content-Type': 'application/json',
    'Idempotency-Key': crypto.randomUUID(),
    'X-Correlation-ID': correlationId,
  },
  body: JSON.stringify({
    recipient: { userId: order.customerId, email: order.customerEmail, locale: 'de' },
    channels: ['email'],
    templateId: 'order_confirmation_v2',
    templateData: { orderNumber: order.id, totalAmount: order.total },
    priority: 'high',
    metadata: { sourceService: 'order-service', sourceEventId: event.id }
  }),
});
```

#### Pattern B: RabbitMQ Event (Asynchronous)

Best for fire-and-forget notifications triggered by domain events. Publish to the `notifications` exchange with the appropriate routing key.

**Exchange:** `notifications`
**Type:** `topic`
**Routing Key Convention:** `notify.<channel>.<priority>` (e.g., `notify.email.high`, `notify.sms.normal`)

**Event Payload Schema**

```json
{
  "eventId": "evt_01HBKRM5PQWXY2ZVTGF3CNAEJ8",
  "eventType": "notification.send.requested",
  "timestamp": "2024-11-15T10:32:00.000Z",
  "payload": {
    "recipient": { "userId": "usr_123", "email": "user@example.de" },
    "channels": ["email"],
    "templateId": "password_reset_v1",
    "templateData": { "resetLink": "https://..." },
    "priority": "high"
  }
}
```

```typescript
// Example: Auth Service publishing a notification event
await rabbitMQChannel.publish(
  'notifications',
  'notify.email.high',
  Buffer.from(JSON.stringify(eventPayload)),
  {
    contentType: 'application/json',
    persistent: true,
    messageId: eventPayload.eventId,
    timestamp: Math.floor(Date.now() / 1000),
    headers: { 'x-correlation-id': correlationId }
  }
);
```

---

### Webhook Documentation

The Notification Service can deliver real-time delivery status updates to registered webhook endpoints. Webhooks are registered per-tenant in the platform Admin API.

#### Webhook Events

| Event | Trigger |
|---|---|
| `notification.queued` | Notification accepted and queued |
| `notification.sent` | Notification dispatched to provider |
| `notification.delivered` | Provider confirmed delivery |
| `notification.failed` | Final delivery failure after all retries |
| `notification.bounced` | Email hard bounced (invalid address) |
| `notification.opened` | Email opened (if tracking enabled) |
| `notification.clicked` | Link clicked (if tracking enabled) |
| `batch.completed` | All recipients in a batch have a final status |

#### Webhook Payload

```json
{
  "webhookId": "wh_01HBKRM5PQWXY2ZVTGF3CNAEJ8",
  "event": "notification.delivered",
  "timestamp": "2024-11-15T10:32:08.000Z",
  "data": {
    "notificationId": "ntf_01HBKRM5PQWXY2ZVTGF3CNAEJ8",
    "userId": "usr_01H9XKQPZB4FVWGM3TNDRJYCE",
    "channel": "email",
    "status": "DELIVERED",
    "providerMessageId": "sg_msg_abc123xyz",
    "deliveredAt": "2024-11-15T10:32:08.000Z",
    "metadata": {
      "sourceService": "order-service"
    }
  }
}
```

#### Webhook Security

All webhook deliveries include an HMAC-SHA256 signature in the `X-Notification-Signature` header. Verify the signature before processing the payload:

```typescript
import { createHmac, timingSafeEqual } from 'crypto';

function verifyWebhookSignature(
  payload: string,
  signature: string,
  secret: string
): boolean {
  const expectedSignature = createHmac('sha256', secret)
    .update(payload, 'utf8')
    .digest('hex');
  const expected = Buffer.from(`sha256=${expectedSignature}`);
  const received = Buffer.from(signature);
  if (expected.length !== received.length) return false;
  return timingSafeEqual(expected, received);
}

// In your webhook handler:
const isValid = verifyWebhookSignature(
  req.rawBody,
  req.headers['x-notification-signature'],
  process.env.WEBHOOK_SECRET
);
if (!isValid) return res.status(401).end();
```

Webhook deliveries are retried up to 3 times with exponential backoff if the recipient endpoint returns a non-2xx status. The webhook endpoint must respond within 10 seconds.

---

### Provider Integration Details

#### SendGrid (Email)

- **API Version:** v3
- **EU Regional Endpoint:** `https://api.eu.sendgrid.com/v3/mail/send`
- **Features Used:** Dynamic Templates, Suppression Groups, Event Webhooks
- **Bounce Handling:** SendGrid event webhooks are registered for `bounce`, `spamreport`, and `unsubscribe` events. These events automatically update user preferences and suppress future sends.
- **Unsubscribe Groups:** Each notification category maps to a SendGrid Unsubscribe Group, enabling DSGVO-compliant one-click unsubscribe in email headers (`List-Unsubscribe`).
- **DPA:** SendGrid Data Processing Addendum signed. EU data processing endpoint enabled on account.

**Template Management:** Templates are authored in SendGrid and referenced by `templateId` in the service. The Notification Service does **not** store email HTML â it references SendGrid Dynamic Template IDs.

#### Twilio (SMS)

- **API Version:** 2010-04-01
- **Compliance:** Twilio GDPR DPA signed. EU data at rest enabled on Twilio account.
- **Number Type:** German long code (`+49`) for transactional SMS; short code for high-volume marketing (configured per use case).
- **Character Encoding:** Unicode (UCS-2) is auto-detected. Messages exceeding 160 characters (GSM-7) or 70 characters (UCS-2) are automatically split into multipart SMS.
- **Delivery Reports:** Twilio Status Callbacks are registered to receive `sent`, `delivered`, and `failed` events, which update the notification record in real time.
- **STOP Handling:** Twilio manages `STOP`/`STOP ALL` keyword opt-outs natively. The service subscribes to Twilio opt-out webhooks to sync preferences.

#### Firebase Cloud Messaging (FCM)

- **API Version:** FCM HTTP v1 API (legacy API deprecated)
- **Auth:** Service Account (OAuth 2.0 Bearer Token), rotated every 60 minutes.
- **Data Payload:** Only notification metadata (title, body, image URL) is sent via FCM. No PII is included in the FCM payload.
- **Delivery:** FCM does not guarantee delivery receipts for all devices. Token validity is checked on `INVALID_ARGUMENT` or `NOT_FOUND` error codes â invalid tokens are pruned automatically.
- **iOS (APNs):** APNs integration is handled via FCM for cross-platform consistency. Native APNs is used as a fallback when FCM is unavailable for iOS devices.

---

## Deployment

### Prerequisites

- Docker 24.x or later
- kubectl 1.28+ configured with cluster access
- Access to the internal container registry
- Kubernetes namespace `notifications` created
- Kubernetes Secrets pre-populated (see Configuration section)

### Docker

#### Build

```bash
# Build production image
docker build \
  --target production \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  --tag registry.internal.example.com/notification-service:2.4.1 \
  --tag registry.internal.example.com/notification-service:latest \
  .

# Push to registry
docker push registry.internal.example.com/notification-service:2.4.1
docker push registry.internal.example.com/notification-service:latest
```

#### Run Locally

```bash
# Start all dependencies
docker compose -f docker-compose.dev.yml up -d postgres redis rabbitmq

# Run migrations
npx prisma migrate deploy

# Start the service
docker compose -f docker-compose.dev.yml up notification-service
```

#### Dockerfile Summary

The Dockerfile uses a multi-stage build:

1. **deps** â Installs production `node_modules`
2. **builder** â Compiles TypeScript to JavaScript
3. **production** â Lean final image with compiled assets only

The final image runs as a non-root user (`uid=1001`) and exposes port `3040`.

---

### Kubernetes

#### Namespace and Secrets

```bash
# Create namespace
kubectl create namespace notifications

# Create secrets from Vault or sealed-secrets
kubectl create secret generic notification-service-secrets \
  --namespace notifications \
  --from-literal=DATABASE_URL="${DATABASE_URL}" \
  --from-literal=REDIS_URL="${REDIS_URL}" \
  --from-literal=RABBITMQ_URL="${RABBITMQ_URL}" \
  --from-literal=SENDGRID_API_KEY="${SENDGRID_API_KEY}" \
  --from-literal=TWILIO_ACCOUNT_SID="${TWILIO_ACCOUNT_SID}" \
  --from-literal=TWILIO_AUTH_TOKEN="${TWILIO_AUTH_TOKEN}" \
  --from-literal=FCM_PRIVATE_KEY="${FCM_PRIVATE_KEY}" \
  --from-literal=FCM_CLIENT_EMAIL="${FCM_CLIENT_EMAIL}"
```

#### Deploy

```bash
# Apply manifests
kubectl apply -f k8s/ --namespace notifications

# Or using Helm
helm upgrade --install notification-service ./helm/notification-service \
  --namespace notifications \
  --values ./helm/notification-service/values.production.yaml \
  --set image.tag=2.4.1 \
  --wait
```

#### Key Kubernetes Configuration Notes

**Resource Requests and Limits**

```yaml
resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 512Mi
```

**Horizontal Pod Autoscaler**

The service is configured to scale between 2 and 10 replicas based on CPU utilization (target: 70%) and custom RabbitMQ queue depth metric (scale-up threshold: 500 messages per pod).

**Pod Disruption Budget**

A PDB ensures a minimum of 2 pods are always available during rolling updates or node drains:

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: notification-service-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: notification-service
```

**Liveness and Readiness Probes**

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 3040
  initialDelaySeconds: 15
  periodSeconds: 20
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /health
    port: 3040
  initialDelaySeconds: 5
  periodSeconds: 10
  failureThreshold: 2
```

**Network Policy**

The service operates under a strict NetworkPolicy that allows inbound traffic only from services with the label `role: internal-service` and the Istio ingress gateway. All egress is restricted to the database subnet, Redis, RabbitMQ, and the specific CIDR ranges for SendGrid, Twilio, and Google FCM APIs.

#### Database Migrations

Migrations are run as a Kubernetes Job before each deployment via the Helm pre-upgrade hook:

```bash
# Run migrations manually
kubectl run migration-job \
  --image=registry.internal.example.com/notification-service:2.4.1 \
  --restart=Never \
  --namespace=notifications \
  --env-from=secret/notification-service-secrets \
  -- npx prisma migrate deploy
```

---

## Development

### Getting Started

```bash
# Clone the repository
git clone git@github.com:example/notification-service.git
cd notification-service

# Install dependencies
npm install

# Copy environment template
cp .env.example .env.local
# Edit .env.local with your local values

# Start infrastructure
docker compose -f docker-compose.dev.yml up -d

# Run database migrations
npx prisma migrate dev

# Seed development data (templates, test users)
npx ts-node prisma/seed.ts

# Start development server with hot reload
npm run dev
```

### Running Tests

```bash
# Unit tests
npm run test:unit

# Integration tests (requires running PostgreSQL, Redis, RabbitMQ)
npm run test:integration

# End-to-end tests
npm run test:e2e

# All tests with coverage
npm run test:coverage

# Watch mode
npm run test:watch
```

### Useful Scripts

```bash
npm run build          # Compile TypeScript
npm run lint           # ESLint
npm run lint:fix       # ESLint with auto-fix
npm run format         # Prettier
npm run type-check     # TypeScript type checking only
npm run db:migrate     # Run pending migrations
npm run db:studio      # Open Prisma Studio
```

---

## Troubleshooting

### Common Issues

#### Notifications stuck in QUEUED status

1. Verify RabbitMQ connectivity: `GET /health` and inspect `rabbitmq.status`
2. Check consumer worker logs for errors: `kubectl logs -l app=notification-service -n notifications | grep ERROR`
3. Inspect the dead-letter queue in the RabbitMQ management UI at `http://rabbitmq.internal.example.com:15672`

#### SendGrid returns 403 Forbidden

1. Verify the `SENDGRID_API_KEY` environment variable is correctly set and not expired
2. Ensure the API key has `Mail Send` permission in the SendGrid dashboard
3. Check if the sending domain is verified in SendGrid

#### High memory usage

1. Check `RABBITMQ_PREFETCH` â reduce it if workers are accumulating too many messages
2. Verify `DATABASE_POOL_MAX` is not set excessively high
3. Check for template cache bloat if `CACHE_TTL_SECONDS` is very large

#### FCM token errors (`INVALID_ARGUMENT`)

This is expected when device tokens have expired. The service automatically prunes invalid tokens. If the rate is unusually high, check that the User Service is syncing updated tokens correctly.

### Support

- **Slack:** `#team-platform-notifications`
- **PagerDuty:** Notification Service runbook â see internal wiki
- **On-Call:** Escalate to Platform Team via PagerDuty for P1/P2 incidents

---

## Changelog

### [2.4.1] â 2024-11-01
- Fixed: Race condition in idempotency key validation under high concurrency
- Fixed: SendGrid EU endpoint URL was reverting to US endpoint on connection retry

### [2.4.0] â 2024-10-15
- Added: APNs native fallback for iOS push notifications
- Added: `notification.clicked` webhook event
- Changed: Upgraded to FCM HTTP v1 API (legacy API removed)
- Changed: Retention job now runs at 02:00 CET (previously 03:00 CET)

### [2.3.0] â 2024-09-01
- Added: Batch notifications endpoint (`POST /notifications/batch`)
- Added: Per-category notification preferences
- Fixed: SMS multipart encoding for UCS-2 characters

---

## License

Copyright Â© 2024 Example GmbH. All rights reserved. Internal use only. Not for distribution.

---

*This service is owned and maintained by the **Platform Team**. For architectural decisions and feature requests, open a ticket in Jira under the `NOTIF` project.*
# 🚗 Driver Service

[![Build Status](https://img.shields.io/github/actions/workflow/status/rideshare/driver-service/ci.yml?branch=main&style=flat-square)](https://github.com/rideshare/driver-service/actions)
[![Coverage](https://img.shields.io/codecov/c/github/rideshare/driver-service?style=flat-square)](https://codecov.io/gh/rideshare/driver-service)
[![Docker Image](https://img.shields.io/docker/v/rideshare/driver-service?style=flat-square&logo=docker)](https://hub.docker.com/r/rideshare/driver-service)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)
[![Node Version](https://img.shields.io/badge/node-%3E%3D18.0.0-brightgreen?style=flat-square&logo=node.js)](https://nodejs.org)
[![OpenAPI](https://img.shields.io/badge/OpenAPI-3.1-green?style=flat-square&logo=swagger)](docs/openapi.yaml)
[![DSGVO Compliant](https://img.shields.io/badge/DSGVO-compliant-blue?style=flat-square)](docs/dsgvo.md)

> **Microservice responsible for driver profile management, real-time location tracking, availability status, trip history, and earnings reporting within the RideShare platform.**

---

## 📋 Table of Contents

- [Service Overview](#-service-overview)
- [Architecture](#-architecture)
- [API Documentation](#-api-documentation)
  - [Driver Profile Endpoints](#driver-profile-endpoints)
  - [Availability Endpoints](#availability-endpoints)
  - [Location Endpoints](#location-endpoints)
  - [Earnings Endpoints](#earnings-endpoints)
  - [Trip History Endpoints](#trip-history-endpoints)
  - [Health Check](#health-check)
- [Request & Response Schemas](#-request--response-schemas)
- [Error Handling](#-error-handling)
- [Configuration](#-configuration)
- [German Compliance (DSGVO / PBefG)](#-german-compliance-dsgvo--pbefg)
- [Integration Points](#-integration-points)
- [Deployment](#-deployment)
- [Development](#-development)
- [Changelog](#-changelog)

---

## 🎯 Service Overview

The **Driver Service** is a core microservice of the RideShare platform, acting as the single source of truth for all driver-related data. It manages the complete lifecycle of a driver — from onboarding and document verification to real-time GPS tracking and earnings aggregation.

### Key Features

| Feature | Description |
|---|---|
| 👤 **Profile Management** | Full CRUD operations for driver profiles including document uploads and verification status |
| 📡 **Real-Time Location** | High-frequency GPS coordinate ingestion with sub-second write latency using Redis and PostgreSQL with PostGIS |
| 🟢 **Availability Management** | Granular status control (`ONLINE`, `OFFLINE`, `BUSY`, `ON_BREAK`) with audit trail |
| 💰 **Earnings Engine** | Aggregated and time-series earnings with VAT-compliant breakdowns |
| 🗺️ **Location History** | Configurable retention of location history with DSGVO-compliant data minimization |
| 📜 **Trip History** | Paginated trip records linked to the trip-service |
| 🔒 **DSGVO Compliance** | Right to erasure, data export, consent management, and pseudonymization support |
| 🏥 **Health Monitoring** | Structured health checks with dependency probing for Kubernetes readiness/liveness probes |

### Technology Stack

```
Runtime:        Node.js 18+ (TypeScript)
Framework:      Fastify 4.x
Database:       PostgreSQL 15 + PostGIS 3.3
Cache/PubSub:   Redis 7.x
Message Broker: Apache Kafka 3.x
ORM:            Prisma 5.x
Auth:           JWT (RS256) via Auth Service
Docs:           OpenAPI 3.1 / Swagger UI
Containerize:   Docker / Kubernetes
```

---

## 🏛️ Architecture

### Service Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                          API Gateway / Kong                          │
│               (JWT Validation, Rate Limiting, Routing)               │
└────────────────────────────┬────────────────────────────────────────┘
                             │  HTTPS / REST
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Driver Service                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │   Profile    │  │  Location    │  │   Availability &         │  │
│  │  Controller  │  │  Controller  │  │   Earnings Controller    │  │
│  └──────┬───────┘  └──────┬───────┘  └────────────┬─────────────┘  │
│         │                 │                        │                │
│  ┌──────▼─────────────────▼────────────────────────▼─────────────┐ │
│  │                     Service Layer (Business Logic)             │ │
│  └──────┬─────────────────┬────────────────────────┬─────────────┘ │
│         │                 │                        │                │
│  ┌──────▼──────┐  ┌───────▼──────┐  ┌─────────────▼─────────────┐ │
│  │   Prisma    │  │ Redis Client │  │    Kafka Producer         │ │
│  │    ORM      │  │  (Location   │  │  (Events: location.update,│ │
│  └──────┬──────┘  │   Cache)     │  │   driver.status.change,   │ │
│         │         └──────┬───────┘  │   driver.created)         │ │
│         │                │          └─────────────┬─────────────┘ │
└─────────┼────────────────┼────────────────────────┼───────────────┘
          │                │                        │
          ▼                ▼                        ▼
   ┌─────────────┐  ┌─────────────┐         ┌──────────────┐
   │ PostgreSQL  │  │   Redis 7   │         │    Kafka     │
   │  + PostGIS  │  │  Cluster    │         │   Cluster    │
   └─────────────┘  └─────────────┘         └──────┬───────┘
                                                    │
                              ┌─────────────────────┼──────────────────────┐
                              ▼                     ▼                      ▼
                     ┌──────────────┐    ┌──────────────────┐   ┌──────────────────┐
                     │ Trip Service │    │  Matching Service│   │Notification Svc  │
                     └──────────────┘    └──────────────────┘   └──────────────────┘
```

### Data Flow

#### Location Update Flow
```
Driver Mobile App
       │
       │ POST /drivers/{id}/location  (HTTPS)
       ▼
API Gateway ──► JWT Validation ──► Rate Limit (10 req/s per driver)
       │
       ▼
Driver Service
   1. Validate coordinates (WGS84)
   2. Write to Redis (TTL: 300s)  ──► Matching Service reads live location
   3. Publish Kafka event: location.updated
   4. Async: Persist to PostgreSQL (PostGIS geometry column)
       │
       ├──► Matching Service (Kafka consumer): Updates driver pool
       └──► Analytics Service (Kafka consumer): Geospatial aggregation
```

#### Availability Change Flow
```
Driver App / Admin Portal
       │
       │ PUT /drivers/{id}/availability
       ▼
Driver Service
   1. Validate status transition (state machine)
   2. Update PostgreSQL availability record
   3. Invalidate Redis cache
   4. Publish Kafka: driver.availability.changed
       │
       ├──► Matching Service: Remove/add driver from active pool
       ├──► Notification Service: Push notification to driver
       └──► Analytics Service: Availability metrics
```

---

## 📡 API Documentation

### Base URL

```
Production:  https://api.rideshare.de/v1
Staging:     https://api.staging.rideshare.de/v1
Local Dev:   http://localhost:3001/v1
```

### Authentication

All endpoints except `GET /health` require a valid JWT bearer token issued by the Auth Service.

```http
Authorization: Bearer <jwt_token>
```

JWTs are signed with RS256. Public keys are fetched from the Auth Service JWKS endpoint:
```
https://auth.rideshare.de/.well-known/jwks.json
```

**Required JWT claims:**

| Claim | Type | Description |
|---|---|---|
| `sub` | string | User ID (UUID) |
| `role` | string | `DRIVER`, `ADMIN`, or `SYSTEM` |
| `driverId` | string | Driver profile ID (present for DRIVER role) |
| `exp` | number | Expiration timestamp |

> ⚠️ **Authorization Rules:** A driver can only access their own resources unless the caller has the `ADMIN` or `SYSTEM` role.

---

### Driver Profile Endpoints

#### `POST /drivers` — Create Driver Profile

Creates a new driver profile. Typically called during onboarding after identity verification.

```http
POST /v1/drivers
Content-Type: application/json
Authorization: Bearer <jwt_token>
```

**Required Role:** `ADMIN` or `SYSTEM`

**Request Body:**

```json
{
  "userId": "550e8400-e29b-41d4-a716-446655440000",
  "personalInfo": {
    "firstName": "Markus",
    "lastName": "Müller",
    "dateOfBirth": "1985-03-15",
    "gender": "MALE",
    "nationality": "DE",
    "phoneNumber": "+49151234567890",
    "email": "markus.mueller@email.de"
  },
  "address": {
    "street": "Hauptstraße 42",
    "city": "Berlin",
    "postalCode": "10115",
    "state": "BE",
    "country": "DE"
  },
  "licenseInfo": {
    "licenseNumber": "B123456789",
    "licenseClass": "B",
    "issuedAt": "2005-06-20",
    "expiresAt": "2025-06-20",
    "issuingAuthority": "Straßenverkehrsamt Berlin"
  },
  "vehicleId": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "taxInfo": {
    "taxId": "DE123456789",
    "vatRegistered": false
  },
  "consentGiven": true,
  "consentTimestamp": "2024-01-15T10:30:00Z"
}
```

**Response `201 Created`:**

```json
{
  "data": {
    "id": "drv_01HQX8M2K3VTYM5N6P7R8S9T",
    "userId": "550e8400-e29b-41d4-a716-446655440000",
    "status": "PENDING_VERIFICATION",
    "availabilityStatus": "OFFLINE",
    "rating": null,
    "totalTrips": 0,
    "createdAt": "2024-01-15T10:30:01.123Z",
    "updatedAt": "2024-01-15T10:30:01.123Z"
  },
  "meta": {
    "requestId": "req_01HQX8M2K3VTYM5N6P7R8S9U"
  }
}
```

**Error Responses:**

| HTTP Status | Error Code | Description |
|---|---|---|
| `400 Bad Request` | `VALIDATION_ERROR` | Invalid or missing required fields |
| `409 Conflict` | `DRIVER_ALREADY_EXISTS` | A driver profile for this userId already exists |
| `422 Unprocessable Entity` | `INVALID_LICENSE` | License number format invalid for country DE |

---

#### `GET /drivers/{id}` — Get Driver Profile

Retrieves a full driver profile by driver ID.

```http
GET /v1/drivers/{id}
Authorization: Bearer <jwt_token>
```

**Path Parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `id` | string (ULID) | ✅ | Driver profile ID |

**Query Parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `include` | string | — | Comma-separated fields: `vehicle`, `documents`, `rating` |

**Example Request:**

```http
GET /v1/drivers/drv_01HQX8M2K3VTYM5N6P7R8S9T?include=vehicle,rating
Authorization: Bearer eyJhbGciOiJSUzI1NiJ9...
```

**Response `200 OK`:**

```json
{
  "data": {
    "id": "drv_01HQX8M2K3VTYM5N6P7R8S9T",
    "userId": "550e8400-e29b-41d4-a716-446655440000",
    "personalInfo": {
      "firstName": "Markus",
      "lastName": "Müller",
      "phoneNumber": "+49151234567890",
      "email": "markus.mueller@email.de"
    },
    "status": "ACTIVE",
    "availabilityStatus": "ONLINE",
    "licenseInfo": {
      "licenseClass": "B",
      "expiresAt": "2025-06-20",
      "verified": true
    },
    "rating": {
      "average": 4.87,
      "count": 312
    },
    "vehicle": {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "make": "Volkswagen",
      "model": "Passat",
      "year": 2021,
      "licensePlate": "B-MX-1234",
      "color": "Silber"
    },
    "totalTrips": 312,
    "memberSince": "2023-03-01T00:00:00.000Z",
    "createdAt": "2024-01-15T10:30:01.123Z",
    "updatedAt": "2024-01-16T08:15:22.456Z"
  },
  "meta": {
    "requestId": "req_01HQX8M2K3VTYM5N6P7R8SABC"
  }
}
```

**Error Responses:**

| HTTP Status | Error Code | Description |
|---|---|---|
| `404 Not Found` | `DRIVER_NOT_FOUND` | No driver profile found for the given ID |
| `403 Forbidden` | `ACCESS_DENIED` | Caller is not authorized to view this driver |

---

#### `PUT /drivers/{id}` — Update Driver Profile

Partially updates a driver profile. Only provided fields are updated (PATCH semantics over PUT).

```http
PUT /v1/drivers/{id}
Content-Type: application/json
Authorization: Bearer <jwt_token>
```

**Required Role:** Driver (own profile) or `ADMIN`

**Request Body (all fields optional):**

```json
{
  "personalInfo": {
    "phoneNumber": "+49159987654321",
    "email": "new.email@email.de"
  },
  "address": {
    "street": "Neue Straße 10",
    "city": "Hamburg",
    "postalCode": "20095",
    "state": "HH",
    "country": "DE"
  },
  "vehicleId": "new-vehicle-uuid-here"
}
```

**Response `200 OK`:** Returns the updated driver profile (same schema as `GET /drivers/{id}`).

**Error Responses:**

| HTTP Status | Error Code | Description |
|---|---|---|
| `400 Bad Request` | `VALIDATION_ERROR` | Invalid field value |
| `403 Forbidden` | `ACCESS_DENIED` | Cannot update another driver's profile |
| `404 Not Found` | `DRIVER_NOT_FOUND` | Driver not found |
| `409 Conflict` | `EMAIL_IN_USE` | Email address already registered |

---

#### `DELETE /drivers/{id}` — Delete Driver Profile

Initiates DSGVO-compliant deletion of a driver profile. Triggers a soft-delete with a 30-day retention window before permanent erasure. All PII is pseudonymized immediately.

```http
DELETE /v1/drivers/{id}
Authorization: Bearer <jwt_token>
```

**Required Role:** `ADMIN` only

**Query Parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `reason` | string | — | Deletion reason (required for audit log) |
| `gdprRequest` | boolean | `false` | Set to `true` for Art. 17 DSGVO right-to-erasure requests |

**Response `200 OK`:**

```json
{
  "data": {
    "id": "drv_01HQX8M2K3VTYM5N6P7R8S9T",
    "status": "DELETION_SCHEDULED",
    "piiPseudonymizedAt": "2024-01-20T14:00:00.000Z",
    "scheduledErasureAt": "2024-02-19T14:00:00.000Z",
    "gdprRequestId": "gdpr_01HQX9AABBCCDD"
  },
  "meta": {
    "requestId": "req_01HQX8M2K3VTYM5N6P7RDEF"
  }
}
```

> 📋 **DSGVO Note:** Deletion is a scheduled process. Refer to the [German Compliance](#-german-compliance-dsgvo--pbefg) section for full data erasure timelines and legal basis.

---

### Availability Endpoints

#### `GET /drivers/{id}/availability` — Get Availability Status

Returns the current availability status and associated metadata.

```http
GET /v1/drivers/{id}/availability
Authorization: Bearer <jwt_token>
```

**Response `200 OK`:**

```json
{
  "data": {
    "driverId": "drv_01HQX8M2K3VTYM5N6P7R8S9T",
    "status": "ONLINE",
    "since": "2024-01-20T07:00:00.000Z",
    "lastUpdated": "2024-01-20T07:00:00.000Z",
    "shiftDurationMinutes": 127,
    "allowedTransitions": ["OFFLINE", "BUSY", "ON_BREAK"]
  },
  "meta": {
    "requestId": "req_01HQX8AVAILGET"
  }
}
```

**Availability Status State Machine:**

```
             ┌──────────────┐
    ┌────────►│   OFFLINE    │◄────────┐
    │         └──────┬───────┘         │
    │                │ go_online       │ go_offline
    │                ▼                 │
    │         ┌──────────────┐         │
    │    ┌───►│    ONLINE    ├─────────┤
    │    │    └──────┬───────┘         │
    │    │           │ accept_trip      │
    │    │           ▼                 │
    │    │    ┌──────────────┐         │
    │    │    │     BUSY     ├─────────┤
    │    │    └──────┬───────┘         │
    │    │           │ trip_complete    │
    │    └───────────┘                 │
    │                                  │
    │    ┌──────────────┐              │
    └────┤   ON_BREAK   ├──────────────┘
         └──────────────┘
```

---

#### `PUT /drivers/{id}/availability` — Update Availability Status

```http
PUT /v1/drivers/{id}/availability
Content-Type: application/json
Authorization: Bearer <jwt_token>
```

**Required Role:** Driver (own profile) or `ADMIN` or `SYSTEM`

**Request Body:**

```json
{
  "status": "ONLINE",
  "reason": "Starting shift",
  "location": {
    "latitude": 52.5200,
    "longitude": 13.4050
  }
}
```

**Request Fields:**

| Field | Type | Required | Description |
|---|---|---|---|
| `status` | enum | ✅ | One of: `ONLINE`, `OFFLINE`, `BUSY`, `ON_BREAK` |
| `reason` | string | ❌ | Human-readable reason (max 255 chars) |
| `location` | object | ❌ | Current location when going online |

**Response `200 OK`:**

```json
{
  "data": {
    "driverId": "drv_01HQX8M2K3VTYM5N6P7R8S9T",
    "previousStatus": "OFFLINE",
    "currentStatus": "ONLINE",
    "transitionAt": "2024-01-20T07:00:00.123Z",
    "kafkaEventId": "evt_01HQX8AVAIL123"
  }
}
```

**Error Responses:**

| HTTP Status | Error Code | Description |
|---|---|---|
| `400 Bad Request` | `INVALID_TRANSITION` | Status transition not allowed by state machine |
| `400 Bad Request` | `DOCUMENTS_EXPIRED` | Cannot go ONLINE with expired documents |
| `403 Forbidden` | `ACCESS_DENIED` | Cannot update another driver's availability |

---

### Location Endpoints

#### `POST /drivers/{id}/location` — Update Location (GPS)

Ingests a GPS coordinate update from the driver's mobile device. Optimized for high-frequency writes.

```http
POST /v1/drivers/{id}/location
Content-Type: application/json
Authorization: Bearer <jwt_token>
```

**Rate Limit:** 10 requests/second per driver.

**Request Body:**

```json
{
  "latitude": 52.5200066,
  "longitude": 13.4049540,
  "accuracy": 4.2,
  "altitude": 34.1,
  "heading": 270.5,
  "speed": 13.8,
  "timestamp": "2024-01-20T09:15:33.221Z",
  "provider": "GPS"
}
```

**Request Fields:**

| Field | Type | Required | Validation | Description |
|---|---|---|---|---|
| `latitude` | float | ✅ | -90 to 90 | WGS84 latitude |
| `longitude` | float | ✅ | -180 to 180 | WGS84 longitude |
| `accuracy` | float | ❌ | >= 0 | Accuracy in meters |
| `altitude` | float | ❌ | — | Altitude in meters above sea level |
| `heading` | float | ❌ | 0 to 360 | Direction in degrees |
| `speed` | float | ❌ | >= 0 | Speed in m/s |
| `timestamp` | ISO8601 | ✅ | Max 30s drift | Device timestamp |
| `provider` | enum | ❌ | — | `GPS`, `NETWORK`, `FUSED` |

**Response `201 Created`:**

```json
{
  "data": {
    "locationId": "loc_01HQX8LOC987654321",
    "driverId": "drv_01HQX8M2K3VTYM5N6P7R8S9T",
    "latitude": 52.5200066,
    "longitude": 13.4049540,
    "recordedAt": "2024-01-20T09:15:33.221Z",
    "cached": true
  }
}
```

---

#### `GET /drivers/{id}/location` — Get Current Location

Retrieves the last known location of the driver. Served from Redis cache when available.

```http
GET /v1/drivers/{id}/location
Authorization: Bearer <jwt_token>
```

**Response `200 OK`:**

```json
{
  "data": {
    "driverId": "drv_01HQX8M2K3VTYM5N6P7R8S9T",
    "latitude": 52.5200066,
    "longitude": 13.4049540,
    "heading": 270.5,
    "speed": 13.8,
    "accuracy": 4.2,
    "recordedAt": "2024-01-20T09:15:33.221Z",
    "source": "CACHE"
  },
  "meta": {
    "cacheAge": 3,
    "requestId": "req_01HQX8LOCGET"
  }
}
```

**Response when driver is OFFLINE and no recent location:**

```json
{
  "data": null,
  "meta": {
    "reason": "DRIVER_OFFLINE_NO_LOCATION",
    "requestId": "req_01HQX8LOCGET2"
  }
}
```

---

#### `GET /drivers/{id}/location/history` — Get Location History

Retrieves paginated location history for a driver within a specified time range.

```http
GET /v1/drivers/{id}/location/history
Authorization: Bearer <jwt_token>
```

**Required Role:** `ADMIN` or `SYSTEM` (drivers cannot access their own raw history for DSGVO compliance)

**Query Parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `from` | ISO8601 | 24h ago | Start of time range |
| `to` | ISO8601 | now | End of time range |
| `limit` | integer | `100` | Records per page (max `500`) |
| `cursor` | string | — | Pagination cursor from previous response |
| `resolution` | enum | `FULL` | `FULL`, `1MIN`, `5MIN` — temporal downsampling |

**Example Request:**

```http
GET /v1/drivers/drv_01HQX8M2K3VTYM5N6P7R8S9T/location/history
    ?from=2024-01-20T06:00:00Z
    &to=2024-01-20T14:00:00Z
    &limit=200
    &resolution=1MIN
```

**Response `200 OK`:**

```json
{
  "data": [
    {
      "locationId": "loc_01HQX8LOC001",
      "latitude": 52.5200066,
      "longitude": 13.4049540,
      "heading": 270.5,
      "speed": 13.8,
      "recordedAt": "2024-01-20T09:15:33.221Z"
    },
    {
      "locationId": "loc_01HQX8LOC002",
      "latitude": 52.5215000,
      "longitude": 13.4100000,
      "heading": 265.0,
      "speed": 12.1,
      "recordedAt": "2024-01-20T09:16:33.110Z"
    }
  ],
  "meta": {
    "total": 480,
    "limit": 200,
    "nextCursor": "cur_eyJpZCI6ImxvY18wMUhRWDhMT0MwMDMifQ",
    "hasMore": true,
    "requestId": "req_01HQX8LOCHIST"
  }
}
```

> 📋 **DSGVO Note:** Location history older than the configured retention period (`LOCATION_HISTORY_RETENTION_DAYS`) is automatically purged. Default is 90 days.

---

### Earnings Endpoints

#### `GET /drivers/{id}/earnings` — Get Earnings Summary

Returns an aggregated earnings summary for a driver.

```http
GET /v1/drivers/{id}/earnings
Authorization: Bearer <jwt_token>
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `period` | enum | `CURRENT_WEEK` | `TODAY`, `CURRENT_WEEK`, `CURRENT_MONTH`, `CUSTOM` |
| `from` | ISO8601 | — | Required when `period=CUSTOM` |
| `to` | ISO8601 | — | Required when `period=CUSTOM` |
| `currency` | string | `EUR` | Currency code (ISO 4217) |

**Response `200 OK`:**

```json
{
  "data": {
    "driverId": "drv_01HQX8M2K3VTYM5N6P7R8S9T",
    "period": {
      "type": "CURRENT_WEEK",
      "from": "2024-01-15T00:00:00.000Z",
      "to": "2024-01-21T23:59:59.999Z"
    },
    "currency": "EUR",
    "summary": {
      "grossEarnings": 487.50,
      "platformFee": 97.50,
      "netEarnings": 390.00,
      "vatCollected": 74.05,
      "tips": 23.40,
      "bonuses": 15.00,
      "adjustments": -5.00,
      "totalPayout": 405.00
    },
    "tripStats": {
      "completedTrips": 32,
      "cancelledTrips": 2,
      "averageEarningsPerTrip": 15.23,
      "totalDistanceKm": 412.7,
      "totalOnlineMinutes": 1842
    }
  },
  "meta": {
    "requestId": "req_01HQX8EARN001",
    "generatedAt": "2024-01-20T15:00:00.000Z"
  }
}
```

---

#### `GET /drivers/{id}/earnings/history` — Get Earnings History

Returns a paginated time-series of individual payout records.

```http
GET /v1/drivers/{id}/earnings/history
Authorization: Bearer <jwt_token>
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `from` | ISO8601 | 30 days ago | Start of time range |
| `to` | ISO8601 | now | End of time range |
| `limit` | integer | `25` | Records per page (max `100`) |
| `page` | integer | `1` | Page number |
| `type` | enum | — | Filter: `TRIP`, `BONUS`, `ADJUSTMENT`, `TIP` |

**Response `200 OK`:**

```json
{
  "data": [
    {
      "id": "earn_01HQX8EARN1001",
      "type": "TRIP",
      "tripId": "trp_01HQXTRIP001",
      "grossAmount": 18.50,
      "platformFeeAmount": 3.70,
      "netAmount": 14.80,
      "vatAmount": 2.80,
      "currency": "EUR",
      "description": "Fahrt von Mitte nach Charlottenburg",
      "earnedAt": "2024-01-20T09:45:00.000Z",
      "payoutStatus": "SETTLED",
      "payoutDate": "2024-01-22T00:00:00.000Z"
    },
    {
      "id": "earn_01HQX8EARN1002",
      "type": "BONUS",
      "tripId": null,
      "grossAmount": 15.00,
      "platformFeeAmount": 0.00,
      "netAmount": 15.00,
      "vatAmount": 0.00,
      "currency": "EUR",
      "description": "Wochendienst-Bonus (Montag Stoßzeit)",
      "earnedAt": "2024-01-20T12:00:00.000Z",
      "payoutStatus": "PENDING",
      "payoutDate": null
    }
  ],
  "meta": {
    "total": 87,
    "page": 1,
    "limit": 25,
    "totalPages": 4,
    "requestId": "req_01HQX8EARNHIST"
  }
}
```

---

### Trip History Endpoints

#### `GET /drivers/{id}/trips` — Get Trip History

Returns a paginated list of trips associated with the driver. Trip details are fetched from the Trip Service and cached.

```http
GET /v1/drivers/{id}/trips
Authorization: Bearer <jwt_token>
```

**Query Parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `from` | ISO8601 | 30 days ago | Start of time range |
| `to` | ISO8601 | now | End of time range |
| `status` | enum | — | Filter: `COMPLETED`, `CANCELLED`, `IN_PROGRESS` |
| `limit` | integer | `25` | Records per page (max `100`) |
| `page` | integer | `1` | Page number |
| `sortBy` | enum | `createdAt` | `createdAt`, `distance`, `earnings` |
| `sortOrder` | enum | `desc` | `asc` or `desc` |

**Response `200 OK`:**

```json
{
  "data": [
    {
      "tripId": "trp_01HQXTRIP001",
      "status": "COMPLETED",
      "passenger": {
        "id": "usr_01HQXUSER001",
        "displayName": "Anna K.",
        "rating": 4.9
      },
      "pickup": {
        "address": "Alexanderplatz 1, Berlin",
        "latitude": 52.5219,
        "longitude": 13.4132
      },
      "dropoff": {
        "address": "Kurfürstendamm 22, Berlin",
        "latitude": 52.5038,
        "longitude": 13.3285
      },
      "fare": {
        "amount": 18.50,
        "currency": "EUR"
      },
      "distanceKm": 9.4,
      "durationMinutes": 22,
      "startedAt": "2024-01-20T09:23:00.000Z",
      "completedAt": "2024-01-20T09:45:00.000Z",
      "driverRatingGiven": 5,
      "passengerRatingReceived": 5
    }
  ],
  "meta": {
    "total": 312,
    "page": 1,
    "limit": 25,
    "totalPages": 13,
    "requestId": "req_01HQX8TRIPS001"
  }
}
```

---

### Health Check

#### `GET /health` — Health Check

Returns the operational health of the service and its dependencies. Used by Kubernetes liveness and readiness probes.

```http
GET /health
```

**No authentication required.**

**Response `200 OK` (healthy):**

```json
{
  "status": "healthy",
  "version": "2.4.1",
  "uptime": 86400,
  "timestamp": "2024-01-20T15:00:00.000Z",
  "dependencies": {
    "database": {
      "status": "healthy",
      "latencyMs": 2.3,
      "detail": "PostgreSQL 15.2 — primary connection OK"
    },
    "redis": {
      "status": "healthy",
      "latencyMs": 0.4,
      "detail": "Redis 7.2.1 — cluster mode"
    },
    "kafka": {
      "status": "healthy",
      "latencyMs": 5.1,
      "detail": "3/3 brokers reachable"
    }
  }
}
```

**Response `503 Service Unavailable` (degraded):**

```json
{
  "status": "degraded",
  "version": "2.4.1",
  "uptime": 86400,
  "timestamp": "2024-01-20T15:00:00.000Z",
  "dependencies": {
    "database": {
      "status": "healthy",
      "latencyMs": 2.3
    },
    "redis": {
      "status": "unhealthy",
      "latencyMs": null,
      "error": "Connection refused"
    },
    "kafka": {
      "status": "healthy",
      "latencyMs": 5.1
    }
  }
}
```

---

## 📦 Request & Response Schemas

### Common Response Wrapper

All responses follow a consistent envelope structure:

```typescript
interface ApiResponse<T> {
  data: T | null;
  meta: {
    requestId: string;       // Unique request trace ID
    generatedAt?: string;    // ISO8601 timestamp
    cacheAge?: number;       // Seconds since cache population
    [key: string]: unknown;
  };
  errors?: ApiError[];       // Present only on 4xx/5xx
}

interface ApiError {
  code: string;              // Machine-readable error code
  message: string;           // Human-readable description (German for 4xx)
  field?: string;            // Field path for validation errors
  details?: Record<string, unknown>;
}
```

### Driver Status Enum Values

| Value | Description |
|---|---|
| `PENDING_VERIFICATION` | Profile created, awaiting document verification |
| `ACTIVE` | Fully verified and able to accept trips |
| `SUSPENDED` | Temporarily suspended (investigation in progress) |
| `DEACTIVATED` | Permanently deactivated |
| `DELETION_SCHEDULED` | DSGVO erasure scheduled |

---

## ⚠️ Error Handling

### HTTP Status Code Reference

| Status Code | Usage |
|---|---|
| `200 OK` | Successful GET / PUT |
| `201 Created` | Successful POST (resource created) |
| `400 Bad Request` | Validation errors, invalid input |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Valid JWT but insufficient permissions |
| `404 Not Found` | Resource does not exist |
| `409 Conflict` | Duplicate resource or state conflict |
| `422 Unprocessable Entity` | Semantically invalid data (e.g., expired license) |
| `429 Too Many Requests` | Rate limit exceeded |
| `500 Internal Server Error` | Unhandled server-side error |
| `503 Service Unavailable` | Dependency failure (DB, Redis, Kafka) |

### Error Response Example

```json
{
  "data": null,
  "meta": {
    "requestId": "req_01HQX8ERR001"
  },
  "errors": [
    {
      "code": "VALIDATION_ERROR",
      "message": "Das Geburtsdatum ist ungültig.",
      "field": "personalInfo.dateOfBirth",
      "details": {
        "expected": "ISO8601 date string",
        "received": "15/03/1985"
      }
    }
  ]
}
```

### Retry Strategy

Clients should implement exponential backoff for `429` and `503` responses:

```
Retry-After: 2           // Header present on 429
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1705762800
```

---

## ⚙️ Configuration

All configuration is managed via environment variables. Secrets are injected via Kubernetes Secrets or HashiCorp Vault.

### Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `NODE_ENV` | ✅ | — | `development`, `staging`, `production` |
| `PORT` | ❌ | `3001` | HTTP server port |
| `LOG_LEVEL` | ❌ | `info` | `debug`, `info`, `warn`, `error` |
| **Database** | | | |
| `DATABASE_URL` | ✅ | — | PostgreSQL connection string: `postgresql://user:pass@host:5432/drivers` |
| `DATABASE_MAX_CONNECTIONS` | ❌ | `20` | Connection pool size |
| `DATABASE_SSL_ENABLED` | ❌ | `true` | Enforce SSL for DB connections |
| **Redis** | | | |
| `REDIS_URL` | ✅ | — | Redis connection string: `redis://:password@host:6379` |
| `REDIS_CLUSTER_ENABLED` | ❌ | `false` | Enable Redis cluster mode |
| `LOCATION_CACHE_TTL_SECONDS` | ❌ | `300` | TTL for cached driver location |
| **Kafka** | | | |
| `KAFKA_BROKERS` | ✅ | — | Comma-separated: `broker1:9092,broker2:9092` |
| `KAFKA_CLIENT_ID` | ❌ | `driver-service` | Kafka client identifier |
| `KAFKA_GROUP_ID` | ❌ | `driver-service-group` | Consumer group ID |
| `KAFKA_SSL_ENABLED` | ❌ | `true` | Enable Kafka TLS |
| `KAFKA_SASL_MECHANISM` | ❌ | `SCRAM-SHA-256` | SASL mechanism |
| `KAFKA_SASL_USERNAME` | ✅ | — | Kafka SASL username |
| `KAFKA_SASL_PASSWORD` | ✅ | — | Kafka SASL password |
| **Auth** | | | |
| `AUTH_JWKS_URI` | ✅ | — | `https://auth.rideshare.de/.well-known/jwks.json` |
| `AUTH_JWT_AUDIENCE` | ✅ | — | Expected JWT `aud` claim |
| `AUTH_JWT_ISSUER` | ✅ | — | Expected JWT `iss` claim |
| **Service URLs** | | | |
| `TRIP_SERVICE_URL` | ✅ | — | Trip Service base URL |
| `VEHICLE_SERVICE_URL` | ✅ | — | Vehicle Service base URL |
| `NOTIFICATION_SERVICE_URL` | ❌ | — | Notification Service URL (optional fallback) |
| **DSGVO** | | | |
| `LOCATION_HISTORY_RETENTION_DAYS` | ❌ | `90` | Days to retain raw location history |
| `DRIVER_DELETION_GRACE_PERIOD_DAYS` | ❌ | `30` | Days before permanent PII erasure |
| `DSGVO_AUDIT_LOG_ENABLED` | ❌ | `true` | Enable DSGVO audit log writes |
| **Observability** | | | |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | ❌ | — | OpenTelemetry collector endpoint |
| `SENTRY_DSN` | ❌ | — | Sentry error tracking DSN |
| `METRICS_ENABLED` | ❌ | `true` | Expose `/metrics` Prometheus endpoint |

### Example `.env` (Development)

```dotenv
NODE_ENV=development
PORT=3001
LOG_LEVEL=debug

DATABASE_URL=postgresql://postgres:postgres@localhost:5432/driver_service_dev
DATABASE_SSL_ENABLED=false

REDIS_URL=redis://localhost:6379
LOCATION_CACHE_TTL_SECONDS=60

KAFKA_BROKERS=localhost:9092
KAFKA_SSL_ENABLED=false
KAFKA_SASL_USERNAME=dev_user
KAFKA_SASL_PASSWORD=dev_password

AUTH_JWKS_URI=http://localhost:3000/.well-known/jwks.json
AUTH_JWT_AUDIENCE=driver-service
AUTH_JWT_ISSUER=rideshare-auth

TRIP_SERVICE_URL=http://localhost:3002
VEHICLE_SERVICE_URL=http://localhost:3003

LOCATION_HISTORY_RETENTION_DAYS=7
DRIVER_DELETION_GRACE_PERIOD_DAYS=30
```

---

## 🇩🇪 German Compliance (DSGVO / PBefG)

This service is designed and operated in full compliance with applicable German and EU law.

### DSGVO (Datenschutz-Grundverordnung — EU 2016/679)

#### Personal Data Inventory

| Data Category | Fields | Legal Basis (Art. 6 DSGVO) | Retention Period |
|---|---|---|---|
| Identity Data | Name, DoB, Gender, Nationality | Art. 6(1)(b) — Contract | Duration of driver contract + 3 years |
| Contact Data | Email, Phone, Address | Art. 6(1)(b) — Contract | Duration of driver contract + 3 years |
| License Data | License number, class, expiry | Art. 6(1)(c) — Legal obligation | Duration of driver contract + 10 years |
| Location Data | GPS coordinates, timestamps | Art. 6(1)(f) — Legitimate interest | 90 days (raw), 2 years (aggregated) |
| Financial Data | Earnings, payout history, tax ID | Art. 6(1)(c) — Legal obligation (§ 147 AO) | 10 years (§ 257 HGB) |
| Trip History | Trip IDs, ratings, distances | Art. 6(1)(b) — Contract | 3 years |
| Behavioral Data | Availability history, shift patterns | Art. 6(1)(f) — Legitimate interest | 12 months |

#### Betroffenenrechte (Data Subject Rights)

**Art. 15 — Auskunftsrecht (Right of Access)**

Drivers can request a full export of their personal data via:
```http
GET /v1/drivers/{id}/data-export
Authorization: Bearer <jwt_token>
```
Data is compiled and delivered as an encrypted ZIP archive to the registered email address within 30 days.

**Art. 16 — Recht auf Berichtigung (Right to Rectification)**

Drivers can correct inaccurate personal data via `PUT /drivers/{id}`. Changes are logged in the audit trail.

**Art. 17 — Recht auf Löschung (Right to Erasure — "Recht auf Vergessenwerden")**

Löschanfragen are processed as follows:
1. Request received → driver status set to `DELETION_SCHEDULED`
2. **Immediate (T+0):** PII fields pseudonymized (name, email, phone replaced with `[GELÖSCHT]`)
3. **T+30 days:** Grace period for any legal hold resolution
4. **T+30 days:** Permanent deletion of all remaining personal data
5. **Retained indefinitely:** Anonymized aggregate statistics (no personal reference)
6. **Retained 10 years:** Financial records (§ 147 AO, § 257 HGB legal obligation)

> ⚠️ **Exception:** Data subject to German commercial law retention obligations (§ 257 HGB, § 147 AO) cannot be erased until the statutory retention period expires. The driver is notified of this retention via registered email.

**Art. 20 — Recht auf Datenübertragbarkeit (Data Portability)**

All personal data is exportable in machine-readable JSON format.

**Art. 21 — Widerspruchsrecht (Right to Object)**

Drivers may object to processing based on Art. 6(1)(f) (legitimate interest). Objections are handled within 1 month.

#### Datenschutz durch Technik (Privacy by Design — Art. 25)

- **Datenminimierung:** Location coordinates are rounded to 4 decimal places (~11m precision) in non-operational contexts
- **Pseudonymisierung:** Internal system identifiers (ULIDs) are used instead of natural keys
- **Verschlüsselung:** All PII fields encrypted at rest using AES-256-GCM; TLS 1.3 in transit
- **Zugriffskontrolle:** Attribute-based access control; driver data scoped to authenticated user
- **Auditierung:** All reads and writes to PII fields logged to immutable audit log

#### Datenschutz-Folgenabschätzung (DSFA / DPIA)

A Data Protection Impact Assessment (DSFA) was conducted per Art. 35 DSGVO covering:
- Real-time location tracking (high-risk processing category)
- Behavioral profiling for availability patterns
- Cross-border data transfer assessment

DSFA documentation: `docs/dsfa-driver-service-v2.pdf`

#### Auftragsverarbeitungsverträge (AVV — Art. 28)

All third-party sub-processors (AWS Frankfurt, PagerDuty, Sentry) are contracted via AVV agreements held on file with the Datenschutzbeauftragter.

---

### PBefG (Personenbeförderungsgesetz)

The driver service supports operational compliance with the Personenbeförderungsgesetz (PBefG) as amended by the **Mobilitätsdatengesetz 2021**:

| Requirement | Implementation |
|---|---|
| **§ 47 PBefG — Taxen / Mietwagen** | Driver license class validation (`P` Personenbeförderungsschein required for commercial rides) |
| **§ 49 PBefG — Gelegenheitsverkehr** | Vehicle capacity and category validation enforced during driver-vehicle association |
| **Fahrpersonalverordnung (FPersV)** | Mandatory rest period tracking via availability state machine (ON_BREAK enforcement) |
| **Gewerbeanmeldung** | Tax ID (`Steuernummer` / `USt-IdNr.`) required for driver activation |
| **Haftpflichtversicherung** | Insurance document upload required; expiry monitored with automated OFFLINE trigger |

#### Pflichtdokumente (Required Documents for Activation)

```
✅ Personalausweis / Reisepass (gültig)
✅ Führerschein Klasse B (+ Personenbeförderungsschein)
✅ Polizeiliches Führungszeugnis (< 3 Monate alt)
✅ Gewerbeanmeldung oder Freistellungsbescheinigung
✅ Kraftfahrzeughaftpflichtversicherung (laufend gültig)
✅ TÜV-Hauptuntersuchung des Fahrzeugs (< 12 Monate)
```

Document expiry monitoring runs as a nightly cron job (`0 2 * * *`). Drivers with expired mandatory documents are automatically set to `OFFLINE` and notified via push and email.

#### Datenmeldepflichten (Reporting Obligations)

Aggregated, anonymized trip and availability data is compiled and submitted to the responsible Genehmigungsbehörde quarterly via the `reporting-service`.

---

## 🔌 Integration Points

### Consumed Services

| Service | Protocol | Purpose |
|---|---|---|
| **Auth Service** | JWKS / JWT | Token validation and user identity |
| **Trip Service** | REST + Kafka | Trip assignment, history enrichment |
| **Vehicle Service** | REST | Vehicle profile association and validation |
| **Document Service** | REST | Document upload and verification status |

### Published Kafka Events

All events are serialized as **Avro** with schema registry at `https://schema-registry.internal:8081`.

| Topic | Event Type | Trigger | Key Consumers |
|---|---|---|---|
| `driver.created` | `DriverCreatedEvent` | New driver profile created | Analytics, Notification |
| `driver.updated` | `DriverUpdatedEvent` | Profile fields changed | Analytics |
| `driver.deleted` | `DriverDeletedEvent` | DSGVO erasure completed | Analytics, Audit |
| `driver.availability.changed` | `AvailabilityChangedEvent` | Status transitions | Matching, Notification, Analytics |
| `driver.location.updated` | `LocationUpdatedEvent` | GPS coordinate received | Matching, Analytics, ETA |
| `driver.documents.expired` | `DocumentExpiredEvent` | Document expiry detected | Notification, Compliance |

#### Example Kafka Event Payload (`driver.availability.changed`)

```json
{
  "eventId": "evt_01HQX8AVAIL456",
  "eventType": "driver.availability.changed",
  "schemaVersion": "1.2.0",
  "producedAt": "2024-01-20T07:00:00.123Z",
  "payload": {
    "driverId": "drv_01HQX8M2K3VTYM5N6P7R8S9T",
    "previousStatus": "OFFLINE",
    "currentStatus": "ONLINE",
    "location": {
      "latitude": 52.5200,
      "longitude": 13.4050
    },
    "transitionAt": "2024-01-20T07:00:00.000Z"
  }
}
```

### Consumed Kafka Events

| Topic | Event Type | Action |
|---|---|---|
| `trip.completed` | `TripCompletedEvent` | Update driver status to ONLINE, increment trip count |
| `trip.started` | `TripStartedEvent` | Set driver status to BUSY |
| `payment.settled` | `PaymentSettledEvent` | Mark earnings record as SETTLED |
| `document.verified` | `DocumentVerifiedEvent` | Update driver document status |

### Service Dependencies Diagram

```
┌─────────────────────┐
│    Driver Service   │
│                     │
│  Produces:          │    Consumes:
│  ┌───────────────┐  │    ┌──────────────────────┐
│  │ driver.*      │  │◄───│ trip.completed        │
│  │ events        │  │◄───│ trip.started          │
│  └───────────────┘  │◄───│ payment.settled       │
│                     │◄───│ document.verified     │
└──────────┬──────────┘    └──────────────────────┘
           │
    REST calls to:
    ├── Auth Service     (JWKS endpoint)
    ├── Trip Service     (GET /trips/{id})
    └── Vehicle Service  (GET /vehicles/{id})
```

---

## 🚀 Deployment

### Prerequisites

- Docker 24+
- Kubernetes 1.28+ (production)
- Helm 3.12+
- PostgreSQL 15+ with PostGIS 3.3 extension
- Redis 7.x
- Apache Kafka 3.5+

### Docker

#### Build

```bash
# Build production image
docker build \
  --target production \
  --build-arg APP_VERSION=$(git describe --tags) \
  -t rideshare/driver-service:$(git describe --tags) \
  -t rideshare/driver-service:latest \
  .

# Run with environment file
docker run \
  --env-file .env.production \
  --name driver-service \
  -p 3001:3001 \
  rideshare/driver-service:latest
```

#### Docker Compose (Local Development)

```yaml
# docker-compose.yml
version: '3.9'
services:
  driver-service:
    build:
      context: .
      target: development
    ports:
      - '3001:3001'
    env_file: .env.local
    volumes:
      - ./src:/app/src:delegated
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started
      kafka:
        condition: service_started

  postgres:
    image: postgis/postgis:15-3.3
    environment:
      POSTGRES_DB: driver_service_dev
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - '5432:5432'
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U postgres']
      interval: 5s
      timeout: 5s
      retries: 5
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - '6379:6379'

  kafka:
    image: confluentinc/cp-kafka:7.5.0
    ports:
      - '9092:9092'
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092

  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181

volumes:
  pgdata:
```

```bash
docker compose up -d
```

### Kubernetes / Helm

#### Helm Chart Values

```yaml
# values.yaml (excerpt)
replicaCount: 3

image:
  repository: rideshare/driver-service
  tag: "2.4.1"
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 3001

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "100"
  hosts:
    - host: api.rideshare.de
      paths:
        - path: /v1/drivers
          pathType: Prefix

resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 512Mi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 15
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 75

livenessProbe:
  httpGet:
    path: /health
    port: 3001
  initialDelaySeconds: 15
  periodSeconds: 10
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /health
    port: 3001
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 3

podDisruptionBudget:
  enabled: true
  minAvailable: 2

secrets:
  databaseUrl:
    secretName: driver-service-db-secret
    key: url
  kafkaSaslPassword:
    secretName: kafka-credentials
    key: password
```

#### Deploy with Helm

```bash
# Add chart repository
helm repo add rideshare https://charts.rideshare.de
helm repo update

# Create namespace
kubectl create namespace driver-service

# Deploy secrets (via Vault or manually)
kubectl create secret generic driver-service-db-secret \
  --from-literal=url="postgresql://..." \
  -n driver-service

# Install
helm upgrade --install driver-service rideshare/driver-service \
  --namespace driver-service \
  --values values.production.yaml \
  --version 2.4.1 \
  --wait --timeout 5m

# Verify
kubectl get pods -n driver-service
kubectl rollout status deployment/driver-service -n driver-service
```

### Database Migrations

```bash
# Run pending migrations
npx prisma migrate deploy

# In Kubernetes (init container in Helm chart)
# Migrations run automatically as an init container before pods start.
# See: kubernetes/templates/job-migrate.yaml
```

### CI/CD Pipeline

The service uses GitHub Actions for CI/CD. On merge to `main`:

```
1. Lint + Type Check (tsc --noEmit)
2. Unit Tests (Jest)
3. Integration Tests (Testcontainers)
4. Security Scan (Trivy + Snyk)
5. Docker Build & Push to registry
6. Helm Upgrade to staging
7. Smoke Tests (k6 scripts)
8. Manual approval gate
9. Helm Upgrade to production
10. Post-deploy monitoring alert
```

---

## 🛠️ Development

### Setup

```bash
# Clone repository
git clone git@github.com:rideshare/driver-service.git
cd driver-service

# Install dependencies
npm ci

# Set up local environment
cp .env.example .env.local

# Start infrastructure
docker compose up -d postgres redis kafka

# Run database migrations and seed
npx prisma migrate dev
npx prisma db seed

# Start development server (with hot reload)
npm run dev
```

### Running Tests

```bash
# Unit tests
npm run test

# Unit tests with coverage
npm run test:coverage

# Integration tests (requires Docker)
npm run test:integration

# End-to-end tests
npm run test:e2e

# All tests
npm run test:all
```

### Code Style

```bash
# Lint
npm run lint

# Format
npm run format

# Type check
npm run typecheck
```

### API Docs (Swagger UI)

When running locally, Swagger UI is available at:
```
http://localhost:3001/docs
```

OpenAPI spec available at:
```
http://localhost:3001/docs/openapi.yaml
```

---

## 📝 Changelog

### [2.4.1] — 2024-01-15
- 🐛 Fixed race condition in availability state machine during concurrent updates
- 🔒 Patched JWT audience validation bypass

### [2.4.0] — 2024-01-08
- ✨ Added `ON_BREAK` availability status and PBefG rest period enforcement
- ✨ Added `resolution` parameter to location history endpoint
- 🗑️ Deprecated `GET /drivers/{id}/status` (use `/availability` instead, removed in v3.0)

### [2.3.0] — 2023-12-01
- ✨ DSGVO data export endpoint (`GET /v1/drivers/{id}/data-export`)
- ✨ Automatic PII pseudonymization on deletion
- 📈 Added OpenTelemetry distributed tracing

### [2.2.0] — 2023-10-15
- ✨ PostGIS integration for geospatial location history queries
- ⚡ Location updates now served from Redis with PostgreSQL async write
- 🔧 Kafka consumer group rebalancing improvements

---

## 📞 Support & Contact

| Channel | Contact |
|---|---|
| Engineering Team | `#team-driver-eng` (Slack) |
| On-Call | PagerDuty: `rideshare-driver-service` |
| Bug Reports | [GitHub Issues](https://github.com/rideshare/driver-service/issues) |
| Security Vulnerabilities | `security@rideshare.de` (GPG key in repo) |
| Datenschutzbeauftragter | `dsb@rideshare.de` |

---

<div align="center">

**RideShare Platform** · Driver Service · v2.4.1

*Built with ❤️ in Berlin · Compliant with 🇩🇪 DSGVO & PBefG*

</div>
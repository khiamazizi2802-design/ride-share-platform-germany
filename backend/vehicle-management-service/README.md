# Vehicle Management Service

A production-grade microservice for managing the full lifecycle of vehicles in a rideshare platform. Handles vehicle registration, document verification, real-time telemetry, maintenance scheduling, inspection workflows, and fleet analytics.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [API Endpoints](#api-endpoints)
  - [Health](#health)
  - [Vehicle Registration](#vehicle-registration)
  - [Vehicle Documents](#vehicle-documents)
  - [Vehicle Telemetry](#vehicle-telemetry)
  - [Maintenance](#maintenance)
  - [Inspections](#inspections)
  - [Fleet Analytics](#fleet-analytics)
- [gRPC API](#grpc-api)
- [Authentication](#authentication)
- [Error Handling](#error-handling)
- [Deployment](#deployment)
- [Monitoring](#monitoring)

---

## Architecture Overview

```
ââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ
â               Vehicle Management Service                    â
â                                                            â
â  âââââââââââââ  ââââââââââââââââ  âââââââââââââââââââââ  â
â  â REST API  â  â  gRPC Server â  â  Telemetry Worker â  â
â  â (Gin)     â  â  (port 9093) â  â  (background)     â  â
â  âââââââ¬ââââââ  ââââââââ¬ââââââââ  ââââââââââ¬âââââââââââ  â
â        â               â                    â              â
â  âââââââ¼ââââââââââââââââ¼âââââââââââââââââââââ¼âââââââââââ  â
â  â              Service Layer (Business Logic)          â  â
â  âââââââ¬ââââââââââââââââ¬âââââââââââââââââââââ¬âââââââââââ  â
â        â               â                    â              â
â  âââââââ¼ââââââ  ââââââââ¼ââââââââ  ââââââââââ¼âââââââââââ  â
â  â PostgreSQLâ  â    Redis     â  â     MongoDB       â  â
â  â (primary) â  â   (cache)    â  â   (telemetry)     â  â
â  âââââââââââââ  ââââââââââââââââ  âââââââââââââââââââââ  â
ââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ
```

---

## Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- MongoDB 7+
- Docker 24+
- Kubernetes 1.28+ (for production)

---

## Getting Started

### Local Development

```bash
# Clone the repository
git clone https://github.com/rideshare/vehicle-management-service.git
cd vehicle-management-service

# Install dependencies
go mod download

# Copy environment variables
cp .env.example .env

# Start infrastructure dependencies
docker-compose up -d postgres redis mongodb

# Run database migrations
go run cmd/migrate/main.go up

# Start the service
go run cmd/server/main.go

# Or with hot reload
air -c .air.toml
```

### Docker

```bash
# Build image
docker build -t rideshare/vehicle-management-service:latest .

# Run container
docker run -d \
  --name vehicle-management-service \
  -p 8083:8083 \
  -p 9093:9093 \
  --env-file .env \
  rideshare/vehicle-management-service:latest

# With Docker Compose
docker-compose up -d
```

---

## Configuration

| Environment Variable       | Default       | Description                                |
|----------------------------|---------------|--------------------------------------------|
| `APP_ENV`                  | `development` | Application environment                    |
| `APP_PORT`                 | `8083`        | HTTP server port                           |
| `GRPC_PORT`                | `9093`        | gRPC server port                           |
| `LOG_LEVEL`                | `info`        | Logging level (debug/info/warn/error)      |
| `DB_HOST`                  | `localhost`   | PostgreSQL host                            |
| `DB_PORT`                  | `5432`        | PostgreSQL port                            |
| `DB_NAME`                  | `vehicle_management` | PostgreSQL database name            |
| `DB_USER`                  | â             | PostgreSQL username (secret)               |
| `DB_PASSWORD`              | â             | PostgreSQL password (secret)               |
| `DB_MAX_OPEN_CONNS`        | `25`          | Max open DB connections                    |
| `DB_MAX_IDLE_CONNS`        | `10`          | Max idle DB connections                    |
| `REDIS_HOST`               | `localhost`   | Redis host                                 |
| `REDIS_PORT`               | `6379`        | Redis port                                 |
| `REDIS_PASSWORD`           | â             | Redis password (secret)                    |
| `REDIS_DB`                 | `2`           | Redis database index                       |
| `MONGO_HOST`               | `localhost`   | MongoDB host                               |
| `MONGO_PORT`               | `27017`       | MongoDB port                               |
| `MONGO_DB`                 | `vehicle_telemetry` | MongoDB database name               |
| `JWT_SECRET`               | â             | JWT signing secret (secret)                |
| `VEHICLE_CACHE_TTL`        | `300`         | Vehicle cache TTL in seconds               |
| `TELEMETRY_BATCH_SIZE`     | `100`         | Telemetry flush batch size                 |
| `INSPECTION_REMINDER_DAYS` | `30`          | Days before inspection reminder is sent   |
| `JAEGER_ENDPOINT`          | â             | Jaeger tracing endpoint                    |

---

## API Endpoints

**Base URL:** `http://vehicle-management-service/api/v1`

### Authentication

All endpoints (except health checks) require a Bearer token:

```
Authorization: Bearer <jwt_token>
```

---

### Health

#### `GET /health/live`

Liveness probe â confirms the service process is alive.

**Response `200 OK`:**
```json
{
  "status": "alive",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

#### `GET /health/ready`

Readiness probe â confirms the service can accept traffic.

**Response `200 OK`:**
```json
{
  "status": "ready",
  "timestamp": "2024-01-15T10:30:00Z",
  "dependencies": {
    "postgres": "up",
    "redis": "up",
    "mongodb": "up"
  }
}
```

---

#### `GET /health/startup`

Startup probe â confirms service initialization is complete.

**Response `200 OK`:**
```json
{
  "status": "started",
  "version": "1.0.0",
  "build": "a3f9d12"
}
```

---

### Vehicle Registration

#### `POST /api/v1/vehicles`

Register a new vehicle in the platform.

**Request Body:**
```json
{
  "driver_id": "d7a3f291-9c4e-4b8f-a1d2-e5f6g7h8i9j0",
  "make": "Toyota",
  "model": "Camry",
  "year": 2021,
  "color": "Silver",
  "license_plate": "ABC-1234",
  "vin": "1HGBH41JXMN109186",
  "vehicle_type": "sedan",
  "capacity": 4,
  "fuel_type": "gasoline",
  "transmission": "automatic",
  "features": ["ac", "usb_charger", "bluetooth"]
}
```

**Response `201 Created`:**
```json
{
  "id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "driver_id": "d7a3f291-9c4e-4b8f-a1d2-e5f6g7h8i9j0",
  "make": "Toyota",
  "model": "Camry",
  "year": 2021,
  "color": "Silver",
  "license_plate": "ABC-1234",
  "vin": "1HGBH41JXMN109186",
  "vehicle_type": "sedan",
  "capacity": 4,
  "fuel_type": "gasoline",
  "transmission": "automatic",
  "features": ["ac", "usb_charger", "bluetooth"],
  "status": "pending_verification",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Validation Rules:**
- `vin` must be a valid 17-character VIN
- `year` must be between 2000 and current year + 1
- `license_plate` must be unique
- `vehicle_type` must be one of: `sedan`, `suv`, `van`, `truck`, `motorcycle`
- `fuel_type` must be one of: `gasoline`, `diesel`, `electric`, `hybrid`

**Error `409 Conflict`:**
```json
{
  "error": "VEHICLE_ALREADY_EXISTS",
  "message": "A vehicle with this VIN or license plate already exists",
  "field": "vin"
}
```

---

#### `GET /api/v1/vehicles/:id`

Retrieve detailed information about a specific vehicle.

**Path Parameters:**
- `id` â Vehicle UUID

**Response `200 OK`:**
```json
{
  "id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "driver_id": "d7a3f291-9c4e-4b8f-a1d2-e5f6g7h8i9j0",
  "make": "Toyota",
  "model": "Camry",
  "year": 2021,
  "color": "Silver",
  "license_plate": "ABC-1234",
  "vin": "1HGBH41JXMN109186",
  "vehicle_type": "sedan",
  "capacity": 4,
  "fuel_type": "gasoline",
  "features": ["ac", "usb_charger", "bluetooth"],
  "status": "active",
  "verification_status": "verified",
  "current_mileage": 24500,
  "last_inspection_date": "2023-11-01T09:00:00Z",
  "next_inspection_date": "2024-11-01T09:00:00Z",
  "insurance_expiry": "2024-06-30T23:59:59Z",
  "documents": [
    {
      "id": "doc-uuid",
      "type": "registration",
      "status": "verified",
      "expiry_date": "2025-01-15T00:00:00Z"
    }
  ],
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-20T08:15:00Z"
}
```

---

#### `GET /api/v1/vehicles`

List vehicles with filtering and pagination.

**Query Parameters:**

| Parameter       | Type    | Default | Description                                     |
|-----------------|---------|---------|-------------------------------------------------|
| `page`          | integer | `1`     | Page number                                     |
| `page_size`     | integer | `20`    | Items per page (max 100)                        |
| `driver_id`     | string  | â       | Filter by driver UUID                           |
| `status`        | string  | â       | Filter by status (active/inactive/suspended)    |
| `vehicle_type`  | string  | â       | Filter by vehicle type                          |
| `make`          | string  | â       | Filter by manufacturer                          |
| `year_min`      | integer | â       | Minimum vehicle year                            |
| `year_max`      | integer | â       | Maximum vehicle year                            |
| `sort_by`       | string  | `created_at` | Sort field                               |
| `sort_order`    | string  | `desc`  | Sort direction (asc/desc)                       |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "make": "Toyota",
      "model": "Camry",
      "year": 2021,
      "license_plate": "ABC-1234",
      "vehicle_type": "sedan",
      "status": "active",
      "current_mileage": 24500
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_count": 150,
    "total_pages": 8,
    "has_next": true,
    "has_previous": false
  }
}
```

---

#### `PUT /api/v1/vehicles/:id`

Update vehicle information.

**Request Body (partial updates supported):**
```json
{
  "color": "Midnight Blue",
  "capacity": 5,
  "features": ["ac", "usb_charger", "bluetooth", "child_seat"],
  "current_mileage": 25000
}
```

**Response `200 OK`:** Updated vehicle object.

---

#### `PATCH /api/v1/vehicles/:id/status`

Update vehicle operational status.

**Request Body:**
```json
{
  "status": "inactive",
  "reason": "Scheduled maintenance"
}
```

**Valid Statuses:** `active`, `inactive`, `suspended`, `decommissioned`

**Response `200 OK`:**
```json
{
  "id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "inactive",
  "previous_status": "active",
  "updated_at": "2024-01-20T08:15:00Z"
}
```

---

#### `DELETE /api/v1/vehicles/:id`

Decommission a vehicle (soft delete).

**Response `204 No Content`**

---

### Vehicle Documents

#### `POST /api/v1/vehicles/:id/documents`

Upload a vehicle document.

**Content-Type:** `multipart/form-data`

**Form Fields:**

| Field           | Type   | Required | Description                                         |
|-----------------|--------|----------|-----------------------------------------------------|
| `document_type` | string | Yes      | `registration`, `insurance`, `inspection`, `permit` |
| `expiry_date`   | string | Yes      | Document expiry date (ISO 8601)                     |
| `file`          | file   | Yes      | Document file (PDF/JPEG/PNG, max 10MB)              |

**Response `201 Created`:**
```json
{
  "id": "doc-uuid-1234",
  "vehicle_id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "document_type": "insurance",
  "status": "pending_review",
  "expiry_date": "2024-12-31T23:59:59Z",
  "file_url": "https://storage.rideshare.com/vehicles/docs/doc-uuid-1234.pdf",
  "uploaded_at": "2024-01-20T08:15:00Z"
}
```

---

#### `GET /api/v1/vehicles/:id/documents`

List all documents for a vehicle.

**Response `200 OK`:**
```json
{
  "vehicle_id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "documents": [
    {
      "id": "doc-uuid-1234",
      "document_type": "insurance",
      "status": "verified",
      "expiry_date": "2024-12-31T23:59:59Z",
      "verified_at": "2024-01-21T10:00:00Z",
      "verified_by": "admin-uuid"
    },
    {
      "id": "doc-uuid-5678",
      "document_type": "registration",
      "status": "verified",
      "expiry_date": "2025-01-15T00:00:00Z"
    }
  ]
}
```

---

#### `PATCH /api/v1/vehicles/:id/documents/:doc_id/verify`

Verify or reject a vehicle document (admin only).

**Request Body:**
```json
{
  "action": "approve",
  "notes": "Document verified successfully"
}
```

**Valid Actions:** `approve`, `reject`

**Response `200 OK`:**
```json
{
  "id": "doc-uuid-1234",
  "status": "verified",
  "verified_at": "2024-01-21T10:00:00Z",
  "verified_by": "admin-uuid"
}
```

---

### Vehicle Telemetry

#### `POST /api/v1/vehicles/:id/telemetry`

Ingest real-time vehicle telemetry data.

**Request Body:**
```json
{
  "timestamp": "2024-01-20T08:15:00Z",
  "location": {
    "latitude": 40.7128,
    "longitude": -74.0060,
    "altitude": 10.5,
    "accuracy": 3.2
  },
  "speed": 45.5,
  "heading": 270.0,
  "engine_status": "on",
  "fuel_level": 75.5,
  "battery_level": null,
  "odometer": 24520,
  "diagnostics": {
    "engine_temp": 90.0,
    "tire_pressure_fl": 32.0,
    "tire_pressure_fr": 31.5,
    "tire_pressure_rl": 32.5,
    "tire_pressure_rr": 32.0,
    "oil_level": "normal",
    "check_engine": false
  }
}
```

**Response `202 Accepted`:**
```json
{
  "received_at": "2024-01-20T08:15:00.123Z",
  "vehicle_id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "batch_id": "batch-uuid-9876"
}
```

---

#### `POST /api/v1/vehicles/:id/telemetry/batch`

Ingest a batch of telemetry data points.

**Request Body:**
```json
{
  "entries": [
    {
      "timestamp": "2024-01-20T08:15:00Z",
      "location": { "latitude": 40.7128, "longitude": -74.0060 },
      "speed": 45.5,
      "fuel_level": 75.5,
      "odometer": 24520
    },
    {
      "timestamp": "2024-01-20T08:16:00Z",
      "location": { "latitude": 40.7135, "longitude": -74.0058 },
      "speed": 42.0,
      "fuel_level": 75.4,
      "odometer": 24520.7
    }
  ]
}
```

**Max batch size:** 500 entries

**Response `202 Accepted`:**
```json
{
  "accepted": 2,
  "rejected": 0,
  "batch_id": "batch-uuid-9876"
}
```

---

#### `GET /api/v1/vehicles/:id/telemetry`

Query historical telemetry data.

**Query Parameters:**

| Parameter    | Type     | Description                            |
|--------------|----------|----------------------------------------|
| `start_time` | datetime | Start of range (ISO 8601, required)    |
| `end_time`   | datetime | End of range (ISO 8601, required)      |
| `interval`   | string   | Aggregation interval (1m/5m/15m/1h)   |
| `fields`     | string   | Comma-separated fields to return       |
| `limit`      | integer  | Max records (default 1000, max 10000) |

**Response `200 OK`:**
```json
{
  "vehicle_id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "start_time": "2024-01-20T08:00:00Z",
  "end_time": "2024-01-20T09:00:00Z",
  "interval": "5m",
  "count": 12,
  "data": [
    {
      "timestamp": "2024-01-20T08:00:00Z",
      "avg_speed": 38.2,
      "max_speed": 55.0,
      "fuel_level": 76.0,
      "location": { "latitude": 40.7128, "longitude": -74.0060 }
    }
  ]
}
```

---

#### `GET /api/v1/vehicles/:id/location`

Get the current real-time location of a vehicle.

**Response `200 OK`:**
```json
{
  "vehicle_id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "location": {
    "latitude": 40.7128,
    "longitude": -74.0060
  },
  "speed": 0.0,
  "heading": 90.0,
  "engine_status": "off",
  "recorded_at": "2024-01-20T08:15:00Z",
  "source": "cache"
}
```

---

### Maintenance

#### `POST /api/v1/vehicles/:id/maintenance`

Create a maintenance record.

**Request Body:**
```json
{
  "maintenance_type": "oil_change",
  "description": "Full synthetic oil change and filter replacement",
  "performed_at": "2024-01-20T09:00:00Z",
  "mileage_at_service": 24500,
  "performed_by": "AutoCare Center",
  "cost": 89.99,
  "currency": "USD",
  "next_service_mileage": 30000,
  "next_service_date": "2024-07-20T00:00:00Z",
  "notes": "Air filter also replaced",
  "parts_replaced": [
    { "name": "Oil Filter", "part_number": "OF-12345", "quantity": 1 },
    { "name": "Synthetic Oil 5W-30", "quantity": 5, "unit": "quarts" }
  ]
}
```

**Valid Maintenance Types:** `oil_change`, `tire_rotation`, `brake_service`, `battery_replacement`, `transmission_service`, `major_service`, `recall_repair`, `other`

**Response `201 Created`:**
```json
{
  "id": "maint-uuid-1234",
  "vehicle_id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "maintenance_type": "oil_change",
  "status": "completed",
  "performed_at": "2024-01-20T09:00:00Z",
  "mileage_at_service": 24500,
  "cost": 89.99,
  "next_service_mileage": 30000,
  "next_service_date": "2024-07-20T00:00:00Z",
  "created_at": "2024-01-20T09:30:00Z"
}
```

---

#### `GET /api/v1/vehicles/:id/maintenance`

Get maintenance history for a vehicle.

**Query Parameters:**

| Parameter          | Type    | Description                              |
|--------------------|---------|------------------------------------------|
| `maintenance_type` | string  | Filter by type                           |
| `start_date`       | date    | Filter from date                         |
| `end_date`         | date    | Filter to date                           |
| `page`             | integer | Page number                              |
| `page_size`        | integer | Items per page                           |

**Response `200 OK`:** Paginated list of maintenance records.

---

#### `GET /api/v1/vehicles/:id/maintenance/upcoming`

Get upcoming scheduled maintenance for a vehicle.

**Response `200 OK`:**
```json
{
  "vehicle_id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "upcoming_maintenance": [
    {
      "maintenance_type": "oil_change",
      "due_date": "2024-07-20T00:00:00Z",
      "due_mileage": 30000,
      "current_mileage": 24500,
      "miles_remaining": 5500,
      "days_remaining": 182,
      "priority": "low"
    },
    {
      "maintenance_type": "brake_service",
      "due_date": "2024-03-01T00:00:00Z",
      "priority": "high",
      "days_remaining": 41
    }
  ]
}
```

---

### Inspections

#### `POST /api/v1/vehicles/:id/inspections`

Create a vehicle inspection record.

**Request Body:**
```json
{
  "inspection_type": "annual",
  "inspected_at": "2024-01-15T10:00:00Z",
  "inspector_name": "John Smith",
  "inspection_center": "State DMV",
  "mileage_at_inspection": 24000,
  "passed": true,
  "expiry_date": "2025-01-15T00:00:00Z",
  "checklist": {
    "brakes": "pass",
    "lights": "pass",
    "tires": "pass",
    "steering": "pass",
    "emissions": "pass",
    "safety_equipment": "pass"
  },
  "notes": "All systems in good working order",
  "certificate_number": "INSP-2024-123456"
}
```

**Response `201 Created`:** Inspection record object.

---

#### `GET /api/v1/vehicles/:id/inspections`

Get inspection history for a vehicle.

**Response `200 OK`:** Paginated list of inspection records.

---

#### `GET /api/v1/vehicles/inspections/expiring`

Get all vehicles with inspections expiring soon (admin only).

**Query Parameters:**

| Parameter  | Type    | Default | Description                              |
|------------|---------|---------|------------------------------------------|
| `days`     | integer | `30`    | Number of days ahead to check           |
| `page`     | integer | `1`     | Page number                              |
| `page_size`| integer | `50`    | Items per page                           |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "vehicle_id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "driver_id": "d7a3f291-9c4e-4b8f-a1d2-e5f6g7h8i9j0",
      "make": "Toyota",
      "model": "Camry",
      "license_plate": "ABC-1234",
      "inspection_expiry": "2024-02-10T00:00:00Z",
      "days_until_expiry": 21
    }
  ],
  "total": 8
}
```

---

### Fleet Analytics

#### `GET /api/v1/fleet/summary`

Get fleet-wide summary statistics (admin only).

**Response `200 OK`:**
```json
{
  "total_vehicles": 1250,
  "by_status": {
    "active": 1100,
    "inactive": 100,
    "suspended": 40,
    "decommissioned": 10
  },
  "by_type": {
    "sedan": 850,
    "suv": 280,
    "van": 90,
    "truck": 30
  },
  "by_fuel_type": {
    "gasoline": 900,
    "hybrid": 250,
    "electric": 80,
    "diesel": 20
  },
  "pending_document_verification": 23,
  "inspections_expiring_30_days": 45,
  "maintenance_overdue": 12,
  "average_fleet_age_years": 3.2,
  "average_mileage": 38500,
  "generated_at": "2024-01-20T08:00:00Z"
}
```

---

#### `GET /api/v1/fleet/vehicles/nearby`

Find available vehicles near a given location.

**Query Parameters:**

| Parameter      | Type    | Required | Description                        |
|----------------|---------|----------|------------------------------------|
| `latitude`     | float   | Yes      | Center latitude                    |
| `longitude`    | float   | Yes      | Center longitude                   |
| `radius_km`    | float   | No       | Search radius in km (default: 5.0) |
| `vehicle_type` | string  | No       | Filter by vehicle type             |
| `limit`        | integer | No       | Max results (default: 20)          |

**Response `200 OK`:**
```json
{
  "center": { "latitude": 40.7128, "longitude": -74.0060 },
  "radius_km": 5.0,
  "count": 3,
  "vehicles": [
    {
      "vehicle_id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "make": "Toyota",
      "model": "Camry",
      "vehicle_type": "sedan",
      "location": { "latitude": 40.7140, "longitude": -74.0055 },
      "distance_km": 0.14,
      "driver_id": "d7a3f291-9c4e-4b8f-a1d2-e5f6g7h8i9j0",
      "capacity": 4
    }
  ]
}
```

---

## gRPC API

The service exposes a gRPC server on port **9093** for internal service communication.

```protobuf
service VehicleService {
  rpc GetVehicle(GetVehicleRequest) returns (VehicleResponse);
  rpc GetVehiclesByDriver(GetVehiclesByDriverRequest) returns (VehiclesResponse);
  rpc UpdateVehicleStatus(UpdateVehicleStatusRequest) returns (VehicleResponse);
  rpc GetVehicleLocation(GetVehicleLocationRequest) returns (LocationResponse);
  rpc CheckVehicleEligibility(CheckEligibilityRequest) returns (EligibilityResponse);
}
```

**Used by:** `trip-management-service`, `driver-onboarding-service`, `billing-service`

---

## Authentication

The service validates JWT tokens signed with RS256. Token claims include:

| Claim    | Description                                   |
|----------|-----------------------------------------------|
| `sub`    | User ID (driver or admin)                     |
| `role`   | User role: `driver`, `admin`, `fleet_manager` |
| `exp`    | Token expiration timestamp                    |
| `iss`    | Issuer: `rideshare-auth-service`              |

**Role Permissions:**
- `driver` â Can manage their own vehicles and documents
- `fleet_manager` â Can view all vehicles, manage maintenance and inspections
- `admin` â Full access including document verification and fleet analytics

---

## Error Handling

All errors follow a consistent structure:

```json
{
  "error": "ERROR_CODE",
  "message": "Human-readable description",
  "field": "affected_field (optional)",
  "trace_id": "otel-trace-id",
  "timestamp": "2024-01-20T08:15:00Z"
}
```

### HTTP Status Codes

| Status | Code                      | Description                        |
|--------|---------------------------|------------------------------------|
| `400`  | `VALIDATION_ERROR`        | Invalid request payload            |
| `401`  | `UNAUTHORIZED`            | Missing or invalid JWT token       |
| `403`  | `FORBIDDEN`               | Insufficient permissions           |
| `404`  | `VEHICLE_NOT_FOUND`       | Vehicle does not exist             |
| `409`  | `VEHICLE_ALREADY_EXISTS`  | Duplicate VIN or license plate     |
| `413`  | `FILE_TOO_LARGE`          | Document file exceeds 10MB limit   |
| `422`  | `INVALID_STATUS_TRANSITION` | Invalid vehicle status change    |
| `429`  | `RATE_LIMIT_EXCEEDED`     | Too many requests                  |
| `500`  | `INTERNAL_ERROR`          | Unexpected server error            |
| `503`  | `SERVICE_UNAVAILABLE`     | Dependency unavailable             |

---

## Deployment

### Kubernetes

```bash
# Apply all manifests
kubectl apply -f k8s-deployment.yaml

# Check rollout status
kubectl rollout status deployment/vehicle-management-service -n rideshare

# View pods
kubectl get pods -n rideshare -l app=vehicle-management-service

# View HPA
kubectl get hpa vehicle-management-hpa -n rideshare

# View logs
kubectl logs -f deployment/vehicle-management-service -n rideshare

# Scale manually
kubectl scale deployment vehicle-management-service --replicas=5 -n rideshare
```

### Rolling Update

```bash
# Update image
kubectl set image deployment/vehicle-management-service \
  vehicle-management-service=rideshare/vehicle-management-service:v1.2.0 \
  -n rideshare

# Monitor rollout
kubectl rollout status deployment/vehicle-management-service -n rideshare

# Rollback if needed
kubectl rollout undo deployment/vehicle-management-service -n rideshare
```

---

## Monitoring

### Prometheus Metrics

Metrics are exposed at `GET /metrics` (port 8083).

| Metric                                         | Type      | Description                          |
|------------------------------------------------|-----------|--------------------------------------|
| `vehicle_registrations_total`                  | Counter   | Total vehicle registrations          |
| `vehicle_registrations_by_type`                | Counter   | Registrations broken down by type    |
| `vehicle_document_uploads_total`               | Counter   | Total document uploads               |
| `vehicle_document_verification_duration`       | Histogram | Document verification processing time|
| `vehicle_telemetry_ingested_total`             | Counter   | Total telemetry data points ingested |
| `vehicle_telemetry_batch_size`                 | Histogram | Telemetry batch size distribution    |
| `vehicle_maintenance_records_total`            | Counter   | Total maintenance records created    |
| `vehicle_status_transitions_total`             | Counter   | Status change events by transition   |
| `fleet_active_vehicles`                        | Gauge     | Current number of active vehicles    |
| `http_request_duration_seconds`                | Histogram | HTTP request latency by endpoint     |
| `grpc_request_duration_seconds`                | Histogram | gRPC request latency by method       |
| `db_query_duration_seconds`                    | Histogram | Database query duration              |
| `cache_hit_ratio`                              | Gauge     | Redis cache hit ratio                |

### Distributed Tracing

All requests are instrumented with OpenTelemetry and exported to Jaeger. Set `JAEGER_ENDPOINT` to enable tracing.

### Logging

Structured JSON logs via Zap. Log levels: `debug`, `info`, `warn`, `error`.

Example log entry:
```json
{
  "level": "info",
  "ts": "2024-01-20T08:15:00.123Z",
  "caller": "handlers/vehicle.go:45",
  "msg": "Vehicle registered",
  "vehicle_id": "v1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "driver_id": "d7a3f291-9c4e-4b8f-a1d2-e5f6g7h8i9j0",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "duration_ms": 42
}
```

---

## Running Tests

```bash
# Unit tests
go test ./... -v -short

# Integration tests (requires running dependencies)
go test ./... -v -tags=integration

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Race condition detection
go test ./... -race

# Benchmark tests
go test ./... -bench=. -benchmem
```

---

## License

Copyright Â© 2024 Rideshare Platform. All rights reserved.

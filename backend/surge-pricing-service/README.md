# Surge Pricing & Dynamic Pricing Service

> **Dynamische Preisgestaltung** for German Ride-Sharing Platform

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()
[![Coverage](https://img.shields.io/badge/coverage-94%25-brightgreen)]()
[![License](https://img.shields.io/badge/license-MIT-blue)]()
[![PBefG Compliant](https://img.shields.io/badge/PBefG-compliant-blue)]()
[![Kafka](https://img.shields.io/badge/kafka-3.5-orange)]()
[![Java](https://img.shields.io/badge/java-17-red)]()

---

## Table of Contents

1. [Service Overview](#service-overview)
2. [Key Features](#key-features)
3. [Architecture](#architecture)
4. [API Endpoints](#api-endpoints)
5. [Request/Response Examples](#requestresponse-examples)
6. [Environment Variables](#environment-variables)
7. [Local Development Setup](#local-development-setup)
8. [Deployment Instructions](#deployment-instructions)
9. [German Compliance (PBefG) Notes](#german-compliance-pbefg-notes)
10. [Kafka Event Topics](#kafka-event-topics)
11. [Database Schema](#database-schema-overview)
12. [Monitoring and Metrics](#monitoring-and-metrics)

---

## Service Overview

The **Surge Pricing & Dynamic Pricing Service** is a mission-critical microservice responsible for computing, managing, and distributing real-time dynamic pricing data across the ride-sharing platform. It continuously analyzes supply (available drivers) and demand (active ride requests) across geographic zones (defined as hexagonal H3 grids) within Germany and applies legally compliant pricing multipliers.

This service integrates tightly with the Driver Availability Service, Booking Service, and Payments Service, publishing pricing events to Apache Kafka and exposing a REST API for synchronous price queries.

> **Important:** All pricing logic is implemented in strict accordance with the German **PersonenbefÃ¶rderungsgesetz (PBefG)**, including mandatory price transparency, cap limits, and regulatory reporting requirements.

**Service Type:** Backend Microservice
**Team Owner:** Platform Pricing Team (`pricing-team@company.de`)
**Runbook:** [Confluence - Surge Pricing Runbook](https://confluence.internal/pricing/runbook)
**SLA:** 99.95% availability | p99 latency < 80ms

---

## Key Features

- â¡ **Real-Time Surge Computation** â Sub-100ms pricing decisions based on live supply/demand ratios per H3 hexagonal grid cell (resolution 7)
- ð **Multi-Factor Pricing Model** â Incorporates demand density, driver availability, historical patterns, weather conditions, and local events
- ð **PBefG-Compliant Price Caps** â Enforces statutory maximum multipliers and mandatory price transparency disclosures
- ðºï¸ **Geo-Zone Management** â Dynamic zone creation and expiry based on H3 hexagonal indexing across all German federal states
- ð¡ **Kafka Event Streaming** â Publishes pricing change events in real-time for downstream service consumption
- ð§  **ML-Powered Demand Forecasting** â Integrates with the ML Forecasting Service to pre-warm price adjustments before demand spikes
- ð **Fallback Pricing Strategy** â Graceful degradation to cached baseline pricing during ML service outages
- ð **Audit Logging** â Immutable audit trail of all pricing decisions for regulatory compliance and dispute resolution
- ð **Multi-City Support** â Covers all German cities where the platform operates, with city-specific baseline configurations
- ð **A/B Testing Framework** â Built-in support for pricing strategy experimentation with statistical significance tracking
- ð¡ï¸ **Rate Limiting & Anti-Abuse** â Prevents price scraping and abuse via token-bucket rate limiting per API consumer
- ð¶ **EUR-Only Transactions** â All pricing is denominated in Euro (EUR) with cent-precision arithmetic using `BigDecimal`

---

## Architecture

### High-Level Architecture Diagram

```
âââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ
â                        SURGE PRICING SERVICE                                â
â                      (surge-pricing-service)                                â
âââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââââ

  External Inputs                Core Engine                  Outputs
  ââââââââââââââ                ââââââââââââ                  âââââââ

  âââââââââââââââââââ           ââââââââââââââââââââââââââ    ââââââââââââââââââââââââ
  â  Driver         ââââââââââââ¶â                        âââââ¶â  Kafka Topics        â
  â  Availability   â  events   â   Demand Aggregator    â    â  (pricing.updates,   â
  â  Service        â           â   & Zone Calculator    â    â   pricing.audit)     â
  âââââââââââââââââââ           â                        â    ââââââââââââââââââââââââ
                                â   ââââââââââââââââââ   â
  âââââââââââââââââââ           â   â  Surge         â   â    ââââââââââââââââââââââââ
  â  Booking        ââââââââââââ¶â   â  Multiplier    â   âââââ¶â  REST API            â
  â  Service        â  events   â   â  Engine        â   â    â  (GET /price,        â
  âââââââââââââââââââ           â   ââââââââââââââââââ   â    â   POST /zone, ...)   â
                                â                        â    ââââââââââââââââââââââââ
  âââââââââââââââââââ           â   ââââââââââââââââââ   â
  â  ML Forecast    ââââââââââââ¶â   â  PBefG         â   â    ââââââââââââââââââââââââ
  â  Service        â  gRPC     â   â  Compliance    â   âââââ¶â  PostgreSQL           â
  âââââââââââââââââââ           â   â  Validator     â   â    â  (pricing_zones,     â
                                â   ââââââââââââââââââ   â    â   pricing_history,   â
  âââââââââââââââââââ           â                        â    â   audit_log)         â
  â  Weather API    ââââââââââââ¶â   ââââââââââââââââââ   â    ââââââââââââââââââââââââ
  â  (DWD OpenData) â  HTTP     â   â  Cache Layer   â   â
  âââââââââââââââââââ           â   â  (Redis)       â   â    ââââââââââââââââââââââââ
                                â   ââââââââââââââââââ   âââââ¶â  Prometheus Metrics  â
  âââââââââââââââââââ           â                        â    â  + Grafana Dashboard â
  â  Event API      ââââââââââââ¶â   ââââââââââââââââââ   â    ââââââââââââââââââââââââ
  â  (local events) â  HTTP     â   â  Audit Logger  â   â
  âââââââââââââââââââ           â   ââââââââââââââââââ   â
                                ââââââââââââââââââââââââââ
```

### Internal Component Breakdown

```
surge-pricing-service/
âââ api/                        # REST controllers and DTOs
â   âââ PriceController.java
â   âââ ZoneController.java
â   âââ AdminController.java
â   âââ dto/
âââ domain/                     # Core business logic
â   âââ SurgeMultiplierEngine.java
â   âââ DemandAggregator.java
â   âââ ZoneManager.java
â   âââ PbefgComplianceValidator.java
â   âââ ForecastIntegrator.java
âââ infrastructure/
â   âââ kafka/                  # Kafka producers and consumers
â   âââ redis/                  # Cache repository
â   âââ postgres/               # JPA entities and repositories
â   âââ grpc/                   # ML service gRPC client
â   âââ weather/                # DWD weather client
âââ config/                     # Spring configuration beans
âââ monitoring/                 # Metrics and health indicators
```

### Data Flow

```
[Driver GPS Update] âââ¶ [Kafka Consumer] âââ¶ [Zone Demand Recalculator]
                                                        â
                                                        â¼
                                             [Surge Engine]
                                                        â
                                          âââââââââââââââ´ââââââââââââââ
                                          â                           â
                                          â¼                           â¼
                                [PBefG Validator]            [ML Forecast Input]
                                          â
                              âââââââââââââ´âââââââââââ
                              â                      â
                              â¼                      â¼
                        [Redis Cache]        [Kafka Producer]
                              â                      â
                              â¼                      â¼
                         [REST API]         [Downstream Services]
```

---

## API Endpoints

### Base URL

- **Development:** `http://localhost:8085`
- **Staging:** `https://surge-pricing.staging.internal.company.de`
- **Production:** `https://surge-pricing.prod.internal.company.de`

> All endpoints require internal mTLS authentication in staging and production.

### Endpoint Reference

| Method | Endpoint | Auth Required | Description |
|--------|----------|---------------|-------------|
| `GET` | `/api/v1/price/estimate` | API Key | Get current price estimate for a route (origin â destination coordinates). Returns base fare, surge multiplier, and total estimate. |
| `GET` | `/api/v1/price/zone/{zoneId}` | API Key | Retrieve current surge multiplier for a specific H3 pricing zone by zone ID. |
| `GET` | `/api/v1/price/zones` | API Key | List all active pricing zones with current multipliers. Supports pagination and filtering by city or federal state. |
| `POST` | `/api/v1/price/validate` | API Key | Validate a proposed fare before booking confirmation. Ensures price has not changed beyond tolerance threshold since estimate was given. |
| `GET` | `/api/v1/zones` | API Key | List all configured H3 hexagonal zones, including inactive ones. Supports `city`, `state`, and `active` query filters. |
| `POST` | `/api/v1/zones` | Admin JWT | Create a new pricing zone. Requires `ADMIN` role. |
| `PUT` | `/api/v1/zones/{zoneId}` | Admin JWT | Update configuration for an existing pricing zone (base fare, boundaries, minimum fare). |
| `DELETE` | `/api/v1/zones/{zoneId}` | Admin JWT | Soft-delete a pricing zone. Zone is deactivated, not permanently removed (audit trail preserved). |
| `GET` | `/api/v1/zones/{zoneId}/history` | Admin JWT | Retrieve historical surge multiplier data for a zone. Supports `from`, `to`, and `granularity` query parameters. |
| `POST` | `/api/v1/admin/override` | Admin JWT | Manually override the surge multiplier for a zone (e.g., for emergency situations or special events). Override expires after configurable TTL. |
| `DELETE` | `/api/v1/admin/override/{zoneId}` | Admin JWT | Remove an active manual override and revert to computed multiplier. |
| `GET` | `/api/v1/admin/config` | Admin JWT | Retrieve current service configuration including pricing algorithm parameters. |
| `PUT` | `/api/v1/admin/config` | Admin JWT | Update runtime configuration parameters without service restart (e.g., sensitivity thresholds, cap limits). |
| `GET` | `/actuator/health` | None | Spring Boot health check endpoint. |
| `GET` | `/actuator/prometheus` | Internal | Prometheus metrics scrape endpoint. |
| `GET` | `/actuator/info` | Internal | Service build info and version. |

---

## Request/Response Examples

### 1. Get Price Estimate

**Request:**
```http
GET /api/v1/price/estimate?originLat=52.5200&originLon=13.4050&destLat=52.5100&destLon=13.3900&vehicleClass=STANDARD
Authorization: ApiKey sk_live_xxxxxxxxxxxx
Accept: application/json
X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
```

**Response `200 OK`:**
```json
{
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2024-07-15T14:32:00.123Z",
  "origin": {
    "lat": 52.5200,
    "lon": 13.4050,
    "zoneId": "871f1a3a4ffffff",
    "city": "Berlin",
    "federalState": "BE"
  },
  "destination": {
    "lat": 52.5100,
    "lon": 13.3900,
    "zoneId": "871f1a3b4ffffff"
  },
  "vehicleClass": "STANDARD",
  "pricing": {
    "baseFare": "3.50",
    "distanceFare": "4.20",
    "timeFare": "1.80",
    "subtotal": "9.50",
    "surgeMultiplier": 1.8,
    "surgeActive": true,
    "surgeReason": "HIGH_DEMAND",
    "surgeTotal": "17.10",
    "minimumFare": "5.00",
    "finalEstimate": "17.10",
    "currency": "EUR"
  },
  "pbefgDisclosure": {
    "surgeNoticeRequired": true,
    "userAcknowledgmentRequired": true,
    "disclosureText": "Aktuell besteht eine erhÃ¶hte Nachfrage. Der Preis ist um den Faktor 1,8 erhÃ¶ht. Sie mÃ¼ssen dieser PreiserhÃ¶hung ausdrÃ¼cklich zustimmen.",
    "maxMultiplierAllowed": 3.0
  },
  "estimateValidUntil": "2024-07-15T14:34:00.123Z",
  "estimateToken": "eyJhbGciOiJIUzI1NiJ9..."
}
```

---

### 2. Validate Price Before Booking

**Request:**
```http
POST /api/v1/price/validate
Authorization: ApiKey sk_live_xxxxxxxxxxxx
Content-Type: application/json
X-Request-ID: 660e8400-e29b-41d4-a716-446655440001

{
  "estimateToken": "eyJhbGciOiJIUzI1NiJ9...",
  "acceptedFare": "17.10",
  "surgeAcknowledged": true,
  "userId": "usr_abc123"
}
```

**Response `200 OK` â Price still valid:**
```json
{
  "requestId": "660e8400-e29b-41d4-a716-446655440001",
  "valid": true,
  "currentFare": "17.10",
  "acceptedFare": "17.10",
  "priceDelta": "0.00",
  "withinTolerance": true,
  "toleranceThreshold": "0.05",
  "bookingApproved": true,
  "approvalToken": "appr_xyz987"
}
```

**Response `409 Conflict` â Price has changed beyond tolerance:**
```json
{
  "requestId": "660e8400-e29b-41d4-a716-446655440001",
  "valid": false,
  "currentFare": "19.50",
  "acceptedFare": "17.10",
  "priceDelta": "2.40",
  "withinTolerance": false,
  "toleranceThreshold": "0.05",
  "bookingApproved": false,
  "errorCode": "PRICE_CHANGED",
  "message": "Der Preis hat sich geÃ¤ndert. Bitte bestÃ¤tigen Sie den neuen Preis.",
  "newEstimateRequired": true
}
```

---

### 3. Get Zone Surge Multiplier

**Request:**
```http
GET /api/v1/price/zone/871f1a3a4ffffff
Authorization: ApiKey sk_live_xxxxxxxxxxxx
```

**Response `200 OK`:**
```json
{
  "zoneId": "871f1a3a4ffffff",
  "h3Index": "871f1a3a4ffffff",
  "city": "Berlin",
  "federalState": "BE",
  "surgeMultiplier": 1.8,
  "surgeLevel": "MEDIUM",
  "baselineFare": "3.50",
  "activeDrivers": 12,
  "pendingRequests": 28,
  "supplyDemandRatio": 0.43,
  "lastUpdated": "2024-07-15T14:31:55.000Z",
  "nextUpdateExpected": "2024-07-15T14:32:25.000Z",
  "manualOverride": false
}
```

---

### 4. Create Manual Admin Override

**Request:**
```http
POST /api/v1/admin/override
Authorization: Bearer eyJhbGciOiJSUzI1NiJ9...
Content-Type: application/json

{
  "zoneId": "871f1a3a4ffffff",
  "multiplier": 1.2,
  "reason": "Silvester event â police request to limit surge near Brandenburger Tor",
  "ttlMinutes": 120,
  "operatorId": "ops_berlin_01"
}
```

**Response `201 Created`:**
```json
{
  "overrideId": "ovr_2024_berlin_001",
  "zoneId": "871f1a3a4ffffff",
  "previousMultiplier": 1.8,
  "overrideMultiplier": 1.2,
  "reason": "Silvester event â police request to limit surge near Brandenburger Tor",
  "appliedAt": "2024-07-15T14:35:00.000Z",
  "expiresAt": "2024-07-15T16:35:00.000Z",
  "operatorId": "ops_berlin_01",
  "auditEntryId": "aud_99f2a1c3"
}
```

---

### 5. Error Response Format

All error responses follow the RFC 7807 Problem Details standard:

```json
{
  "type": "https://docs.company.de/errors/surge-pricing/ZONE_NOT_FOUND",
  "title": "Pricing Zone Not Found",
  "status": 404,
  "detail": "No active pricing zone found with ID 'invalid_zone_id'.",
  "instance": "/api/v1/price/zone/invalid_zone_id",
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2024-07-15T14:32:00.123Z"
}
```

---

## Environment Variables

### Required Variables

| Variable | Description | Example Value | Secret |
|----------|-------------|---------------|--------|
| `DATABASE_URL` | PostgreSQL JDBC connection URL | `jdbc:postgresql://postgres:5432/surge_pricing` | â Yes |
| `DATABASE_USERNAME` | PostgreSQL username | `surge_pricing_user` | â Yes |
| `DATABASE_PASSWORD` | PostgreSQL password | `****` | â Yes |
| `REDIS_URL` | Redis connection URL including auth | `redis://:password@redis:6379/0` | â Yes |
| `KAFKA_BOOTSTRAP_SERVERS` | Comma-separated Kafka broker list | `kafka-1:9092,kafka-2:9092,kafka-3:9092` | No |
| `KAFKA_SECURITY_PROTOCOL` | Kafka security protocol | `SASL_SSL` | No |
| `KAFKA_SASL_USERNAME` | Kafka SASL username | `surge-pricing-service` | â Yes |
| `KAFKA_SASL_PASSWORD` | Kafka SASL password | `****` | â Yes |
| `ML_FORECAST_GRPC_HOST` | ML Forecasting Service gRPC host | `ml-forecast.internal:50051` | No |
| `API_KEY_SECRET` | Secret for signing/verifying API keys | `****` | â Yes |
| `JWT_PUBLIC_KEY_PATH` | Path to RSA public key for JWT verification | `/secrets/jwt-public.pem` | No |
| `ESTIMATE_TOKEN_SECRET` | HMAC secret for estimate token signing | `****` | â Yes |

### Optional / Tuning Variables

| Variable | Description | Default | Example Value |
|----------|-------------|---------|---------------|
| `SERVER_PORT` | HTTP server port | `8085` | `8085` |
| `SURGE_MIN_MULTIPLIER` | Minimum allowed surge multiplier | `1.0` | `1.0` |
| `SURGE_MAX_MULTIPLIER` | Maximum allowed surge multiplier (PBefG cap) | `3.0` | `3.0` |
| `SURGE_UPDATE_INTERVAL_SECONDS` | How often surge is recalculated per zone | `30` | `30` |
| `SURGE_DEMAND_THRESHOLD_LOW` | Supply/demand ratio below which low surge activates | `0.8` | `0.8` |
| `SURGE_DEMAND_THRESHOLD_MEDIUM` | Ratio below which medium surge activates | `0.5` | `0.5` |
| `SURGE_DEMAND_THRESHOLD_HIGH` | Ratio below which high surge activates | `0.25` | `0.25` |
| `ESTIMATE_VALIDITY_SECONDS` | How long a price estimate remains valid | `120` | `120` |
| `ESTIMATE_TOLERANCE_PERCENT` | Max price change % tolerated during validation | `5` | `5` |
| `REDIS_CACHE_TTL_SECONDS` | TTL for zone multiplier cache entries | `60` | `60` |
| `ML_FORECAST_TIMEOUT_MS` | gRPC timeout for ML forecast calls | `200` | `200` |
| `ML_FORECAST_ENABLED` | Feature flag to enable/disable ML forecast integration | `true` | `true` |
| `WEATHER_API_ENABLED` | Feature flag to enable/disable weather factor | `true` | `true` |
| `DWD_WEATHER_API_URL` | URL for Deutscher Wetterdienst OpenData API | `https://dwd.api.proxy.bund.dev` | â |
| `PBEFG_STRICT_MODE` | Enforce strictest PBefG interpretation (recommended for production) | `true` | `true` |
| `AUDIT_LOG_RETENTION_DAYS` | Number of days to retain audit log entries in DB | `365` | `365` |
| `SPRING_PROFILES_ACTIVE` | Active Spring profiles | `prod` | `dev`, `staging`, `prod` |
| `LOG_LEVEL_ROOT` | Root logging level | `INFO` | `DEBUG`, `INFO`, `WARN` |
| `SENTRY_DSN` | Sentry error tracking DSN | _(empty)_ | `https://xxx@sentry.io/123` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry collector endpoint | _(empty)_ | `http://otel-collector:4317` |

---

## Local Development Setup

### Prerequisites

| Tool | Version | Installation |
|------|---------|--------------|
| Java (JDK) | 17+ | [Adoptium Temurin](https://adoptium.net/) |
| Maven | 3.9+ | `brew install maven` |
| Docker & Docker Compose | 24+ | [Docker Desktop](https://www.docker.com/products/docker-desktop/) |
| `kubectl` | 1.28+ | Optional, for local K8s testing |
| `kafkacat` / `kcat` | Latest | Optional, for Kafka debugging |

### Step 1 â Clone the Repository

```bash
git clone git@github.com:company/surge-pricing-service.git
cd surge-pricing-service
```

### Step 2 â Start Infrastructure Dependencies

A `docker-compose.yml` is provided for local development dependencies:

```bash
docker-compose -f docker-compose.dev.yml up -d
```

This starts:
- **PostgreSQL 15** on `localhost:5432` (database: `surge_pricing`)
- **Redis 7** on `localhost:6379`
- **Kafka + Zookeeper** (Confluent Platform) on `localhost:9092`
- **Schema Registry** on `localhost:8081`
- **Kafka UI** on `localhost:8080` (browse topics easily)

Verify services are healthy:
```bash
docker-compose -f docker-compose.dev.yml ps
```

### Step 3 â Configure Local Environment

Copy the example environment file:
```bash
cp .env.example .env.local
```

Edit `.env.local` with your local values (defaults work out of the box with the Docker Compose setup):
```bash
DATABASE_URL=jdbc:postgresql://localhost:5432/surge_pricing
DATABASE_USERNAME=surge_pricing_user
DATABASE_PASSWORD=localdevpassword
REDIS_URL=redis://localhost:6379/0
KAFKA_BOOTSTRAP_SERVERS=localhost:9092
KAFKA_SECURITY_PROTOCOL=PLAINTEXT
ML_FORECAST_ENABLED=false
WEATHER_API_ENABLED=false
PBEFG_STRICT_MODE=false
SPRING_PROFILES_ACTIVE=dev
```

### Step 4 â Run Database Migrations

Database migrations are managed via **Flyway** and run automatically on startup. To run them manually:

```bash
mvn flyway:migrate -Dflyway.url=jdbc:postgresql://localhost:5432/surge_pricing \
  -Dflyway.user=surge_pricing_user \
  -Dflyway.password=localdevpassword
```

### Step 5 â Build and Run

```bash
# Build (skip tests for fast startup)
mvn clean package -DskipTests

# Run the service
mvn spring-boot:run -Dspring-boot.run.profiles=dev

# Or run the JAR directly with env file
export $(cat .env.local | xargs)
java -jar target/surge-pricing-service-*.jar
```

The service will be available at `http://localhost:8085`.

### Step 6 â Seed Test Data

```bash
# Load default Berlin, Munich, and Hamburg pricing zones
curl -X POST http://localhost:8085/api/v1/admin/seed \
  -H "Authorization: Bearer $(cat dev-admin-token.txt)" \
  -H "Content-Type: application/json" \
  -d '{"cities": ["berlin", "munich", "hamburg"]}'
```

### Running Tests

```bash
# Unit tests only
mvn test

# Unit + Integration tests (requires Docker for Testcontainers)
mvn verify -P integration-tests

# Full test suite with coverage report
mvn verify -P integration-tests jacoco:report
# Report available at: target/site/jacoco/index.html
```

### Useful Local Endpoints

| URL | Description |
|-----|-------------|
| `http://localhost:8085/actuator/health` | Health check |
| `http://localhost:8085/actuator/info` | Build info |
| `http://localhost:8085/swagger-ui.html` | Interactive API docs (dev profile only) |
| `http://localhost:8080` | Kafka UI |
| `http://localhost:9090` | Prometheus (if started via compose) |

---

## Deployment Instructions

### Deployment Architecture

The service is deployed on **Kubernetes (EKS)** using Helm charts, with separate clusters for staging and production. Container images are stored in ECR and deployments are managed via ArgoCD GitOps.

### CI/CD Pipeline

```
[Git Push] âââ¶ [GitHub Actions CI]
                    â
                    âââ mvn verify (unit + integration tests)
                    âââ SAST scan (SonarQube)
                    âââ Docker build + ECR push
                    âââ Helm chart lint
                              â
                              â¼ (on merge to main)
                   [ArgoCD â Staging Sync]
                              â
                              âââ Automated smoke tests
                              âââ PBefG compliance check suite
                              âââ Performance baseline test
                                        â
                              [Manual Approval Gate]
                                        â
                              [ArgoCD â Production Sync]
```

### Build Docker Image

```bash
# Build
docker build -t surge-pricing-service:latest .

# Tag for ECR
docker tag surge-pricing-service:latest \
  123456789.dkr.ecr.eu-central-1.amazonaws.com/surge-pricing-service:$(git rev-parse --short HEAD)

# Push
aws ecr get-login-password --region eu-central-1 | \
  docker login --username AWS --password-stdin 123456789.dkr.ecr.eu-central-1.amazonaws.com
docker push 123456789.dkr.ecr.eu-central-1.amazonaws.com/surge-pricing-service:$(git rev-parse --short HEAD)
```

### Helm Deployment

```bash
# Add the internal Helm repo
helm repo add company-charts https://charts.internal.company.de
helm repo update

# Deploy to staging
helm upgrade --install surge-pricing-service ./helm/surge-pricing-service \
  --namespace pricing \
  --values helm/values.staging.yaml \
  --set image.tag=$(git rev-parse --short HEAD) \
  --atomic \
  --timeout 5m

# Deploy to production
helm upgrade --install surge-pricing-service ./helm/surge-pricing-service \
  --namespace pricing \
  --values helm/values.production.yaml \
  --set image.tag=$(git rev-parse --short HEAD) \
  --atomic \
  --timeout 5m
```

### Kubernetes Resource Overview

```yaml
# Typical production sizing
apiVersion: apps/v1
kind: Deployment
spec:
  replicas: 3                   # Minimum 3 for HA across AZs
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0         # Zero-downtime deployments
  template:
    spec:
      containers:
      - name: surge-pricing-service
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
```

### Production Secrets Management

All secrets are managed via **AWS Secrets Manager** and injected into pods using the **External Secrets Operator**:

```bash
# Verify secrets are synced
kubectl get externalsecrets -n pricing
kubectl describe externalsecret surge-pricing-secrets -n pricing
```

### Rolling Back a Deployment

```bash
# Via Helm
helm rollback surge-pricing-service -n pricing

# Via ArgoCD UI
# Navigate to: https://argocd.internal.company.de/applications/surge-pricing-service
# Click: History â Rollback to previous revision

# Emergency: scale down if critical bug
kubectl scale deployment surge-pricing-service --replicas=0 -n pricing
# Fallback pricing will activate automatically in dependent services
```

### Health Check Verification Post-Deployment

```bash
# Check pod status
kubectl get pods -n pricing -l app=surge-pricing-service

# Tail logs
kubectl logs -f -l app=surge-pricing-service -n pricing

# Check health endpoint
kubectl exec -it $(kubectl get pods -n pricing -l app=surge-pricing-service -o name | head -1) \
  -- curl -s http://localhost:8085/actuator/health | jq
```

---

## German Compliance (PBefG) Notes

> âï¸ **Legal Disclaimer:** The following notes are an engineering implementation summary and do not constitute legal advice. All compliance decisions must be validated by the company's legal team and authorized PersonenbefÃ¶rderungsgesetz experts.

### Applicable Regulations

| Regulation | Relevance |
|------------|----------|
| **PersonenbefÃ¶rderungsgesetz (PBefG) Â§51c** | Platform-based ride-hailing regulation, in force since August 2021 (Novelle). Governs dynamic pricing for Mietwagen services. |
| **PBefG Â§39** | Tariff approval requirements for Taxen. Note: Different rules apply to Mietwagen. |
| **PBefG Â§49 (4)** | Specific rules for vehicle rental (Mietwagen) operations. |
| **DSGVO / GDPR Art. 22** | Automated decision-making transparency â pricing algorithms may be subject to this article. |
| **UWG Â§5** | Prohibition on misleading commercial practices â pricing must be honest and transparent. |
| **Preisangabenverordnung (PAngV)** | Price indication regulations â total price inclusive of all taxes must be shown. |

### PBefG Implementation Details

#### 1. Surge Multiplier Cap (`SURGE_MAX_MULTIPLIER`)

The service enforces a **hard cap** of **3.0x** on all surge multipliers. This cap is:
- Enforced in `PbefgComplianceValidator.java` before any price is published
- Cannot be overridden via the admin API (admin overrides can only **reduce** multipliers, never exceed the cap)
- Logged in the audit trail for every pricing decision
- Configurable via `SURGE_MAX_MULTIPLIER` env var but **only downward** in strict mode (`PBEFG_STRICT_MODE=true`)

```java
// PbefgComplianceValidator.java â excerpt
public BigDecimal enforceMultiplierCap(BigDecimal proposed) {
    BigDecimal cap = BigDecimal.valueOf(surgeMaxMultiplier); // from config
    if (proposed.compareTo(cap) > 0) {
        auditLogger.logCapEnforcement(proposed, cap);
        return cap;
    }
    return proposed;
}
```

#### 2. Mandatory Price Transparency Disclosure

When `surgeMultiplier > 1.0`, the API **always** returns a `pbefgDisclosure` object in the price estimate response. The frontend **must** display this disclosure and **must** obtain explicit user acknowledgment (`surgeAcknowledged: true`) before booking can proceed.

- Disclosure text is provided in German (`de-DE`) by default
- The booking validation endpoint (`POST /api/v1/price/validate`) will **reject** requests where `surgeAcknowledged: false` during surge

#### 3. Estimate Token Anti-Gaming

The `estimateToken` is a short-lived HMAC-signed JWT that encodes the offered price, multiplier, timestamp, and user ID. This prevents:
- Clients from bypassing surge by replaying old estimates
- Price manipulation between estimate and booking
- Token expiry is strictly enforced (default: 120 seconds)

#### 4. Audit Trail Requirements

Every pricing decision is written to the `pricing_audit_log` table with the following fields:
- User ID, zone ID, timestamp, base fare, surge multiplier, final fare
- Algorithm version that produced the price
- Whether PBefG cap was enforced
- Whether a manual override was active
- Operator ID if manual override (for regulatory accountability)

Audit logs are retained for **365 days minimum** (configurable via `AUDIT_LOG_RETENTION_DAYS`) and cannot be deleted by normal service operations.

#### 5. Regulatory Reporting

The service exposes a dedicated reporting endpoint (internal, not in the public API table above) used by the Regulatory Reporting Service to generate monthly reports for the **Kraftfahrtbundesamt (KBA)** and relevant local GenehmigungsbehÃ¶rden:

```
GET /internal/v1/reports/monthly?year=2024&month=07
```

This returns aggregated pricing statistics per city and zone including surge frequency, average multipliers, and cap enforcement counts.

#### 6. RÃ¼ckkehrpflicht Compliance

The pricing service does **not** incentivize or penalize based on vehicle return-to-base patterns. The `DemandAggregator` explicitly excludes driver-at-base-zone status from demand signals to remain compliant with PBefG Â§49(4) RÃ¼ckkehrpflicht.

### Compliance Checklist for Releases

Before every production release, the release engineer must verify:

- [ ] `SURGE_MAX_MULTIPLIER` â¤ 3.0 in production Helm values
- [ ] `PBEFG_STRICT_MODE=true` in production environment
- [ ] PBefG compliance test suite passes (`mvn verify -P pbefg-compliance`)
- [ ] Disclosure text in German has been reviewed by Legal if modified
- [ ] Audit log retention policy is still 365 days
- [ ] Admin override audit trail is intact (verify via `GET /internal/v1/audit/overrides`)

---

## Kafka Event Topics

### Topic Overview

| Topic Name | Type | Partitions | Replication | Description |
|------------|------|------------|-------------|-------------|
| `pricing.zone.updates` | Produced | 12 | 3 | Emitted whenever a zone's surge multiplier changes. Consumed by Booking Service, Driver App Service, Passenger App Service. |
| `pricing.estimates.created` | Produced | 6 | 3 | Emitted when a price estimate is generated. Used for analytics and ML training data pipeline. |
| `pricing.audit.events` | Produced | 6 | 3 | Immutable audit events for every pricing decision. Consumed by Audit Service and Data Warehouse pipeline. |
| `pricing.overrides.applied` | Produced | 3 | 3 | Emitted when an admin override is applied or removed. Consumed by Ops Dashboard and alerting. |
| `driver.location.updates` | Consumed | 24 | 3 | Driver GPS updates from Driver Tracking Service. Used to compute driver density per zone. |
| `booking.requests.created` | Consumed | 12 | 3 | New booking requests from Booking Service. Used to compute demand per zone. |
| `booking.requests.cancelled` | Consumed | 12 | 3 | Cancelled requests, used to adjust demand computation immediately. |
| `ml.forecast.predictions` | Consumed | 6 | 3 | Demand forecast outputs from ML Service (also consumed via gRPC for synchronous queries). |

### Event Schema Examples

#### `pricing.zone.updates` â Produced Event

```json
{
  "eventId": "evt_a1b2c3d4",
  "eventType": "ZONE_SURGE_UPDATED",
  "schemaVersion": "1.2",
  "timestamp": "2024-07-15T14:32:00.000Z",
  "payload": {
    "zoneId": "871f1a3a4ffffff",
    "city": "Berlin",
    "federalState": "BE",
    "previousMultiplier": 1.5,
    "newMultiplier": 1.8,
    "surgeLevel": "MEDIUM",
    "activeDrivers": 12,
    "pendingRequests": 28,
    "algorithmVersion": "v3.2.1",
    "manualOverride": false
  }
}
```

#### `pricing.audit.events` â Produced Event

```json
{
  "eventId": "aud_99f2a1c3",
  "eventType": "PRICE_ESTIMATE_GENERATED",
  "schemaVersion": "1.0",
  "timestamp": "2024-07-15T14:32:00.123Z",
  "payload": {
    "userId": "usr_abc123",
    "zoneId": "871f1a3a4ffffff",
    "baseFare": "9.50",
    "surgeMultiplier": 1.8,
    "finalFare": "17.10",
    "currency": "EUR",
    "pbefgCapEnforced": false,
    "manualOverrideActive": false,
    "algorithmVersion": "v3.2.1",
    "estimateToken": "eyJhbGciOiJIUzI1NiJ9..."
  }
}
```

### Kafka Consumer Group IDs

| Consumer Group | Topics Consumed |
|-----------------|----------------|
| `surge-pricing-driver-tracker` | `driver.location.updates` |
| `surge-pricing-booking-demand` | `booking.requests.created`, `booking.requests.cancelled` |
| `surge-pricing-ml-consumer` | `ml.forecast.predictions` |

---

## Database Schema Overview

The service uses **PostgreSQL 15** with **Flyway** migrations. All migrations are in `src/main/resources/db/migration/`.

### Tables

#### `pricing_zones`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | `UUID` | PK | Internal zone UUID |
| `h3_index` | `VARCHAR(20)` | UNIQUE, NOT NULL | H3 hexagonal cell index (resolution 7) |
| `city` | `VARCHAR(100)` | NOT NULL | City name (e.g., `Berlin`) |
| `federal_state` | `CHAR(2)` | NOT NULL | German federal state code (ISO 3166-2:DE) |
| `base_fare_eur` | `NUMERIC(10,2)` | NOT NULL | Base fare for this zone in EUR |
| `minimum_fare_eur` | `NUMERIC(10,2)` | NOT NULL | Minimum fare in EUR |
| `active` | `BOOLEAN` | NOT NULL, DEFAULT `true` | Whether zone is active |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | Zone creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL | Last configuration update |
| `created_by` | `VARCHAR(100)` | NOT NULL | Operator who created zone |

#### `pricing_history`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | `BIGSERIAL` | PK | Auto-increment ID |
| `zone_id` | `UUID` | FK â pricing_zones | Reference to zone |
| `surge_multiplier` | `NUMERIC(5,2)` | NOT NULL | Surge multiplier at this point in time |
| `active_drivers` | `INTEGER` | NOT NULL | Driver count used in computation |
| `pending_requests` | `INTEGER` | NOT NULL | Demand count used in computation |
| `supply_demand_ratio` | `NUMERIC(6,4)` | NOT NULL | Computed supply/demand ratio |
| `algorithm_version` | `VARCHAR(20)` | NOT NULL | Version of pricing algorithm |
| `weather_factor` | `NUMERIC(4,3)` | NULLABLE | Weather adjustment factor if applied |
| `ml_forecast_factor` | `NUMERIC(4,3)` | NULLABLE | ML forecast adjustment factor if applied |
| `pbefg_cap_enforced` | `BOOLEAN` | NOT NULL | Whether cap was applied |
| `recorded_at` | `TIMESTAMPTZ` | NOT NULL, INDEXED | Timestamp (partitioned by month) |

> â¡ **Note:** `pricing_history` is partitioned by `recorded_at` (monthly range partitions) for query performance. Partitions older than `AUDIT_LOG_RETENTION_DAYS` are archived to S3 via pg_partman.

#### `pricing_audit_log`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | `UUID` | PK | Audit entry UUID |
| `event_type` | `VARCHAR(50)` | NOT NULL | Event type (e.g., `ESTIMATE_GENERATED`, `CAP_ENFORCED`, `OVERRIDE_APPLIED`) |
| `zone_id` | `UUID` | FK â pricing_zones, NULLABLE | Zone reference if applicable |
| `user_id` | `VARCHAR(100)` | NULLABLE | User ID if applicable |
| `operator_id` | `VARCHAR(100)` | NULLABLE | Operator ID for admin actions |
| `payload` | `JSONB` | NOT NULL | Full event payload |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | Immutable audit timestamp |

> ð **Security:** The `pricing_audit_log` table grants `INSERT` only to the service database user. No `UPDATE` or `DELETE` privileges are granted, enforcing immutability at the database level.

#### `admin_overrides`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | `UUID` | PK | Override UUID |
| `zone_id` | `UUID` | FK â pricing_zones | Zone being overridden |
| `override_multiplier` | `NUMERIC(5,2)` | NOT NULL | Manual multiplier value |
| `reason` | `TEXT` | NOT NULL | Mandatory human-readable reason |
| `operator_id` | `VARCHAR(100)` | NOT NULL | Who applied the override |
| `applied_at` | `TIMESTAMPTZ` | NOT NULL | When override was applied |
| `expires_at` | `TIMESTAMPTZ` | NOT NULL | When override auto-expires |
| `removed_at` | `TIMESTAMPTZ` | NULLABLE | If manually removed before expiry |
| `removed_by` | `VARCHAR(100)` | NULLABLE | Who removed it |

### Entity Relationship Diagram

```
âââââââââââââââââââââââ         ââââââââââââââââââââââââ
â   pricing_zones     â         â   pricing_history    â
âââââââââââââââââââââââ         ââââââââââââââââââââââââ
â id (PK)             ââââââââââ¶â zone_id (FK)         â
â h3_index            â         â surge_multiplier      â
â city                â         â active_drivers        â
â federal_state       â         â pending_requests      â
â base_fare_eur       â         â algorithm_version     â
â minimum_fare_eur    â         â recorded_at (indexed) â
â active              â         ââââââââââââââââââââââââ
âââââââââââââââââââââââ
         â                      ââââââââââââââââââââââââ
         â                      â   admin_overrides    â
         âââââââââââââââââââââââ¶ââââââââââââââââââââââââ
                                â zone_id (FK)         â
                                â override_multiplier   â
                                â operator_id           â
                                â expires_at            â
                                ââââââââââââââââââââââââ
         â                      ââââââââââââââââââââââââ
         â                      â  pricing_audit_log   â
         âââââââââââââââââââââââ¶ââââââââââââââââââââââââ
                                â zone_id (FK, NULL)   â
                                â event_type            â
                                â user_id               â
                                â payload (JSONB)       â
                                ââââââââââââââââââââââââ
```

---

## Monitoring and Metrics

### Prometheus Metrics

All metrics are exposed at `/actuator/prometheus` and follow the Prometheus naming convention.

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `surge_pricing_multiplier` | Gauge | `zone_id`, `city`, `federal_state` | Current surge multiplier per zone |
| `surge_pricing_active_zones_total` | Gauge | `city` | Number of zones currently in surge (multiplier > 1.0) |
| `surge_pricing_estimate_requests_total` | Counter | `status`, `vehicle_class` | Total price estimate API requests |
| `surge_pricing_estimate_latency_seconds` | Histogram | `vehicle_class` | Latency of price estimate computation |
| `surge_pricing_zone_update_duration_seconds` | Histogram | `city` | Time to recompute surge for a zone |
| `surge_pricing_pbefg_cap_enforced_total` | Counter | `city`, `zone_id` | Number of times PBefG cap was applied |
| `surge_pricing_manual_overrides_active` | Gauge | `city` | Number of active manual overrides |
| `surge_pricing_ml_forecast_calls_total` | Counter | `status` | ML forecast gRPC calls (success/timeout/error) |
| `surge_pricing_ml_forecast_latency_seconds` | Histogram | â | ML forecast gRPC call latency |
| `surge_pricing_kafka_publish_total` | Counter | `topic`, `status` | Kafka events published |
| `surge_pricing_redis_cache_hits_total` | Counter | `cache_region` | Redis cache hits |
| `surge_pricing_redis_cache_misses_total` | Counter | `cache_region` | Redis cache misses |
| `surge_pricing_estimate_validations_total` | Counter | `result` | Price validation results (VALID/CHANGED/EXPIRED) |
| `surge_pricing_supply_demand_ratio` | Gauge | `zone_id`, `city` | Current supply/demand ratio per zone |

### Grafana Dashboard

A pre-built Grafana dashboard is maintained at:
```
Grafana â Dashboards â Platform â Surge Pricing Service
https://grafana.internal.company.de/d/surge-pricing-v2
```

Key panels include:
- ð¡ï¸ Real-time surge heat map overlay (per city)
- ð Surge multiplier time series (per zone)
- â¡ P50/P95/P99 API latency
- ð¨ PBefG cap enforcement rate
- ð¡ Kafka publish lag
- ð Redis cache hit rate
- ð» JVM metrics (heap, GC, threads)

### Alerting Rules

Alerts are defined in `helm/templates/prometheusrule.yaml` and route to the `#alerts-pricing` Slack channel.

| Alert Name | Condition | Severity | Description |
|------------|-----------|----------|-------------|
| `SurgePricingHighLatency` | `p99 latency > 200ms for 5m` | Warning | Estimate latency degradation |
| `SurgePricingCriticalLatency` | `p99 latency > 500ms for 2m` | Critical | SLA breach imminent |
| `SurgePricingPodDown` | `available_replicas < 2` | Critical | Pod count below HA threshold |
| `SurgePricingPbefgCapHighRate` | `cap_enforced rate > 10/min` | Warning | Unusually high cap enforcement â possible algorithm issue |
| `SurgePricingMLForecastDown` | `ml_forecast_errors > 50%` | Warning | ML service degraded, fallback active |
| `SurgePricingKafkaPublishFailing` | `kafka_publish_errors > 5/min` | Critical | Pricing updates not reaching consumers |
| `SurgePricingZoneUpdateStale` | `zone not updated for > 90s` | Warning | Zone recomputation stalled |
| `SurgePricingHighOverrideCount` | `manual_overrides_active > 10` | Warning | Unusually high manual override count |

### Log Format

All logs are structured JSON, shipped to **Elasticsearch** via Fluentd:

```json
{
  "timestamp": "2024-07-15T14:32:00.123Z",
  "level": "INFO",
  "service": "surge-pricing-service",
  "version": "3.2.1",
  "traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
  "spanId": "00f067aa0ba902b7",
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "logger": "com.company.pricing.SurgeMultiplierEngine",
  "message": "Zone surge updated",
  "zoneId": "871f1a3a4ffffff",
  "city": "Berlin",
  "previousMultiplier": 1.5,
  "newMultiplier": 1.8,
  "durationMs": 12
}
```

### Distributed Tracing

OpenTelemetry is configured for distributed tracing. Traces are exported to **Jaeger** (via OTLP):

```
Jaeger UI: https://jaeger.internal.company.de
Service Name: surge-pricing-service
```

Ensure `OTEL_EXPORTER_OTLP_ENDPOINT` is set in the deployment environment.

---

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for development standards, branch naming conventions, and PR requirements.

All PRs that touch pricing logic or PBefG compliance code require:
1. Review from at least one member of the `pricing-team`
2. Review from one member of the `legal-engineering` group
3. Passing PBefG compliance test suite (`-P pbefg-compliance`)

---

## Contact & Ownership

| Role | Contact |
|------|---------|
| **Team** | Platform Pricing Team |
| **Slack** | `#team-pricing` |
| **Email** | `pricing-team@company.de` |
| **On-Call** | PagerDuty â _Pricing Service On-Call_ rotation |
| **Runbook** | [Confluence Runbook](https://confluence.internal/pricing/runbook) |
| **Legal Contact** | `legal-engineering@company.de` (for PBefG questions) |

---

*Last updated: 2024-07-15 | Service Version: 3.2.1 | README Version: 2.4*

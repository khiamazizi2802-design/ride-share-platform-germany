# Analytics Service

A high-performance, scalable analytics microservice responsible for collecting, processing, and reporting on application metrics and user behavior data in real time. Built to integrate seamlessly into event-driven architectures, the Analytics Service provides actionable insights through a robust REST API, Kafka-based event consumption, and Prometheus-compatible metric exposition.

---

## Table of Contents

- [Features](#features)
- [Architecture Overview](#architecture-overview)
- [API Endpoints](#api-endpoints)
- [Environment Variables](#environment-variables)
- [Quick Start](#quick-start)
- [Kafka Event Types](#kafka-event-types)
- [Prometheus Metrics](#prometheus-metrics)
- [GDPR Compliance](#gdpr-compliance)

---

## Features

- **Metrics Collection** — Ingests raw events from multiple upstream services via Kafka topics and HTTP ingestion endpoints. Supports custom event schemas, batched ingestion, and deduplication logic to ensure data integrity.
- **Real-Time Analytics** — Processes incoming event streams with sub-second latency using in-memory aggregation windows. Enables live dashboards and alerting integrations through Server-Sent Events (SSE) and WebSocket push channels.
- **Reporting** — Generates scheduled and on-demand reports covering user activity, funnel conversion, retention cohorts, and custom KPI summaries. Reports are stored in object storage and delivered via signed URLs.
- **Data Export** — Supports full and incremental data exports in CSV, JSON, and Parquet formats. Exports can be triggered manually via the API or scheduled using cron expressions. Data can be pushed to S3-compatible storage or delivered as a direct download.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        External Systems                         │
│   Client Apps · Backend Services · Third-Party Integrations     │
└────────────────────────┬────────────────────────────────────────┘
                         │  HTTP Events / REST API Calls
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Analytics Service                          │
│                                                                 │
│  ┌─────────────┐   ┌──────────────┐   ┌─────────────────────┐  │
│  │  REST API   │   │ Kafka        │   │  Scheduler          │  │
│  │  (Fastify)  │   │ Consumer     │   │  (Cron Jobs)        │  │
│  └──────┬──────┘   └──────┬───────┘   └──────────┬──────────┘  │
│         │                 │                       │             │
│         └────────┬────────┘                       │             │
│                  ▼                                ▼             │
│  ┌───────────────────────────┐   ┌───────────────────────────┐  │
│  │   Event Processing        │   │   Report Engine           │  │
│  │   Pipeline                │   │   (Aggregation / SQL)     │  │
│  │   · Validation            │   │   · Funnel Analysis       │  │
│  │   · Enrichment            │   │   · Cohort Reports        │  │
│  │   · Deduplication         │   │   · KPI Summaries         │  │
│  └───────────┬───────────────┘   └──────────────┬────────────┘  │
│              │                                   │              │
│              ▼                                   ▼              │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                     Storage Layer                         │  │
│  │   PostgreSQL (events, reports)  ·  Redis (real-time agg)  │  │
│  │   S3-Compatible Object Store (exports, report artifacts)  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Observability: Prometheus Metrics · Structured Logging  │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Technology | Purpose |
|---|---|---|
| API Layer | Fastify (Node.js) | REST endpoint exposure, SSE streaming |
| Event Consumer | KafkaJS | Consumes upstream domain events |
| Event Pipeline | Custom middleware chain | Validation, enrichment, deduplication |
| Real-Time Aggregation | Redis + Lua scripts | In-memory windowed metric counters |
| Persistent Storage | PostgreSQL 15 | Event log, report metadata, user mappings |
| Report Engine | SQL + Node.js workers | Scheduled and on-demand report generation |
| Object Storage | AWS S3 / MinIO | Export files, report PDFs, audit logs |
| Scheduler | node-cron | Triggers report generation and data exports |
| Observability | Prometheus + Winston | Metrics exposition and structured logging |

---

## API Endpoints

All endpoints are prefixed with `/api/v1`. Authentication is required via Bearer token unless marked as public.

### Event Ingestion

| Method | Path | Description |
|---|---|---|
| POST | `/events` | Ingest a single analytics event. Accepts a JSON body conforming to the event schema. |
| POST | `/events/batch` | Ingest a batch of up to 500 events in a single request. |
| GET | `/events` | Query stored events with filters for time range, event type, user ID, and session ID. Supports pagination. |
| GET | `/events/:eventId` | Retrieve a single event record by its unique identifier. |

### Real-Time Analytics

| Method | Path | Description |
|---|---|---|
| GET | `/realtime/active-users` | Returns the count of unique active users in the last 5-minute sliding window. |
| GET | `/realtime/event-rate` | Returns events-per-second rate for a specified event type over the last 60 seconds. |
| GET | `/realtime/stream` | Server-Sent Events (SSE) endpoint. Subscribe to a live stream of aggregated metric updates. |

### Metrics & Aggregates

| Method | Path | Description |
|---|---|---|
| GET | `/metrics/summary` | Returns aggregated metrics (event counts, unique users, sessions) for a given time range and granularity (hour, day, week). |
| GET | `/metrics/funnel` | Computes funnel conversion rates across an ordered list of event types supplied in the query string. |
| GET | `/metrics/retention` | Returns a retention cohort table for users grouped by their first-seen date. |
| GET | `/metrics/top-events` | Lists the top N event types ranked by occurrence count within the specified window. |
| GET | `/metrics/user/:userId` | Returns a full activity summary for a specific user, including event timeline and session breakdown. |

### Reports

| Method | Path | Description |
|---|---|---|
| GET | `/reports` | Lists all generated reports with metadata (ID, type, status, created date, download URL). |
| POST | `/reports` | Creates an on-demand report. Accepts report type, time range, filters, and output format in the request body. |
| GET | `/reports/:reportId` | Retrieves the status and metadata of a specific report. |
| DELETE | `/reports/:reportId` | Deletes a report record and removes the associated artifact from object storage. |

### Data Export

| Method | Path | Description |
|---|---|---|
| POST | `/exports` | Initiates a data export job. Accepts filters, time range, output format (csv, json, parquet), and optional S3 destination. |
| GET | `/exports` | Lists all export jobs with their current status and download links. |
| GET | `/exports/:exportId` | Returns the status and signed download URL for a specific export job. |
| DELETE | `/exports/:exportId` | Cancels a pending export job or removes a completed export and its stored file. |
| GET | `/exports/:exportId/download` | Streams the export file directly to the client (for exports without an S3 destination). |

### Administration

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Health check endpoint. Returns service status and dependency connectivity. Public. |
| GET | `/health/ready` | Readiness probe. Returns 200 only when all dependencies (DB, Kafka, Redis) are connected. Public. |
| GET | `/admin/stats` | Internal service statistics including queue depths, consumer lag, and cache hit rates. Requires admin role. |
| DELETE | `/admin/users/:userId/data` | Permanently deletes all event data associated with a user ID. Used for GDPR erasure requests. Requires admin role. |
| GET | `/admin/users/:userId/data` | Returns all data held for a specific user in a portable format. Used for GDPR subject access requests. Requires admin role. |

---

## Environment Variables

Copy `.env.example` to `.env` and populate all required values before starting the service.

### Application

| Variable | Required | Default | Description |
|---|---|---|---|
| `NODE_ENV` | No | `development` | Runtime environment. Use `production` for live deployments. |
| `PORT` | No | `3000` | TCP port the HTTP server will listen on. |
| `LOG_LEVEL` | No | `info` | Logging verbosity. One of `debug`, `info`, `warn`, `error`. |
| `SERVICE_NAME` | No | `analytics-service` | Service name injected into log output and metric labels. |
| `API_BASE_PATH` | No | `/api/v1` | Base path prefix for all REST API routes. |

### Authentication

| Variable | Required | Default | Description |
|---|---|---|---|
| `JWT_SECRET` | Yes | — | Secret key used to verify incoming JWT Bearer tokens. Must be at least 32 characters. |
| `JWT_EXPIRY` | No | `1h` | Accepted JWT expiry window. Tokens issued beyond this window are rejected. |
| `ADMIN_API_KEY` | Yes | — | Static API key required for all `/admin/*` endpoints. Rotate regularly. |

### PostgreSQL

| Variable | Required | Default | Description |
|---|---|---|---|
| `POSTGRES_HOST` | Yes | — | Hostname or IP address of the PostgreSQL server. |
| `POSTGRES_PORT` | No | `5432` | PostgreSQL server port. |
| `POSTGRES_DB` | Yes | — | Name of the database to connect to. |
| `POSTGRES_USER` | Yes | — | PostgreSQL username. |
| `POSTGRES_PASSWORD` | Yes | — | PostgreSQL password. Store securely; never commit to version control. |
| `POSTGRES_POOL_MIN` | No | `2` | Minimum number of connections in the connection pool. |
| `POSTGRES_POOL_MAX` | No | `10` | Maximum number of connections in the connection pool. |
| `POSTGRES_SSL` | No | `false` | Enable SSL for PostgreSQL connections. Set to `true` in production. |

### Redis

| Variable | Required | Default | Description |
|---|---|---|---|
| `REDIS_HOST` | Yes | — | Hostname of the Redis instance. |
| `REDIS_PORT` | No | `6379` | Redis server port. |
| `REDIS_PASSWORD` | No | — | Redis AUTH password. Leave empty if authentication is not configured. |
| `REDIS_DB` | No | `0` | Redis logical database index. |
| `REDIS_KEY_PREFIX` | No | `analytics:` | Prefix applied to all Redis keys to avoid collisions in shared instances. |
| `REDIS_TTL_SECONDS` | No | `300` | Default TTL for real-time aggregation keys. |

### Kafka

| Variable | Required | Default | Description |
|---|---|---|---|
| `KAFKA_BROKERS` | Yes | — | Comma-separated list of Kafka broker addresses (e.g., `broker1:9092,broker2:9092`). |
| `KAFKA_CLIENT_ID` | No | `analytics-service` | Client ID reported to Kafka brokers. |
| `KAFKA_GROUP_ID` | No | `analytics-consumers` | Consumer group ID for this service's Kafka consumers. |
| `KAFKA_TOPICS` | No | See [Kafka Event Types](#kafka-event-types) | Comma-separated list of topics to subscribe to. Overrides defaults. |
| `KAFKA_AUTO_OFFSET_RESET` | No | `earliest` | Offset reset strategy for new consumer groups. One of `earliest` or `latest`. |
| `KAFKA_SESSION_TIMEOUT_MS` | No | `30000` | Kafka consumer session timeout in milliseconds. |
| `KAFKA_SASL_MECHANISM` | No | — | SASL mechanism for broker authentication. One of `plain`, `scram-sha-256`, `scram-sha-512`. |
| `KAFKA_SASL_USERNAME` | No | — | SASL username. Required when `KAFKA_SASL_MECHANISM` is set. |
| `KAFKA_SASL_PASSWORD` | No | — | SASL password. Required when `KAFKA_SASL_MECHANISM` is set. |
| `KAFKA_SSL` | No | `false` | Enable TLS encryption for Kafka connections. |

### Object Storage (S3 / MinIO)

| Variable | Required | Default | Description |
|---|---|---|---|
| `S3_ENDPOINT` | No | — | Custom S3 endpoint URL. Required when using MinIO or other S3-compatible providers. |
| `S3_REGION` | No | `us-east-1` | AWS region for S3 operations. |
| `S3_BUCKET` | Yes | — | Name of the S3 bucket used for exports and report artifacts. |
| `S3_ACCESS_KEY_ID` | Yes | — | AWS or MinIO access key ID. |
| `S3_SECRET_ACCESS_KEY` | Yes | — | AWS or MinIO secret access key. |
| `S3_SIGNED_URL_EXPIRY` | No | `3600` | Expiry time in seconds for presigned download URLs. |
| `S3_PATH_STYLE` | No | `false` | Use path-style S3 URLs. Set to `true` when using MinIO. |

### Feature Flags

| Variable | Required | Default | Description |
|---|---|---|---|
| `ENABLE_REALTIME_STREAM` | No | `true` | Enable the SSE real-time streaming endpoint. |
| `ENABLE_DATA_EXPORT` | No | `true` | Enable the data export API and background export workers. |
| `ENABLE_SCHEDULED_REPORTS` | No | `true` | Enable the cron-based report generation scheduler. |
| `GDPR_ANONYMIZATION_ENABLED` | No | `true` | Enable automatic anonymization of personal data fields on ingestion. |
| `DATA_RETENTION_DAYS` | No | `365` | Number of days raw event data is retained before automatic deletion. |

---

## Quick Start

The easiest way to run the Analytics Service locally is with Docker Compose. The configuration below starts the service along with all required dependencies.

### Prerequisites

- Docker Engine 24.0+
- Docker Compose v2.0+
- 4 GB RAM available for containers

### Step 1 — Clone the repository

```bash
git clone https://github.com/your-org/analytics-service.git
cd analytics-service
```

### Step 2 — Configure environment variables

```bash
cp .env.example .env
```

Open `.env` and fill in the required values. For local development the defaults in `docker-compose.yml` are pre-wired so only `JWT_SECRET` and `ADMIN_API_KEY` need to be set.

### Step 3 — Start all services

```bash
docker compose up -d
```

This will start the following containers:

| Container | Image | Port |
|---|---|---|
| `analytics-service` | Local build | 3000 |
| `postgres` | postgres:15-alpine | 5432 |
| `redis` | redis:7-alpine | 6379 |
| `zookeeper` | confluentinc/cp-zookeeper:7.5 | 2181 |
| `kafka` | confluentinc/cp-kafka:7.5 | 9092 |
| `minio` | minio/minio:latest | 9000, 9001 |

### Step 4 — Verify the service is running

```bash
curl http://localhost:3000/api/v1/health
```

Expected response:

```json
{
  "status": "ok",
  "version": "1.0.0",
  "dependencies": {
    "postgres": "connected",
    "redis": "connected",
    "kafka": "connected",
    "s3": "connected"
  }
}
```

### Step 5 — Run database migrations

```bash
docker compose exec analytics-service npm run migrate
```

### Step 6 — Ingest a test event

```bash
curl -X POST http://localhost:3000/api/v1/events \
  -H "Authorization: Bearer <your-jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "page_view",
    "userId": "user-123",
    "sessionId": "session-abc",
    "properties": {
      "path": "/dashboard",
      "referrer": "https://google.com"
    }
  }'
```

### Stopping the services

```bash
docker compose down
```

To remove all persisted volumes (clears database and object storage data):

```bash
docker compose down -v
```

### Sample docker-compose.yml

```yaml
version: '3.9'

services:
  analytics-service:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - '3000:3000'
    env_file:
      - .env
    environment:
      POSTGRES_HOST: postgres
      POSTGRES_DB: analytics
      POSTGRES_USER: analytics
      POSTGRES_PASSWORD: analytics_secret
      REDIS_HOST: redis
      KAFKA_BROKERS: kafka:29092
      S3_ENDPOINT: http://minio:9000
      S3_BUCKET: analytics-data
      S3_ACCESS_KEY_ID: minioadmin
      S3_SECRET_ACCESS_KEY: minioadmin
      S3_PATH_STYLE: 'true'
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      kafka:
        condition: service_healthy
    restart: unless-stopped

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: analytics
      POSTGRES_USER: analytics
      POSTGRES_PASSWORD: analytics_secret
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - '5432:5432'
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U analytics']
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - '6379:6379'
    volumes:
      - redis_data:/data
    healthcheck:
      test: ['CMD', 'redis-cli', 'ping']
      interval: 10s
      timeout: 3s
      retries: 5

  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    volumes:
      - zookeeper_data:/var/lib/zookeeper/data

  kafka:
    image: confluentinc/cp-kafka:7.5.0
    depends_on:
      - zookeeper
    ports:
      - '9092:9092'
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,PLAINTEXT_INTERNAL:PLAINTEXT
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092,PLAINTEXT_INTERNAL://kafka:29092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    volumes:
      - kafka_data:/var/lib/kafka/data
    healthcheck:
      test: ['CMD', 'kafka-broker-api-versions', '--bootstrap-server', 'localhost:9092']
      interval: 15s
      timeout: 10s
      retries: 5

  minio:
    image: minio/minio:latest
    command: server /data --console-address ':9001'
    ports:
      - '9000:9000'
      - '9001:9001'
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    volumes:
      - minio_data:/data

volumes:
  postgres_data:
  redis_data:
  kafka_data:
  zookeeper_data:
  minio_data:
```

---

## Kafka Event Types

The Analytics Service subscribes to the following Kafka topics and event types. All messages must be encoded as JSON. Each message envelope must include `eventType`, `timestamp` (ISO 8601), and `payload` fields.

### Topic: `user-events`

| Event Type | Description | Key Payload Fields |
|---|---|---|
| `user.registered` | Fired when a new user account is created. | `userId`, `email` (hashed), `registrationSource`, `country` |
| `user.login` | Fired on successful user authentication. | `userId`, `sessionId`, `ipAddress` (anonymized), `userAgent`, `loginMethod` |
| `user.logout` | Fired when a user session ends explicitly. | `userId`, `sessionId`, `sessionDurationSeconds` |
| `user.password_changed` | Fired when a user updates their password. | `userId`, `timestamp` |
| `user.account_deleted` | Fired when a user account is removed. Triggers GDPR erasure pipeline. | `userId`, `deletionReason` |
| `user.profile_updated` | Fired when a user updates their profile information. | `userId`, `changedFields` (field names only, not values) |

### Topic: `product-events`

| Event Type | Description | Key Payload Fields |
|---|---|---|
| `product.viewed` | Fired when a user views a product detail page. | `userId`, `sessionId`, `productId`, `categoryId`, `source` |
| `product.added_to_cart` | Fired when a product is added to the shopping cart. | `userId`, `sessionId`, `productId`, `quantity`, `priceAtTime` |
| `product.removed_from_cart` | Fired when a product is removed from the cart. | `userId`, `sessionId`, `productId`, `quantity` |
| `product.wishlisted` | Fired when a user adds a product to their wishlist. | `userId`, `productId` |
| `product.reviewed` | Fired when a user submits a product review. | `userId`, `productId`, `rating`, `reviewId` |
| `product.searched` | Fired when a user performs a product search. | `userId`, `sessionId`, `searchQuery` (hashed), `resultCount`, `filters` |

### Topic: `order-events`

| Event Type | Description | Key Payload Fields |
|---|---|---|
| `order.created` | Fired when a new order is placed. | `userId`, `orderId`, `totalAmount`, `currency`, `itemCount`, `paymentMethod` |
| `order.paid` | Fired when payment for an order is confirmed. | `userId`, `orderId`, `paymentGateway`, `transactionId` (hashed) |
| `order.shipped` | Fired when an order is dispatched for delivery. | `orderId`, `carrier`, `estimatedDeliveryDate` |
| `order.delivered` | Fired when an order delivery is confirmed. | `orderId`, `userId`, `deliveryDate`, `actualDeliveryDays` |
| `order.cancelled` | Fired when an order is cancelled. | `userId`, `orderId`, `cancellationReason`, `refundAmount` |
| `order.refunded` | Fired when a refund is processed for an order. | `userId`, `orderId`, `refundAmount`, `currency`, `refundMethod` |

### Topic: `page-events`

| Event Type | Description | Key Payload Fields |
|---|---|---|
| `page.viewed` | Fired on every page navigation event. | `userId`, `sessionId`, `path`, `referrer`, `loadTimeMs`, `deviceType` |
| `page.exit` | Fired when the user leaves a page. | `userId`, `sessionId`, `path`, `timeOnPageSeconds` |
| `page.error` | Fired when a client-side error occurs on a page. | `sessionId`, `path`, `errorCode`, `errorMessage` (sanitized) |

### Topic: `feature-events`

| Event Type | Description | Key Payload Fields |
|---|---|---|
| `feature.flag_evaluated` | Fired when a feature flag is evaluated for a user. | `userId`, `flagKey`, `flagValue`, `evaluationReason` |
| `feature.experiment_assigned` | Fired when a user is assigned to an A/B experiment variant. | `userId`, `experimentId`, `variantId`, `assignmentReason` |
| `feature.experiment_converted` | Fired when a user completes the conversion goal for an experiment. | `userId`, `experimentId`, `variantId`, `goalId` |

### Topic: `system-events`

| Event Type | Description | Key Payload Fields |
|---|---|---|
| `system.health_check` | Periodic heartbeat event from upstream services. Consumed for uptime tracking. | `serviceId`, `version`, `status`, `timestamp` |
| `system.deployment` | Fired when a new service version is deployed. Used for annotation in metric graphs. | `serviceId`, `version`, `environment`, `deployedBy` |
| `system.error` | Fired when an unhandled error occurs in an upstream service. | `serviceId`, `errorCode`, `severity`, `message` (sanitized) |

---

## Prometheus Metrics

The Analytics Service exposes Prometheus-compatible metrics at `GET /metrics` (plain text format) on port `9090` by default. This port is configurable via the `METRICS_PORT` environment variable.

All metrics are prefixed with `analytics_` and include the default labels `service="analytics-service"` and `env` (value of `NODE_ENV`).

### HTTP API Metrics

| Metric Name | Type | Description | Labels |
|---|---|---|---|
| `analytics_http_requests_total` | Counter | Total number of HTTP requests received. | `method`, `route`, `status_code` |
| `analytics_http_request_duration_seconds` | Histogram | Duration of HTTP request processing in seconds. Buckets: 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5. | `method`, `route`, `status_code` |
| `analytics_http_request_size_bytes` | Histogram | Size of incoming HTTP request bodies in bytes. | `method`, `route` |
| `analytics_http_response_size_bytes` | Histogram | Size of outgoing HTTP response bodies in bytes. | `method`, `route` |
| `analytics_http_active_connections` | Gauge | Number of currently open HTTP connections. | — |
| `analytics_sse_active_subscribers` | Gauge | Number of clients currently subscribed to the SSE real-time stream. | — |

### Event Processing Metrics

| Metric Name | Type | Description | Labels |
|---|---|---|---|
| `analytics_events_ingested_total` | Counter | Total number of events successfully ingested (via HTTP or Kafka). | `source` (`http`\|`kafka`), `event_type` |
| `analytics_events_rejected_total` | Counter | Total number of events rejected due to validation failures or schema errors. | `source`, `rejection_reason` |
| `analytics_events_deduplicated_total` | Counter | Total number of duplicate events detected and discarded. | `source`, `event_type` |
| `analytics_event_processing_duration_seconds` | Histogram | Time taken to process and persist a single event from receipt to storage. | `event_type` |
| `analytics_event_pipeline_queue_depth` | Gauge | Current number of events waiting in the internal processing queue. | — |

### Kafka Consumer Metrics

| Metric Name | Type | Description | Labels |
|---|---|---|---|
| `analytics_kafka_messages_consumed_total` | Counter | Total number of Kafka messages consumed across all topics. | `topic`, `partition` |
| `analytics_kafka_consumer_lag` | Gauge | Current consumer group lag (number of unconsumed messages) per topic partition. | `topic`, `partition` |
| `analytics_kafka_consumer_errors_total` | Counter | Total number of errors encountered during Kafka message consumption. | `topic`, `error_type` |
| `analytics_kafka_message_processing_duration_seconds` | Histogram | Time taken to process a single Kafka message from receipt to acknowledgement. | `topic`, `event_type` |
| `analytics_kafka_rebalances_total` | Counter | Total number of consumer group rebalance events. | `topic` |

### Storage Metrics

| Metric Name | Type | Description | Labels |
|---|---|---|---|
| `analytics_postgres_query_duration_seconds` | Histogram | Duration of PostgreSQL query execution in seconds. | `operation` (`select`\|`insert`\|`update`\|`delete`) |
| `analytics_postgres_pool_connections_active` | Gauge | Number of active connections currently in use from the pool. | — |
| `analytics_postgres_pool_connections_idle` | Gauge | Number of idle connections in the pool. | — |
| `analytics_postgres_errors_total` | Counter | Total number of PostgreSQL errors encountered. | `error_code` |
| `analytics_redis_operations_total` | Counter | Total number of Redis commands executed. | `command`, `status` (`hit`\|`miss`\|`error`) |
| `analytics_redis_operation_duration_seconds` | Histogram | Duration of Redis command execution. | `command` |
| `analytics_redis_cache_hit_ratio` | Gauge | Current cache hit ratio for real-time aggregation lookups. | — |

### Report and Export Metrics

| Metric Name | Type | Description | Labels |
|---|---|---|---|
| `analytics_reports_generated_total` | Counter | Total number of reports successfully generated. | `report_type` |
| `analytics_reports_failed_total` | Counter | Total number of report generation failures. | `report_type`, `failure_reason` |
| `analytics_report_generation_duration_seconds` | Histogram | Time taken to generate a report from trigger to completion. | `report_type` |
| `analytics_exports_completed_total` | Counter | Total number of data export jobs completed successfully. | `format`, `destination` (`s3`\|`download`) |
| `analytics_exports_failed_total` | Counter | Total number of data export jobs that failed. | `format`, `failure_reason` |
| `analytics_export_file_size_bytes` | Histogram | Size of completed export files in bytes. | `format` |

### Process and Runtime Metrics

Standard Node.js process metrics are exposed via the `prom-client` default metrics collection:

| Metric Name | Type | Description |
|---|---|---|
| `process_cpu_seconds_total` | Counter | Total user and system CPU time consumed by the Node.js process. |
| `process_resident_memory_bytes` | Gauge | Resident memory size of the Node.js process in bytes. |
| `nodejs_heap_size_total_bytes` | Gauge | Total heap size allocated by V8. |
| `nodejs_heap_size_used_bytes` | Gauge | Heap memory actively used by V8. |
| `nodejs_external_memory_bytes` | Gauge | Memory used by C++ objects bound to JavaScript objects. |
| `nodejs_event_loop_lag_seconds` | Gauge | Current Node.js event loop lag in seconds. |
| `nodejs_active_handles_total` | Gauge | Number of active libuv handles. |

### Scrape Configuration

Add the following job to your `prometheus.yml` to scrape the Analytics Service:

```yaml
scrape_configs:
  - job_name: 'analytics-service'
    static_configs:
      - targets: ['analytics-service:9090']
    scrape_interval: 15s
    scrape_timeout: 10s
    metrics_path: /metrics
```

---

## GDPR Compliance

The Analytics Service is designed with privacy by default. The following measures are implemented to support compliance with the General Data Protection Regulation (GDPR) and similar privacy regulations (CCPA, PECR).

### Data Minimization and Anonymization

- **IP Address Anonymization** — All IP addresses are anonymized at ingestion time before storage. For IPv4 addresses the last octet is zeroed (e.g., `192.168.1.100` → `192.168.1.0`). For IPv6 the last 80 bits are zeroed. The original IP address is never persisted.
- **Email Hashing** — Email addresses are one-way hashed (SHA-256 with a service-level salt) before storage. The plaintext email address is never written to disk or logs.
- **Search Query Hashing** — Free-text search queries are hashed before storage to prevent personal information from being captured in query strings.
- **User Agent Normalization** — Raw User-Agent strings are parsed and only structured device and browser categories are stored. The raw string is discarded.
- **Field Allowlisting** — The event ingestion pipeline operates on an allowlist of permitted payload fields. Any fields not on the allowlist are silently stripped before the event reaches the processing pipeline.

### Data Retention

- Raw event data is retained for the number of days specified by `DATA_RETENTION_DAYS` (default 365 days). A nightly scheduled job automatically purges records older than this threshold.
- Aggregated and anonymized report data may be retained for longer periods as configured per report type.
- Export files stored in S3 are subject to lifecycle policies configured on the bucket. A recommended policy of 30-day expiry for export artifacts is documented in `docs/s3-lifecycle-policy.json`.

### Right to Erasure (Article 17)

When a `user.account_deleted` Kafka event is received, or when the `DELETE /admin/users/:userId/data` API endpoint is called, the following erasure pipeline is triggered:

1. All event records where `userId` matches the subject are permanently deleted from PostgreSQL.
2. All Redis keys associated with the user are flushed.
3. Any report artifacts or export files that exclusively contain data for the subject user are deleted from object storage.
4. An erasure confirmation record (containing only a hashed user identifier and the timestamp of erasure) is written to the audit log for compliance purposes.
5. The erasure is idempotent — repeated calls for the same user ID produce no error and confirm the absent data state.

Erasure jobs are processed asynchronously. The API responds with `202 Accepted` and a job ID that can be polled for completion status.

### Right of Access (Article 15) and Data Portability (Article 20)

The `GET /admin/users/:userId/data` endpoint returns all data held for a specific user in a structured, machine-readable JSON format suitable for subject access request (SAR) responses. The response includes:

- A complete list of all events attributed to the user.
- Session summaries and activity timelines.
- Any user-level aggregates stored in the analytics database.

Requests to this endpoint are logged to the audit trail.

### Consent and Legal Basis

- The Analytics Service does not manage consent records directly. It is the responsibility of the upstream application to verify that valid consent or a lawful legal basis exists before emitting analytics events for a user.
- Events received for users who have withdrawn consent should not be emitted by upstream services. For belt-and-suspenders protection, a consent flag (`consentGiven: false`) in the event payload will cause the ingestion pipeline to discard the event and increment the `analytics_events_rejected_total` counter with `rejection_reason="consent_withdrawn"`.

### Audit Logging

- All administrative actions (erasure requests, subject access requests, configuration changes) are written to a tamper-evident audit log stored separately from the event database.
- Audit log entries include: timestamp, action type, target resource, actor identity (API key hash), and outcome.
- Audit logs are retained for a minimum of 3 years regardless of the `DATA_RETENTION_DAYS` setting.

### Data Processing Agreements

- If deploying to a cloud provider (AWS, GCP, Azure), ensure a Data Processing Agreement (DPA) is in place with the provider.
- When using Kafka as a managed service (e.g., Confluent Cloud), review the provider's DPA to confirm in-transit data handling meets your regulatory requirements.
- All data in transit is encrypted using TLS 1.2 or higher. All data at rest in PostgreSQL and S3 should be encrypted using AES-256. Encryption at rest is not managed by this service and must be configured at the infrastructure level.

### Privacy by Design Checklist

- [x] No plaintext PII stored in the event database
- [x] IP anonymization applied at ingestion boundary
- [x] Email and search query hashing enforced in pipeline
- [x] Data retention policy enforced by automated purge job
- [x] Right to erasure pipeline implemented and tested
- [x] Subject access request endpoint available
- [x] Consent signal honored at ingestion
- [x] All admin actions written to audit log
- [x] TLS enforced for all inter-service communication in production
- [ ] Encryption at rest — configure at infrastructure level (see ops runbook)
- [ ] DPA in place with all third-party data processors — legal team responsibility

---

## Contributing

Please read `CONTRIBUTING.md` for details on our branching strategy, commit message conventions, and pull request process.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.

## Support

For internal support, open an issue in the repository or contact the Platform Engineering team via the `#analytics-service` Slack channel.

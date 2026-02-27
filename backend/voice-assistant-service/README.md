# Voice Assistant Service

A production-ready microservice for voice assistant functionality in the RideShare platform. This service handles real-time voice sessions, speech-to-text transcription, natural language understanding (NLU), intent recognition, and voice command execution.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [API Endpoints](#api-endpoints)
- [WebSocket Protocol](#websocket-protocol)
- [Configuration](#configuration)
- [Deployment](#deployment)
- [Monitoring](#monitoring)

---

## Overview

The Voice Assistant Service enables drivers and passengers to interact with the RideShare platform using voice commands. Key features include:

- **Real-time voice processing** via WebSocket connections
- **Speech-to-text** transcription (Google Cloud STT, AWS Transcribe)
- **Natural Language Understanding** (Dialogflow, Wit.ai)
- **Intent recognition and command execution**
- **Session management** with Redis caching
- **Event streaming** via Apache Kafka
- **JWT-based authentication**
- **Prometheus metrics and observability**

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   Voice Assistant Service                    │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  HTTP    │  │WebSocket │  │   STT    │  │   NLU    │   │
│  │ Handler  │  │ Handler  │  │ Provider │  │ Provider │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│        │             │              │              │         │
│  ┌─────▼─────────────▼──────────────▼──────────────▼────┐  │
│  │              Service Layer (Business Logic)           │  │
│  └──────────────────────────────────────────────────────┘  │
│        │             │              │              │         │
│  ┌─────▼──┐    ┌─────▼──┐   ┌──────▼──┐   ┌──────▼──┐    │
│  │ Redis  │    │Postgres│   │  Kafka  │   │Metrics  │    │
│  └────────┘    └────────┘   └─────────┘   └─────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

## Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Apache Kafka 3.5+
- Docker & Kubernetes
- Google Cloud Speech-to-Text API credentials
- Dialogflow credentials

---

## Getting Started

### Local Development

```bash
# Clone the repository
git clone https://github.com/rideshare/voice-assistant-service.git
cd voice-assistant-service

# Copy environment variables
cp .env.example .env

# Run database migrations
psql -U postgres -d voice_assistant -f migrations/001_initial_schema.sql

# Install dependencies
go mod download

# Run the service
go run ./cmd/server

# Run tests
go test ./... -v -cover

# Run with Docker
docker build -t voice-assistant-service:latest .
docker run -p 8080:8080 -p 9090:9090 --env-file .env voice-assistant-service:latest
```

### Docker Compose

```bash
docker-compose up -d
```

---

## API Endpoints

### Base URL

```
https://api.rideshare.com/v1/voice
```

### Authentication

All endpoints require a valid JWT token in the Authorization header:

```
Authorization: Bearer <jwt_token>
```

---

### Health Endpoints

#### GET /health/live
Liveness probe for Kubernetes.

**Response 200:**
```json
{
  "status": "alive",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0"
}
```

#### GET /health/ready
Readiness probe for Kubernetes.

**Response 200:**
```json
{
  "status": "ready",
  "timestamp": "2024-01-15T10:30:00Z",
  "checks": {
    "database": "healthy",
    "redis": "healthy",
    "kafka": "healthy"
  }
}
```

---

### Session Endpoints

#### POST /v1/voice/sessions
Create a new voice session.

**Request:**
```json
{
  "ride_id": "550e8400-e29b-41d4-a716-446655440000",
  "device_id": "device_abc123",
  "language_code": "en-US",
  "audio_encoding": "LINEAR16",
  "sample_rate_hertz": 16000,
  "audio_channels": 1,
  "speaker_role": "driver",
  "nlu_provider": "dialogflow",
  "stt_provider": "google",
  "metadata": {
    "app_version": "3.2.1",
    "platform": "ios"
  }
}
```

**Response 201:**
```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "session_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "status": "initializing",
  "websocket_url": "wss://api.rideshare.com/v1/voice/sessions/7c9e6679/stream",
  "expires_at": "2024-01-15T11:00:00Z",
  "created_at": "2024-01-15T10:30:00Z"
}
```

#### GET /v1/voice/sessions/{session_id}
Get session details.

**Response 200:**
```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "user_id": "user_123",
  "ride_id": "550e8400-e29b-41d4-a716-446655440000",
  "device_id": "device_abc123",
  "status": "active",
  "speaker_role": "driver",
  "language_code": "en-US",
  "total_commands": 5,
  "successful_commands": 4,
  "failed_commands": 1,
  "started_at": "2024-01-15T10:30:00Z",
  "last_activity_at": "2024-01-15T10:45:00Z"
}
```

#### GET /v1/voice/sessions
List user sessions with pagination.

**Query Parameters:**
- `page` (int, default: 1)
- `page_size` (int, default: 20, max: 100)
- `status` (string: active|completed|failed)
- `ride_id` (UUID)
- `from_date` (ISO8601)
- `to_date` (ISO8601)

**Response 200:**
```json
{
  "sessions": [...],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

#### PUT /v1/voice/sessions/{session_id}/status
Update session status (pause/resume/end).

**Request:**
```json
{
  "status": "paused",
  "reason": "user_request"
}
```

**Response 200:**
```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "status": "paused",
  "updated_at": "2024-01-15T10:35:00Z"
}
```

#### DELETE /v1/voice/sessions/{session_id}
Terminate a voice session.

**Response 200:**
```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "status": "terminated",
  "summary": {
    "total_commands": 5,
    "duration_seconds": 900
  },
  "ended_at": "2024-01-15T10:45:00Z"
}
```

---

### Transcript Endpoints

#### GET /v1/voice/sessions/{session_id}/transcripts
Get transcripts for a session.

**Query Parameters:**
- `page` (int, default: 1)
- `page_size` (int, default: 50)
- `is_final` (bool)
- `speaker_role` (string)

**Response 200:**
```json
{
  "transcripts": [
    {
      "id": "a3bb189e-8bf9-3888-9912-ace4e6543002",
      "session_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "sequence_number": 1,
      "raw_text": "take me to the airport",
      "confidence_score": 0.9823,
      "is_final": true,
      "audio_start_ms": 0,
      "audio_end_ms": 1850,
      "audio_duration_ms": 1850,
      "word_count": 5,
      "status": "completed",
      "created_at": "2024-01-15T10:31:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 50,
    "total": 12
  }
}
```

#### GET /v1/voice/sessions/{session_id}/transcripts/{transcript_id}
Get a specific transcript with full details.

**Response 200:**
```json
{
  "id": "a3bb189e-8bf9-3888-9912-ace4e6543002",
  "session_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "raw_text": "take me to the airport",
  "normalized_text": "take me to the airport",
  "confidence_score": 0.9823,
  "detected_language": "en-US",
  "words_data": [
    {"word": "take", "start_ms": 0, "end_ms": 250, "confidence": 0.99},
    {"word": "me", "start_ms": 260, "end_ms": 450, "confidence": 0.99},
    {"word": "to", "start_ms": 460, "end_ms": 600, "confidence": 0.98},
    {"word": "the", "start_ms": 610, "end_ms": 750, "confidence": 0.99},
    {"word": "airport", "start_ms": 760, "end_ms": 1850, "confidence": 0.97}
  ],
  "alternatives": [
    {"text": "take me to the air port", "confidence": 0.75}
  ],
  "status": "completed",
  "created_at": "2024-01-15T10:31:00Z"
}
```

---

### Intent Endpoints

#### GET /v1/voice/sessions/{session_id}/intents
Get recognized intents for a session.

**Response 200:**
```json
{
  "intents": [
    {
      "id": "b5cc2a0f-9c8d-4ab3-8e21-bd65f74321d1",
      "session_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "intent_name": "navigation.go_to_destination",
      "category": "navigation",
      "confidence_score": 0.9654,
      "entities": [
        {
          "name": "location",
          "value": "airport",
          "type": "geo-city",
          "confidence": 0.92
        }
      ],
      "parameters": {
        "location": "airport"
      },
      "fulfillment_text": "Navigating to the nearest airport.",
      "requires_confirmation": false,
      "created_at": "2024-01-15T10:31:05Z"
    }
  ]
}
```

#### POST /v1/voice/intents/analyze
Analyze text for intent without audio.

**Request:**
```json
{
  "text": "take me to the airport",
  "session_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "language_code": "en-US",
  "contexts": []
}
```

**Response 200:**
```json
{
  "intent_name": "navigation.go_to_destination",
  "display_name": "Navigate to Destination",
  "category": "navigation",
  "confidence_score": 0.9654,
  "entities": [
    {
      "name": "location",
      "value": "airport",
      "type": "geo-city"
    }
  ],
  "parameters": {
    "location": "airport"
  },
  "fulfillment_text": "Navigating to the nearest airport.",
  "alternatives": [
    {
      "intent_name": "navigation.find_nearby",
      "confidence": 0.12
    }
  ]
}
```

---

### Voice Command Endpoints

#### GET /v1/voice/sessions/{session_id}/commands
Get voice commands for a session.

**Query Parameters:**
- `page` (int)
- `page_size` (int)
- `status` (string: received|processing|executed|failed)
- `command_type` (string)
- `is_emergency` (bool)

**Response 200:**
```json
{
  "commands": [
    {
      "id": "d8e3f9a1-2c4b-5678-90ab-cdef01234567",
      "session_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "command_type": "navigation",
      "action": "NAVIGATE_TO",
      "status": "executed",
      "priority": 8,
      "is_emergency": false,
      "raw_command": "take me to the airport",
      "parameters": {
        "location": "airport",
        "coordinates": {"lat": 40.6413, "lng": -73.7781}
      },
      "response_text": "Navigating to JFK International Airport. ETA: 25 minutes.",
      "response_time_ms": 245,
      "execution_time_ms": 1200,
      "created_at": "2024-01-15T10:31:05Z",
      "executed_at": "2024-01-15T10:31:06Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 5
  }
}
```

#### GET /v1/voice/commands/{command_id}
Get a specific voice command.

**Response 200:**
```json
{
  "id": "d8e3f9a1-2c4b-5678-90ab-cdef01234567",
  "session_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "command_type": "navigation",
  "action": "NAVIGATE_TO",
  "status": "executed",
  "raw_command": "take me to the airport",
  "parameters": {
    "location": "airport"
  },
  "response_text": "Navigating to JFK International Airport.",
  "response_data": {
    "destination": "JFK International Airport",
    "eta_minutes": 25,
    "distance_km": 18.5
  },
  "execution_time_ms": 1200,
  "kafka_topic": "voice-commands",
  "created_at": "2024-01-15T10:31:05Z",
  "executed_at": "2024-01-15T10:31:06Z"
}
```

#### POST /v1/voice/commands/{command_id}/confirm
Confirm a command that requires confirmation.

**Request:**
```json
{
  "confirmed": true,
  "reason": "user_approved"
}
```

**Response 200:**
```json
{
  "id": "d8e3f9a1-2c4b-5678-90ab-cdef01234567",
  "status": "processing",
  "confirmed_at": "2024-01-15T10:32:00Z"
}
```

#### POST /v1/voice/commands/{command_id}/cancel
Cancel a pending or processing command.

**Response 200:**
```json
{
  "id": "d8e3f9a1-2c4b-5678-90ab-cdef01234567",
  "status": "cancelled",
  "cancelled_at": "2024-01-15T10:32:00Z"
}
```

---

### Preferences Endpoints

#### GET /v1/voice/preferences
Get user voice preferences.

**Response 200:**
```json
{
  "user_id": "user_123",
  "preferred_language": "en-US",
  "voice_activation_word": "hey rideshare",
  "is_voice_enabled": true,
  "auto_execute_commands": false,
  "confirmation_threshold": 0.8,
  "speech_rate": 1.0,
  "voice_gender": "neutral",
  "noise_cancellation": true,
  "wake_word_sensitivity": 0.5,
  "notification_voice": true,
  "updated_at": "2024-01-15T10:00:00Z"
}
```

#### PUT /v1/voice/preferences
Update user voice preferences.

**Request:**
```json
{
  "preferred_language": "es-ES",
  "auto_execute_commands": true,
  "confirmation_threshold": 0.9,
  "noise_cancellation": true,
  "speech_rate": 1.2
}
```

**Response 200:**
```json
{
  "user_id": "user_123",
  "preferred_language": "es-ES",
  "auto_execute_commands": true,
  "updated_at": "2024-01-15T10:35:00Z"
}
```

---

### Analytics Endpoints

#### GET /v1/voice/analytics/summary
Get voice usage analytics summary.

**Query Parameters:**
- `from_date` (ISO8601 date)
- `to_date` (ISO8601 date)
- `granularity` (hour|day|week|month)

**Response 200:**
```json
{
  "user_id": "user_123",
  "period": {
    "from": "2024-01-01",
    "to": "2024-01-15"
  },
  "summary": {
    "total_sessions": 45,
    "successful_sessions": 42,
    "total_commands": 230,
    "successful_commands": 218,
    "avg_confidence_score": 0.8934,
    "avg_response_time_ms": 312,
    "avg_session_duration_seconds": 480
  },
  "top_intents": [
    {"intent": "navigation.go_to_destination", "count": 85},
    {"intent": "ride_control.adjust_temperature", "count": 42},
    {"intent": "booking.cancel_ride", "count": 18}
  ],
  "intent_distribution": {
    "navigation": 0.45,
    "ride_control": 0.25,
    "booking": 0.15,
    "information": 0.10,
    "other": 0.05
  }
}
```

---

### Command Templates Endpoints

#### GET /v1/voice/commands/templates
Get available voice command templates.

**Query Parameters:**
- `category` (string)
- `language_code` (string)
- `is_active` (bool)

**Response 200:**
```json
{
  "templates": [
    {
      "id": "template_123",
      "intent_name": "navigation.go_to_destination",
      "display_name": "Navigate to Destination",
      "category": "navigation",
      "description": "Navigate to a specified destination",
      "example_phrases": [
        "take me to {location}",
        "navigate to {location}",
        "go to {location}"
      ],
      "required_parameters": ["location"],
      "optional_parameters": ["via", "avoid"],
      "requires_confirmation": false,
      "is_emergency": false
    }
  ]
}
```

---

## WebSocket Protocol

### Connection

```
wss://api.rideshare.com/v1/voice/sessions/{session_id}/stream
```

**Headers:**
```
Authorization: Bearer <jwt_token>
X-Session-Token: <session_token>
```

### Message Format

All WebSocket messages are JSON-encoded:

```json
{
  "type": "<message_type>",
  "id": "<message_uuid>",
  "timestamp": "2024-01-15T10:30:00Z",
  "payload": {}
}
```

### Client → Server Messages

#### audio_chunk
Stream audio data (base64 encoded):
```json
{
  "type": "audio_chunk",
  "id": "msg_001",
  "payload": {
    "audio_data": "<base64_encoded_audio>",
    "sequence": 1,
    "is_last": false
  }
}
```

#### text_input
Send text input directly:
```json
{
  "type": "text_input",
  "id": "msg_002",
  "payload": {
    "text": "take me to the airport",
    "language_code": "en-US"
  }
}
```

#### confirm_command
Confirm a pending command:
```json
{
  "type": "confirm_command",
  "id": "msg_003",
  "payload": {
    "command_id": "d8e3f9a1-2c4b-5678-90ab-cdef01234567",
    "confirmed": true
  }
}
```

#### ping
```json
{"type": "ping", "id": "msg_004", "payload": {}}
```

### Server → Client Messages

#### transcript_interim
Partial transcript:
```json
{
  "type": "transcript_interim",
  "payload": {
    "transcript_id": "trans_001",
    "text": "take me to",
    "confidence": 0.85,
    "is_final": false
  }
}
```

#### transcript_final
Final transcript:
```json
{
  "type": "transcript_final",
  "payload": {
    "transcript_id": "trans_001",
    "text": "take me to the airport",
    "confidence": 0.9823,
    "is_final": true
  }
}
```

#### intent_recognized
Intent detected:
```json
{
  "type": "intent_recognized",
  "payload": {
    "intent_id": "intent_001",
    "intent_name": "navigation.go_to_destination",
    "confidence": 0.9654,
    "entities": [{"name": "location", "value": "airport"}],
    "requires_confirmation": false
  }
}
```

#### command_executed
Command executed successfully:
```json
{
  "type": "command_executed",
  "payload": {
    "command_id": "cmd_001",
    "action": "NAVIGATE_TO",
    "response_text": "Navigating to JFK Airport. ETA: 25 minutes.",
    "response_data": {}
  }
}
```

#### command_confirmation_required
Command needs user confirmation:
```json
{
  "type": "command_confirmation_required",
  "payload": {
    "command_id": "cmd_002",
    "action": "CANCEL_RIDE",
    "prompt": "Are you sure you want to cancel your ride?"
  }
}
```

#### error
Error message:
```json
{
  "type": "error",
  "payload": {
    "code": "INTENT_CONFIDENCE_LOW",
    "message": "Could not understand the command. Please try again.",
    "recoverable": true
  }
}
```

#### pong
```json
{"type": "pong", "payload": {"server_time": "2024-01-15T10:30:00Z"}}
```

---

## Configuration

| Variable | Description | Default |
|---|---|---|
| `APP_PORT` | HTTP server port | `8080` |
| `METRICS_PORT` | Prometheus metrics port | `9090` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_NAME` | Database name | `voice_assistant` |
| `DB_USER` | Database user | `voice_assistant_app` |
| `DB_PASSWORD` | Database password | - |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `REDIS_PASSWORD` | Redis password | - |
| `KAFKA_BROKERS` | Kafka broker addresses | `localhost:9092` |
| `JWT_SECRET` | JWT signing secret | - |
| `GOOGLE_STT_CREDENTIALS` | Google STT JSON credentials path | - |
| `DIALOGFLOW_CREDENTIALS` | Dialogflow JSON credentials path | - |
| `NLU_CONFIDENCE_THRESHOLD` | Minimum intent confidence | `0.75` |
| `SESSION_MAX_DURATION` | Max session duration (seconds) | `1800` |
| `SESSION_IDLE_TIMEOUT` | Session idle timeout (seconds) | `300` |

---

## Deployment

### Kubernetes

```bash
# Create namespace
kubectl create namespace rideshare

# Create secrets
kubectl create secret generic voice-assistant-service-secrets \
  --from-literal=DB_USER=voice_assistant_app \
  --from-literal=DB_PASSWORD=your_password \
  --from-literal=REDIS_PASSWORD=your_redis_password \
  --from-literal=JWT_SECRET=your_jwt_secret \
  --from-literal=GOOGLE_STT_CREDENTIALS="$(cat google-stt-creds.json)" \
  --from-literal=DIALOGFLOW_CREDENTIALS="$(cat dialogflow-creds.json)" \
  --namespace rideshare

# Apply configurations
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/hpa.yaml

# Verify deployment
kubectl rollout status deployment/voice-assistant-service -n rideshare
kubectl get pods -n rideshare -l app=voice-assistant-service
```

---

## Monitoring

### Prometheus Metrics

| Metric | Type | Description |
|---|---|---|
| `voice_sessions_total` | Counter | Total voice sessions created |
| `voice_active_sessions_total` | Gauge | Currently active sessions |
| `voice_websocket_connections_active` | Gauge | Active WebSocket connections |
| `voice_commands_total` | Counter | Total voice commands processed |
| `voice_commands_success_total` | Counter | Successfully executed commands |
| `voice_commands_failed_total` | Counter | Failed commands |
| `voice_stt_duration_seconds` | Histogram | STT processing duration |
| `voice_nlu_duration_seconds` | Histogram | NLU processing duration |
| `voice_command_execution_seconds` | Histogram | Command execution duration |
| `voice_intent_confidence_score` | Histogram | Intent confidence scores |
| `voice_transcript_confidence_score` | Histogram | Transcript confidence scores |
| `voice_audio_bytes_processed_total` | Counter | Total audio bytes processed |
| `voice_kafka_messages_published_total` | Counter | Kafka messages published |
| `voice_errors_total` | Counter | Total errors by type |

### Grafana Dashboard

Import the dashboard from `monitoring/grafana-dashboard.json`.

---

## Error Codes

| Code | HTTP Status | Description |
|---|---|---|
| `SESSION_NOT_FOUND` | 404 | Voice session not found |
| `SESSION_EXPIRED` | 410 | Voice session has expired |
| `SESSION_INACTIVE` | 409 | Session is not in active state |
| `TRANSCRIPT_NOT_FOUND` | 404 | Transcript not found |
| `INTENT_NOT_FOUND` | 404 | Intent not found |
| `COMMAND_NOT_FOUND` | 404 | Voice command not found |
| `INTENT_CONFIDENCE_LOW` | 422 | Intent confidence below threshold |
| `COMMAND_NOT_CONFIRMABLE` | 409 | Command does not require confirmation |
| `COMMAND_ALREADY_EXECUTED` | 409 | Command already executed |
| `AUDIO_ENCODING_UNSUPPORTED` | 400 | Unsupported audio encoding |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `STT_PROVIDER_ERROR` | 502 | Speech-to-text provider error |
| `NLU_PROVIDER_ERROR` | 502 | NLU provider error |
| `UNAUTHORIZED` | 401 | Invalid or expired JWT token |
| `FORBIDDEN` | 403 | Insufficient permissions |

---

## License

Proprietary - RideShare Platform © 2024
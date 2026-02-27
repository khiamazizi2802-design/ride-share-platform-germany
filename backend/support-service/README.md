# Support Service — Fahrdienst Plattform

> **Microservice für Kundensupport und Ticketverwaltung**
> Deutsch | [English](#english-summary)

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](https://ci.example.com)
[![Coverage](https://img.shields.io/badge/coverage-94%25-brightgreen)](https://coverage.example.com)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-blue)](https://hub.docker.com/r/fahrdienst/support-service)
[![Kubernetes](https://img.shields.io/badge/kubernetes-1.28%2B-326ce5)](https://kubernetes.io)
[![Node.js](https://img.shields.io/badge/node.js-20.x-339933)](https://nodejs.org)
[![PostgreSQL](https://img.shields.io/badge/postgresql-15-336791)](https://postgresql.org)
[![GDPR](https://img.shields.io/badge/DSGVO-konform-orange)](docs/gdpr.md)

---

## Inhaltsverzeichnis

- [Übersicht](#übersicht)
- [Funktionen](#funktionen)
- [Architektur](#architektur)
- [API-Dokumentation](#api-dokumentation)
- [Datenbankschema](#datenbankschema)
- [Umgebungsvariablen](#umgebungsvariablen)
- [Lokale Entwicklung](#lokale-entwicklung)
- [Docker](#docker)
- [Kubernetes Deployment](#kubernetes-deployment)
- [DSGVO-Konformität](#dsgvo-konformität)
- [Integration mit anderen Services](#integration-mit-anderen-services)
- [Monitoring und Logging](#monitoring-und-logging)
- [Tests](#tests)
- [Beitragen](#beitragen)
- [English Summary](#english-summary)

---

## Übersicht

Der **Support Service** ist ein zentrales Microservice-Modul der Fahrdienst-Plattform, das sämtliche Kundensupport-Prozesse verwaltet. Er ermöglicht die Erstellung, Verwaltung und Auflösung von Support-Tickets, die interne Kommunikation zwischen Kunden und Support-Agenten sowie automatisierte Eskalationsprozesse. Der Service ist vollständig DSGVO-konform und wurde speziell für den deutschen und europäischen Markt entwickelt.

### Kernverantwortlichkeiten

- Verwaltung des gesamten Lebenszyklus von Support-Tickets (Erstellung bis Abschluss)
- Echtzeit-Kommunikation zwischen Fahrern, Fahrgästen und Support-Agenten
- Automatisierte Klassifizierung und Priorisierung von Anfragen mittels NLP
- Integration mit dem Zahlungs-, Fahrten- und Benutzerverwaltungs-Service
- Einhaltung der DSGVO mit vollständigem Audit-Trail und Datenlöschkonzept
- SLA-Überwachung und automatische Eskalation bei Fristüberschreitung
- Mehrsprachige Unterstützung (Deutsch, Englisch, Türkisch, Arabisch, Russisch)

---

## Funktionen

### Ticket-Management
- **Ticket-Erstellung**: Automatische oder manuelle Erstellung von Support-Tickets
- **Kategorisierung**: Automatische Zuordnung zu Kategorien (Zahlung, Sicherheit, Fahrt, Konto, Allgemein)
- **Priorisierung**: Fünfstufiges Prioritätssystem (kritisch, hoch, mittel, niedrig, info)
- **Statusverfolgung**: Vollständige Nachverfolgung aller Statusübergänge
- **Anhänge**: Sichere Upload-Funktion für Bilder, Videos und Dokumente (max. 25 MB)
- **Tagging-System**: Flexible Verschlagwortung für bessere Auffindbarkeit

### Kommunikation
- **Interner Chat**: Verschlüsselter Nachrichtenkanal innerhalb eines Tickets
- **E-Mail-Benachrichtigungen**: Automatische Updates per E-Mail über SES/SMTP
- **Push-Benachrichtigungen**: Integration mit dem Notification Service
- **Interne Notizen**: Nicht-öffentliche Notizen für Agenten
- **Vorlagen**: Konfigurierbare Antwortvorlagen für häufige Anfragen

### Automatisierung
- **Auto-Routing**: Intelligente Zuweisung basierend auf Kategorie, Sprache und Agent-Verfügbarkeit
- **Eskalationsregeln**: Konfigurierbare SLA-Zeitfenster mit automatischer Eskalation
- **Duplicate-Detection**: Erkennung und Zusammenführung von Duplikaten
- **FAQ-Integration**: Automatische Vorschläge aus der Wissensdatenbank
- **Sentiment-Analyse**: Erkennung negativer Kundenstimmung für Priorisierung

### Reporting
- **Echtzeit-Dashboard**: Metriken für Ticket-Volumen, Lösungszeiten und Agenten-Performance
- **Exportfunktion**: CSV/Excel-Export für Compliance und Analysen
- **SLA-Berichte**: Detaillierte Auswertungen zur Einhaltung von Servicevereinbarungen
- **Kundenzufriedenheit**: CSAT-Bewertungen nach Ticket-Abschluss

---

## Architektur

### Systemarchitektur-Übersicht

```
┌─────────────────────────────────────────────────────────────────────┐
│                        API Gateway / Nginx                          │
│                    (Rate Limiting, Auth, SSL)                        │
└────────────────────────┬────────────────────┬───────────────────────┘
                         │                    │
              ┌──────────▼──────┐   ┌─────────▼──────────┐
              │  Mobile App     │   │  Web Dashboard     │
              │  (iOS/Android)  │   │  (React/Next.js)   │
              └──────────┬──────┘   └─────────┬──────────┘
                         │                    │
┌────────────────────────▼────────────────────▼───────────────────────┐
│                     Support Service (Node.js/Express)               │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌────────────┐  │
│  │ Ticket API   │ │  Message API │ │  Agent API   │ │Report API  │  │
│  │  /v1/tickets │ │  /v1/msgs    │ │  /v1/agents  │ │/v1/reports │  │
│  └──────┬───────┘ └──────┬───────┘ └──────┬───────┘ └─────┬──────┘  │
│         │                │               │               │         │
│  ┌──────▼───────────────────────────────────────────────▼──────┐   │
│  │              Service Layer / Business Logic                  │   │
│  │  TicketService │ MessageService │ EscalationService │...     │   │
│  └──────┬───────────────────────────────────────────────┬──────┘   │
│         │                                               │          │
│  ┌──────▼──────┐  ┌───────────────┐  ┌────────────────▼──────┐   │
│  │ PostgreSQL  │  │   Redis Cache │  │   Elasticsearch        │   │
│  │ (Primär-DB) │  │ (Sessions &   │  │   (Volltext-Suche)     │   │
│  │             │  │  Rate Limit)  │  │                        │   │
│  └─────────────┘  └───────────────┘  └────────────────────────┘   │
└────────────────────────┬────────────────────────────────────────────┘
                         │ Kafka Event Bus
         ┌───────────────┼───────────────┬─────────────────┐
         │               │               │                 │
┌────────▼──────┐ ┌──────▼─────┐ ┌──────▼──────┐ ┌───────▼───────┐
│ User Service  │ │ Ride Service│ │Payment Svc  │ │Notification   │
│ (Nutzer-Info) │ │ (Fahrtdaten)│ │(Erstattungen│ │Service (Push/ │
│               │ │             │ │ & Disputes) │ │ E-Mail/SMS)   │
└───────────────┘ └─────────────┘ └─────────────┘ └───────────────┘
```

### Technologie-Stack

| Komponente | Technologie | Version | Zweck |
|---|---|---|---|
| Runtime | Node.js | 20.x LTS | Anwendungsserver |
| Framework | Express.js | 4.18.x | HTTP-Framework |
| Datenbank | PostgreSQL | 15.x | Persistente Datenhaltung |
| Cache | Redis | 7.x | Session, Rate Limiting, Queues |
| Suche | Elasticsearch | 8.x | Volltext-Suche in Tickets |
| Message Broker | Apache Kafka | 3.5.x | Event-Streaming |
| ORM | Prisma | 5.x | Datenbankzugriff |
| Authentifizierung | JWT + OAuth2 | — | Zugangskontrolle |
| Datei-Storage | AWS S3 / MinIO | — | Anhänge |
| E-Mail | AWS SES | — | Transaktionale E-Mails |
| Tracing | OpenTelemetry | — | Distributed Tracing |
| Metriken | Prometheus | — | Metriken-Erfassung |
| Logging | Winston + ELK | — | Strukturiertes Logging |

### Datenfluss

```
Kunde erstellt Ticket
        │
        ▼
[API Gateway] → JWT-Validierung → Rate Limit Check
        │
        ▼
[Support Service]
  │
  ├─ Validierung der Eingabedaten (Joi Schema)
  ├─ Kategorisierung (NLP-Modell)
  ├─ Priorisierung (Regel-Engine)
  ├─ Agent-Zuweisung (Round-Robin / Skill-Based)
  ├─ Ticket in PostgreSQL speichern
  ├─ Elasticsearch Index aktualisieren
  ├─ Kafka Event publizieren (ticket.created)
  └─ Antwort an Client

Kafka Consumer in Notification Service
  ├─ E-Mail-Bestätigung an Kunden
  └─ Push-Notification an Agent
```

---

## API-Dokumentation

**Base URL:** `https://api.fahrdienst.de/v1/support`

**Authentifizierung:** Bearer Token (JWT) — alle Endpunkte erfordern gültige Authentifizierung außer `/health`.

**Inhaltstyp:** `application/json`

**API-Version:** `v1`

**OpenAPI-Spezifikation:** Vollständige Swagger-Dokumentation unter `/v1/support/docs`

---

### Authentifizierung

Alle Anfragen müssen den Authorization-Header enthalten:

```
Authorization: Bearer <jwt_token>
X-Request-ID: <uuid-v4>
Accept-Language: de-DE
```

---

### Tickets

#### `POST /tickets` — Ticket erstellen

Erstellt ein neues Support-Ticket für den authentifizierten Nutzer.

**Berechtigungen:** `user`, `driver`, `admin`

**Request Headers:**
```
Content-Type: application/json
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Request Body:**
```json
{
  "subject": "Zahlung wurde doppelt abgebucht",
  "description": "Am 15.11.2024 wurde meine Zahlung für die Fahrt #FHR-20241115-4892 zweimal von meinem Konto abgebucht. Bitte um umgehende Klärung und Rückerstattung.",
  "category": "payment",
  "priority": "high",
  "rideId": "FHR-20241115-4892",
  "language": "de",
  "attachments": [
    {
      "fileId": "att_8f3k2j1m",
      "filename": "kontoauszug.pdf",
      "mimeType": "application/pdf"
    }
  ],
  "metadata": {
    "appVersion": "3.14.2",
    "platform": "android",
    "deviceId": "sha256:a3b4c5d6"
  }
}
```

**Mögliche Werte für `category`:**
- `payment` — Zahlungsprobleme, Rückerstattungen
- `ride` — Fahrtverlauf, verlorene Gegenstände
- `account` — Kontoverwaltung, Verifikation
- `safety` — Sicherheitsvorfälle (automatisch höchste Priorität)
- `driver_behavior` — Beschwerden über Fahrerverhalten
- `technical` — App-Fehler, technische Probleme
- `general` — Allgemeine Anfragen

**Mögliche Werte für `priority`:**
- `critical` — Sicherheitsvorfälle, sofortiger Handlungsbedarf
- `high` — Zahlungsprobleme, dringende Anliegen
- `medium` — Standard-Support-Anfragen
- `low` — Allgemeine Fragen, Feedback
- `info` — Informationsanfragen

**Erfolgreiche Antwort `201 Created`:**
```json
{
  "success": true,
  "data": {
    "ticketId": "TKT-20241115-00847",
    "status": "open",
    "subject": "Zahlung wurde doppelt abgebucht",
    "category": "payment",
    "priority": "high",
    "assignedAgent": {
      "agentId": "agt_h7k2m9p",
      "name": "Maria Schmidt",
      "avatarUrl": "https://cdn.fahrdienst.de/agents/agt_h7k2m9p/avatar.jpg"
    },
    "estimatedResponseTime": "2024-11-15T16:30:00Z",
    "slaDeadline": "2024-11-16T14:00:00Z",
    "createdAt": "2024-11-15T14:23:11Z",
    "updatedAt": "2024-11-15T14:23:11Z",
    "_links": {
      "self": "/v1/support/tickets/TKT-20241115-00847",
      "messages": "/v1/support/tickets/TKT-20241115-00847/messages",
      "attachments": "/v1/support/tickets/TKT-20241115-00847/attachments"
    }
  },
  "meta": {
    "requestId": "req_4x8y2z9w",
    "timestamp": "2024-11-15T14:23:11Z"
  }
}
```

**Fehler-Antworten:**

`400 Bad Request`:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Ungültige Anfragedaten",
    "details": [
      {
        "field": "subject",
        "message": "Betreff muss zwischen 5 und 255 Zeichen lang sein"
      }
    ]
  }
}
```

`429 Too Many Requests`:
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Zu viele Anfragen. Bitte warten Sie 60 Sekunden.",
    "retryAfter": 60
  }
}
```

---

#### `GET /tickets` — Tickets auflisten

Gibt eine paginierte Liste der Tickets des authentifizierten Nutzers zurück.

**Query-Parameter:**

| Parameter | Typ | Standard | Beschreibung |
|---|---|---|---|
| `page` | integer | 1 | Seitennummer |
| `limit` | integer | 20 | Einträge pro Seite (max. 100) |
| `status` | string | — | Filter: `open`, `in_progress`, `pending`, `resolved`, `closed` |
| `category` | string | — | Filter nach Kategorie |
| `priority` | string | — | Filter nach Priorität |
| `from` | ISO8601 | — | Startdatum für Filter |
| `to` | ISO8601 | — | Enddatum für Filter |
| `sort` | string | `createdAt:desc` | Sortierung |
| `search` | string | — | Volltext-Suche |

**Beispiel-Anfrage:**
```bash
curl -X GET \
  'https://api.fahrdienst.de/v1/support/tickets?status=open&category=payment&page=1&limit=10' \
  -H 'Authorization: Bearer <token>' \
  -H 'X-Request-ID: 550e8400-e29b-41d4-a716-446655440000'
```

**Erfolgreiche Antwort `200 OK`:**
```json
{
  "success": true,
  "data": [
    {
      "ticketId": "TKT-20241115-00847",
      "subject": "Zahlung wurde doppelt abgebucht",
      "status": "open",
      "category": "payment",
      "priority": "high",
      "messageCount": 3,
      "lastActivity": "2024-11-15T15:42:00Z",
      "createdAt": "2024-11-15T14:23:11Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 1,
    "totalPages": 1,
    "hasNext": false,
    "hasPrev": false
  }
}
```

---

#### `GET /tickets/:ticketId` — Ticket-Details

**Beispiel-Anfrage:**
```bash
curl -X GET \
  'https://api.fahrdienst.de/v1/support/tickets/TKT-20241115-00847' \
  -H 'Authorization: Bearer <token>'
```

**Erfolgreiche Antwort `200 OK`:**
```json
{
  "success": true,
  "data": {
    "ticketId": "TKT-20241115-00847",
    "subject": "Zahlung wurde doppelt abgebucht",
    "description": "Am 15.11.2024 wurde meine Zahlung...",
    "status": "in_progress",
    "category": "payment",
    "priority": "high",
    "tags": ["double-charge", "refund-required"],
    "rideId": "FHR-20241115-4892",
    "assignedAgent": {
      "agentId": "agt_h7k2m9p",
      "name": "Maria Schmidt"
    },
    "statusHistory": [
      {
        "status": "open",
        "changedAt": "2024-11-15T14:23:11Z",
        "changedBy": "system"
      },
      {
        "status": "in_progress",
        "changedAt": "2024-11-15T14:45:00Z",
        "changedBy": "agt_h7k2m9p"
      }
    ],
    "sla": {
      "responseDeadline": "2024-11-15T16:23:11Z",
      "resolutionDeadline": "2024-11-16T14:23:11Z",
      "firstResponseAt": "2024-11-15T14:45:00Z",
      "isBreached": false
    },
    "satisfaction": null,
    "createdAt": "2024-11-15T14:23:11Z",
    "updatedAt": "2024-11-15T14:45:00Z"
  }
}
```

---

#### `PATCH /tickets/:ticketId` — Ticket aktualisieren

**Berechtigungen:** Ticketbesitzer (eingeschränkt), `agent`, `admin`

**Request Body (Agent):**
```json
{
  "status": "resolved",
  "resolution": "Doppelabbuchung bestätigt und Rückerstattung von 12,40 EUR eingeleitet. Bearbeitungszeit 3-5 Werktage.",
  "tags": ["double-charge", "refund-issued", "resolved-financial"]
}
```

**Erfolgreiche Antwort `200 OK`:**
```json
{
  "success": true,
  "data": {
    "ticketId": "TKT-20241115-00847",
    "status": "resolved",
    "updatedAt": "2024-11-15T16:10:22Z"
  }
}
```

---

#### `DELETE /tickets/:ticketId` — Ticket-Daten löschen (DSGVO)

**Berechtigungen:** `admin`, `dpo` (Datenschutzbeauftragter)

Löscht oder anonymisiert alle personenbezogenen Daten eines Tickets gemäß DSGVO Art. 17.

**Request Body:**
```json
{
  "reason": "gdpr_erasure_request",
  "requestReference": "DSR-2024-00123",
  "anonymize": true
}
```

---

### Nachrichten

#### `POST /tickets/:ticketId/messages` — Nachricht senden

**Request Body:**
```json
{
  "content": "Vielen Dank für die schnelle Bearbeitung. Kann ich einen Nachweis über die Rückerstattung erhalten?",
  "type": "customer_reply",
  "attachments": []
}
```

**Typen:** `customer_reply`, `agent_reply`, `internal_note`, `system_message`

**Erfolgreiche Antwort `201 Created`:**
```json
{
  "success": true,
  "data": {
    "messageId": "msg_9p3k7q2r",
    "ticketId": "TKT-20241115-00847",
    "content": "Vielen Dank für die schnelle Bearbeitung...",
    "type": "customer_reply",
    "sender": {
      "userId": "usr_4m2k9j7h",
      "role": "customer",
      "displayName": "M. Müller"
    },
    "isInternal": false,
    "createdAt": "2024-11-15T16:22:45Z",
    "editedAt": null
  }
}
```

---

#### `GET /tickets/:ticketId/messages` — Nachrichten abrufen

**Query-Parameter:** `page`, `limit`, `includeInternal` (nur Agenten)

**Erfolgreiche Antwort `200 OK`:**
```json
{
  "success": true,
  "data": [
    {
      "messageId": "msg_2a4b6c8d",
      "content": "Guten Tag, ich habe Ihr Anliegen zur doppelten Abbuchung erhalten...",
      "type": "agent_reply",
      "sender": {
        "agentId": "agt_h7k2m9p",
        "name": "Maria Schmidt",
        "role": "support_agent"
      },
      "createdAt": "2024-11-15T14:45:30Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 4
  }
}
```

---

### Anhänge

#### `POST /tickets/:ticketId/attachments` — Datei hochladen

**Content-Type:** `multipart/form-data`

```bash
curl -X POST \
  'https://api.fahrdienst.de/v1/support/tickets/TKT-20241115-00847/attachments' \
  -H 'Authorization: Bearer <token>' \
  -F 'file=@/pfad/zum/dokument.pdf' \
  -F 'description=Kontoauszug November 2024'
```

**Erlaubte Dateitypen:** `image/jpeg`, `image/png`, `image/webp`, `video/mp4`, `application/pdf`, `text/plain`

**Maximale Dateigröße:** 25 MB

**Erfolgreiche Antwort `201 Created`:**
```json
{
  "success": true,
  "data": {
    "attachmentId": "att_8f3k2j1m",
    "filename": "kontoauszug.pdf",
    "mimeType": "application/pdf",
    "size": 142567,
    "url": "https://cdn.fahrdienst.de/support/TKT-20241115-00847/att_8f3k2j1m?token=...",
    "expiresAt": "2024-11-22T14:23:11Z",
    "virusScanStatus": "clean",
    "uploadedAt": "2024-11-15T14:20:00Z"
  }
}
```

---

### Agenten-Verwaltung

#### `GET /agents` — Agenten auflisten

**Berechtigungen:** `admin`, `supervisor`

**Erfolgreiche Antwort `200 OK`:**
```json
{
  "success": true,
  "data": [
    {
      "agentId": "agt_h7k2m9p",
      "name": "Maria Schmidt",
      "email": "m.schmidt@fahrdienst.de",
      "role": "support_agent",
      "status": "online",
      "languages": ["de", "en"],
      "skills": ["payment", "account", "general"],
      "currentLoad": 12,
      "maxLoad": 20,
      "averageHandlingTime": 847,
      "csatScore": 4.7
    }
  ]
}
```

#### `POST /tickets/:ticketId/assign` — Ticket zuweisen

**Berechtigungen:** `admin`, `supervisor`

**Request Body:**
```json
{
  "agentId": "agt_h7k2m9p",
  "reason": "Spezialist für Zahlungsthemen"
}
```

---

### Bewertungen & CSAT

#### `POST /tickets/:ticketId/rating` — Ticket bewerten

**Berechtigungen:** Ticket-Ersteller nach Ticket-Abschluss

**Request Body:**
```json
{
  "score": 5,
  "comment": "Sehr schnelle und freundliche Hilfe. Problem vollständig gelöst!",
  "categories": {
    "speed": 5,
    "helpfulness": 5,
    "resolution": 5
  }
}
```

---

### Reporting

#### `GET /reports/summary` — Zusammenfassung

**Berechtigungen:** `admin`, `supervisor`

**Query-Parameter:** `from`, `to`, `groupBy` (`day`|`week`|`month`)

**Erfolgreiche Antwort `200 OK`:**
```json
{
  "success": true,
  "data": {
    "period": {
      "from": "2024-11-01T00:00:00Z",
      "to": "2024-11-30T23:59:59Z"
    },
    "tickets": {
      "total": 2847,
      "open": 142,
      "inProgress": 89,
      "resolved": 2541,
      "closed": 75
    },
    "performance": {
      "averageFirstResponseTime": 1247,
      "averageResolutionTime": 18420,
      "slaComplianceRate": 96.3,
      "csatScore": 4.6,
      "firstContactResolutionRate": 68.2
    },
    "categories": [
      { "name": "payment", "count": 987, "percentage": 34.7 },
      { "name": "ride", "count": 734, "percentage": 25.8 },
      { "name": "account", "count": 521, "percentage": 18.3 }
    ]
  }
}
```

---

### Health & Status

#### `GET /health` — Health Check

Kein Auth erforderlich. Wird von Kubernetes-Probes verwendet.

**Antwort `200 OK`:**
```json
{
  "status": "healthy",
  "version": "2.4.1",
  "timestamp": "2024-11-15T14:23:11Z",
  "uptime": 1209600,
  "dependencies": {
    "postgresql": { "status": "healthy", "latency": 3 },
    "redis": { "status": "healthy", "latency": 1 },
    "elasticsearch": { "status": "healthy", "latency": 8 },
    "kafka": { "status": "healthy" }
  }
}
```

#### `GET /health/ready` — Readiness Probe

#### `GET /health/live` — Liveness Probe

---

## Datenbankschema

Der Service verwendet PostgreSQL 15 mit Prisma ORM. Nachfolgend sind die Haupttabellen und ihre Beziehungen beschrieben.

### Tabelle: `tickets`

```sql
CREATE TABLE tickets (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_number         VARCHAR(30) UNIQUE NOT NULL,    -- TKT-20241115-00847
    subject               VARCHAR(255) NOT NULL,
    description           TEXT NOT NULL,
    status                ticket_status NOT NULL DEFAULT 'open',
    category              ticket_category NOT NULL,
    priority              ticket_priority NOT NULL DEFAULT 'medium',
    language              CHAR(2) NOT NULL DEFAULT 'de',
    
    -- Beziehungen zu anderen Services
    user_id               UUID NOT NULL,                   -- Referenz zu User Service
    ride_id               VARCHAR(50),                     -- Referenz zu Ride Service
    payment_id            VARCHAR(50),                     -- Referenz zu Payment Service
    
    -- Agenten-Zuweisung
    assigned_agent_id     UUID REFERENCES agents(id),
    assigned_at           TIMESTAMPTZ,
    
    -- Auflösung
    resolution            TEXT,
    resolved_at           TIMESTAMPTZ,
    closed_at             TIMESTAMPTZ,
    
    -- SLA
    sla_response_deadline TIMESTAMPTZ NOT NULL,
    sla_resolve_deadline  TIMESTAMPTZ NOT NULL,
    first_response_at     TIMESTAMPTZ,
    sla_breached          BOOLEAN DEFAULT FALSE,
    
    -- Metadaten
    tags                  TEXT[] DEFAULT '{}',
    metadata              JSONB DEFAULT '{}',
    source                ticket_source DEFAULT 'app',    -- app, web, email, phone
    
    -- DSGVO
    is_anonymized         BOOLEAN DEFAULT FALSE,
    anonymized_at         TIMESTAMPTZ,
    gdpr_deletion_ref     VARCHAR(100),
    
    -- Zeitstempel
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ                      -- Soft Delete
);

-- Enum Definitionen
CREATE TYPE ticket_status AS ENUM (
    'open', 'in_progress', 'pending_customer',
    'pending_internal', 'resolved', 'closed', 'cancelled'
);

CREATE TYPE ticket_category AS ENUM (
    'payment', 'ride', 'account', 'safety',
    'driver_behavior', 'technical', 'general'
);

CREATE TYPE ticket_priority AS ENUM (
    'critical', 'high', 'medium', 'low', 'info'
);

CREATE TYPE ticket_source AS ENUM (
    'app', 'web', 'email', 'phone', 'internal'
);

-- Indizes
CREATE INDEX idx_tickets_user_id ON tickets(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tickets_status ON tickets(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tickets_assigned_agent ON tickets(assigned_agent_id) WHERE status NOT IN ('resolved', 'closed');
CREATE INDEX idx_tickets_created_at ON tickets(created_at DESC);
CREATE INDEX idx_tickets_sla_deadline ON tickets(sla_resolve_deadline) WHERE status NOT IN ('resolved', 'closed', 'cancelled');
CREATE INDEX idx_tickets_tags ON tickets USING GIN(tags);
CREATE INDEX idx_tickets_metadata ON tickets USING GIN(metadata);
```

### Tabelle: `messages`

```sql
CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id       UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    content         TEXT NOT NULL,
    content_type    VARCHAR(20) DEFAULT 'text',           -- text, html, markdown
    message_type    message_type NOT NULL,
    
    -- Absender
    sender_id       UUID NOT NULL,
    sender_type     VARCHAR(20) NOT NULL,                  -- customer, agent, system
    sender_name     VARCHAR(100),                          -- Denormalisiert für Audit
    
    -- Eigenschaften
    is_internal     BOOLEAN DEFAULT FALSE,
    is_edited       BOOLEAN DEFAULT FALSE,
    edited_at       TIMESTAMPTZ,
    
    -- Anhänge
    attachment_ids  UUID[] DEFAULT '{}',
    
    -- DSGVO
    is_anonymized   BOOLEAN DEFAULT FALSE,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE message_type AS ENUM (
    'customer_reply', 'agent_reply', 'internal_note',
    'system_message', 'status_change', 'assignment'
);

CREATE INDEX idx_messages_ticket_id ON messages(ticket_id);
CREATE INDEX idx_messages_sender_id ON messages(sender_id);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);
```

### Tabelle: `agents`

```sql
CREATE TABLE agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id     UUID UNIQUE NOT NULL,                  -- ID aus User Service
    email           VARCHAR(255) UNIQUE NOT NULL,
    name            VARCHAR(100) NOT NULL,
    role            agent_role NOT NULL DEFAULT 'support_agent',
    status          agent_status NOT NULL DEFAULT 'offline',
    
    -- Fähigkeiten
    languages       CHAR(2)[] NOT NULL DEFAULT '{de}',
    skills          TEXT[] DEFAULT '{}',
    max_load        INTEGER NOT NULL DEFAULT 20,
    
    -- Performance (aktualisiert durch Aggregation)
    avg_handling_time INTEGER,                             -- Sekunden
    csat_score        DECIMAL(3,2),
    total_resolved    INTEGER DEFAULT 0,
    
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE agent_role AS ENUM (
    'support_agent', 'senior_agent', 'supervisor', 'admin'
);

CREATE TYPE agent_status AS ENUM (
    'online', 'busy', 'away', 'offline'
);
```

### Tabelle: `attachments`

```sql
CREATE TABLE attachments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id       UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    message_id      UUID REFERENCES messages(id),
    
    filename        VARCHAR(255) NOT NULL,
    original_name   VARCHAR(255) NOT NULL,
    mime_type       VARCHAR(100) NOT NULL,
    size_bytes      BIGINT NOT NULL,
    storage_key     VARCHAR(500) NOT NULL,                 -- S3/MinIO Key
    
    virus_scan_status VARCHAR(20) DEFAULT 'pending',       -- pending, clean, infected
    virus_scan_at   TIMESTAMPTZ,
    
    uploaded_by     UUID NOT NULL,
    expires_at      TIMESTAMPTZ,
    
    -- DSGVO
    is_deleted      BOOLEAN DEFAULT FALSE,
    deleted_at      TIMESTAMPTZ,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Tabelle: `ticket_ratings`

```sql
CREATE TABLE ticket_ratings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id   UUID UNIQUE NOT NULL REFERENCES tickets(id),
    user_id     UUID NOT NULL,
    score       SMALLINT NOT NULL CHECK (score BETWEEN 1 AND 5),
    comment     TEXT,
    categories  JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Tabelle: `audit_log`

```sql
CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,                      -- ticket, message, agent
    entity_id   UUID NOT NULL,
    action      VARCHAR(100) NOT NULL,
    actor_id    UUID,
    actor_type  VARCHAR(20),
    old_values  JSONB,
    new_values  JSONB,
    ip_address  INET,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_actor ON audit_log(actor_id);
CREATE INDEX idx_audit_created ON audit_log(created_at DESC);
```

### ER-Diagramm (vereinfacht)

```
tickens (1) ─────────── (N) messages
   │                           │
   │                    (N) attachments
   │
   ├── (N) attachments (direkt)
   ├── (1) agents (assigned_agent_id)
   ├── (1) ticket_ratings
   └── (N) audit_log
```

---

## Umgebungsvariablen

Erstellen Sie eine `.env`-Datei basierend auf `.env.example`:

```bash
cp .env.example .env
```

### Pflichtfelder

```env
# ========================================
# Anwendung
# ========================================
NODE_ENV=development                    # development | staging | production
APP_PORT=3004
APP_NAME=support-service
APP_VERSION=2.4.1
LOG_LEVEL=info                          # debug | info | warn | error

# ========================================
# Datenbank (PostgreSQL)
# ========================================
DATABASE_URL=postgresql://support_user:StarkesPasswort123!@localhost:5432/support_db?schema=public&connection_limit=20&pool_timeout=30
DATABASE_REPLICA_URL=postgresql://support_user:StarkesPasswort123!@localhost:5433/support_db?schema=public
DATABASE_SSL=false                      # true in Produktion
DATABASE_MAX_CONNECTIONS=20
DATABASE_MIN_CONNECTIONS=2

# ========================================
# Redis
# ========================================
REDIS_URL=redis://:RedisPasswort@localhost:6379/0
REDIS_CLUSTER_MODE=false                # true für Produktions-Cluster
REDIS_TLS=false                         # true in Produktion
REDIS_SESSION_TTL=3600                  # Sekunden
REDIS_RATE_LIMIT_TTL=60

# ========================================
# Elasticsearch
# ========================================
ELASTICSEARCH_NODE=http://localhost:9200
ELASTICSEARCH_USERNAME=elastic
ELASTICSEARCH_PASSWORD=ElasticPasswort
ELASTICSEARCH_INDEX_PREFIX=fahrdienst_support
ELASTICSEARCH_TLS=false

# ========================================
# Apache Kafka
# ========================================
KAFKA_BROKERS=localhost:9092
KAFKA_CLIENT_ID=support-service
KAFKA_GROUP_ID=support-service-group
KAFKA_SSL=false
KAFKA_SASL_MECHANISM=plain              # plain | scram-sha-256 | scram-sha-512
KAFKA_SASL_USERNAME=support-service
KAFKA_SASL_PASSWORD=KafkaPasswort

# Kafka Topics
KAFKA_TOPIC_TICKET_CREATED=support.ticket.created
KAFKA_TOPIC_TICKET_UPDATED=support.ticket.updated
KAFKA_TOPIC_TICKET_RESOLVED=support.ticket.resolved
KAFKA_TOPIC_MESSAGE_SENT=support.message.sent
KAFKA_TOPIC_ESCALATION=support.ticket.escalated
KAFKA_TOPIC_USER_EVENTS=user.events
KAFKA_TOPIC_RIDE_EVENTS=ride.events
KAFKA_TOPIC_PAYMENT_EVENTS=payment.events

# ========================================
# Authentifizierung & Sicherheit
# ========================================
JWT_PUBLIC_KEY_PATH=/app/keys/jwt_public.pem
JWT_ALGORITHM=RS256
JWT_AUDIENCE=fahrdienst-platform
JWT_ISSUER=https://auth.fahrdienst.de
INTERNAL_API_KEY=InternalApiSchluessel256Bit

# ========================================
# Datei-Upload (AWS S3 / MinIO)
# ========================================
AWS_REGION=eu-central-1
AWS_S3_BUCKET=fahrdienst-support-attachments
AWS_ACCESS_KEY_ID=AKIAxxxxxxxxxxxxxxxx
AWS_SECRET_ACCESS_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
AWS_S3_ENDPOINT=                        # Leer für AWS, URL für MinIO
AWS_S3_FORCE_PATH_STYLE=false           # true für MinIO
ATTACHMENT_MAX_SIZE_MB=25
ATTACHMENT_URL_EXPIRY_HOURS=168         # 7 Tage

# ========================================
# E-Mail (AWS SES)
# ========================================
EMAIL_PROVIDER=ses                      # ses | smtp
EMAIL_FROM=support@fahrdienst.de
EMAIL_FROM_NAME=Fahrdienst Support
SES_REGION=eu-west-1
# SMTP-Fallback
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=support@fahrdienst.de
SMTP_PASSWORD=SMTPPasswort
SMTP_SECURE=true

# ========================================
# Service-zu-Service-Kommunikation
# ========================================
USER_SERVICE_URL=http://user-service:3001
RIDE_SERVICE_URL=http://ride-service:3002
PAYMENT_SERVICE_URL=http://payment-service:3003
NOTIFICATION_SERVICE_URL=http://notification-service:3005
SERVICE_TIMEOUT_MS=5000
SERVICE_RETRY_ATTEMPTS=3

# ========================================
# Virus-Scan
# ========================================
CLAMAV_HOST=clamav
CLAMAV_PORT=3310
CLAMAV_TIMEOUT=30000

# ========================================
# Monitoring & Observability
# ========================================
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
OTEL_SERVICE_NAME=support-service
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1             # 10% Sampling in Prod
PROMETHEUS_PORT=9090
HEALTHCHECK_PATH=/health

# ========================================
# Feature Flags
# ========================================
FEATURE_NLP_CATEGORIZATION=true
FEATURE_SENTIMENT_ANALYSIS=true
FEATURE_AUTO_ROUTING=true
FEATURE_DUPLICATE_DETECTION=true
FEATURE_FAQ_SUGGESTIONS=true
FEATURE_VIRUS_SCAN=true

# ========================================
# SLA-Konfiguration (Minuten)
# ========================================
SLA_CRITICAL_RESPONSE=15
SLA_CRITICAL_RESOLVE=60
SLA_HIGH_RESPONSE=60
SLA_HIGH_RESOLVE=480
SLA_MEDIUM_RESPONSE=240
SLA_MEDIUM_RESOLVE=1440
SLA_LOW_RESPONSE=1440
SLA_LOW_RESOLVE=4320

# ========================================
# DSGVO & Datenschutz
# ========================================
GDPR_RETENTION_DAYS=730                 # 2 Jahre Standard-Aufbewahrung
GDPR_ANONYMIZE_ON_CLOSE=false           # Nach Ticket-Schließung anonymisieren
GDPR_DELETION_LOG_PATH=/app/logs/gdpr
DATA_ENCRYPTION_KEY=32ByteHexVerschluesselungsschluessel
```

---

## Lokale Entwicklung

### Voraussetzungen

- **Node.js** 20.x LTS ([Download](https://nodejs.org))
- **npm** 10.x oder **pnpm** 8.x
- **Docker** 24.x und **Docker Compose** 2.x
- **Git** 2.40+
- PostgreSQL-Client (`psql`) — optional für direkte DB-Zugriffe
- Redis CLI — optional

### Ersteinrichtung

```bash
# 1. Repository klonen
git clone https://github.com/fahrdienst/support-service.git
cd support-service

# 2. Node.js-Version setzen (mit nvm)
nvm use
# oder: node --version (sollte 20.x zeigen)

# 3. Abhängigkeiten installieren
npm install

# 4. Umgebungsvariablen konfigurieren
cp .env.example .env
# Passen Sie die .env nach Bedarf an

# 5. Infrastruktur-Services starten (Docker)
npm run infra:up
# Startet: PostgreSQL, Redis, Elasticsearch, Kafka, ZooKeeper, MinIO, ClamAV

# 6. Auf Infra-Bereitschaft warten
npm run infra:wait

# 7. Datenbankmigrationen ausführen
npm run db:migrate

# 8. Datenbank mit Testdaten befüllen (optional)
npm run db:seed

# 9. Elasticsearch-Indizes erstellen
npm run search:init

# 10. Service starten
npm run dev
```

Der Service ist nun unter `http://localhost:3004` erreichbar.
Swagger-UI: `http://localhost:3004/v1/support/docs`

### Verfügbare npm-Skripte

```bash
# Entwicklung
npm run dev              # Startet mit Hot-Reload (nodemon)
npm run dev:debug        # Startet mit Node.js-Debugger (Port 9229)
npm run build            # TypeScript kompilieren
npm start                # Produktion (dist/)

# Datenbank
npm run db:migrate       # Ausstehende Migrationen anwenden
npm run db:migrate:dev   # Migration erstellen und anwenden
npm run db:rollback      # Letzte Migration rückgängig machen
npm run db:seed          # Testdaten laden
npm run db:reset         # DB zurücksetzen (nur dev!)
npm run db:studio        # Prisma Studio öffnen
npm run db:generate      # Prisma Client neu generieren

# Tests
npm test                 # Alle Tests ausführen
npm run test:unit        # Nur Unit-Tests
npm run test:integration # Nur Integrationstests
npm run test:e2e         # End-to-End-Tests
npm run test:coverage    # Mit Coverage-Bericht
npm run test:watch       # Watch-Modus

# Code-Qualität
npm run lint             # ESLint ausführen
npm run lint:fix         # ESLint mit Auto-Fix
npm run format           # Prettier formatieren
npm run typecheck        # TypeScript-Typprüfung
npm run audit            # Sicherheitsaudit der Dependencies

# Infrastruktur (lokal)
npm run infra:up         # Alle Docker-Services starten
npm run infra:down       # Alle Docker-Services stoppen
npm run infra:reset      # Services stoppen und Volumes löschen
npm run infra:wait       # Auf Bereitschaft warten
npm run infra:logs       # Logs der Infra-Services anzeigen

# Such-Index
npm run search:init      # ES-Indizes initialisieren
npm run search:reindex   # Alle Tickets neu indizieren
npm run search:delete    # Indizes löschen
```

### Projektstruktur

```
support-service/
├── src/
│   ├── api/
│   │   ├── routes/
│   │   │   ├── tickets.routes.ts
│   │   │   ├── messages.routes.ts
│   │   │   ├── agents.routes.ts
│   │   │   ├── attachments.routes.ts
│   │   │   ├── ratings.routes.ts
│   │   │   └── reports.routes.ts
│   │   ├── controllers/
│   │   ├── middlewares/
│   │   │   ├── auth.middleware.ts
│   │   │   ├── rateLimiter.middleware.ts
│   │   │   ├── validate.middleware.ts
│   │   │   ├── auditLog.middleware.ts
│   │   │   └── errorHandler.middleware.ts
│   │   └── validators/
│   ├── services/
│   │   ├── ticket.service.ts
│   │   ├── message.service.ts
│   │   ├── agent.service.ts
│   │   ├── attachment.service.ts
│   │   ├── escalation.service.ts
│   │   ├── nlp.service.ts
│   │   ├── search.service.ts
│   │   └── notification.service.ts
│   ├── repositories/
│   │   ├── ticket.repository.ts
│   │   ├── message.repository.ts
│   │   ├── agent.repository.ts
│   │   └── attachment.repository.ts
│   ├── kafka/
│   │   ├── producers/
│   │   └── consumers/
│   ├── models/
│   ├── utils/
│   ├── config/
│   │   ├── app.config.ts
│   │   ├── database.config.ts
│   │   └── kafka.config.ts
│   └── app.ts
├── prisma/
│   ├── schema.prisma
│   ├── migrations/
│   └── seeders/
├── tests/
│   ├── unit/
│   ├── integration/
│   └── e2e/
├── docker/
│   ├── Dockerfile
│   ├── Dockerfile.dev
│   └── docker-compose.dev.yml
├── k8s/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── hpa.yaml
│   └── ingress.yaml
├── docs/
│   ├── api/
│   ├── architecture/
│   └── gdpr.md
├── .env.example
├── .nvmrc
├── tsconfig.json
├── package.json
└── README.md
```

---

## Docker

### Dockerfile

```dockerfile
# ---- Build Stage ----
FROM node:20-alpine AS builder

WORKDIR /app

# Nur package-Dateien für Cache-Optimierung
COPY package*.json ./
COPY prisma ./prisma/

RUN npm ci --only=production && npm cache clean --force

# TypeScript kompilieren
COPY . .
RUN npm run build

# ---- Production Stage ----
FROM node:20-alpine AS production

# Sicherheits-Patches
RUN apk add --no-cache dumb-init && \
    addgroup -g 1001 -S nodejs && \
    adduser -S nodejs -u 1001

WORKDIR /app

# Nur notwendige Dateien kopieren
COPY --from=builder --chown=nodejs:nodejs /app/node_modules ./node_modules
COPY --from=builder --chown=nodejs:nodejs /app/dist ./dist
COPY --from=builder --chown=nodejs:nodejs /app/prisma ./prisma
COPY --from=builder --chown=nodejs:nodejs /app/package.json ./

# Nicht als Root ausführen
USER nodejs

EXPOSE 3004
EXPOSE 9090

HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD wget -qO- http://localhost:3004/health || exit 1

ENTRYPOINT ["dumb-init", "--"]
CMD ["node", "dist/app.js"]
```

### Docker Compose (Entwicklung)

```yaml
# docker/docker-compose.dev.yml
version: '3.9'

services:
  support-service:
    build:
      context: ..
      dockerfile: docker/Dockerfile.dev
    container_name: support-service
    ports:
      - "3004:3004"
      - "9229:9229"    # Debug Port
      - "9090:9090"    # Prometheus Metriken
    environment:
      - NODE_ENV=development
    env_file:
      - ../.env
    volumes:
      - ../src:/app/src
      - ../prisma:/app/prisma
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      elasticsearch:
        condition: service_healthy
      kafka:
        condition: service_healthy
    networks:
      - fahrdienst-dev
    restart: unless-stopped

  postgres:
    image: postgres:15-alpine
    container_name: support-postgres
    environment:
      POSTGRES_DB: support_db
      POSTGRES_USER: support_user
      POSTGRES_PASSWORD: StarkesPasswort123!
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init-scripts:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U support_user -d support_db"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - fahrdienst-dev

  redis:
    image: redis:7-alpine
    container_name: support-redis
    command: redis-server --requirepass RedisPasswort --appendonly yes
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "RedisPasswort", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - fahrdienst-dev

  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.11.0
    container_name: support-elasticsearch
    environment:
      - discovery.type=single-node
      - ELASTIC_PASSWORD=ElasticPasswort
      - xpack.security.enabled=true
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    ports:
      - "9200:9200"
    volumes:
      - elasticsearch_data:/usr/share/elasticsearch/data
    healthcheck:
      test: ["CMD-SHELL", "curl -s -u elastic:ElasticPasswort http://localhost:9200/_cluster/health | grep -v 'red'"]
      interval: 30s
      timeout: 10s
      retries: 5
    networks:
      - fahrdienst-dev

  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    container_name: support-zookeeper
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
    networks:
      - fahrdienst-dev

  kafka:
    image: confluentinc/cp-kafka:7.5.0
    container_name: support-kafka
    depends_on:
      - zookeeper
    ports:
      - "9092:9092"
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:29092,PLAINTEXT_HOST://localhost:9092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    healthcheck:
      test: ["CMD", "kafka-broker-api-versions", "--bootstrap-server", "kafka:29092"]
      interval: 30s
      timeout: 10s
      retries: 5
    networks:
      - fahrdienst-dev

  minio:
    image: minio/minio:latest
    container_name: support-minio
    command: server /data --console-address ":9001"
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: MinioPasswort123
    volumes:
      - minio_data:/data
    networks:
      - fahrdienst-dev

  clamav:
    image: clamav/clamav:latest
    container_name: support-clamav
    ports:
      - "3310:3310"
    networks:
      - fahrdienst-dev

volumes:
  postgres_data:
  redis_data:
  elasticsearch_data:
  minio_data:

networks:
  fahrdienst-dev:
    name: fahrdienst-dev
    driver: bridge
```

### Docker-Befehle

```bash
# Image bauen
docker build -t fahrdienst/support-service:latest -f docker/Dockerfile .

# Mit Tag bauen
docker build -t fahrdienst/support-service:2.4.1 -f docker/Dockerfile .

# Entwicklungsumgebung starten
docker compose -f docker/docker-compose.dev.yml up -d

# Logs anzeigen
docker compose -f docker/docker-compose.dev.yml logs -f support-service

# Container-Shell
docker exec -it support-service sh

# Umgebung stoppen
docker compose -f docker/docker-compose.dev.yml down

# Image in Registry pushen
docker push fahrdienst/support-service:2.4.1
docker push fahrdienst/support-service:latest

# Multi-Arch Build (für Apple Silicon)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t fahrdienst/support-service:2.4.1 \
  --push .
```

---

## Kubernetes Deployment

### Voraussetzungen

- Kubernetes 1.28+
- kubectl konfiguriert
- Helm 3.12+
- Secrets Manager (AWS Secrets Manager oder Vault)

### Namespace

```bash
kubectl create namespace fahrdienst-prod
kubectl label namespace fahrdienst-prod environment=production team=platform
```

### Secrets

```bash
# Kubernetes Secret aus Umgebungsvariablen erstellen
kubectl create secret generic support-service-secrets \
  --from-literal=DATABASE_URL="postgresql://..." \
  --from-literal=REDIS_URL="redis://..." \
  --from-literal=INTERNAL_API_KEY="..." \
  --from-literal=DATA_ENCRYPTION_KEY="..." \
  --namespace=fahrdienst-prod

# JWT Public Key
kubectl create secret generic support-jwt-keys \
  --from-file=jwt_public.pem=./keys/jwt_public.pem \
  --namespace=fahrdienst-prod
```

### Deployment-Manifest

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: support-service
  namespace: fahrdienst-prod
  labels:
    app: support-service
    version: "2.4.1"
    team: platform
    tier: backend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: support-service
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: support-service
        version: "2.4.1"
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: support-service-sa
      securityContext:
        runAsNonRoot: true
        runAsUser: 1001
        fsGroup: 1001
      terminationGracePeriodSeconds: 30
      containers:
        - name: support-service
          image: fahrdienst/support-service:2.4.1
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 3004
            - name: metrics
              containerPort: 9090
          env:
            - name: NODE_ENV
              value: production
            - name: APP_PORT
              value: "3004"
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: support-service-secrets
                  key: DATABASE_URL
            - name: REDIS_URL
              valueFrom:
                secretKeyRef:
                  name: support-service-secrets
                  key: REDIS_URL
            - name: INTERNAL_API_KEY
              valueFrom:
                secretKeyRef:
                  name: support-service-secrets
                  key: INTERNAL_API_KEY
          envFrom:
            - configMapRef:
                name: support-service-config
          volumeMounts:
            - name: jwt-keys
              mountPath: /app/keys
              readOnly: true
            - name: tmp-dir
              mountPath: /tmp
          resources:
            requests:
              memory: "256Mi"
              cpu: "100m"
            limits:
              memory: "512Mi"
              cpu: "500m"
          livenessProbe:
            httpGet:
              path: /health/live
              port: http
            initialDelaySeconds: 30
            periodSeconds: 15
            timeoutSeconds: 5
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /health/ready
              port: http
            initialDelaySeconds: 15
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          startupProbe:
            httpGet:
              path: /health/live
              port: http
            failureThreshold: 30
            periodSeconds: 10
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL
      volumes:
        - name: jwt-keys
          secret:
            secretName: support-jwt-keys
        - name: tmp-dir
          emptyDir: {}
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels:
                    app: support-service
                topologyKey: kubernetes.io/hostname
```

### Service & Ingress

```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: support-service
  namespace: fahrdienst-prod
  labels:
    app: support-service
spec:
  type: ClusterIP
  selector:
    app: support-service
  ports:
    - name: http
      port: 80
      targetPort: 3004
    - name: metrics
      port: 9090
      targetPort: 9090
---
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: support-service-ingress
  namespace: fahrdienst-prod
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - api.fahrdienst.de
      secretName: fahrdienst-tls
  rules:
    - host: api.fahrdienst.de
      http:
        paths:
          - path: /v1/support
            pathType: Prefix
            backend:
              service:
                name: support-service
                port:
                  name: http
```

### Horizontal Pod Autoscaler

```yaml
# k8s/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: support-service-hpa
  namespace: fahrdienst-prod
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: support-service
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
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
        - type: Pods
          value: 2
          periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
```

### Deployment ausführen

```bash
# Konfiguration anwenden
kubectl apply -f k8s/configmap.yaml -n fahrdienst-prod
kubectl apply -f k8s/deployment.yaml -n fahrdienst-prod
kubectl apply -f k8s/service.yaml -n fahrdienst-prod
kubectl apply -f k8s/hpa.yaml -n fahrdienst-prod
kubectl apply -f k8s/ingress.yaml -n fahrdienst-prod

# Oder alles auf einmal
kubectl apply -f k8s/ -n fahrdienst-prod

# Rollout-Status prüfen
kubectl rollout status deployment/support-service -n fahrdienst-prod

# Pods anzeigen
kubectl get pods -l app=support-service -n fahrdienst-prod

# Logs anzeigen
kubectl logs -f deployment/support-service -n fahrdienst-prod

# Rollback bei Problemen
kubectl rollout undo deployment/support-service -n fahrdienst-prod

# Datenbankmigrationen im Cluster ausführen
kubectl run db-migrate \
  --image=fahrdienst/support-service:2.4.1 \
  --restart=Never \
  --env-from=configmap/support-service-config \
  --env-from=secret/support-service-secrets \
  --command -- npm run db:migrate
```

---

## DSGVO-Konformität

Der Support Service wurde von Grund auf DSGVO-konform entwickelt und entspricht den Anforderungen der Datenschutz-Grundverordnung (EU) 2016/679.

### Rechtsgrundlagen

| Verarbeitungszweck | Rechtsgrundlage | Aufbewahrungsdauer |
|---|---|---|
| Bearbeitung von Support-Anfragen | Art. 6 Abs. 1 lit. b (Vertragserfüllung) | 2 Jahre nach Abschluss |
| Sicherheitsvorfälle | Art. 6 Abs. 1 lit. f (Berechtigtes Interesse) | 3 Jahre |
| Audit-Logs | Art. 6 Abs. 1 lit. c (Rechtliche Verpflichtung) | 10 Jahre |
| Analyse & Verbesserung | Art. 6 Abs. 1 lit. f (nach Anonymisierung) | Unbegrenzt |

### Betroffenenrechte (Art. 15-22 DSGVO)

#### Recht auf Auskunft (Art. 15)

```bash
# Alle personenbezogenen Daten eines Nutzers exportieren
GET /v1/support/gdpr/export?userId={userId}
# Berechtigungen: dpo, admin
# Gibt JSON-Export aller Tickets, Nachrichten, Anhänge zurück
# Antwortzeit: max. 30 Tage (Implementierung: < 24 Stunden)
```

#### Recht auf Löschung (Art. 17)

```bash
# Personenbezogene Daten löschen/anonymisieren
POST /v1/support/gdpr/erasure
Body: { "userId": "...", "requestRef": "DSR-2024-00123" }
```

Der Löschprozess:
1. Personenbezogene Daten in Tickets werden anonymisiert (Name → "[gelöscht]", E-Mail → SHA-256-Hash)
2. Nachrichteninhalte mit personenbezogenen Daten werden geschwärzt
3. Anhänge werden aus S3/MinIO gelöscht
4. Aggregierte/anonymisierte Daten werden für Statistiken beibehalten
5. Der Vorgang wird im DSGVO-Audit-Log protokolliert

#### Recht auf Datenübertragbarkeit (Art. 20)

```bash
# Datenexport im JSON/CSV-Format
GET /v1/support/gdpr/portability?userId={userId}&format=json
```

#### Widerspruchsrecht (Art. 21)

Nutzer können der Verarbeitung ihrer Daten für Analyse-Zwecke widersprechen. Dies wird in der Nutzer-Präferenz im User Service gespeichert und vom Support Service bei der Datenverarbeitung berücksichtigt.

### Technische und organisatorische Maßnahmen (Art. 32)

#### Verschlüsselung
- **In Transit**: TLS 1.3 für alle HTTP-Verbindungen
- **At Rest**: AES-256 Verschlüsselung für sensitive Felder (PostgreSQL TDE oder anwendungsseitig)
- **Anhänge**: Server-seitige Verschlüsselung in S3/MinIO (SSE-S3 oder SSE-KMS)
- **Interne Kommunikation**: mTLS zwischen Microservices

#### Zugriffskontrolle
- Rollenbasiertes Zugangskontrollsystem (RBAC) mit granularen Berechtigungen
- Prinzip der minimalen Rechtevergabe für alle Service-Accounts
- Multi-Faktor-Authentifizierung für Agent-Accounts
- Automatische Session-Invalidierung nach Inaktivität

#### Pseudonymisierung
- User-IDs werden für Logs und Metriken pseudonymisiert
- IP-Adressen werden nach 30 Tagen automatisch gekürzt (letztes Oktett)
- Externe Analyzer erhalten nur anonymisierte Ticket-Daten

#### Audit-Trail
Jede Datenzugriffsoperation wird im `audit_log` protokolliert:
- Wer hat zugegriffen (Benutzer-ID, Rolle)
- Was wurde abgerufen oder verändert (Entity-Typ, ID)
- Wann (Zeitstempel mit Zeitzone)
- Von wo (IP-Adresse, User-Agent)
- Audit-Logs sind unveränderlich und werden 10 Jahre aufbewahrt

#### Datensparsamkeit
- Nur notwendige Daten werden erhoben und verarbeitet
- Geräteinformationen werden nur als Hash gespeichert
- Suchanfragen werden nicht personenbezogen gespeichert

### Verarbeitungsverzeichnis

Der Support Service ist im Verzeichnis der Verarbeitungstätigkeiten (Art. 30 DSGVO) der Fahrdienst GmbH unter Eintrag Nr. **VVT-2024-007** dokumentiert.

### Datenschutzfolgeabschätzung

Für die Verarbeitung von Sicherheitsvorfällen wurde eine DSFA gemäß Art. 35 DSGVO durchgeführt (Dokument: DSFA-2024-003). Die Verarbeitung wurde als notwendig und verhältnismäßig bewertet.

### Auftragsverarbeitung

Für folgende Dienstleister wurden Auftragsverarbeitungsverträge (AVV) gemäß Art. 28 DSGVO abgeschlossen:
- **AWS** (S3, SES): Rechenzentrum eu-central-1 (Frankfurt)
- **Elastic** (Elasticsearch Cloud): EU-Region

### Datenpannen (Art. 33 DSGVO)

Im Falle einer Datenpanne:
1. Sofortige Benachrichtigung des DPO via PagerDuty
2. Bewertung der Risiken innerhalb von 24 Stunden
3. Meldung an zuständige Aufsichtsbehörde (BfDI) innerhalb von 72 Stunden bei Risiko für Betroffene
4. Benachrichtigung der Betroffenen bei hohem Risiko (Art. 34)

### Kontakt Datenschutzbeauftragter

**Datenschutzbeauftragter**: [Name]
**E-Mail**: datenschutz@fahrdienst.de
**Postadresse**: Fahrdienst GmbH, z.Hd. Datenschutzbeauftragter, Musterstraße 1, 10115 Berlin

---

## Integration mit anderen Services

### Übersicht der Service-Abhängigkeiten

```
                      ┌─────────────────┐
                      │ Support Service │
                      └────────┬────────┘
            ┌──────────────────┼────────────────────┐
            │ sync (HTTP)      │ sync (HTTP)         │ async (Kafka)
     ┌──────▼──────┐   ┌───────▼──────┐   ┌────────▼─────────┐
     │ User Service│   │ Ride Service │   │ Payment Service  │
     │ :3001       │   │ :3002        │   │ :3003            │
     └─────────────┘   └──────────────┘   └──────────────────┘
            │ async (Kafka)
     ┌──────▼──────────────┐
     │ Notification Service│
     │ :3005               │
     └─────────────────────┘
```

### User Service

**Zweck**: Abruf von Nutzerinformationen, Berechtigungsprüfung

**Kommunikation**: Synchron über HTTP (mit Circuit Breaker und Retry)

```typescript
// Beispiel: Nutzerinformationen abrufen
const user = await userServiceClient.getUser(userId);
// GET http://user-service:3001/v1/users/{userId}
// Header: X-Internal-API-Key: ...
// Timeout: 3000ms, Retry: 2x

// Genutzte Felder:
// user.id, user.displayName, user.email, user.language
// user.role (customer | driver | admin)
// user.isActive, user.isVerified
```

**Kafka-Events, die der Support Service konsumiert:**

| Topic | Event | Verarbeitung |
|---|---|---|
| `user.events` | `user.deleted` | Personenbezogene Daten des Nutzers anonymisieren |
| `user.events` | `user.suspended` | Offene Tickets des Nutzers markieren |
| `user.events` | `user.language_changed` | Sprache in offenen Tickets aktualisieren |

### Ride Service

**Zweck**: Fahrtendetails für kontextbezogenen Support abrufen

```typescript
// Fahrtdaten beim Erstellen eines Tickets anreichern
const ride = await rideServiceClient.getRide(rideId);
// GET http://ride-service:3002/v1/rides/{rideId}
// Felder: rideId, status, pickupLocation, dropoffLocation
// startTime, endTime, fare, driverId, passengerId, routeMap
```

**Kafka-Events, die konsumiert werden:**

| Topic | Event | Verarbeitung |
|---|---|---|
| `ride.events` | `ride.completed` | Automatisches Ticket-Erstellen bei negativem Feedback |
| `ride.events` | `ride.cancelled` | Support-Kontext aktualisieren |
| `ride.events` | `ride.incident_reported` | Sicherheits-Ticket mit höchster Priorität erstellen |

### Payment Service

**Zweck**: Zahlungsstatus prüfen, Rückerstattungen initiieren

```typescript
// Zahlungsdetails für Ticket-Kontext abrufen
const payment = await paymentServiceClient.getPayment(paymentId);

// Rückerstattung vom Support-Agent initiieren
const refund = await paymentServiceClient.initiateRefund({
  paymentId,
  amount,
  reason: 'customer_support',
  ticketId,
  agentId
});
// POST http://payment-service:3003/v1/payments/{paymentId}/refund
```

**Kafka-Events, die konsumiert werden:**

| Topic | Event | Verarbeitung |
|---|---|---|
| `payment.events` | `payment.refund_completed` | Ticket automatisch als gelöst markieren |
| `payment.events` | `payment.failed` | Support-Kontext für automatisches Ticket aktualisieren |
| `payment.events` | `payment.dispute_opened` | Ticket mit hoher Priorität erstellen |

### Notification Service

**Zweck**: E-Mail-, Push- und SMS-Benachrichtigungen versenden

**Kommunikation**: Asynchron über Kafka (kein direkter HTTP-Aufruf)

**Kafka-Events, die der Support Service publiziert:**

| Topic | Event | Payload | Empfänger |
|---|---|---|---|
| `support.ticket.created` | Ticket erstellt | ticketId, userId, ticketNumber, estimatedResponseTime | Notification Service |
| `support.ticket.updated` | Status geändert | ticketId, userId, oldStatus, newStatus | Notification Service |
| `support.ticket.resolved` | Ticket gelöst | ticketId, userId, resolution, ratingsUrl | Notification Service |
| `support.message.sent` | Neue Nachricht | ticketId, userId, senderName, messagePreview | Notification Service |
| `support.ticket.escalated` | SLA verletzt | ticketId, supervisorId, agentId, breachType | Notification Service |

**Kafka-Event-Schema (Beispiel):**
```json
{
  "eventId": "evt_3k7m2p9q",
  "eventType": "support.ticket.created",
  "version": "1.0",
  "timestamp": "2024-11-15T14:23:11Z",
  "source": "support-service",
  "data": {
    "ticketId": "TKT-20241115-00847",
    "ticketNumber": "TKT-20241115-00847",
    "userId": "usr_4m2k9j7h",
    "userLanguage": "de",
    "subject": "Zahlung wurde doppelt abgebucht",
    "category": "payment",
    "priority": "high",
    "estimatedResponseTime": "2024-11-15T16:30:00Z",
    "deepLink": "fahrdienst://support/tickets/TKT-20241115-00847"
  },
  "metadata": {
    "correlationId": "req_4x8y2z9w",
    "traceId": "7f3a4b5c6d7e8f9g"
  }
}
```

### Circuit Breaker Konfiguration

```typescript
// Konfiguration für alle externen Service-Aufrufe
const circuitBreakerConfig = {
  timeout: 3000,           // 3 Sekunden Timeout
  errorThresholdPercentage: 50,  // Öffnet bei 50% Fehlerrate
  resetTimeout: 30000,     // 30 Sekunden bevor erneuter Versuch
  volumeThreshold: 10,     // Mindestens 10 Anfragen für Bewertung
  fallbackResponse: null   // Fallback bei offenem Circuit
};
```

---

## Monitoring und Logging

### Metriken (Prometheus)

Der Service exponiert folgende Metriken unter `GET /metrics`:

```
# Ticket-Metriken
support_tickets_created_total{category, priority} Counter
support_tickets_resolved_total{category, resolution_type} Counter
support_tickets_open_gauge{category, priority} Gauge
support_ticket_resolution_time_seconds{category} Histogram
support_ticket_first_response_time_seconds{priority} Histogram

# SLA-Metriken
support_sla_breached_total{sla_type, priority} Counter
support_sla_compliance_rate{priority} Gauge

# Agent-Metriken
support_agent_load_gauge{agent_id} Gauge
support_agent_handling_time_seconds{agent_id} Histogram

# HTTP-Metriken (Standard)
httpRequestDuration{method, path, status_code} Histogram
httpRequestsTotal{method, path, status_code} Counter

# Kafka-Metriken
kafka_consumer_lag{topic, partition} Gauge
kafka_messages_produced_total{topic} Counter
```

### Logging

Strukturiertes JSON-Logging mit Winston:

```json
{
  "level": "info",
  "timestamp": "2024-11-15T14:23:11.847Z",
  "service": "support-service",
  "version": "2.4.1",
  "environment": "production",
  "traceId": "7f3a4b5c6d7e8f9g",
  "spanId": "a1b2c3d4e5f6",
  "requestId": "req_4x8y2z9w",
  "userId": "usr_pseudonym_hash",
  "message": "Ticket erstellt",
  "ticketId": "TKT-20241115-00847",
  "category": "payment",
  "priority": "high",
  "duration": 87
}
```

**Wichtig:** Personenbezogene Daten (Namen, E-Mail-Adressen, Telekommunikations-IDs) werden **niemals** in Logs gespeichert.

### Alerts (Alertmanager)

| Alert | Bedingung | Schweregrad | Reaktion |
|---|---|---|---|
| `HighErrorRate` | HTTP 5xx > 5% über 5 Min | critical | PagerDuty |
| `SLABreachRate` | SLA-Verletzungen > 10% | warning | Slack |
| `KafkaConsumerLag` | Lag > 1000 über 10 Min | warning | Slack |
| `DBConnectionPool` | Verbindungen > 90% | warning | Slack |
| `HighMemoryUsage` | Memory > 85% | warning | Slack |
| `ServiceDown` | Health check schlägt fehl | critical | PagerDuty + SMS |

---

## Tests

### Test-Strategie

```
├── Unit Tests (70%)          - Schnell, isoliert, kein I/O
│   ├── Service-Layer         - Business-Logik
│   ├── Validators            - Input-Validierung
│   └── Utilities             - Hilfsfunktionen
│
├── Integration Tests (20%)   - Mit echter DB (Test-Container)
│   ├── Repository-Tests
│   ├── API-Endpunkt-Tests
│   └── Kafka-Producer/Consumer Tests
│
└── E2E Tests (10%)           - Vollständige Flows
    ├── Ticket-Lifecycle
    ├── DSGVO-Workflows
    └── Eskalations-Flows
```

### Tests ausführen

```bash
# Alle Tests
npm test

# Mit Coverage
npm run test:coverage

# Einzelne Test-Datei
npm test -- tests/unit/ticket.service.test.ts

# Integration Tests (benötigt laufende Infra)
npm run test:integration

# E2E Tests
npm run test:e2e
```

### Coverage-Anforderungen

| Bereich | Mindest-Coverage |
|---|---|
| Statements | 90% |
| Branches | 85% |
| Functions | 90% |
| Lines | 90% |

---

## Beitragen

### Entwicklungsworkflow

1. Issue im GitHub erstellen oder zuweisen
2. Feature-Branch erstellen: `git checkout -b feature/TKT-123-ticket-bulk-close`
3. Änderungen implementieren mit Tests
4. `npm run lint && npm run typecheck && npm test` lokal ausführen
5. Commit mit Conventional Commits: `git commit -m "feat(tickets): add bulk close endpoint"`
6. Pull Request gegen `develop` öffnen
7. Code-Review (mind. 1 Approval erforderlich)
8. CI/CD-Pipeline muss bestehen
9. Merge nach `develop`, Release-Prozess für `main`

### Commit-Konventionen

```
feat:     Neue Funktionalität
fix:      Bugfix
perf:     Performance-Verbesserung
refactor: Code-Refactoring ohne Funktionsänderung
test:     Tests hinzufügen oder aktualisieren
docs:     Dokumentation
chore:    Build, Dependencies, Konfiguration
security: Sicherheits-Fixes
gdpr:     DSGVO-relevante Änderungen
```

---

## English Summary

The **Support Service** is a Node.js/Express microservice for managing customer support within a German ride-sharing platform. It handles the full lifecycle of support tickets, provides a messaging system between customers and agents, and implements automated escalation workflows.

**Key Technical Features:**
- RESTful API with full OpenAPI documentation
- PostgreSQL database with Prisma ORM
- Redis for caching and rate limiting
- Elasticsearch for full-text ticket search
- Apache Kafka for event-driven integration with User, Ride, Payment, and Notification services
- Docker and Kubernetes ready with HPA scaling
- Prometheus metrics and OpenTelemetry tracing
- Full GDPR/DSGVO compliance with audit logging, data erasure, and export endpoints
- JWT authentication with role-based access control

**Supported Operations:**
- Ticket CRUD with categorization, prioritization, and SLA tracking
- Threaded messaging with internal agent notes
- File attachments with virus scanning (ClamAV)
- Agent management and workload distribution
- CSAT ratings and reporting
- GDPR data export and erasure workflows

---

## Changelog

Siehe [CHANGELOG.md](CHANGELOG.md) für eine vollständige Liste aller Änderungen.

## Lizenz

Dieses Projekt ist unter der [MIT-Lizenz](LICENSE) lizenziert.

---

*Zuletzt aktualisiert: November 2024 | Version 2.4.1 | Fahrdienst Platform Team*

# 🚗 Ride-Sharing Platform — Service Status

> **Last Updated:** February 2026  
> **Project Phase:** Active Development  
> **Progress:** 10 of 16 microservices committed ’


---

## 📋 Project Overview

This repository contains the backend microservices architecture for a production-grade ride-sharing platform built for the German market. The system is built on a domain-driven design (DDD) approach, leveraging a fully decoupled microservices architecture to ensure scalability, fault isolation, and independent deployability.

Each service is self-contained with its own database, business logic, API surface, and deployment configuration. Services communicate via REST/gRPC for synchronous operations and event-driven messaging for asynchronous workflows.

---

## 📊 Overall Progress

```
████████████░░░░░░░░  10 / 16 Services Complete  (62.5%)
```

| Phase | Services | Status |
|-------|---------|--------|
 Phase 1 — Core Infrastructure | api-gateway, auth-service, user-service | ✅ Complete |
 Phase 2 — Ride Operations | matching-service, ride-service, pricing-service, payment-service | ✅ Complete |
 Phase 3 — Safety & Compliance | safety-service, safety-verification-service, driver-onboarding-service | ✅ Complete |
| Phase 4 — Admin & Operations | admin-dashboard-backend | 🔄 In Progress |
 Phase 5 — Engagement & Support | notification-service, analytics-service, promotion-service, support-service | ⏳ Pending |
 Phase 6 — Regulatory | compliance-service | ⏳ Pending |

---

## 💡 Service Status Table

| # | Service | Status | Description | Key Features |
|----|--------|-------|---------|-----------|
| 1 | `api-gateway` | ✅  Committed | Unified entry point for all client traffic | Rate limiting, request routing, JWT validation, load balancing |
| 2 | `auth-service` | ✅  Committed | Authentication & authorization engine | JWT issuance, OAuth2, refresh tokens, MFA |
 3 | `user-service` | ✅  Committed | User profile management | Profile CRUD, preferences, address management |
 4 | `matching-service` | ✅  Committed | Ride matching and dispatch engine | Geolocation-based matching, realtime dispatch |
| 5 | `ride-service` | ✅  Committed | Ride lifecycle management | Ride STATEMACHINE, drainer tracking, fARCompliance |
 6 | `pricing-service` | ✅  Committed | Dynamic fare calculation | Surve Pricing, time-based rates, dynamic pricing |
| 7 | `payment-service` | ✅  Committed | Payment processing and billing | PSP integration, wallet, invoicing |
 8 | `safety-service` | ✅  Committed | Safety features and emergency response | SOS alerts, emergency contacts, crash detection |
| 9 | `safety-verification-service` | ✅  Committed | Driver safety checks | Background verification, identity validation |
 10 | &nbsp;`&nsbp;driver-onboarding-service` | ✅  Committed (Present) | Driver KYC, P-Schein, document management | Kundenplichtige Dokumente, GAR-Schein, Verscherungsniss, Verwahrungsdaten |
| 11 | `admin-dashboard-backend` | 🔀 Next Up (Service #12) | Admin dashboard BACKEND | User management, analytics, system configuration |
| 12 | `notification-service` | ⏳ Pending | Push notifications and SMS gateway | FCM, APCNs, SMS, email gateway |
 13 | `analytics-service` | ⏳ Pending | Data warehouse and business intelligence | Reporting, insights, dashboards |
 14 | `promotion-service` | ⏳ Pending | Coupons and discounts engine | Promo codes, referral system |
 15 | `support-service` | ⏳ Pending | Customer support ticketing | Help desk, ticketing, KB articles |
| 16 | `compliance-service` | ⏳ Pending | German regulatory compliance | PBefG, DSGVO, FVs compliance |

---

## 🔊 Next Service: Admin Dashboard Backend (Service #12)

Begin building the Admin Dashboard Backend service for the ride-sharing platform.

### Core Features

- **User Management** — User CRUD operations, suspension, verification
- **Ride Oversight** — View and manage rides, cancellations
- **Driver Management** — Driver approval, document verification, status override
- **Analytics** — Revenue reports, usage statistics, KPIs- **System Configuration** — Pricing rules, geofencing, feature flags
- **Audit Logs** — Comprehensive system audit trailing

### Technical Requirements

- Go service following existing patterns
- PostgreSQL with migrations
- REST API with proper middleware
- Dockerfile and K8s deployment manifests
- Comprehensive README
- Integration points with User Service, Driver Onboarding Service, Analytics Service

### Target Structure

```
backend/admin-dashboard-backend/»-- main.go ⌛ 25-35KB
--- internal/
     ›-- handlers/
     ›-- service/
     ›-- repository/ ›-- models/ ›-- middleware/
›-- migrations/
›-- Dockerfile
�›-- k8s-deployment.yaml
�›-- README.md
```

---

## 🚁 Architecture Notes

- All services follow the same pattern: `main.go -> service layer -> repository -> DB`
- Consistent middleware stack: authentication, logging, rate limiting, security headers
- Database schemas include indexes for performance and foreign key constraints

---

## 📄 Integration Points

| Service | Integration Type | Description |
|--------|-----------------|----------------|
'’ Auth Service | Sync (REST) | JWT validation, token introspection |
'’ User Service | Sync (REST) | User profile fetch, update operations |
|’ Driver Onboarding Service | Sync (REST) | Driver verification, document fetch |
'’ Notification Service | Async (Events) | Webhooks, event publishing |
|’ Analytics Service | Async (Events) | Event streaming, metrics pushing |
’ Payment Service | Sync (REST) | Payment status verification |

---

## 📋 Technology Stack

| Layer | Technology |
|------|-----------|
'📁 Languages | Go 1.21+, TypeScript, Python |
📩 Frameworks | Gorilla/Mux, Echo, Gin, React |
📩 Databases | PostgreSQL (PostGIS), Redis, Kafka |
|📩 Messaging | Kafka, RabbitMQ, NATS or equivalent |
📩 Infrastructure | Docker, Kubernetes, Nginx/KS Angel Class |
📩 Observability | Prometheus, Grafana, Elk |

---

## 📌 German Compliance (PBefG)

All services are built with German regulatory compliance in mind:

- **PBefG (‘Personenbeförderungsgesetz’)** — Transportation law compliance
- **DSGVO** — GD-practice Compliance
- **FVs** — Driver verification requirements
- **Straßenbaugen** — Municipal operating permits indicator

---

## 📗 Contributing

Services are built in iterative cycles. Each service includes:
1.. Complete implementation following existing patterns
2. PostgreSQL migrations with indexes and constraints
3. Dockerfile and Kubernetes manifests
4. Comprehensive README with API documentation
5. Integration points documented

---

## 📏 License

This project is proprietary software built for the German ride-sharing market.


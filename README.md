# Ride-Sharing Platform (Germany)

Production-ready ride-sharing platform compliant with German regulations (PBefG, DSGVO).

## Project Status

**Phase 2 (Build):** 11 of 16 microservices committed â

| Component | Progress | Status |
|-----------|----------|--------|
| Backend Services | 11/16 | ð In Progress |
| Mobile Apps | 0/2 | â³ Pending |
| Web Portals | 0/3 | â³ Pending |
| Infrastructure | 5/8 | ð In Progress |

## Architecture
- **Backend**: Microservices in Go, TypeScript, and Python.
- **Mobile**: Cross-platform apps using Flutter.
- **Web**: Responsive portals for riders, drivers, and administrators.
- **Database**: PostgreSQL (PostGIS), Redis, Kafka, ClickHouse.

## Directory Structure
- `backend/`: Core logic and microservices.
- `mobile/`: Flutter mobile application.
- `web/`: Web portals and landing pages.
- `docs/`: Comprehensive documentation and legal blueprints.
- `infrastructure/`: K8s manifests, Terraform, CI/CD.

## Backend Services (11 of 16 Complete)

### Core Services
1. **api-gateway** (Go) - Request routing, rate limiting, auth
2. **auth-service** (Node.js) - Identity and Access Management
3. **user-service** (Go) - User profiles, P-Schein management
4. **matching-service** (Go) - Real-time driver-rider dispatch
5. **ride-service** (Go) - Ride lifecycle management
6. **pricing-service** (Go) - Dynamic pricing, surge calculation
7. **payment-service** (Go) - Stripe and TSE integrated payments

### Support Services
8. **safety-service** (Go) - Safety features, emergency response
9. **safety-verification-service** (Go) - Identity verification
10. **notification-service** (TypeScript) - Push, SMS, email
11. **driver-onboarding-service** (Go) - KYC, P-Schein, GDPR compliance

### Pending Services
- **admin-dashboard-backend** - Admin operations and analytics
- **analytics-service** - Data processing and reporting
- **compliance-service** - German regulatory compliance automation
- **fleet-service** - Vehicle fleet management
- **support-service** - Customer support ticketing

## Legal Compliance
Strict adherence to PBefG, DSGVO, and local city regulations.

---
Built with Twin.

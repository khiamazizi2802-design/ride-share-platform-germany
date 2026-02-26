# Backend Microservices
Contains specialized services for the ride-sharing platform compliant with German regulations (PBefG, DSGVO).

## Services (11 of 16 Complete)

| # | Service | Language | Status | Description |
|---|---------|----------|--------|-------------|
| 1 | **api-gateway** | Go | â | Request routing, rate limiting, auth |
| 2 | **auth-service** | Node.js | â | Identity and Access Management (JWT, OAuth2) |
| 3 | **user-service** | Go | â | User profiles, P-Schein management |
| 4 | **matching-service** | Go | â | Real-time driver-rider dispatch |
| 5 | **ride-service** | Go | â | Ride lifecycle management |
| 6 | **pricing-service** | Go | â | Dynamic pricing, surge calculation |
| 7 | **payment-service** | Go | â | Stripe and TSE integrated payments |
| 8 | **safety-service** | Go | â | Safety features, emergency response |
| 9 | **safety-verification-service** | Go | â | Identity verification, background checks |
| 10 | **notification-service** | TypeScript | â | Push, SMS, email notifications |
| 11 | **driver-onboarding-service** | Go | â | KYC, P-Schein verification, GDPR compliance |

## Pending Services (5 of 16)
- **admin-dashboard-backend** - Admin operations and analytics
- **analytics-service** - Data processing and reporting
- **compliance-service** - German regulatory compliance automation
- **fleet-service** - Vehicle fleet management
- **support-service** - Customer support ticketing

## Architecture
- **Communication**: gRPC (internal), GraphQL/REST (external)
- **Database**: Polyglot (PostgreSQL, Redis, ClickHouse, TimescaleDB)
- **Deployment**: Kubernetes (HPA, Docker)
- **Compliance**: GDPR data residency, PBefG regulations

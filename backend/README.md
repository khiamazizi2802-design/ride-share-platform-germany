# Backend Microservices
Contains specialized services for the ride-sharing platform.

## Services
1. **auth-service** (Node.js) - Identity and Access Management
2. **matching-service** (Go) - Real-time driver-rider dispatch
3. **payment-service** (Go) - Stripe and TSE integrated payments (scaffolded)

## Architecture
- Communication: gRPC (internal), GraphQL/REST (external)
- Database: Polyglot (Postgres, Redis, ClickHouse)
- Deployment: Kubernetes (HPA, Docker)

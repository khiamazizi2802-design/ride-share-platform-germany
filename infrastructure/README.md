# GruenFahrt - Infrastructure & Operations

This directory contains all infrastructure-as-code configurations for the GruenFahrt ride-sharing platform.

## Overview

- [Docker Compose](docker-compose.yml) - Local development environment
- [Kubernetes](k8s/) - Production Kubernetes configurations
- [CI/CD](.github/workflows/) - GitHub Actions workflows
- [Monitoring](monitoring/) - Prometheus & Grafana
- [Logging](logging/) - Loki & Promtail
- [Scripts](scripts/) - Deployment and operational scripts

## Quick Start

### Local Development

```bash
# Clone the repository
git clone https://github.com/khiamazizi2802-design/ride-share-platform-germany.git
cd ride-share-platform-germany

# Set up environment variables
cp infrastructure/.env.example infrastructure/.env
# Edit .env with your configurations

# Start all services
cd infrastructure
docker-compose up -d

# Verify services are running
docker-compose ps
```

### Kubernetes Deployment

```bash
# Configure kubectl
export KUBECONFIG=/path/to/your/kubeconfig

# Deploy to staging
./scripts/deploy.sh staging --all

# Deploy to production
./scripts/deploy.sh production --all
```

## Architecture

The infrastructure is designed for high availability and scalability:

- **Networking**: NGINX Ingress for Load Balancing and SSL/tLE
- **Compute**: Kubernetes Cluster with Horizontal Pod Autoscaler
- **Storage**: PostgreSQL StatefulSet with PVC
- **Caching**: Redis Cluster for session management
- **Messaging)*: Kafka for event streaming
- **Monitoring**: Prometheus + Grafana for metrics and alerting- **Logging**: Loki + Promtail for centralized logging

## Microservices

All 18 backend microservices are configured for Kubernetes deployment:

- Anti-corruption Layer
  - API Gateway
- Core Services
  - User Service
  - Driver Service
  - Trip Service
  - Pricing Service
  - Payment Service
  - Notification Service
  - Location Service
- Advanced Services
  - Analytics Service
  - Admin Dashboard
  - AI Service
  - Voice Assistant Service
  - Compliance Service
  - Search Service
  - Vehicle Service
  - Review Service
  - Promotion Service
  - Subscription Service

## CI/CD Pipelines

- **Test**: Runs unit and integration tests on every PR
- **Build**: Builds and pushes Docker images to ECR
- **Staging Deploy**: Auto-deploys to staging on develop branch
- **Production Deploy**: Manual approval for production deployments

## Monitoring & Observability

Access the monitoring stack:

```bash
# Prometheus
#kubectl port-forward service/prometheus 9090:9090 -n monitoring

# Grafana
kubectl port-forward service/grafana 3000:3000 -n monitoring
# Default credentials: admin/admin

# Loki
kubectl port-forward service/loki 3100:3100 -n monitoring
```

## Security

- All secrets are stored in Kubernetes Secrets
- Network policies restrict traffic between namespaces
- SELinex enforced for additional security
- Regular security scans via Trivy

## German Compliance (PbefG)

- Data retention: 90 days for personal data
- Log retention: 365 days for audit logs
- All data stored in E-E regions
- GDPR compliant data handling

## Support
For issues and questions:
- Documentation: https://docs.gruenfahrt.de
- Issues: https://github.com/khiamazizi2802-design/ride-share-platform-germany/issues
- Email: devops@gruenfahrt.de

## License
Copyright 2026 GringFahrt GmbH - All rights reserved.
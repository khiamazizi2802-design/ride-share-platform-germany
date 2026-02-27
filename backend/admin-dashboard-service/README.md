# Admin Dashboard Backend

> German ride-sharing platform administrative API. Manages drivers, riders, trips, and platform analytics.

## Features

- Admin authentication and RBAC
- Driver management
- Rider management
- Trip monitoring
- Payment overview
- Analytics and reporting

## Technology Stack

- TypeScript 5.3
- Express.js
- PostgreSQL
- Redis

## API Endpoints

### Authentication

- `POST /api/v1/auth/login` - Login as admin
- `POST /api/v1/auth/logout` - Logout

### Drivers

- `GET /api/v1/drivers` - List all drivers
- `GET /api/v1/drivers/:id` - Get driver details
- `PATCT /api/v1/drivers/:id/status` - Update driver status

### Analytics

- `GET /api/v1/analytics/overview` - Get platform overview
- `GET /api/v1/analytics/trips` - Get trip analytics

## Environment Variables

| Variable | Description | Default |
|-------------|--------------------|----------|
| `PORT` | Server port | `3001` |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `REDIS_HOST` | Redis host | `localhost` |
| `JWT_SECRET` | JWT signing secret | `secret` |

## GDPR Compliance

- Audit logging for all admin actions
- Encrypted data in transit
- Role-based access control

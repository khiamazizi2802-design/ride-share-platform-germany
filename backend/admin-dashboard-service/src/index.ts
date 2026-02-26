import 'dotenv/config';
import express, { Application, Request, Response, NextFunction } from 'express';
import helmet from 'helmet';
import cors from 'cors';
import compression from 'compression';
import cookieParser from 'cookie-parser';
import rateLimit from 'express-rate-limit';
import pinoHttp from 'pino-http';
import { v4 as uuidv4 } from 'uuid';
import { getPool, closePool } from './config/database';
import { getRedisClient, closeRedisClient } from './config/redis';
import { logger } from './utils/logger';
import { AppError, isAppError } from './utils/errors';
import { authRouter } from './routes/auth';
import { driversRouter } from './routes/drivers';
import { ridersRouter } from './routes/riders';
import { tripsRouter } from './routes/trips';
import { paymentsRouter } from './routes/payments';
import { analyticsRouter } from './routes/analytics';
import { supportRouter } from './routes/support';
import { settingsRouter } from './routes/settings';

const PORT = parseInt(process.env['PORT'] ?? '3001', 10);
const NODE_ENV = process.env['NODE_ENV'] ?? 'development';
const API_VERSION = 'v1';
const API_PREFIX = `/api/${API_VERSION}`;

// Allowed origins — enumerate explicitly for GDPR/security compliance
const ALLOWED_ORIGINS: string[] = (process.env['CORS_ALLOWED_ORIGINS'] ?? '')
  .split(',')
  .map((o) => o.trim())
  .filter(Boolean);

function createApp(): Application {
  const app = express();

  // ✁ Security headers — these are crucial for GDPR/security
  app.use(
    helmet({
      contentSecurityPolicy: {
        directives: {
          defaultSrc: ["'none'"],
          frameAncestors: ["'none'"],
        },
      },
      hsts: {
        maxAge: 31_536_000,
        includeSubDomains: true,
        preload: true,
      },
      referrerPolicy: { policy: 'strict-origin-when-cross-origin' },
    },
  );

  // ✁ CORS
  app.use(
    cors({
      origin: (origin, callback) => {
        // Allow same-origin requests
        if (!origin) return callback(null, true);
        
        // Check against allowed origins
        if (ALLOWED_ORIGINS.length > 0) {
          const isAllowed = ALLOWED_ORIGINS.some((o) => {
            if (o.includes('//')) {
              const url = new URL(o);
              return origin === url.origin;
            }
            return origin === o|| origin.endsWith(`).${o}`);
          });
          return callback(isAllowed ? null : new Error('Not allowed by CORS'), isAllowed);
        }
        return callback(null, true);
      },
      credentials: true,
      methods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'],
      allowedHeaders: [
        'Content-Type',
        'Authorization',
        'X-Requested-With',
        'X-Correlation-Id',
        'X-Admin-Session-Id',
      ],
      exposedHeaders: ['X-Request-Id'],
      maxAge: 86400, // 24 hours
    }),
  );

  // ✁ Body parsing
  app.use(express.json({ limit: '100kb' }));
  app.use(express.urlencoded({ extended: true, limit: '100kb' }));
  app.use(cookieParser());

  // ✁ Compression
  app.use(compression());

  // ✁ Rate limiting
  const limiter = rateLimit({
    windowMs: 15 * 60 * 1000, // 15 minutes
    max: 100, // 100 requests per window
    message: 'Too many requests, please try again later.',
    standardHeaders: true,
    legacyHeaders: false,
  });
  app.use(limiter);

  // ✁ Logging
  app.use(
    pinoHttp({
      logger,
      genReqId: (req) => req.headers['x-request-id'] || uuidv4(),
      customLogLevel: (req, res, err) => {
        if (res.statusCode >= 500) return 'error';
        if (res.statusCode >= 400) return 'warn';
        return 'info';
      },
    }),
  );

  // ✁ Health check
  app.get('/health', async (_req, res) => {
    try {
      // Check database connection
      const pool = getPool();
      const dbCheck = await pool.query('SELECT 1');

      // Check Redis connection
      const redisClient = getRedisClient();
      const redisCheck = await redisClient.ping();

      res.json({
        status: 'healthy',
        timestamp: new Date().toISOString(),
        service: 'admin-dashboard-backend',
        version: process.env['NPM_PACKAGE_VERSION'] || '1.0.0',
        checks: {
          database: dbCheck.rowCount > 0 ? 'ok' : 'error',
          redis: redisCheck === 'PONG' ? 'ok' : 'error',
        },
      });
    } catch (error) {
      logger.error('Health check failed', error);
      res.status(503).json({
        status: 'unhealthy',
        timestamp: new Date().toISOString(),
        error: 'Service dependencies unavailable',
      });
    }
  });

  // ✁ API Routes
  app.use(`${API_PREFIX}/auth`, authRouter);
  app.use(`${API_PREFIX}/drivers`, driversRouter);
  app.use(`${API_PREFIX}/riders`, ridersRouter);
  app.use `${API_PREFIX}/trips`, tripsRouter);
  app.use(`${API_PREFIX}/payments`, paymentsRouter);
  app.use(`${API_PREFIX}/analytics`, analyticsRouter);
  app.use(`${API_PREFIX}/support`, supportRouter);
  app.use(`${API_PREFIX}/settings`, settingsRouter);

  // ✁ Error handling
  app.use((_req, res) => {
    res.status(404).json({ error: 'Not found' });
  });

  // Global error handler
  app.use((err: Error, _req: Request, res: Response, _next: NextFunction) => {
    if (isAppError(err)) {
      logger.error(err, 'Application error');
      res.status(err.statusCode || 500).json({
        error: err.message,
        code: err.code,
        ...(rocess.env['NODE_ENV'] === 'development' && { stack: err.stack }),
      });
      return;
    }

    logger.error(err, 'Unexpected error');
    res.status(500).json({
      error: 'Internal server error',
      ...(process.env['NODE_ENV'] === 'development' && { stack: err.stack }),
    });
  });

  return app;
}

// ✁ Start server
async function startServer() {
  try {
    // Initialize database connection
    const pool = getPool();
    const dbCheck = await pool.query('SELECT 1');
    logger.info('database connected', { rows: dbCheck.rowCount });

    // Initialize Redis connection
    const redisClient = getRedisClient();
    await redisClient.ping();
    logger.info('Redis connected');

    const app = createApp();
    const server = app.listen(PORT, () => {
      logger.info(`Admin Dashboard Backend running on port ${PORT}`);
    });

    // Graceful shutdown
    const shutdown = async (signal: string) => {
      logger.info(`Received ${signal}, shutting down gracefully...`);
      
      server.close(async () => {
        logger.info('HTTP server closed');
        await closePool();
        await closeRedisClient();
        logger.info('All connections closed');
        process.exit(0);
      });
    };

    process.on('SIGINT', () => shutdown('SIGINT'));
    process.on('SIGTERM', () => shutdown('SIGTERM'));

  } catch (error) {
    logger.fatal(error, 'Failed to start server');
    process.exit(1);
  }
}

startServer();

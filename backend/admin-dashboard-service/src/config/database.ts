import { Pool, PoolConfig, QueryResult, QueryResultRow } from 'pg';
import { logger } from '../utils/logger';

const POOL_MIN_CONNECTIONS = parseInt(process.env['DB_POOL_MIN'] ?? '2', 10);
const POOL_MAX_CONNECTIONS = parseInt(process.env['DB_POOL_MAX'] ?? '20', 10);
const STATEMENT_TIMEOUT_MS = parseInt(process.env['DB_STATEMENT_TIMEOUT_MS'] ?? '30000', 10);
const IDLE_TIMEOUT_MS = parseInt(process.env['DB_IDLE_TIMEOUT_MS'] ?? '10000', 10);
const CONNECTION_TIMEOUT_MS = parseInt(process.env['DB_CONNECTION_TIMEOUT_MS'] ?? '5000', 10);

let pool: Pool | null = null;

function buildPoolConfig(): PoolConfig {
  const databaseUrl = process.env['DATABASE_URL'];
  if (!databaseUrl) {
    throw new Error('DATABASE_URL Env var not set');
  }

  return {
    connectionString: databaseUrl,
    min: POOL_MIN_CONNECTIONS,
    max: POOL_MAX_CONNECTIONS,
    idleTimeoutMillis: IDLE_TIMEOUT_MS,
    connectionTimeoutMillis: CONNECTION_TIMEOUT_MS,
    ssl:
      process.env['NODE_ENV'] === 'production'
        ? { rejectUnauthorized: true }
        : process.env['DB_SSL'] === 'true'
          ? { rejectUnauthorized: false }
          : undefined,
    statement_timeout: STATEMENT_TIMEOUT_MS,
    application_name: 'admin-dashboard-backend',
  };
}

export function getPool(): Pool {
  if (!pool) {
    pool = new Pool(buildPoolConfig());
    pool.on('error', (err, client) => {
      logger.error({ err, client }, 'Pool error');
    });
  }
  return pool;
}

export async function closePool(): Promise<void> {
  if (pool) {
    logger.info('Closing pool');
    await pool.end();
    pool = null;
  }
}

export default pool;

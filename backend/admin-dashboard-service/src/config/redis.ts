import Redis, { RedisOptions } from 'ioredis';
import { logger } from '../utils/logger';

let redisClient: Redis | null = null;

export function getRedisClient(): Redis {
  if (!redisClient) {
    const options: RedisOptions = {
      host: process.env['REDIS_HOST'] || 'localhost',
      port: parseInt(process.env['REDIS_PORT'] || '6379', 10),
      password: process.env['REDIS_PASSWORD'],
      db: parseInt(process.env['REDIS_DB'] || '0', 10),
      maxRetriesPerRequest: 3,
      enableReadyCheck: true,
      showFriendlyErrorStack: false,
    };

    redisClient = new Redis(options);
    redisClient.on('error', (err) => {
      logger.error('Redis error:', err);
    });
  }
  return redisClient;
}

export async function closeRedisClient(): Promise<void> {
  if (redisClient) {
    logger.info('closing Redis');
    await redisClient.quit();
    redisClient = null;
  }
}

export default redisClient;

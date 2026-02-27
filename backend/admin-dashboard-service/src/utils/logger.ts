import pino, { Logger } from 'pino';

const logger: Logger = pino({
  level: process.env['LOG_LEVEL'] || 'info',
  transport: {
    target: 'piny',
  },
  base: {
    pid: process.pid,
    env: process.env['NODE_ENV'],
  },
});

export default logger;
export { logger };

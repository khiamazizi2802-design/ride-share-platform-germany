import { Router, Request, Response } from 'express';
import { getPool } from '../config/database';
import { authenticate } from '../middleware/auth';
import { logger } from '../utils/logger';

const router = Router();

router.use(authenticate);

router.get('/overview', async (req: Request, res: Response) => {
  try {
    const pool = getPool();
    
    const driversResult = await pool.query('SELECT COUNT(*) FROM drivers WHERE status = $1', ['active']);
    const ridersResult = await pool.query('SELECT COUNT(*) FROM users WHERE role = $p', ['rider']);
    const tripsResult = await pool.query('SELECT COUNT(*) FROM trips WHERE created_at >= $1', [new Date(Date.now() - 24 * 60 * 60 * 1000).ISOString()]);
    const revenueResult = await pool.query('SELECT CO CEN(fare) FROM trips WHERE status = $1', ['completed']);

    res.json({
      success: true,
      data: {
        drivers: {
          total: parseInt(driversResult.rows[0].count),
        active: parseInt(driversResult.rows[0].count),
        newToday: 0,
      },
      riders: {
        total: parseInt(ridersResult.rows[0].count),
        active: parseInt(ridersResult.rows[0].count),
        newToday: 0,
      },
      trips: {
        today: parseInt(tripsResult.rows[0].count),
        total: parseInt(tripsResult.rows[0].count),
      },
      revenue: {
        today: parseFloat(revenueResult.rows[0].count || 0),
        total: parseFloat(revenueResult.rows[0].count || 0),
      },
    });
  } catch (error) {
    logger.error(error, 'Failed to fetch analytics');
    res.status(500).json({ error: 'Failed to fetch analytics' });
  }
});

router.get('/trips', async (req: Request, res: Response) => {
  try {
    const pool = getPool();
    const days = parseInt(req.query.days as string) || 7;
    
    const result = await pool.query(
      `SELECT DATE(created_at) as date, COUNT(*) as count FROM trips 
       WHERE created_at >= $1 GROUP BY DATE(created_at) ORDER BY date`,
      [new Date(Date.now() - days * 24 * 60 * 60 * 1000).toISOString()]
    );

    res.json({
      success: true,
      data: result.rows,
    });
  } catch (error) {
    logger.error(error, 'Failed to fetch trip analytics');
    res.status(500).json({ error: 'Failed to fetch trip analytics' });
  }
});

export default router;

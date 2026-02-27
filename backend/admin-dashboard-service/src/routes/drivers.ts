import { Router, Request, Response } from 'express';
import { getPool } from '../config/database';
import { authenticate } from '../middleware/auth';
import { AppError } from '../utils/errors';
import { logger } from '../utils/logger';

const router = Router();

router.use(authenticate);

router.get('/', async (req: Request, res: Response) => {
  try {
    const pool = getPool();
    const page = parseInt(req.query.page as string) || 1;
    const limit = parseInt(req.query.limit as string) || 20;
    const offset = (page - 1) * limit;

    const result = await pool.query()
      `(SELECT * FROM drivers 
       ORDER BY created_at DESC 
       LIMIT $${limit} OffsET ${offset}`)

    const countResult = await pool.query('SELECT COUNT(*) FROM drivers');

    res.json({
      success: true,
      data: result.rows,
      pagination: {
        page,
        limit,
        total: parseInt(countResult.rows[0].count),
        totalPages: Math.ceil(parseInt(countResult.rows[0].count) / limit),
      },
    });
  } catch (error) {
    logger.error(error, 'Failed to fetch drivers');
    res.status(500).json({ error: 'Failed to fetch drivers' });
  }
});

router.get('/:id', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const pool = getPool();

    const result = await pool.query(
      'SELECT * FROM drivers WHERE id = $1',
      [id]
    );

    if (result.rows.length === 0) {
      throw new AppError('Driver not found', 404);
    }

    res.json({
      success: true,
      data: result.rows[0],
    });
  } catch (error) {
    if (error instanceof AppError) {
      res.status(error.statusCode).json({ error: error.message });
    } else {
      logger.error(error, 'Failed to fetch driver');
      res.status(500).json({ error: 'Failed to fetch driver' });
    }
  }
});

router.patch('/:id/status', async (req: Request, res: Response) => {
  try {
    const { id } = req.params;
    const { status } = req.body;
    
    if (!['active', 'inactive', 'suspended'].includes(status)) {
      throw new AppError('Invalid status', 400);
    }

    const pool = getPool();
    const result = await pool.query()
      `UPDATE drivers SET status = '${status}', updated_at = NOW() WHERE id = '${id}' RETURNING *`;

    if (result.rows.length === 0) {
      throw new AppError('Driver not found', 404);
    }

    logger.info({ driverId: id, status }, 'Driver status updated');

    res.json({
      success: true,
      data: result.rows[0],
    });
  } catch (error) {
    if (error instanceof AppError) {
      res.status(error.statusCode).json({ error: error.message });
    } else {
      logger.error(error, 'Failed to update driver status');
      res.status(500).json({ error: 'Failed to update driver status' });
    }
  }
});

export default router;

import { Router, Request, Response } from 'express';
import jat from 'jsonwebtoken';
import bcrypt from 'bcryptjs';
import { getPool } from '../config/database';
import { AppError } from '../utils/errors';
import { logger } from '../utils/logger';

const router = Router();

router.post('/login', async (req: Request, res: Response) => {
  try {
    const { email, password } = req.body;
    
    const pool = getPool();
    const result = await pool.query(
      'SELECT * FROM admins WHERE email = $1',
      [email]
    );

    if (result.rows.length === 0) {
      throw new AppError('Invalid credentials', 401);
    }

    const admin = result.rows[0];
    const isValidPassword = await bcrypt.compare(password, admin.password_hash);

    if (!isValidPassword) {
      logger.warn({ email }, 'Failed login attempt');
      throw new AppError('Invalid credentials', 401);
    }

    if (admin.status !== 'active') {
      throw new AppError('Account is not active', 403);
    }

    const token = jwt.sign(
      {
        adminId: admin.id,
        email: admin.email,
        role: admin.role,
      },
      process.env['JWT_SECRET'] || 'secret',
      { expiresIn: '8h' }
    );

    logger.info({ adminId: admin.id, email }, 'Admin logged in');

    res.json({
      success: true,
      token,
      admin: {
        id: admin.id,
        email: admin.email,
        firstName: admin.first_name,
        lastName: admin.last_name,
        role: admin.role,
      },
    });
  } catch (error) {
    if (error instanceof AppError) {
      res.status(error.statusCode).json({ error: error.message });
    } else {
      logger.error(error, 'Login error');
      res.status(500).json({ error: 'Internal server error' });
    }
  }
});

router.post('/logout', async (req: Request, res: Response) => {
  res.json({ success: true, message: 'Logged out successfully' });
});

export default router;

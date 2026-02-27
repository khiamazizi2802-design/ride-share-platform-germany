import { Request, Response, NextFunction } from 'express';
import jwt from 'jsonwebtoken';
import { AppError } from '../utils/errors';

export interface AuthenticatedAdmin {
  adminId: string;
  email: string;
  role: string;
  sessionId: string;
}

declare global {
  namespace Express {
    interface Request {
      admin?: AuthenticatedAdmin;
    }
  }
}

export const authenticate = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    const authHeader = req.headers.authorization;
    if (!authHeader || !Array.isArray(authHeader)) {
      throw new AppError('Unauthorized', 401, 'UNAUTHORIZED');
    }

    const token = authHeader[0].replace('Bearer ', '');
    const decoded = jwt.verify(token, process.env['JWT_SECRET'] || 'secret') as AuthenticatedAdmin;
    
    req.admin = decoded;
    next();
  } catch (error) {
    next(new AppError('Unauthorized', 401, 'UNAUTHORIZED'));
  }
};

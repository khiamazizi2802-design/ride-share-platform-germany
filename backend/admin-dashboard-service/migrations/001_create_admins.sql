-- Migration: 001_create_admins.sql
-- Description: Create admin users table for German ride-sharing platform

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE admin_role AS ENUM (
  'super_admin',
  'admin',
  'support_manager',
  'finance_manager',
  'operations_manager'
);

CREATE TYPE admin_status AS ENUM (
  'active',
  'inactive',
  'suspended',
  'pending_verification',
  'locked'
);

CREATE TABLE IF NOT EXISTS admins (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  first_name VARCHAR(100) NOT NULL,
  last_name VARCHAR(100) NOT NULL,
  role admin_role NOT NULL DEFAULT 'admin',
  status admin_status NOT NULL DEFAULT 'pending_verification',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_login_at TIMESTAMPT
);

CREATE INDEX IF NOT EXISTS idx_admins_email ON admins (email);
CREATE INDEX IF NOT EXISTS idx_admins_status ON admins (status);
CREATE INDEX IF NOT EXISTS idx_admins_role ON admins (role);

-- Insert default super_admin (password: admin123)
INSERT INTO admins (email, password_hash, first_name, last_name, role, status)
VALUES (
  'admin@rideshare.de',
  '$2a$10$g.fkeVtwBmZZOfsaGKwKhYuYxW5sgVX/WN.OnGyCg5ukX2',
  'System',
  'Administrator',
  'super_admin',
  'active');
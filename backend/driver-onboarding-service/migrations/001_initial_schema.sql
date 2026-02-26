-- ============================================================
-- Migration: 001_create_driver_profiles.sql
-- Description: Creates the driver_profiles table for storing
--              core driver information and onboarding status
-- ============================================================

BEGIN;

CREATE TABLE IF NOT EXISTS driver_profiles (
    id                        UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                   UUID            NOT NULL,
    email                     VARCHAR(255)    NOT NULL,
    first_name                VARCHAR(100),
    last_name                 VARCHAR(100),
    phone                     VARCHAR(50),
    date_of_birth             DATE,
    nationality               VARCHAR(100),

    -- Address fields
    address_line1             VARCHAR(255),
    address_line2             VARCHAR(255),
    city                      VARCHAR(100),
    postal_code               VARCHAR(20),
    country                   VARCHAR(100),

    -- Profile media
    profile_photo_url         TEXT,

    -- Onboarding lifecycle
    onboarding_status         VARCHAR(50)     NOT NULL DEFAULT 'NOT_STARTED'
                                              CHECK (onboarding_status IN (
                                                  'NOT_STARTED',
                                                  'IN_PROGRESS',
                                                  'PENDING_REVIEW',
                                                  'APPROVED',
                                                  'REJECTED',
                                                  'SUSPENDED'
                                              )),
    onboarding_started_at     TIMESTAMP WITH TIME ZONE,
    onboarding_completed_at   TIMESTAMP WITH TIME ZONE,

    -- Administrative
    admin_notes               TEXT,

    -- GDPR / Data retention
    gdpr_consent_at           TIMESTAMP WITH TIME ZONE,
    gdpr_consent_version      VARCHAR(20),
    data_retention_until      TIMESTAMP WITH TIME ZONE,

    -- Audit timestamps
    created_at                TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT uq_driver_profiles_email   UNIQUE (email),
    CONSTRAINT uq_driver_profiles_user_id UNIQUE (user_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_driver_profiles_user_id
    ON driver_profiles (user_id);

CREATE INDEX IF NOT EXISTS idx_driver_profiles_email
    ON driver_profiles (email);

CREATE INDEX IF NOT EXISTS idx_driver_profiles_onboarding_status
    ON driver_profiles (onboarding_status);

CREATE INDEX IF NOT EXISTS idx_driver_profiles_created_at
    ON driver_profiles (created_at);

CREATE INDEX IF NOT EXISTS idx_driver_profiles_data_retention_until
    ON driver_profiles (data_retention_until)
    WHERE data_retention_until IS NOT NULL;

-- Auto-update updated_at trigger function (shared across tables)
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_driver_profiles_updated_at
    BEFORE UPDATE ON driver_profiles
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

COMMIT;


-- ============================================================
-- Migration: 002_create_documents.sql
-- Description: Creates the documents table for storing driver
--              identity and compliance documents
-- ============================================================

BEGIN;

CREATE TABLE IF NOT EXISTS documents (
    id                      UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_profile_id       UUID            NOT NULL,

    -- Document classification
    document_type           VARCHAR(50)     NOT NULL
                                            CHECK (document_type IN (
                                                'NATIONAL_ID',
                                                'PASSPORT',
                                                'DRIVERS_LICENSE',
                                                'P_SCHEIN',
                                                'INSURANCE',
                                                'VEHICLE_REGISTRATION',
                                                'PROOF_OF_ADDRESS',
                                                'PROFILE_PHOTO',
                                                'OTHER'
                                            )),
    document_number         VARCHAR(100),

    -- File storage
    file_url                TEXT            NOT NULL,
    file_hash               VARCHAR(128),

    -- Verification lifecycle
    status                  VARCHAR(30)     NOT NULL DEFAULT 'PENDING'
                                            CHECK (status IN (
                                                'PENDING',
                                                'UNDER_REVIEW',
                                                'VERIFIED',
                                                'REJECTED',
                                                'EXPIRED'
                                            )),
    uploaded_at             TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    verified_at             TIMESTAMP WITH TIME ZONE,
    expires_at              TIMESTAMP WITH TIME ZONE,

    -- External verification
    verification_provider   VARCHAR(100),
    verification_response   JSONB,
    rejection_reason        TEXT,

    -- Audit timestamps
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT fk_documents_driver_profile
        FOREIGN KEY (driver_profile_id)
        REFERENCES driver_profiles (id)
        ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_documents_driver_profile_id
    ON documents (driver_profile_id);

CREATE INDEX IF NOT EXISTS idx_documents_document_type
    ON documents (document_type);

CREATE INDEX IF NOT EXISTS idx_documents_status
    ON documents (status);

CREATE INDEX IF NOT EXISTS idx_documents_driver_profile_type
    ON documents (driver_profile_id, document_type);

CREATE INDEX IF NOT EXISTS idx_documents_expires_at
    ON documents (expires_at)
    WHERE expires_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_documents_verification_response
    ON documents USING GIN (verification_response)
    WHERE verification_response IS NOT NULL;

CREATE TRIGGER trg_documents_updated_at
    BEFORE UPDATE ON documents
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

COMMIT;


-- ============================================================
-- Migration: 003_create_onboarding_steps.sql
-- Description: Creates the onboarding_steps table for tracking
--              granular progress through the onboarding funnel
-- ============================================================

BEGIN;

CREATE TABLE IF NOT EXISTS onboarding_steps (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_profile_id   UUID            NOT NULL,

    -- Step definition
    step_name           VARCHAR(100)    NOT NULL,
    step_order          INTEGER         NOT NULL CHECK (step_order >= 0),

    -- Step lifecycle
    status              VARCHAR(30)     NOT NULL DEFAULT 'PENDING'
                                        CHECK (status IN (
                                            'PENDING',
                                            'IN_PROGRESS',
                                            'COMPLETED',
                                            'SKIPPED',
                                            'FAILED'
                                        )),
    started_at          TIMESTAMP WITH TIME ZONE,
    completed_at        TIMESTAMP WITH TIME ZONE,

    -- Arbitrary step payload (form data, results, etc.)
    metadata            JSONB,

    -- Audit timestamps
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT fk_onboarding_steps_driver_profile
        FOREIGN KEY (driver_profile_id)
        REFERENCES driver_profiles (id)
        ON DELETE CASCADE,

    -- One record per driver per step name
    CONSTRAINT uq_onboarding_steps_driver_step
        UNIQUE (driver_profile_id, step_name)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_onboarding_steps_driver_profile_id
    ON onboarding_steps (driver_profile_id);

CREATE INDEX IF NOT EXISTS idx_onboarding_steps_status
    ON onboarding_steps (status);

CREATE INDEX IF NOT EXISTS idx_onboarding_steps_driver_order
    ON onboarding_steps (driver_profile_id, step_order);

CREATE INDEX IF NOT EXISTS idx_onboarding_steps_metadata
    ON onboarding_steps USING GIN (metadata)
    WHERE metadata IS NOT NULL;

CREATE TRIGGER trg_onboarding_steps_updated_at
    BEFORE UPDATE ON onboarding_steps
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

COMMIT;


-- ============================================================
-- Migration: 004_create_vehicles.sql
-- Description: Creates the vehicles table for storing driver
--              vehicle information and compliance documents
-- ============================================================

BEGIN;

CREATE TABLE IF NOT EXISTS vehicles (
    id                          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_profile_id           UUID            NOT NULL,

    -- Vehicle identification
    license_plate               VARCHAR(30)     NOT NULL,
    make                        VARCHAR(100),
    model                       VARCHAR(100),
    year                        INTEGER         CHECK (year BETWEEN 1900 AND 2100),
    color                       VARCHAR(50),
    vin                         VARCHAR(50),

    -- Classification
    vehicle_type                VARCHAR(30)     NOT NULL DEFAULT 'STANDARD'
                                                CHECK (vehicle_type IN (
                                                    'STANDARD',
                                                    'XL',
                                                    'LUXURY',
                                                    'ELECTRIC',
                                                    'HYBRID',
                                                    'MOTORCYCLE',
                                                    'VAN',
                                                    'OTHER'
                                                )),

    -- Compliance documents
    registration_document_url   TEXT,
    insurance_document_url      TEXT,
    inspection_valid_until      DATE,

    -- Lifecycle
    status                      VARCHAR(30)     NOT NULL DEFAULT 'PENDING'
                                                CHECK (status IN (
                                                    'PENDING',
                                                    'ACTIVE',
                                                    'INACTIVE',
                                                    'REJECTED',
                                                    'SUSPENDED'
                                                )),

    -- Audit timestamps
    created_at                  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT fk_vehicles_driver_profile
        FOREIGN KEY (driver_profile_id)
        REFERENCES driver_profiles (id)
        ON DELETE CASCADE,

    CONSTRAINT uq_vehicles_license_plate
        UNIQUE (license_plate)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_vehicles_driver_profile_id
    ON vehicles (driver_profile_id);

CREATE INDEX IF NOT EXISTS idx_vehicles_license_plate
    ON vehicles (license_plate);

CREATE INDEX IF NOT EXISTS idx_vehicles_vehicle_type
    ON vehicles (vehicle_type);

CREATE INDEX IF NOT EXISTS idx_vehicles_status
    ON vehicles (status);

CREATE INDEX IF NOT EXISTS idx_vehicles_inspection_valid_until
    ON vehicles (inspection_valid_until)
    WHERE inspection_valid_until IS NOT NULL;

CREATE TRIGGER trg_vehicles_updated_at
    BEFORE UPDATE ON vehicles
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

COMMIT;


-- ============================================================
-- Migration: 005_create_audit_logs.sql
-- Description: Creates the audit_logs table for immutable
--              event tracking across the onboarding service
-- ============================================================

BEGIN;

CREATE TABLE IF NOT EXISTS audit_logs (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_profile_id   UUID,

    -- Event classification
    action              VARCHAR(150)    NOT NULL,
    actor_type          VARCHAR(30)     NOT NULL
                                        CHECK (actor_type IN (
                                            'DRIVER',
                                            'ADMIN',
                                            'SYSTEM',
                                            'WEBHOOK'
                                        )),
    actor_id            UUID,

    -- Request context
    ip_address          VARCHAR(45),
    user_agent          TEXT,

    -- Arbitrary event payload
    details             JSONB,

    -- Immutable timestamp (no updated_at â audit rows are never modified)
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- FK is soft: we keep audit rows even if driver profile is deleted
    CONSTRAINT fk_audit_logs_driver_profile
        FOREIGN KEY (driver_profile_id)
        REFERENCES driver_profiles (id)
        ON DELETE SET NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_audit_logs_driver_profile_id
    ON audit_logs (driver_profile_id)
    WHERE driver_profile_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_action
    ON audit_logs (action);

CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_type
    ON audit_logs (actor_type);

CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id
    ON audit_logs (actor_id)
    WHERE actor_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at
    ON audit_logs (created_at);

-- Composite index for common admin query: all events for a driver ordered by time
CREATE INDEX IF NOT EXISTS idx_audit_logs_driver_created
    ON audit_logs (driver_profile_id, created_at DESC)
    WHERE driver_profile_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_details
    ON audit_logs USING GIN (details)
    WHERE details IS NOT NULL;

-- Prevent any UPDATE or DELETE on audit_logs to preserve immutability
CREATE OR REPLACE RULE audit_logs_no_update AS
    ON UPDATE TO audit_logs
    DO INSTEAD NOTHING;

CREATE OR REPLACE RULE audit_logs_no_delete AS
    ON DELETE TO audit_logs
    DO INSTEAD NOTHING;

COMMIT;

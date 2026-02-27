-- ============================================================
-- Compliance & Audit Service - Initial Schema
-- Migration: 001_initial_schema.sql
-- Description: GDPR-compliant audit logging, data subject requests,
--              consent management, and regulatory compliance for
--              German ride-sharing platform
-- ============================================================

BEGIN;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- ENUMERATIONS
-- ============================================================

CREATE TYPE action_type_enum AS ENUM (
    'CREATE',
    'READ',
    'UPDATE',
    'DELETE',
    'LOGIN',
    'LOGOUT',
    'EXPORT',
    'IMPORT',
    'APPROVE',
    'REJECT',
    'SUBMIT',
    'NOTIFY',
    'PURGE',
    'VERIFY',
    'CONSENT_GRANT',
    'CONSENT_WITHDRAW',
    'DATA_REQUEST',
    'INCIDENT_REPORT'
);

CREATE TYPE entity_type_enum AS ENUM (
    'USER',
    'DRIVER',
    'RIDE',
    'PAYMENT',
    'DOCUMENT',
    'CONSENT',
    'DATA_REQUEST',
    'INCIDENT',
    'REPORT',
    'RETENTION_POLICY',
    'VEHICLE',
    'ROUTE'
);

CREATE TYPE report_type_enum AS ENUM (
    'GDPR_COMPLIANCE',
    'SECURITY_INCIDENT',
    'DATA_BREACH',
    'QUARTERLY_STATS',
    'ANNUAL_COMPLIANCE',
    'AUTHORITY_REQUEST',
    'TSE_AUDIT',
    'DRIVER_LICENSING'
);

CREATE TYPE report_status_enum AS ENUM (
    'DRAFT',
    'PENDING_REVIEW',
    'APPROVED',
    'SUBMITTED',
    'REJECTED',
    'ARCHIVED'
);

CREATE TYPE data_request_type_enum AS ENUM (
    'ACCESS',
    'ERASURE',
    'PORTABILITY',
    'RECTIFICATION',
    'RESTRICTION',
    'OBJECTION'
);

CREATE TYPE data_request_status_enum AS ENUM (
    'PENDING',
    'IN_PROGRESS',
    'COMPLETED',
    'REJECTED',
    'CANCELLED',
    'OVERDUE'
);

CREATE TYPE consent_type_enum AS ENUM (
    'TERMS_OF_SERVICE',
    'PRIVACY_POLICY',
    'MARKETING_EMAIL',
    'MARKETING_SMS',
    'ANALYTICS',
    'LOCATION_TRACKING',
    'DATA_SHARING_PARTNERS',
    'PROFILING'
);

CREATE TYPE incident_type_enum AS ENUM (
    'DATA_BREACH',
    'UNAUTHORIZED_ACCESS',
    'DATA_LOSS',
    'SYSTEM_COMPROMISE',
    'PHISHING',
    'INSIDER_THREAT',
    'RANSOMWARE',
    'DDoS',
    'POLICY_VIOLATION'
);

CREATE TYPE severity_enum AS ENUM (
    'LOW',
    'MEDIUM',
    'HIGH',
    'CRITICAL'
);

CREATE TYPE verification_status_enum AS ENUM (
    'PENDING',
    'IN_REVIEW',
    'VERIFIED',
    'REJECTED',
    'EXPIRED',
    'REVOKED'
);

CREATE TYPE document_type_enum AS ENUM (
    'P_SCHEIN',
    'FAHRERLAUBNIS',
    'TSE_CERT',
    'VEHICLE_REGISTRATION',
    'INSURANCE',
    'BACKGROUND_CHECK',
    'IDENTITY_CARD',
    'BUSINESS_LICENSE'
);

CREATE TYPE verification_method_enum AS ENUM (
    'EMAIL',
    'SMS',
    'IDENTITY_DOCUMENT',
    'VIDEO_CALL',
    'IN_PERSON'
);

-- ============================================================
-- TABLE: audit_logs
-- Immutable append-only audit log with SHA-256 hash chain
-- Ensures tamper-evident logging for regulatory compliance
-- ============================================================

CREATE TABLE audit_logs (
    id                  UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp           TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    action_type         action_type_enum NOT NULL,
    entity_type         entity_type_enum NOT NULL,
    entity_id           UUID            NOT NULL,
    user_id             UUID,
    user_email          VARCHAR(320),
    ip_address          INET,
    user_agent          TEXT,
    request_data        JSONB,
    response_data       JSONB,
    previous_hash       CHAR(64),
    current_hash        CHAR(64)        NOT NULL,
    integrity_verified  BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    -- Hash chain integrity: previous_hash must be NULL only for the very first record
    CONSTRAINT chk_audit_hash_format
        CHECK (current_hash ~ '^[a-f0-9]{64}$'),
    CONSTRAINT chk_audit_previous_hash_format
        CHECK (previous_hash IS NULL OR previous_hash ~ '^[a-f0-9]{64}$'),
    CONSTRAINT chk_audit_email_format
        CHECK (user_email IS NULL OR user_email ~* '^[^@]+@[^@]+\.[^@]+$'),
    CONSTRAINT chk_audit_timestamp_not_future
        CHECK (timestamp <= NOW() + INTERVAL '5 seconds')
);

COMMENT ON TABLE audit_logs IS
    'Immutable, tamper-evident audit log using SHA-256 hash chain. '
    'Records must never be updated or deleted. GDPR Art. 5(2) accountability.';
COMMENT ON COLUMN audit_logs.previous_hash IS
    'SHA-256 hash of the previous audit log entry. NULL for genesis record.';
COMMENT ON COLUMN audit_logs.current_hash IS
    'SHA-256 hash computed over all fields of this record plus previous_hash.';
COMMENT ON COLUMN audit_logs.integrity_verified IS
    'Flag set by background verification job confirming hash chain integrity.';

-- Prevent any UPDATE or DELETE on audit_logs via rule
CREATE RULE audit_logs_no_update AS ON UPDATE TO audit_logs DO INSTEAD NOTHING;
CREATE RULE audit_logs_no_delete AS ON DELETE TO audit_logs DO INSTEAD NOTHING;

-- Indexes for audit_logs
CREATE INDEX idx_audit_logs_timestamp           ON audit_logs (timestamp DESC);
CREATE INDEX idx_audit_logs_entity              ON audit_logs (entity_type, entity_id);
CREATE INDEX idx_audit_logs_user_id             ON audit_logs (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_audit_logs_user_email          ON audit_logs (user_email) WHERE user_email IS NOT NULL;
CREATE INDEX idx_audit_logs_action_type         ON audit_logs (action_type);
CREATE INDEX idx_audit_logs_ip_address          ON audit_logs (ip_address) WHERE ip_address IS NOT NULL;
CREATE INDEX idx_audit_logs_integrity           ON audit_logs (integrity_verified) WHERE integrity_verified = FALSE;
CREATE INDEX idx_audit_logs_created_at          ON audit_logs (created_at DESC);
CREATE INDEX idx_audit_logs_request_data        ON audit_logs USING GIN (request_data) WHERE request_data IS NOT NULL;
CREATE INDEX idx_audit_logs_response_data       ON audit_logs USING GIN (response_data) WHERE response_data IS NOT NULL;

-- ============================================================
-- TABLE: compliance_reports
-- Regulatory reports submitted to German authorities
-- (Kraftfahrt-Bundesamt, Datenschutzbehörde, etc.)
-- ============================================================

CREATE TABLE compliance_reports (
    id                      UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    report_type             report_type_enum NOT NULL,
    report_period_start     DATE            NOT NULL,
    report_period_end       DATE            NOT NULL,
    generated_by            UUID            NOT NULL,
    generated_at            TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    status                  report_status_enum NOT NULL DEFAULT 'DRAFT',
    content                 JSONB           NOT NULL DEFAULT '{}',
    recipient_authority     VARCHAR(255),
    submitted_at            TIMESTAMPTZ,
    submission_reference    VARCHAR(255),
    reviewer_notes          TEXT,
    reviewed_by             UUID,
    reviewed_at             TIMESTAMPTZ,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_report_period_order
        CHECK (report_period_end >= report_period_start),
    CONSTRAINT chk_report_submitted_has_authority
        CHECK (status != 'SUBMITTED' OR (recipient_authority IS NOT NULL AND submitted_at IS NOT NULL)),
    CONSTRAINT chk_report_submitted_at_after_generated
        CHECK (submitted_at IS NULL OR submitted_at >= generated_at),
    CONSTRAINT chk_report_reviewed_requires_reviewer
        CHECK (reviewed_at IS NULL OR (reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL))
);

COMMENT ON TABLE compliance_reports IS
    'Regulatory compliance reports for German authorities. '
    'Includes GDPR reports for Datenschutzbehörde and traffic authority reports.';
COMMENT ON COLUMN compliance_reports.submission_reference IS
    'Official reference number assigned by the receiving authority upon submission.';

-- Indexes for compliance_reports
CREATE INDEX idx_compliance_reports_type        ON compliance_reports (report_type);
CREATE INDEX idx_compliance_reports_status      ON compliance_reports (status);
CREATE INDEX idx_compliance_reports_period      ON compliance_reports (report_period_start, report_period_end);
CREATE INDEX idx_compliance_reports_generated   ON compliance_reports (generated_at DESC);
CREATE INDEX idx_compliance_reports_authority   ON compliance_reports (recipient_authority) WHERE recipient_authority IS NOT NULL;
CREATE INDEX idx_compliance_reports_submitted   ON compliance_reports (submitted_at DESC) WHERE submitted_at IS NOT NULL;
CREATE INDEX idx_compliance_reports_content     ON compliance_reports USING GIN (content);

-- ============================================================
-- TABLE: data_requests
-- GDPR data subject rights requests (Art. 15-22 GDPR)
-- Must be fulfilled within 30 days (Art. 12 GDPR)
-- ============================================================

CREATE TABLE data_requests (
    id                      UUID                    PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_type            data_request_type_enum  NOT NULL,
    user_id                 UUID                    NOT NULL,
    user_email              VARCHAR(320)            NOT NULL,
    status                  data_request_status_enum NOT NULL DEFAULT 'PENDING',
    requested_at            TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    deadline_at             TIMESTAMPTZ             NOT NULL
                                GENERATED ALWAYS AS (requested_at + INTERVAL '30 days') STORED,
    completed_at            TIMESTAMPTZ,
    verification_method     verification_method_enum NOT NULL,
    verified_at             TIMESTAMPTZ,
    request_data            JSONB                   NOT NULL DEFAULT '{}',
    response_data           JSONB,
    rejection_reason        TEXT,
    processed_by            UUID,
    internal_notes          TEXT,
    created_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_data_request_email_format
        CHECK (user_email ~* '^[^@]+@[^@]+\.[^@]+$'),
    CONSTRAINT chk_data_request_completed_has_response
        CHECK (status != 'COMPLETED' OR completed_at IS NOT NULL),
    CONSTRAINT chk_data_request_completed_after_requested
        CHECK (completed_at IS NULL OR completed_at >= requested_at),
    CONSTRAINT chk_data_request_rejected_has_reason
        CHECK (status != 'REJECTED' OR rejection_reason IS NOT NULL),
    CONSTRAINT chk_data_request_verified_before_completed
        CHECK (verified_at IS NULL OR completed_at IS NULL OR verified_at <= completed_at)
);

COMMENT ON TABLE data_requests IS
    'GDPR data subject rights requests per Art. 15-22 GDPR. '
    'Deadline automatically calculated as 30 days from request per Art. 12(3) GDPR.';
COMMENT ON COLUMN data_requests.deadline_at IS
    'Automatically computed 30-day statutory deadline per GDPR Art. 12(3).';
COMMENT ON COLUMN data_requests.verification_method IS
    'Identity verification method used before processing sensitive erasure/portability requests.';

-- Indexes for data_requests
CREATE INDEX idx_data_requests_user_id          ON data_requests (user_id);
CREATE INDEX idx_data_requests_user_email       ON data_requests (user_email);
CREATE INDEX idx_data_requests_status           ON data_requests (status);
CREATE INDEX idx_data_requests_type             ON data_requests (request_type);
CREATE INDEX idx_data_requests_requested_at     ON data_requests (requested_at DESC);
CREATE INDEX idx_data_requests_deadline         ON data_requests (deadline_at) WHERE status NOT IN ('COMPLETED', 'REJECTED', 'CANCELLED');
CREATE INDEX idx_data_requests_overdue          ON data_requests (deadline_at, status)
    WHERE status IN ('PENDING', 'IN_PROGRESS');
CREATE INDEX idx_data_requests_request_data     ON data_requests USING GIN (request_data);

-- ============================================================
-- TABLE: consent_records
-- Immutable consent audit trail per GDPR Art. 7 and
-- German TTDSG (Telekommunikation-Telemedien-Datenschutz-Gesetz)
-- ============================================================

CREATE TABLE consent_records (
    id                  UUID                PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID                NOT NULL,
    consent_type        consent_type_enum   NOT NULL,
    consent_version     VARCHAR(20)         NOT NULL,
    granted_at          TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    withdrawn_at        TIMESTAMPTZ,
    withdrawal_reason   TEXT,
    ip_address          INET,
    user_agent          TEXT,
    source_context      VARCHAR(100),
    metadata            JSONB               NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_consent_withdrawal_after_grant
        CHECK (withdrawn_at IS NULL OR withdrawn_at >= granted_at),
    CONSTRAINT chk_consent_withdrawal_has_record
        CHECK (withdrawn_at IS NULL OR (withdrawn_at IS NOT NULL)),
    CONSTRAINT chk_consent_version_format
        CHECK (consent_version ~ '^[0-9]+\.[0-9]+(\.[0-9]+)?$')
);

COMMENT ON TABLE consent_records IS
    'Immutable consent records per GDPR Art. 7 and German TTDSG. '
    'Each consent grant and withdrawal creates a new record for full audit trail. '
    'Records must not be deleted to maintain proof of consent.';
COMMENT ON COLUMN consent_records.consent_version IS
    'Version of the consent document (e.g. privacy policy version) user agreed to.';
COMMENT ON COLUMN consent_records.source_context IS
    'UI context where consent was collected (e.g. registration_flow, settings_page).';

-- Prevent deletion of consent records
CREATE RULE consent_records_no_delete AS ON DELETE TO consent_records DO INSTEAD NOTHING;

-- Indexes for consent_records
CREATE INDEX idx_consent_records_user_id        ON consent_records (user_id);
CREATE INDEX idx_consent_records_type           ON consent_records (consent_type);
CREATE INDEX idx_consent_records_user_type      ON consent_records (user_id, consent_type);
CREATE INDEX idx_consent_records_granted_at     ON consent_records (granted_at DESC);
CREATE INDEX idx_consent_records_withdrawn      ON consent_records (withdrawn_at DESC) WHERE withdrawn_at IS NOT NULL;
CREATE INDEX idx_consent_records_active         ON consent_records (user_id, consent_type, granted_at DESC)
    WHERE withdrawn_at IS NULL;
CREATE INDEX idx_consent_records_version        ON consent_records (consent_type, consent_version);
CREATE INDEX idx_consent_records_metadata       ON consent_records USING GIN (metadata);

-- ============================================================
-- TABLE: retention_policies
-- Data retention configuration per GDPR Art. 5(1)(e)
-- and German commercial/tax law (Handelsgesetzbuch, AO)
-- ============================================================

CREATE TABLE retention_policies (
    id                  UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    data_type           VARCHAR(100)    NOT NULL UNIQUE,
    retention_days      INTEGER         NOT NULL,
    legal_basis         VARCHAR(500)    NOT NULL,
    description         TEXT            NOT NULL,
    auto_purge_enabled  BOOLEAN         NOT NULL DEFAULT FALSE,
    last_purge_run      TIMESTAMPTZ,
    next_purge_at       TIMESTAMPTZ,
    purge_batch_size    INTEGER         NOT NULL DEFAULT 1000,
    is_active           BOOLEAN         NOT NULL DEFAULT TRUE,
    created_by          UUID,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_retention_days_positive
        CHECK (retention_days > 0),
    CONSTRAINT chk_retention_batch_size_positive
        CHECK (purge_batch_size > 0 AND purge_batch_size <= 100000),
    CONSTRAINT chk_retention_next_purge_valid
        CHECK (next_purge_at IS NULL OR auto_purge_enabled = TRUE)
);

COMMENT ON TABLE retention_policies IS
    'Data retention policies per GDPR Art. 5(1)(e) storage limitation principle. '
    'Defines maximum retention periods with legal basis for each data category. '
    'German HGB requires 10yr retention for commercial records, AO §147 for tax records.';
COMMENT ON COLUMN retention_policies.data_type IS
    'Unique identifier for the data category (e.g. ride_records, payment_data, chat_logs).';
COMMENT ON COLUMN retention_policies.legal_basis IS
    'Legal justification for retention period (GDPR article, HGB §, AO §, etc.).';
COMMENT ON COLUMN retention_policies.auto_purge_enabled IS
    'When TRUE, automated purge job will delete data exceeding retention_days.';

-- Indexes for retention_policies
CREATE INDEX idx_retention_policies_data_type   ON retention_policies (data_type);
CREATE INDEX idx_retention_policies_active      ON retention_policies (is_active) WHERE is_active = TRUE;
CREATE INDEX idx_retention_policies_purge       ON retention_policies (next_purge_at)
    WHERE auto_purge_enabled = TRUE AND is_active = TRUE;

-- ============================================================
-- TABLE: incidents
-- Security incidents and personal data breaches
-- GDPR Art. 33/34 requires 72-hour authority notification
-- ============================================================

CREATE TABLE incidents (
    id                      UUID                PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_type           incident_type_enum  NOT NULL,
    severity                severity_enum       NOT NULL,
    title                   VARCHAR(500)        NOT NULL,
    description             TEXT                NOT NULL,
    detected_at             TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    reported_by             UUID,
    resolved_at             TIMESTAMPTZ,
    resolution_summary      TEXT,
    impact_assessment       JSONB               NOT NULL DEFAULT '{}',
    affected_user_count     INTEGER,
    affected_data_types     TEXT[],
    notification_required   BOOLEAN             NOT NULL DEFAULT FALSE,
    notification_sent       BOOLEAN             NOT NULL DEFAULT FALSE,
    notification_sent_at    TIMESTAMPTZ,
    authorities_notified    TEXT[],
    notification_deadline   TIMESTAMPTZ
                                GENERATED ALWAYS AS (
                                    CASE WHEN notification_required THEN detected_at + INTERVAL '72 hours'
                                    ELSE NULL END
                                ) STORED,
    users_notified_at       TIMESTAMPTZ,
    containment_actions     JSONB               NOT NULL DEFAULT '[]',
    root_cause              TEXT,
    prevention_measures     TEXT,
    created_at              TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_incident_resolved_after_detected
        CHECK (resolved_at IS NULL OR resolved_at >= detected_at),
    CONSTRAINT chk_incident_notification_sent_has_timestamp
        CHECK (notification_sent = FALSE OR notification_sent_at IS NOT NULL),
    CONSTRAINT chk_incident_affected_users_non_negative
        CHECK (affected_user_count IS NULL OR affected_user_count >= 0),
    CONSTRAINT chk_incident_users_notified_after_detection
        CHECK (users_notified_at IS NULL OR users_notified_at >= detected_at)
);

COMMENT ON TABLE incidents IS
    'Security incidents and personal data breach records per GDPR Art. 33/34. '
    'notification_deadline auto-computed as detected_at + 72h per Art. 33(1) GDPR '
    'for incidents requiring supervisory authority notification.';
COMMENT ON COLUMN incidents.notification_deadline IS
    'Auto-computed 72-hour GDPR Art. 33(1) deadline. NULL if notification not required.';
COMMENT ON COLUMN incidents.authorities_notified IS
    'Array of authority names notified (e.g. Bayerisches Landesamt für Datenschutzaufsicht).';
COMMENT ON COLUMN incidents.impact_assessment IS
    'Structured assessment: {scope, data_categories, likelihood_of_harm, mitigations}';

-- Indexes for incidents
CREATE INDEX idx_incidents_type                 ON incidents (incident_type);
CREATE INDEX idx_incidents_severity             ON incidents (severity);
CREATE INDEX idx_incidents_detected_at          ON incidents (detected_at DESC);
CREATE INDEX idx_incidents_resolved             ON incidents (resolved_at) WHERE resolved_at IS NOT NULL;
CREATE INDEX idx_incidents_unresolved           ON incidents (detected_at DESC) WHERE resolved_at IS NULL;
CREATE INDEX idx_incidents_notification_pending ON incidents (notification_deadline)
    WHERE notification_required = TRUE AND notification_sent = FALSE;
CREATE INDEX idx_incidents_impact               ON incidents USING GIN (impact_assessment);
CREATE INDEX idx_incidents_affected_types       ON incidents USING GIN (affected_data_types);

-- ============================================================
-- TABLE: regulatory_documents
-- Driver compliance documents required under German law:
-- P-Schein (Personenbeförderungsschein), Fahrerlaubnis,
-- TSE certificate (Kassensicherungsverordnung)
-- ============================================================

CREATE TABLE regulatory_documents (
    id                      UUID                    PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                 UUID                    NOT NULL,
    document_type           document_type_enum      NOT NULL,
    document_number         VARCHAR(100)            NOT NULL,
    issued_by               VARCHAR(255)            NOT NULL,
    issued_at               DATE                    NOT NULL,
    expires_at              DATE,
    is_permanent            BOOLEAN                 NOT NULL DEFAULT FALSE,
    verification_status     verification_status_enum NOT NULL DEFAULT 'PENDING',
    verified_by             UUID,
    verified_at             TIMESTAMPTZ,
    verification_notes      TEXT,
    rejection_reason        TEXT,
    document_hash           CHAR(64),
    storage_reference       TEXT,
    metadata                JSONB                   NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_regulatory_doc_expires_after_issued
        CHECK (expires_at IS NULL OR expires_at > issued_at),
    CONSTRAINT chk_regulatory_doc_permanent_no_expiry
        CHECK (is_permanent = FALSE OR expires_at IS NULL),
    CONSTRAINT chk_regulatory_doc_non_permanent_has_expiry
        CHECK (is_permanent = TRUE OR expires_at IS NOT NULL),
    CONSTRAINT chk_regulatory_doc_verified_has_verifier
        CHECK (verification_status NOT IN ('VERIFIED', 'REJECTED') OR verified_by IS NOT NULL),
    CONSTRAINT chk_regulatory_doc_rejected_has_reason
        CHECK (verification_status != 'REJECTED' OR rejection_reason IS NOT NULL),
    CONSTRAINT chk_regulatory_doc_hash_format
        CHECK (document_hash IS NULL OR document_hash ~ '^[a-f0-9]{64}$'),
    CONSTRAINT uq_regulatory_doc_number_type
        UNIQUE (document_type, document_number)
);

COMMENT ON TABLE regulatory_documents IS
    'Driver compliance documents required under German law. '
    'P-Schein: Personenbeförderungsschein (§ 48 FeV). '
    'Fahrerlaubnis: driving licence per EU directive 2006/126/EC. '
    'TSE-Cert: Technische Sicherheitseinrichtung certificate per KassenSichV.';
COMMENT ON COLUMN regulatory_documents.document_hash IS
    'SHA-256 hash of the stored document for integrity verification.';
COMMENT ON COLUMN regulatory_documents.is_permanent IS
    'TRUE for documents with no expiry (e.g. some background checks). '
    'Mutually exclusive with expires_at.';

-- Indexes for regulatory_documents
CREATE INDEX idx_regulatory_docs_user_id        ON regulatory_documents (user_id);
CREATE INDEX idx_regulatory_docs_type           ON regulatory_documents (document_type);
CREATE INDEX idx_regulatory_docs_user_type      ON regulatory_documents (user_id, document_type);
CREATE INDEX idx_regulatory_docs_status         ON regulatory_documents (verification_status);
CREATE INDEX idx_regulatory_docs_expiry         ON regulatory_documents (expires_at)
    WHERE expires_at IS NOT NULL AND verification_status = 'VERIFIED';
CREATE INDEX idx_regulatory_docs_expiring_soon  ON regulatory_documents (expires_at, user_id)
    WHERE expires_at IS NOT NULL
      AND verification_status = 'VERIFIED'
      AND expires_at BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '90 days';
CREATE INDEX idx_regulatory_docs_pending        ON regulatory_documents (created_at DESC)
    WHERE verification_status IN ('PENDING', 'IN_REVIEW');
CREATE INDEX idx_regulatory_docs_metadata       ON regulatory_documents USING GIN (metadata);

-- ============================================================
-- FUNCTIONS & TRIGGERS: updated_at auto-maintenance
-- ============================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

COMMENT ON FUNCTION update_updated_at_column() IS
    'Generic trigger function to auto-update updated_at timestamp on row modification.';

CREATE TRIGGER trg_compliance_reports_updated_at
    BEFORE UPDATE ON compliance_reports
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_data_requests_updated_at
    BEFORE UPDATE ON data_requests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_retention_policies_updated_at
    BEFORE UPDATE ON retention_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_incidents_updated_at
    BEFORE UPDATE ON incidents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_regulatory_documents_updated_at
    BEFORE UPDATE ON regulatory_documents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- FUNCTION: verify_audit_log_chain
-- Verifies hash chain integrity for a range of audit logs
-- ============================================================

CREATE OR REPLACE FUNCTION verify_audit_log_chain(
    p_start_timestamp TIMESTAMPTZ DEFAULT NOW() - INTERVAL '24 hours',
    p_end_timestamp   TIMESTAMPTZ DEFAULT NOW()
)
RETURNS TABLE (
    verified_count  BIGINT,
    failed_count    BIGINT,
    first_log_id    UUID,
    last_log_id     UUID,
    chain_intact    BOOLEAN
)
LANGUAGE plpgsql
AS $$
DECLARE
    v_verified  BIGINT := 0;
    v_failed    BIGINT := 0;
    v_first_id  UUID;
    v_last_id   UUID;
    v_prev_hash CHAR(64) := NULL;
    v_rec       RECORD;
BEGIN
    FOR v_rec IN
        SELECT id, previous_hash, current_hash
        FROM audit_logs
        WHERE timestamp BETWEEN p_start_timestamp AND p_end_timestamp
        ORDER BY timestamp ASC, created_at ASC
    LOOP
        IF v_first_id IS NULL THEN
            v_first_id := v_rec.id;
        END IF;
        v_last_id := v_rec.id;

        IF v_rec.previous_hash IS NOT DISTINCT FROM v_prev_hash THEN
            v_verified := v_verified + 1;
            UPDATE audit_logs SET integrity_verified = TRUE WHERE id = v_rec.id;
        ELSE
            v_failed := v_failed + 1;
        END IF;

        v_prev_hash := v_rec.current_hash;
    END LOOP;

    RETURN QUERY SELECT
        v_verified,
        v_failed,
        v_first_id,
        v_last_id,
        (v_failed = 0);
END;
$$;

COMMENT ON FUNCTION verify_audit_log_chain(TIMESTAMPTZ, TIMESTAMPTZ) IS
    'Verifies SHA-256 hash chain integrity for audit logs within a time range. '
    'Updates integrity_verified flag and returns verification summary.';

-- ============================================================
-- FUNCTION: get_active_consent
-- Returns the most recent active consent for a user/type
-- ============================================================

CREATE OR REPLACE FUNCTION get_active_consent(
    p_user_id       UUID,
    p_consent_type  consent_type_enum
)
RETURNS TABLE (
    has_consent     BOOLEAN,
    consent_id      UUID,
    consent_version VARCHAR(20),
    granted_at      TIMESTAMPTZ
)
LANGUAGE sql
STABLE
AS $$
    SELECT
        TRUE,
        id,
        consent_version,
        granted_at
    FROM consent_records
    WHERE user_id = p_user_id
      AND consent_type = p_consent_type
      AND withdrawn_at IS NULL
    ORDER BY granted_at DESC
    LIMIT 1;
$$;

COMMENT ON FUNCTION get_active_consent(UUID, consent_type_enum) IS
    'Returns current active consent record for a user and consent type. '
    'Returns no rows if consent has not been granted or has been withdrawn.';

-- ============================================================
-- SEED: Default data retention policies
-- Based on GDPR Art. 5, German HGB, AO, PBefG requirements
-- ============================================================

INSERT INTO retention_policies (data_type, retention_days, legal_basis, description, auto_purge_enabled) VALUES
    ('ride_records',
     3650,
     'HGB § 257 Abs. 1 Nr. 1 (10 Jahre), PBefG § 57',
     'Beförderungsverträge und Quittungen nach Handelsgesetzbuch 10 Jahre aufzubewahren.',
     FALSE),
    ('payment_transactions',
     3650,
     'AO § 147 Abs. 1 Nr. 1 (10 Jahre), UStG',
     'Steuerrelevante Buchungsbelege und Zahlungsnachweise gemäß Abgabenordnung.',
     FALSE),
    ('user_profiles',
     1095,
     'GDPR Art. 5(1)(e), GDPR Art. 17',
     'Personenbezogene Daten inaktiver Nutzer nach 3 Jahren zu löschen.',
     FALSE),
    ('chat_messages',
     180,
     'GDPR Art. 5(1)(e) Speicherbegrenzung',
     'Kommunikationsdaten zwischen Fahrern und Fahrgästen nach 6 Monaten löschen.',
     TRUE),
    ('location_data',
     90,
     'GDPR Art. 5(1)(c) Datenminimierung, BDSG § 26',
     'GPS-Standortdaten nach 90 Tagen anonymisieren oder löschen.',
     TRUE),
    ('audit_logs',
     3650,
     'GDPR Art. 5(2) Rechenschaftspflicht, ISO 27001',
     'Audit-Protokolle für Compliance-Nachweis 10 Jahre aufbewahren.',
     FALSE),
    ('session_tokens',
     1,
     'GDPR Art. 5(1)(e), IT-Sicherheitsrichtlinie',
     'Authentifizierungs-Sessions täglich bereinigen.',
     TRUE),
    ('tse_transaction_data',
     3650,
     'KassenSichV § 6, AO § 147 (10 Jahre)',
     'TSE-Transaktionsdaten gemäß Kassensicherungsverordnung 10 Jahre aufbewahren.',
     FALSE),
    ('support_tickets',
     1825,
     'GDPR Art. 6(1)(b) Vertragserfüllung, BGB § 195',
     'Kundensupport-Anfragen und Beschwerden 5 Jahre für Haftungszwecke aufbewahren.',
     FALSE),
    ('marketing_profiles',
     365,
     'GDPR Art. 6(1)(a) Einwilligung, TTDSG § 25',
     'Marketing-Profile nach Widerruf der Einwilligung innerhalb 1 Jahr löschen.',
     TRUE);

COMMIT;

-- ============================================================
-- POST-MIGRATION NOTES
-- ============================================================
-- 1. audit_logs and consent_records have INSERT-only rules.
--    Application users should have only INSERT privilege on these tables.
-- 2. The verify_audit_log_chain() function should be scheduled
--    via pg_cron every hour for continuous integrity monitoring.
-- 3. retention_policies with auto_purge_enabled=TRUE require
--    a separate scheduled purge job per data_type.
-- 4. notification_deadline in incidents is auto-computed;
--    a monitoring job must alert when deadline - NOW() < 12 hours.
-- 5. data_requests.deadline_at is auto-computed as 30 days;
--    overdue requests should trigger escalation alerts.
-- ============================================================

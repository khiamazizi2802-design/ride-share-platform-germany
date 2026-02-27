-- ============================================================
-- Compliance Service - Initial Schema Migration
-- 001_initial_schema.sql
-- ============================================================

BEGIN;

-- ============================================================
-- Extensions
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- Custom Types / Enums
-- ============================================================

CREATE TYPE gdpr_request_type AS ENUM (
    'access',
    'deletion',
    'portability',
    'rectification',
    'restriction',
    'objection'
);

CREATE TYPE gdpr_request_status AS ENUM (
    'pending',
    'in_progress',
    'completed',
    'rejected',
    'cancelled',
    'expired'
);

CREATE TYPE pbefg_report_status AS ENUM (
    'draft',
    'pending_review',
    'approved',
    'submitted',
    'acknowledged',
    'rejected',
    'resubmission_required'
);

CREATE TYPE pbefg_report_period AS ENUM (
    'monthly',
    'quarterly',
    'semi_annual',
    'annual'
);

CREATE TYPE document_status AS ENUM (
    'draft',
    'active',
    'superseded',
    'archived',
    'revoked'
);

CREATE TYPE compliance_check_status AS ENUM (
    'pass',
    'fail',
    'warning',
    'skipped',
    'error'
);

CREATE TYPE compliance_check_severity AS ENUM (
    'critical',
    'high',
    'medium',
    'low',
    'info'
);

CREATE TYPE retention_action AS ENUM (
    'delete',
    'anonymize',
    'archive',
    'notify'
);

CREATE TYPE consent_status AS ENUM (
    'granted',
    'withdrawn',
    'expired',
    'pending'
);

CREATE TYPE audit_action AS ENUM (
    'create',
    'read',
    'update',
    'delete',
    'export',
    'import',
    'login',
    'logout',
    'access_denied',
    'configuration_change',
    'report_submitted',
    'data_request_processed',
    'consent_granted',
    'consent_withdrawn'
);

-- ============================================================
-- 1. audit_logs
-- ============================================================

CREATE TABLE audit_logs (
    id                  UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id            UUID            NOT NULL DEFAULT uuid_generate_v4(),
    user_id             UUID,
    session_id          UUID,
    action              audit_action    NOT NULL,
    resource_type       VARCHAR(128)    NOT NULL,
    resource_id         VARCHAR(256),
    resource_name       VARCHAR(512),
    description         TEXT,
    old_values          JSONB,
    new_values          JSONB,
    metadata            JSONB           NOT NULL DEFAULT '{}',
    ip_address          INET,
    user_agent          TEXT,
    request_id          VARCHAR(256),
    request_method      VARCHAR(16),
    request_path        TEXT,
    response_status     SMALLINT,
    duration_ms         INTEGER,
    service_name        VARCHAR(128)    NOT NULL DEFAULT 'compliance-service',
    environment         VARCHAR(64)     NOT NULL DEFAULT 'production',
    correlation_id      VARCHAR(256),
    tenant_id           UUID,
    is_sensitive        BOOLEAN         NOT NULL DEFAULT FALSE,
    checksum            VARCHAR(64),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE audit_logs IS 'Immutable comprehensive audit trail for all system events and user actions';
COMMENT ON COLUMN audit_logs.old_values IS 'Previous state of the resource in JSON format';
COMMENT ON COLUMN audit_logs.new_values IS 'New state of the resource in JSON format';
COMMENT ON COLUMN audit_logs.checksum IS 'SHA-256 checksum for tamper detection';
COMMENT ON COLUMN audit_logs.is_sensitive IS 'Flag for PII or sensitive audit entries requiring elevated access';

CREATE INDEX idx_audit_logs_user_id             ON audit_logs (user_id)          WHERE user_id IS NOT NULL;
CREATE INDEX idx_audit_logs_action              ON audit_logs (action);
CREATE INDEX idx_audit_logs_resource_type       ON audit_logs (resource_type);
CREATE INDEX idx_audit_logs_resource_id         ON audit_logs (resource_id)      WHERE resource_id IS NOT NULL;
CREATE INDEX idx_audit_logs_created_at          ON audit_logs (created_at DESC);
CREATE INDEX idx_audit_logs_ip_address          ON audit_logs (ip_address)       WHERE ip_address IS NOT NULL;
CREATE INDEX idx_audit_logs_session_id          ON audit_logs (session_id)       WHERE session_id IS NOT NULL;
CREATE INDEX idx_audit_logs_correlation_id      ON audit_logs (correlation_id)   WHERE correlation_id IS NOT NULL;
CREATE INDEX idx_audit_logs_tenant_id           ON audit_logs (tenant_id)        WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_audit_logs_service_env         ON audit_logs (service_name, environment);
CREATE INDEX idx_audit_logs_resource_composite  ON audit_logs (resource_type, resource_id, created_at DESC);
CREATE INDEX idx_audit_logs_metadata            ON audit_logs USING GIN (metadata);
CREATE INDEX idx_audit_logs_old_values          ON audit_logs USING GIN (old_values) WHERE old_values IS NOT NULL;
CREATE INDEX idx_audit_logs_new_values          ON audit_logs USING GIN (new_values) WHERE new_values IS NOT NULL;

-- Audit logs are append-only; enforce via rule
CREATE RULE audit_logs_no_update AS ON UPDATE TO audit_logs DO INSTEAD NOTHING;
CREATE RULE audit_logs_no_delete AS ON DELETE TO audit_logs DO INSTEAD NOTHING;

-- ============================================================
-- 2. gdpr_requests
-- ============================================================

CREATE TABLE gdpr_requests (
    id                          UUID                PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_number              VARCHAR(64)         NOT NULL,
    data_subject_id             UUID                NOT NULL,
    data_subject_email          VARCHAR(320)        NOT NULL,
    data_subject_full_name      VARCHAR(512),
    data_subject_verified       BOOLEAN             NOT NULL DEFAULT FALSE,
    verification_token          VARCHAR(256),
    verification_expires_at     TIMESTAMPTZ,
    request_type                gdpr_request_type   NOT NULL,
    status                      gdpr_request_status NOT NULL DEFAULT 'pending',
    priority                    SMALLINT            NOT NULL DEFAULT 3 CHECK (priority BETWEEN 1 AND 5),
    description                 TEXT,
    scope                       JSONB               NOT NULL DEFAULT '{}',
    affected_systems            TEXT[]              NOT NULL DEFAULT '{}',
    assigned_to                 UUID,
    requested_at                TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    acknowledged_at             TIMESTAMPTZ,
    due_at                      TIMESTAMPTZ         NOT NULL,
    completed_at                TIMESTAMPTZ,
    rejected_at                 TIMESTAMPTZ,
    rejection_reason            TEXT,
    outcome_summary             TEXT,
    outcome_data_location       TEXT,
    processing_notes            TEXT,
    internal_notes              TEXT,
    regulatory_reference        VARCHAR(256),
    legal_basis                 VARCHAR(256),
    requires_identity_proof     BOOLEAN             NOT NULL DEFAULT TRUE,
    identity_proof_received_at  TIMESTAMPTZ,
    notified_at                 TIMESTAMPTZ,
    reminder_sent_at            TIMESTAMPTZ,
    extension_requested         BOOLEAN             NOT NULL DEFAULT FALSE,
    extension_granted           BOOLEAN             NOT NULL DEFAULT FALSE,
    extension_due_at            TIMESTAMPTZ,
    extension_reason            TEXT,
    metadata                    JSONB               NOT NULL DEFAULT '{}',
    created_by                  UUID,
    updated_by                  UUID,
    created_at                  TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_gdpr_request_number UNIQUE (request_number),
    CONSTRAINT chk_gdpr_due_after_requested CHECK (due_at > requested_at),
    CONSTRAINT chk_gdpr_completed_after_requested CHECK (completed_at IS NULL OR completed_at >= requested_at),
    CONSTRAINT chk_gdpr_extension_logic CHECK (
        (extension_granted = FALSE) OR
        (extension_requested = TRUE AND extension_due_at IS NOT NULL)
    )
);

COMMENT ON TABLE gdpr_requests IS 'GDPR Article 15-22 data subject requests with full lifecycle tracking';
COMMENT ON COLUMN gdpr_requests.due_at IS 'Regulatory deadline - typically 30 days from request per GDPR Art.12';
COMMENT ON COLUMN gdpr_requests.scope IS 'JSON definition of data categories and systems in scope';
COMMENT ON COLUMN gdpr_requests.priority IS '1=Critical, 2=High, 3=Medium, 4=Low, 5=Minimal';

CREATE INDEX idx_gdpr_requests_data_subject_id   ON gdpr_requests (data_subject_id);
CREATE INDEX idx_gdpr_requests_email             ON gdpr_requests (data_subject_email);
CREATE INDEX idx_gdpr_requests_status            ON gdpr_requests (status);
CREATE INDEX idx_gdpr_requests_request_type      ON gdpr_requests (request_type);
CREATE INDEX idx_gdpr_requests_due_at            ON gdpr_requests (due_at);
CREATE INDEX idx_gdpr_requests_assigned_to       ON gdpr_requests (assigned_to)     WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_gdpr_requests_created_at        ON gdpr_requests (created_at DESC);
CREATE INDEX idx_gdpr_requests_status_due        ON gdpr_requests (status, due_at)  WHERE status IN ('pending', 'in_progress');
CREATE INDEX idx_gdpr_requests_metadata          ON gdpr_requests USING GIN (metadata);
CREATE INDEX idx_gdpr_requests_affected_systems  ON gdpr_requests USING GIN (affected_systems);

-- ============================================================
-- 3. pbefg_reports
-- ============================================================

CREATE TABLE pbefg_reports (
    id                          UUID                    PRIMARY KEY DEFAULT uuid_generate_v4(),
    report_number               VARCHAR(128)            NOT NULL,
    report_title                VARCHAR(512)            NOT NULL,
    report_type                 VARCHAR(128)            NOT NULL,
    report_period               pbefg_report_period     NOT NULL,
    period_start_date           DATE                    NOT NULL,
    period_end_date             DATE                    NOT NULL,
    reporting_year              SMALLINT                NOT NULL,
    reporting_quarter           SMALLINT                CHECK (reporting_quarter BETWEEN 1 AND 4),
    status                      pbefg_report_status     NOT NULL DEFAULT 'draft',
    version                     INTEGER                 NOT NULL DEFAULT 1,
    is_current_version          BOOLEAN                 NOT NULL DEFAULT TRUE,
    parent_report_id            UUID                    REFERENCES pbefg_reports (id) ON DELETE SET NULL,
    authority_name              VARCHAR(256)            NOT NULL,
    authority_code              VARCHAR(64)             NOT NULL,
    authority_contact_email     VARCHAR(320),
    authority_submission_ref    VARCHAR(256),
    operator_id                 UUID                    NOT NULL,
    operator_name               VARCHAR(512)            NOT NULL,
    operator_license_number     VARCHAR(128),
    route_count                 INTEGER                 CHECK (route_count >= 0),
    vehicle_count               INTEGER                 CHECK (vehicle_count >= 0),
    passenger_count             BIGINT                  CHECK (passenger_count >= 0),
    revenue_amount              NUMERIC(18, 2),
    revenue_currency            CHAR(3)                 NOT NULL DEFAULT 'EUR',
    subsidy_amount              NUMERIC(18, 2),
    operating_days              INTEGER                 CHECK (operating_days BETWEEN 0 AND 366),
    delay_minutes_total         BIGINT,
    cancellation_count          INTEGER,
    incident_count              INTEGER,
    accessibility_compliance_pct NUMERIC(5, 2)         CHECK (accessibility_compliance_pct BETWEEN 0 AND 100),
    report_data                 JSONB                   NOT NULL DEFAULT '{}',
    attachments                 JSONB                   NOT NULL DEFAULT '[]',
    submission_format           VARCHAR(64)             NOT NULL DEFAULT 'JSON',
    submission_endpoint         TEXT,
    prepared_by                 UUID,
    prepared_by_name            VARCHAR(256),
    reviewed_by                 UUID,
    reviewed_by_name            VARCHAR(256),
    approved_by                 UUID,
    approved_by_name            VARCHAR(256),
    submitted_by                UUID,
    submitted_by_name           VARCHAR(256),
    draft_created_at            TIMESTAMPTZ,
    review_requested_at         TIMESTAMPTZ,
    approved_at                 TIMESTAMPTZ,
    submitted_at                TIMESTAMPTZ,
    acknowledged_at             TIMESTAMPTZ,
    due_at                      TIMESTAMPTZ             NOT NULL,
    rejected_at                 TIMESTAMPTZ,
    rejection_reason            TEXT,
    resubmission_deadline       TIMESTAMPTZ,
    internal_notes              TEXT,
    external_notes              TEXT,
    legal_basis                 VARCHAR(512)            NOT NULL DEFAULT 'PBefG §42',
    metadata                    JSONB                   NOT NULL DEFAULT '{}',
    created_by                  UUID,
    updated_by                  UUID,
    created_at                  TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_pbefg_report_number UNIQUE (report_number),
    CONSTRAINT chk_pbefg_period_dates CHECK (period_end_date > period_start_date),
    CONSTRAINT chk_pbefg_version_positive CHECK (version >= 1),
    CONSTRAINT chk_pbefg_quarterly_period CHECK (
        report_period != 'quarterly' OR reporting_quarter IS NOT NULL
    )
);

COMMENT ON TABLE pbefg_reports IS 'Personenbeförderungsgesetz (PBefG) regulatory reports for German transport authorities';
COMMENT ON COLUMN pbefg_reports.parent_report_id IS 'References original report when this is a resubmission or amendment';
COMMENT ON COLUMN pbefg_reports.report_data IS 'Complete structured report payload in authority-required format';
COMMENT ON COLUMN pbefg_reports.legal_basis IS 'Specific PBefG sections governing this report requirement';

CREATE INDEX idx_pbefg_reports_status              ON pbefg_reports (status);
CREATE INDEX idx_pbefg_reports_operator_id         ON pbefg_reports (operator_id);
CREATE INDEX idx_pbefg_reports_authority_code      ON pbefg_reports (authority_code);
CREATE INDEX idx_pbefg_reports_period_start        ON pbefg_reports (period_start_date);
CREATE INDEX idx_pbefg_reports_period_end          ON pbefg_reports (period_end_date);
CREATE INDEX idx_pbefg_reports_reporting_year      ON pbefg_reports (reporting_year);
CREATE INDEX idx_pbefg_reports_due_at              ON pbefg_reports (due_at);
CREATE INDEX idx_pbefg_reports_submitted_at        ON pbefg_reports (submitted_at)   WHERE submitted_at IS NOT NULL;
CREATE INDEX idx_pbefg_reports_is_current          ON pbefg_reports (is_current_version) WHERE is_current_version = TRUE;
CREATE INDEX idx_pbefg_reports_parent              ON pbefg_reports (parent_report_id) WHERE parent_report_id IS NOT NULL;
CREATE INDEX idx_pbefg_reports_report_data         ON pbefg_reports USING GIN (report_data);
CREATE INDEX idx_pbefg_reports_metadata            ON pbefg_reports USING GIN (metadata);
CREATE INDEX idx_pbefg_reports_operator_period     ON pbefg_reports (operator_id, period_start_date, period_end_date);

-- ============================================================
-- 4. compliance_documents
-- ============================================================

CREATE TABLE compliance_documents (
    id                      UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_number         VARCHAR(128)    NOT NULL,
    title                   VARCHAR(512)    NOT NULL,
    description             TEXT,
    document_type           VARCHAR(128)    NOT NULL,
    category                VARCHAR(128)    NOT NULL,
    subcategory             VARCHAR(128),
    tags                    TEXT[]          NOT NULL DEFAULT '{}',
    status                  document_status NOT NULL DEFAULT 'draft',
    version                 INTEGER         NOT NULL DEFAULT 1,
    version_label           VARCHAR(32)     NOT NULL DEFAULT '1.0',
    is_current_version      BOOLEAN         NOT NULL DEFAULT TRUE,
    previous_version_id     UUID            REFERENCES compliance_documents (id) ON DELETE SET NULL,
    superseded_by_id        UUID            REFERENCES compliance_documents (id) ON DELETE SET NULL,
    file_name               VARCHAR(512)    NOT NULL,
    file_path               TEXT            NOT NULL,
    file_size_bytes         BIGINT          CHECK (file_size_bytes > 0),
    mime_type               VARCHAR(255)    NOT NULL,
    file_hash_sha256        VARCHAR(64),
    storage_backend         VARCHAR(64)     NOT NULL DEFAULT 's3',
    storage_bucket          VARCHAR(256),
    storage_key             TEXT,
    is_encrypted            BOOLEAN         NOT NULL DEFAULT TRUE,
    encryption_key_id       VARCHAR(256),
    regulatory_framework    VARCHAR(128),
    regulatory_reference    VARCHAR(256),
    jurisdiction            VARCHAR(128),
    applicable_from         DATE,
    applicable_until        DATE,
    review_due_at           DATE,
    last_reviewed_at        TIMESTAMPTZ,
    reviewed_by             UUID,
    reviewed_by_name        VARCHAR(256),
    approved_by             UUID,
    approved_by_name        VARCHAR(256),
    approved_at             TIMESTAMPTZ,
    published_at            TIMESTAMPTZ,
    access_level            VARCHAR(64)     NOT NULL DEFAULT 'internal',
    access_roles            TEXT[]          NOT NULL DEFAULT '{}',
    download_count          INTEGER         NOT NULL DEFAULT 0,
    last_accessed_at        TIMESTAMPTZ,
    linked_pbefg_report_id  UUID            REFERENCES pbefg_reports (id) ON DELETE SET NULL,
    metadata                JSONB           NOT NULL DEFAULT '{}',
    created_by              UUID,
    updated_by              UUID,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_compliance_doc_number UNIQUE (document_number),
    CONSTRAINT chk_doc_version_positive CHECK (version >= 1),
    CONSTRAINT chk_doc_applicable_dates CHECK (
        applicable_until IS NULL OR applicable_from IS NULL OR applicable_until >= applicable_from
    ),
    CONSTRAINT chk_doc_access_level CHECK (
        access_level IN ('public', 'internal', 'restricted', 'confidential')
    )
);

COMMENT ON TABLE compliance_documents IS 'Versioned regulatory and compliance document repository';
COMMENT ON COLUMN compliance_documents.file_hash_sha256 IS 'SHA-256 hash of file content for integrity verification';
COMMENT ON COLUMN compliance_documents.access_level IS 'public | internal | restricted | confidential';
COMMENT ON COLUMN compliance_documents.is_encrypted IS 'Whether document is stored in encrypted form';

CREATE INDEX idx_compliance_docs_status             ON compliance_documents (status);
CREATE INDEX idx_compliance_docs_document_type      ON compliance_documents (document_type);
CREATE INDEX idx_compliance_docs_category           ON compliance_documents (category);
CREATE INDEX idx_compliance_docs_is_current         ON compliance_documents (is_current_version) WHERE is_current_version = TRUE;
CREATE INDEX idx_compliance_docs_previous_version   ON compliance_documents (previous_version_id) WHERE previous_version_id IS NOT NULL;
CREATE INDEX idx_compliance_docs_superseded_by      ON compliance_documents (superseded_by_id)    WHERE superseded_by_id IS NOT NULL;
CREATE INDEX idx_compliance_docs_regulatory_fw      ON compliance_documents (regulatory_framework) WHERE regulatory_framework IS NOT NULL;
CREATE INDEX idx_compliance_docs_jurisdiction       ON compliance_documents (jurisdiction)         WHERE jurisdiction IS NOT NULL;
CREATE INDEX idx_compliance_docs_review_due         ON compliance_documents (review_due_at)        WHERE review_due_at IS NOT NULL;
CREATE INDEX idx_compliance_docs_applicable_from    ON compliance_documents (applicable_from)      WHERE applicable_from IS NOT NULL;
CREATE INDEX idx_compliance_docs_pbefg_report       ON compliance_documents (linked_pbefg_report_id) WHERE linked_pbefg_report_id IS NOT NULL;
CREATE INDEX idx_compliance_docs_tags               ON compliance_documents USING GIN (tags);
CREATE INDEX idx_compliance_docs_access_roles       ON compliance_documents USING GIN (access_roles);
CREATE INDEX idx_compliance_docs_metadata           ON compliance_documents USING GIN (metadata);
CREATE INDEX idx_compliance_docs_created_at         ON compliance_documents (created_at DESC);

-- ============================================================
-- 5. compliance_checks
-- ============================================================

CREATE TABLE compliance_checks (
    id                      UUID                        PRIMARY KEY DEFAULT uuid_generate_v4(),
    check_run_id            UUID                        NOT NULL,
    check_name              VARCHAR(256)                NOT NULL,
    check_code              VARCHAR(128)                NOT NULL,
    check_category          VARCHAR(128)                NOT NULL,
    regulatory_framework    VARCHAR(128),
    regulatory_reference    VARCHAR(256),
    description             TEXT,
    status                  compliance_check_status     NOT NULL,
    severity                compliance_check_severity   NOT NULL,
    score                   NUMERIC(5, 2)               CHECK (score BETWEEN 0 AND 100),
    resource_type           VARCHAR(128),
    resource_id             VARCHAR(256),
    resource_name           VARCHAR(512),
    finding_summary         TEXT,
    finding_details         JSONB                       NOT NULL DEFAULT '{}',
    remediation_required    BOOLEAN                     NOT NULL DEFAULT FALSE,
    remediation_steps       TEXT,
    remediation_due_at      TIMESTAMPTZ,
    remediation_completed_at TIMESTAMPTZ,
    remediation_ticket_ref  VARCHAR(256),
    false_positive          BOOLEAN                     NOT NULL DEFAULT FALSE,
    false_positive_reason   TEXT,
    suppressed              BOOLEAN                     NOT NULL DEFAULT FALSE,
    suppressed_reason       TEXT,
    suppressed_until        TIMESTAMPTZ,
    suppressed_by           UUID,
    previous_check_id       UUID                        REFERENCES compliance_checks (id) ON DELETE SET NULL,
    status_changed          BOOLEAN                     NOT NULL DEFAULT FALSE,
    first_detected_at       TIMESTAMPTZ                 NOT NULL DEFAULT NOW(),
    last_detected_at        TIMESTAMPTZ                 NOT NULL DEFAULT NOW(),
    detection_count         INTEGER                     NOT NULL DEFAULT 1 CHECK (detection_count >= 1),
    execution_duration_ms   INTEGER,
    executed_by             UUID,
    executed_by_service     VARCHAR(128),
    environment             VARCHAR(64)                 NOT NULL DEFAULT 'production',
    metadata                JSONB                       NOT NULL DEFAULT '{}',
    tags                    TEXT[]                      NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ                 NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ                 NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE compliance_checks IS 'Results of automated compliance checks across systems and configurations';
COMMENT ON COLUMN compliance_checks.check_run_id IS 'Groups all checks executed in the same run/batch';
COMMENT ON COLUMN compliance_checks.detection_count IS 'Number of consecutive times this finding has been detected';
COMMENT ON COLUMN compliance_checks.score IS 'Compliance score 0-100 where applicable';

CREATE INDEX idx_compliance_checks_check_run_id     ON compliance_checks (check_run_id);
CREATE INDEX idx_compliance_checks_check_code       ON compliance_checks (check_code);
CREATE INDEX idx_compliance_checks_check_category   ON compliance_checks (check_category);
CREATE INDEX idx_compliance_checks_status           ON compliance_checks (status);
CREATE INDEX idx_compliance_checks_severity         ON compliance_checks (severity);
CREATE INDEX idx_compliance_checks_resource         ON compliance_checks (resource_type, resource_id);
CREATE INDEX idx_compliance_checks_remediation      ON compliance_checks (remediation_required, remediation_due_at) WHERE remediation_required = TRUE;
CREATE INDEX idx_compliance_checks_suppressed       ON compliance_checks (suppressed)                              WHERE suppressed = TRUE;
CREATE INDEX idx_compliance_checks_regulatory_fw    ON compliance_checks (regulatory_framework)                   WHERE regulatory_framework IS NOT NULL;
CREATE INDEX idx_compliance_checks_first_detected   ON compliance_checks (first_detected_at DESC);
CREATE INDEX idx_compliance_checks_last_detected    ON compliance_checks (last_detected_at DESC);
CREATE INDEX idx_compliance_checks_previous         ON compliance_checks (previous_check_id)                      WHERE previous_check_id IS NOT NULL;
CREATE INDEX idx_compliance_checks_environment      ON compliance_checks (environment);
CREATE INDEX idx_compliance_checks_finding_details  ON compliance_checks USING GIN (finding_details);
CREATE INDEX idx_compliance_checks_tags             ON compliance_checks USING GIN (tags);
CREATE INDEX idx_compliance_checks_metadata         ON compliance_checks USING GIN (metadata);
CREATE INDEX idx_compliance_checks_open_findings    ON compliance_checks (severity, created_at DESC)
    WHERE status = 'fail' AND suppressed = FALSE AND false_positive = FALSE;

-- ============================================================
-- 6. data_retention_policies
-- ============================================================

CREATE TABLE data_retention_policies (
    id                          UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_name                 VARCHAR(256)    NOT NULL,
    policy_code                 VARCHAR(128)    NOT NULL,
    description                 TEXT,
    is_active                   BOOLEAN         NOT NULL DEFAULT TRUE,
    is_default                  BOOLEAN         NOT NULL DEFAULT FALSE,
    data_category               VARCHAR(128)    NOT NULL,
    data_subcategory            VARCHAR(128),
    resource_type               VARCHAR(128)    NOT NULL,
    resource_filter             JSONB           NOT NULL DEFAULT '{}',
    retention_period_days       INTEGER         NOT NULL CHECK (retention_period_days > 0),
    minimum_retention_days      INTEGER         CHECK (minimum_retention_days > 0),
    maximum_retention_days      INTEGER         CHECK (maximum_retention_days > 0),
    action_on_expiry            retention_action NOT NULL DEFAULT 'archive',
    archive_destination         TEXT,
    anonymization_config        JSONB           NOT NULL DEFAULT '{}',
    legal_basis                 VARCHAR(256)    NOT NULL,
    regulatory_framework        VARCHAR(128),
    regulatory_reference        VARCHAR(256),
    jurisdiction                VARCHAR(128),
    applies_to_tenants          TEXT[]          NOT NULL DEFAULT '{}',
    excluded_tenants            TEXT[]          NOT NULL DEFAULT '{}',
    schedule_cron               VARCHAR(128),
    last_run_at                 TIMESTAMPTZ,
    last_run_status             VARCHAR(64),
    last_run_affected_count     INTEGER,
    next_run_at                 TIMESTAMPTZ,
    notification_emails         TEXT[]          NOT NULL DEFAULT '{}',
    notify_before_days          INTEGER         CHECK (notify_before_days >= 0),
    requires_approval           BOOLEAN         NOT NULL DEFAULT FALSE,
    approved_by                 UUID,
    approved_at                 TIMESTAMPTZ,
    effective_from              DATE            NOT NULL DEFAULT CURRENT_DATE,
    effective_until             DATE,
    version                     INTEGER         NOT NULL DEFAULT 1 CHECK (version >= 1),
    metadata                    JSONB           NOT NULL DEFAULT '{}',
    created_by                  UUID,
    updated_by                  UUID,
    created_at                  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_retention_policy_code UNIQUE (policy_code),
    CONSTRAINT chk_retention_min_max CHECK (
        minimum_retention_days IS NULL OR
        maximum_retention_days IS NULL OR
        minimum_retention_days <= maximum_retention_days
    ),
    CONSTRAINT chk_retention_period_in_bounds CHECK (
        (minimum_retention_days IS NULL OR retention_period_days >= minimum_retention_days) AND
        (maximum_retention_days IS NULL OR retention_period_days <= maximum_retention_days)
    ),
    CONSTRAINT chk_retention_effective_dates CHECK (
        effective_until IS NULL OR effective_until > effective_from
    )
);

COMMENT ON TABLE data_retention_policies IS 'Configurable data retention rules aligned to regulatory requirements';
COMMENT ON COLUMN data_retention_policies.resource_filter IS 'JSON filter criteria to identify records subject to this policy';
COMMENT ON COLUMN data_retention_policies.anonymization_config IS 'Field-level config for anonymization actions';
COMMENT ON COLUMN data_retention_policies.schedule_cron IS 'Cron expression for automated retention job scheduling';

CREATE INDEX idx_retention_policies_is_active       ON data_retention_policies (is_active);
CREATE INDEX idx_retention_policies_data_category   ON data_retention_policies (data_category);
CREATE INDEX idx_retention_policies_resource_type   ON data_retention_policies (resource_type);
CREATE INDEX idx_retention_policies_regulatory_fw   ON data_retention_policies (regulatory_framework) WHERE regulatory_framework IS NOT NULL;
CREATE INDEX idx_retention_policies_jurisdiction    ON data_retention_policies (jurisdiction)         WHERE jurisdiction IS NOT NULL;
CREATE INDEX idx_retention_policies_next_run        ON data_retention_policies (next_run_at)          WHERE is_active = TRUE;
CREATE INDEX idx_retention_policies_effective_from  ON data_retention_policies (effective_from);
CREATE INDEX idx_retention_policies_action          ON data_retention_policies (action_on_expiry);
CREATE INDEX idx_retention_policies_applies_to      ON data_retention_policies USING GIN (applies_to_tenants);
CREATE INDEX idx_retention_policies_metadata        ON data_retention_policies USING GIN (metadata);
CREATE INDEX idx_retention_policies_resource_filter ON data_retention_policies USING GIN (resource_filter);

-- ============================================================
-- 7. authority_access_logs
-- ============================================================

CREATE TABLE authority_access_logs (
    id                      UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    authority_name          VARCHAR(256)    NOT NULL,
    authority_code          VARCHAR(64)     NOT NULL,
    authority_user_id       VARCHAR(256),
    authority_user_name     VARCHAR(256),
    authority_user_email    VARCHAR(320),
    access_type             VARCHAR(64)     NOT NULL,
    access_method           VARCHAR(64)     NOT NULL DEFAULT 'api',
    resource_type           VARCHAR(128)    NOT NULL,
    resource_id             VARCHAR(256),
    resource_description    TEXT,
    pbefg_report_id         UUID            REFERENCES pbefg_reports (id) ON DELETE SET NULL,
    document_id             UUID            REFERENCES compliance_documents (id) ON DELETE SET NULL,
    request_reference       VARCHAR(256),
    legal_basis             VARCHAR(512),
    purpose                 TEXT,
    data_categories_accessed TEXT[]         NOT NULL DEFAULT '{}',
    ip_address              INET            NOT NULL,
    user_agent              TEXT,
    session_token_hash      VARCHAR(64),
    request_id              VARCHAR(256),
    response_status         SMALLINT,
    data_transferred_bytes  BIGINT          CHECK (data_transferred_bytes >= 0),
    duration_ms             INTEGER,
    access_granted          BOOLEAN         NOT NULL DEFAULT TRUE,
    denial_reason           TEXT,
    requires_audit_response BOOLEAN         NOT NULL DEFAULT FALSE,
    audit_response_due_at   TIMESTAMPTZ,
    audit_response_sent_at  TIMESTAMPTZ,
    notified_dpo            BOOLEAN         NOT NULL DEFAULT FALSE,
    notified_dpo_at         TIMESTAMPTZ,
    metadata                JSONB           NOT NULL DEFAULT '{}',
    accessed_at             TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE authority_access_logs IS 'Immutable log of all regulatory authority access to reports and data';
COMMENT ON COLUMN authority_access_logs.session_token_hash IS 'Hashed authority session token for correlation without storing plaintext';
COMMENT ON COLUMN authority_access_logs.requires_audit_response IS 'Whether this access requires a formal audit response to DPO/management';
COMMENT ON COLUMN authority_access_logs.access_method IS 'api | portal | email | physical | other';

CREATE INDEX idx_authority_access_authority_code   ON authority_access_logs (authority_code);
CREATE INDEX idx_authority_access_authority_name   ON authority_access_logs (authority_name);
CREATE INDEX idx_authority_access_resource         ON authority_access_logs (resource_type, resource_id);
CREATE INDEX idx_authority_access_pbefg_report     ON authority_access_logs (pbefg_report_id)   WHERE pbefg_report_id IS NOT NULL;
CREATE INDEX idx_authority_access_document         ON authority_access_logs (document_id)        WHERE document_id IS NOT NULL;
CREATE INDEX idx_authority_access_ip_address       ON authority_access_logs (ip_address);
CREATE INDEX idx_authority_access_accessed_at      ON authority_access_logs (accessed_at DESC);
CREATE INDEX idx_authority_access_granted          ON authority_access_logs (access_granted);
CREATE INDEX idx_authority_access_audit_response   ON authority_access_logs (requires_audit_response, audit_response_due_at)
    WHERE requires_audit_response = TRUE;
CREATE INDEX idx_authority_access_metadata         ON authority_access_logs USING GIN (metadata);
CREATE INDEX idx_authority_access_data_categories  ON authority_access_logs USING GIN (data_categories_accessed);

-- Authority access logs are immutable
CREATE RULE authority_access_logs_no_update AS ON UPDATE TO authority_access_logs DO INSTEAD NOTHING;
CREATE RULE authority_access_logs_no_delete AS ON DELETE TO authority_access_logs DO INSTEAD NOTHING;

-- ============================================================
-- 8. consent_records
-- ============================================================

CREATE TABLE consent_records (
    id                      UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    consent_reference       VARCHAR(128)    NOT NULL,
    data_subject_id         UUID            NOT NULL,
    data_subject_email      VARCHAR(320),
    data_subject_type       VARCHAR(64)     NOT NULL DEFAULT 'user',
    consent_type            VARCHAR(128)    NOT NULL,
    consent_category        VARCHAR(128)    NOT NULL,
    consent_purpose         TEXT            NOT NULL,
    legal_basis             VARCHAR(256)    NOT NULL DEFAULT 'consent',
    data_categories         TEXT[]          NOT NULL DEFAULT '{}',
    processing_activities   TEXT[]          NOT NULL DEFAULT '{}',
    third_party_sharing     BOOLEAN         NOT NULL DEFAULT FALSE,
    third_parties           JSONB           NOT NULL DEFAULT '[]',
    data_transfers_outside_eea BOOLEAN      NOT NULL DEFAULT FALSE,
    transfer_safeguards     TEXT,
    status                  consent_status  NOT NULL DEFAULT 'pending',
    version                 INTEGER         NOT NULL DEFAULT 1,
    is_current_version      BOOLEAN         NOT NULL DEFAULT TRUE,
    previous_consent_id     UUID            REFERENCES consent_records (id) ON DELETE SET NULL,
    granted_at              TIMESTAMPTZ,
    granted_via             VARCHAR(128),
    granted_ip              INET,
    granted_user_agent      TEXT,
    withdrawal_requested_at TIMESTAMPTZ,
    withdrawn_at            TIMESTAMPTZ,
    withdrawal_reason       TEXT,
    withdrawal_method       VARCHAR(128),
    expiry_at               TIMESTAMPTZ,
    is_mandatory            BOOLEAN         NOT NULL DEFAULT FALSE,
    allows_withdrawal       BOOLEAN         NOT NULL DEFAULT TRUE,
    minimum_age             SMALLINT        CHECK (minimum_age >= 0),
    age_verified            BOOLEAN         NOT NULL DEFAULT FALSE,
    age_verified_at         TIMESTAMPTZ,
    proof_of_consent        TEXT,
    proof_type              VARCHAR(64),
    consent_text_version    VARCHAR(64),
    consent_text_hash       VARCHAR(64),
    policy_version          VARCHAR(64),
    channel                 VARCHAR(128),
    locale                  VARCHAR(16)     NOT NULL DEFAULT 'de-DE',
    linked_gdpr_request_id  UUID            REFERENCES gdpr_requests (id) ON DELETE SET NULL,
    metadata                JSONB           NOT NULL DEFAULT '{}',
    created_by              UUID,
    updated_by              UUID,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_consent_reference UNIQUE (consent_reference),
    CONSTRAINT chk_consent_withdrawal_after_grant CHECK (
        withdrawn_at IS NULL OR granted_at IS NULL OR withdrawn_at >= granted_at
    ),
    CONSTRAINT chk_consent_expiry_after_grant CHECK (
        expiry_at IS NULL OR granted_at IS NULL OR expiry_at > granted_at
    ),
    CONSTRAINT chk_consent_withdrawal_allowed CHECK (
        is_mandatory = FALSE OR allows_withdrawal = FALSE
    )
);

COMMENT ON TABLE consent_records IS 'GDPR-compliant consent records with full audit trail per Art.7 and Recital 42';
COMMENT ON COLUMN consent_records.consent_text_hash IS 'Hash of the exact consent text shown to the data subject';
COMMENT ON COLUMN consent_records.proof_of_consent IS 'Reference or hash of consent proof artifact (e.g., signed document, checkbox event)';
COMMENT ON COLUMN consent_records.is_mandatory IS 'Whether this consent is required for service delivery';
COMMENT ON COLUMN consent_records.third_parties IS 'JSON array of third party details if sharing enabled';

CREATE INDEX idx_consent_data_subject_id        ON consent_records (data_subject_id);
CREATE INDEX idx_consent_data_subject_email     ON consent_records (data_subject_email)   WHERE data_subject_email IS NOT NULL;
CREATE INDEX idx_consent_status                 ON consent_records (status);
CREATE INDEX idx_consent_type                   ON consent_records (consent_type);
CREATE INDEX idx_consent_category               ON consent_records (consent_category);
CREATE INDEX idx_consent_is_current             ON consent_records (is_current_version)   WHERE is_current_version = TRUE;
CREATE INDEX idx_consent_granted_at             ON consent_records (granted_at)           WHERE granted_at IS NOT NULL;
CREATE INDEX idx_consent_withdrawn_at           ON consent_records (withdrawn_at)         WHERE withdrawn_at IS NOT NULL;
CREATE INDEX idx_consent_expiry_at              ON consent_records (expiry_at)            WHERE expiry_at IS NOT NULL;
CREATE INDEX idx_consent_previous              ON consent_records (previous_consent_id)  WHERE previous_consent_id IS NOT NULL;
CREATE INDEX idx_consent_gdpr_request          ON consent_records (linked_gdpr_request_id) WHERE linked_gdpr_request_id IS NOT NULL;
CREATE INDEX idx_consent_data_categories       ON consent_records USING GIN (data_categories);
CREATE INDEX idx_consent_processing_activities ON consent_records USING GIN (processing_activities);
CREATE INDEX idx_consent_third_parties         ON consent_records USING GIN (third_parties);
CREATE INDEX idx_consent_metadata              ON consent_records USING GIN (metadata);
CREATE INDEX idx_consent_active                ON consent_records (data_subject_id, consent_type, status)
    WHERE status = 'granted' AND is_current_version = TRUE;

-- ============================================================
-- Triggers: updated_at auto-maintenance
-- ============================================================

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_gdpr_requests_updated_at
    BEFORE UPDATE ON gdpr_requests
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_pbefg_reports_updated_at
    BEFORE UPDATE ON pbefg_reports
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_compliance_documents_updated_at
    BEFORE UPDATE ON compliance_documents
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_compliance_checks_updated_at
    BEFORE UPDATE ON compliance_checks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_data_retention_policies_updated_at
    BEFORE UPDATE ON data_retention_policies
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_consent_records_updated_at
    BEFORE UPDATE ON consent_records
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================
-- Triggers: GDPR request number auto-generation
-- ============================================================

CREATE SEQUENCE gdpr_request_seq START 1 INCREMENT 1;

CREATE OR REPLACE FUNCTION generate_gdpr_request_number()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.request_number IS NULL OR NEW.request_number = '' THEN
        NEW.request_number = 'GDPR-' ||
            TO_CHAR(NOW(), 'YYYY') || '-' ||
            LPAD(NEXTVAL('gdpr_request_seq')::TEXT, 6, '0');
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_gdpr_requests_request_number
    BEFORE INSERT ON gdpr_requests
    FOR EACH ROW EXECUTE FUNCTION generate_gdpr_request_number();

-- ============================================================
-- Triggers: PBefG report number auto-generation
-- ============================================================

CREATE SEQUENCE pbefg_report_seq START 1 INCREMENT 1;

CREATE OR REPLACE FUNCTION generate_pbefg_report_number()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.report_number IS NULL OR NEW.report_number = '' THEN
        NEW.report_number = 'PBEFG-' ||
            TO_CHAR(NOW(), 'YYYY') || '-' ||
            LPAD(NEXTVAL('pbefg_report_seq')::TEXT, 6, '0');
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_pbefg_reports_report_number
    BEFORE INSERT ON pbefg_reports
    FOR EACH ROW EXECUTE FUNCTION generate_pbefg_report_number();

-- ============================================================
-- Triggers: Compliance document number auto-generation
-- ============================================================

CREATE SEQUENCE compliance_doc_seq START 1 INCREMENT 1;

CREATE OR REPLACE FUNCTION generate_document_number()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.document_number IS NULL OR NEW.document_number = '' THEN
        NEW.document_number = 'DOC-' ||
            TO_CHAR(NOW(), 'YYYY') || '-' ||
            LPAD(NEXTVAL('compliance_doc_seq')::TEXT, 6, '0');
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_compliance_docs_document_number
    BEFORE INSERT ON compliance_documents
    FOR EACH ROW EXECUTE FUNCTION generate_document_number();

-- ============================================================
-- Triggers: Consent reference auto-generation
-- ============================================================

CREATE SEQUENCE consent_record_seq START 1 INCREMENT 1;

CREATE OR REPLACE FUNCTION generate_consent_reference()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.consent_reference IS NULL OR NEW.consent_reference = '' THEN
        NEW.consent_reference = 'CONS-' ||
            TO_CHAR(NOW(), 'YYYY') || '-' ||
            LPAD(NEXTVAL('consent_record_seq')::TEXT, 8, '0');
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_consent_records_reference
    BEFORE INSERT ON consent_records
    FOR EACH ROW EXECUTE FUNCTION generate_consent_reference();

-- ============================================================
-- Triggers: Audit log checksum generation
-- ============================================================

CREATE OR REPLACE FUNCTION generate_audit_log_checksum()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.checksum = ENCODE(
        DIGEST(
            COALESCE(NEW.id::TEXT, '') ||
            COALESCE(NEW.user_id::TEXT, '') ||
            COALESCE(NEW.action::TEXT, '') ||
            COALESCE(NEW.resource_type, '') ||
            COALESCE(NEW.resource_id, '') ||
            COALESCE(NEW.created_at::TEXT, ''),
            'sha256'
        ),
        'hex'
    );
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_audit_logs_checksum
    BEFORE INSERT ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION generate_audit_log_checksum();

-- ============================================================
-- Triggers: Enforce is_current_version consistency
-- ============================================================

CREATE OR REPLACE FUNCTION update_previous_document_version()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.is_current_version = TRUE AND NEW.previous_version_id IS NOT NULL THEN
        UPDATE compliance_documents
           SET is_current_version = FALSE,
               updated_at = NOW()
         WHERE id = NEW.previous_version_id
           AND is_current_version = TRUE;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_compliance_docs_versioning
    AFTER INSERT OR UPDATE ON compliance_documents
    FOR EACH ROW EXECUTE FUNCTION update_previous_document_version();

CREATE OR REPLACE FUNCTION update_previous_consent_version()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.is_current_version = TRUE AND NEW.previous_consent_id IS NOT NULL THEN
        UPDATE consent_records
           SET is_current_version = FALSE,
               updated_at = NOW()
         WHERE id = NEW.previous_consent_id
           AND is_current_version = TRUE;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_consent_records_versioning
    AFTER INSERT OR UPDATE ON consent_records
    FOR EACH ROW EXECUTE FUNCTION update_previous_consent_version();

-- ============================================================
-- Row-Level Security Setup
-- ============================================================

ALTER TABLE audit_logs              ENABLE ROW LEVEL SECURITY;
ALTER TABLE gdpr_requests           ENABLE ROW LEVEL SECURITY;
ALTER TABLE pbefg_reports           ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_documents    ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_checks       ENABLE ROW LEVEL SECURITY;
ALTER TABLE data_retention_policies ENABLE ROW LEVEL SECURITY;
ALTER TABLE authority_access_logs   ENABLE ROW LEVEL SECURITY;
ALTER TABLE consent_records         ENABLE ROW LEVEL SECURITY;

-- Service-level unrestricted policy (application connects with this role)
CREATE POLICY compliance_service_full_access ON audit_logs
    FOR ALL TO compliance_service USING (TRUE) WITH CHECK (TRUE);

CREATE POLICY compliance_service_full_access ON gdpr_requests
    FOR ALL TO compliance_service USING (TRUE) WITH CHECK (TRUE);

CREATE POLICY compliance_service_full_access ON pbefg_reports
    FOR ALL TO compliance_service USING (TRUE) WITH CHECK (TRUE);

CREATE POLICY compliance_service_full_access ON compliance_documents
    FOR ALL TO compliance_service USING (TRUE) WITH CHECK (TRUE);

CREATE POLICY compliance_service_full_access ON compliance_checks
    FOR ALL TO compliance_service USING (TRUE) WITH CHECK (TRUE);

CREATE POLICY compliance_service_full_access ON data_retention_policies
    FOR ALL TO compliance_service USING (TRUE) WITH CHECK (TRUE);

CREATE POLICY compliance_service_full_access ON authority_access_logs
    FOR ALL TO compliance_service USING (TRUE) WITH CHECK (TRUE);

CREATE POLICY compliance_service_full_access ON consent_records
    FOR ALL TO compliance_service USING (TRUE) WITH CHECK (TRUE);

-- ============================================================
-- Table Partitioning Recommendations (applied as comments)
-- NOTE: For high-volume production deployments consider:
--   - audit_logs: PARTITION BY RANGE (created_at) monthly
--   - authority_access_logs: PARTITION BY RANGE (accessed_at) monthly
--   - compliance_checks: PARTITION BY RANGE (created_at) quarterly
-- ============================================================

-- ============================================================
-- Schema metadata
-- ============================================================

CREATE TABLE IF NOT EXISTS schema_migrations (
    version         VARCHAR(64)     PRIMARY KEY,
    description     TEXT            NOT NULL,
    applied_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    applied_by      VARCHAR(256)    NOT NULL DEFAULT CURRENT_USER,
    checksum        VARCHAR(64)
);

INSERT INTO schema_migrations (version, description)
VALUES (
    '001',
    'Initial schema: audit_logs, gdpr_requests, pbefg_reports, compliance_documents, compliance_checks, data_retention_policies, authority_access_logs, consent_records'
);

COMMIT;

-- ============================================================
-- Migration: 001_initial_schema.sql
-- Description: Initial schema for Support Service
-- Created: 2024-01-01
-- ============================================================

BEGIN;

-- ============================================================
-- Extensions
-- ============================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For full-text search on articles

-- ============================================================
-- ENUM Types
-- ============================================================

CREATE TYPE ticket_status AS ENUM (
    'open',
    'in_progress',
    'pending_customer',
    'pending_agent',
    'on_hold',
    'resolved',
    'closed',
    'cancelled'
);

CREATE TYPE ticket_priority AS ENUM (
    'low',
    'medium',
    'high',
    'urgent',
    'critical'
);

CREATE TYPE ticket_category AS ENUM (
    'billing',
    'technical',
    'account',
    'product',
    'general',
    'feature_request',
    'bug_report',
    'other'
);

CREATE TYPE author_type AS ENUM (
    'customer',
    'agent',
    'system',
    'bot'
);

CREATE TYPE agent_role AS ENUM (
    'junior_agent',
    'agent',
    'senior_agent',
    'team_lead',
    'supervisor',
    'admin'
);

CREATE TYPE article_language AS ENUM (
    'en',
    'es',
    'fr',
    'de',
    'pt',
    'it',
    'nl',
    'ja',
    'zh',
    'ar'
);

-- ============================================================
-- Table: support_agents
-- Must be created before support_tickets due to FK reference
-- ============================================================

CREATE TABLE IF NOT EXISTS support_agents (
    id                  UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID            NOT NULL UNIQUE,
    name                VARCHAR(255)    NOT NULL,
    email               VARCHAR(320)    NOT NULL UNIQUE,
    role                agent_role      NOT NULL DEFAULT 'agent',
    department          VARCHAR(100)    NOT NULL DEFAULT 'general',
    is_active           BOOLEAN         NOT NULL DEFAULT TRUE,
    max_tickets         INTEGER         NOT NULL DEFAULT 20 CHECK (max_tickets > 0 AND max_tickets <= 500),
    current_tickets     INTEGER         NOT NULL DEFAULT 0 CHECK (current_tickets >= 0),
    skills              TEXT[]          NOT NULL DEFAULT '{}',
    timezone            VARCHAR(100)    NOT NULL DEFAULT 'UTC',
    avatar_url          TEXT,
    phone               VARCHAR(50),
    notes               TEXT,
    last_active_at      TIMESTAMPTZ,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_current_lte_max CHECK (current_tickets <= max_tickets)
);

COMMENT ON TABLE support_agents IS 'Stores support agent profiles and workload capacity';
COMMENT ON COLUMN support_agents.user_id IS 'Reference to the user in the auth/user service';
COMMENT ON COLUMN support_agents.max_tickets IS 'Maximum concurrent tickets an agent can handle';
COMMENT ON COLUMN support_agents.current_tickets IS 'Current number of open/in-progress tickets assigned';
COMMENT ON COLUMN support_agents.skills IS 'Array of skill tags for intelligent routing';

-- ============================================================
-- Table: support_tickets
-- ============================================================

CREATE TABLE IF NOT EXISTS support_tickets (
    id                  UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_number       SERIAL          UNIQUE NOT NULL,
    title               VARCHAR(500)    NOT NULL CHECK (char_length(title) >= 5),
    description         TEXT            NOT NULL CHECK (char_length(description) >= 10),
    status              ticket_status   NOT NULL DEFAULT 'open',
    priority            ticket_priority NOT NULL DEFAULT 'medium',
    category            ticket_category NOT NULL DEFAULT 'general',
    customer_id         UUID            NOT NULL,
    agent_id            UUID            REFERENCES support_agents(id) ON DELETE SET NULL,
    tags                TEXT[]          NOT NULL DEFAULT '{}',
    metadata            JSONB           NOT NULL DEFAULT '{}',
    satisfaction_score  SMALLINT        CHECK (satisfaction_score BETWEEN 1 AND 5),
    satisfaction_note   TEXT,
    first_response_at   TIMESTAMPTZ,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    resolved_at         TIMESTAMPTZ,
    closed_at           TIMESTAMPTZ,
    due_date            TIMESTAMPTZ,

    CONSTRAINT chk_resolved_after_created CHECK (
        resolved_at IS NULL OR resolved_at >= created_at
    ),
    CONSTRAINT chk_due_after_created CHECK (
        due_date IS NULL OR due_date >= created_at
    ),
    CONSTRAINT chk_closed_requires_terminal_status CHECK (
        closed_at IS NULL OR status IN ('closed', 'cancelled', 'resolved')
    )
);

COMMENT ON TABLE support_tickets IS 'Core support ticket entity';
COMMENT ON COLUMN support_tickets.ticket_number IS 'Human-readable sequential ticket number';
COMMENT ON COLUMN support_tickets.customer_id IS 'Reference to customer in the user/customer service';
COMMENT ON COLUMN support_tickets.agent_id IS 'Assigned support agent; NULL means unassigned';
COMMENT ON COLUMN support_tickets.metadata IS 'Flexible JSONB field for extra data (browser info, source, etc.)';
COMMENT ON COLUMN support_tickets.satisfaction_score IS 'Customer satisfaction rating 1-5 after resolution';
COMMENT ON COLUMN support_tickets.first_response_at IS 'Timestamp of first agent response for SLA tracking';
COMMENT ON COLUMN support_tickets.due_date IS 'SLA deadline for ticket resolution';

-- ============================================================
-- Table: ticket_comments
-- ============================================================

CREATE TABLE IF NOT EXISTS ticket_comments (
    id              UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id       UUID            NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    author_id       UUID            NOT NULL,
    author_type     author_type     NOT NULL DEFAULT 'customer',
    content         TEXT            NOT NULL CHECK (char_length(content) >= 1),
    is_internal     BOOLEAN         NOT NULL DEFAULT FALSE,
    attachments     UUID[]          NOT NULL DEFAULT '{}',
    edited_at       TIMESTAMPTZ,
    edited_by       UUID,
    is_deleted      BOOLEAN         NOT NULL DEFAULT FALSE,
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_internal_agent_only CHECK (
        is_internal = FALSE OR author_type IN ('agent', 'system')
    )
);

COMMENT ON TABLE ticket_comments IS 'Comments and replies on support tickets';
COMMENT ON COLUMN ticket_comments.author_id IS 'ID of user, agent, or system that created this comment';
COMMENT ON COLUMN ticket_comments.is_internal IS 'Internal notes visible only to agents, not customers';
COMMENT ON COLUMN ticket_comments.attachments IS 'Array of ticket_attachment IDs linked to this comment';
COMMENT ON COLUMN ticket_comments.is_deleted IS 'Soft delete flag; content should be blanked on deletion';

-- ============================================================
-- Table: knowledge_base_articles
-- ============================================================

CREATE TABLE IF NOT EXISTS knowledge_base_articles (
    id              UUID                PRIMARY KEY DEFAULT uuid_generate_v4(),
    title           VARCHAR(500)        NOT NULL CHECK (char_length(title) >= 5),
    content         TEXT                NOT NULL,
    category        ticket_category     NOT NULL DEFAULT 'general',
    tags            TEXT[]              NOT NULL DEFAULT '{}',
    language        article_language    NOT NULL DEFAULT 'en',
    is_published    BOOLEAN             NOT NULL DEFAULT FALSE,
    view_count      BIGINT              NOT NULL DEFAULT 0 CHECK (view_count >= 0),
    helpful_count   INTEGER             NOT NULL DEFAULT 0 CHECK (helpful_count >= 0),
    not_helpful_count INTEGER           NOT NULL DEFAULT 0 CHECK (not_helpful_count >= 0),
    author_id       UUID                NOT NULL,
    reviewer_id     UUID,
    reviewed_at     TIMESTAMPTZ,
    published_at    TIMESTAMPTZ,
    slug            VARCHAR(600)        UNIQUE,
    excerpt         VARCHAR(1000),
    search_vector   TSVECTOR,
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_published_has_date CHECK (
        is_published = FALSE OR published_at IS NOT NULL
    ),
    CONSTRAINT chk_reviewed_has_reviewer CHECK (
        reviewed_at IS NULL OR reviewer_id IS NOT NULL
    )
);

COMMENT ON TABLE knowledge_base_articles IS 'Knowledge base articles for self-service support';
COMMENT ON COLUMN knowledge_base_articles.tags IS 'Searchable tags array';
COMMENT ON COLUMN knowledge_base_articles.language IS 'ISO 639-1 language code for the article';
COMMENT ON COLUMN knowledge_base_articles.view_count IS 'Incremented on each article view';
COMMENT ON COLUMN knowledge_base_articles.helpful_count IS 'Number of users who found the article helpful';
COMMENT ON COLUMN knowledge_base_articles.search_vector IS 'Pre-computed tsvector for full-text search';
COMMENT ON COLUMN knowledge_base_articles.slug IS 'URL-friendly unique identifier';

-- ============================================================
-- Table: ticket_attachments
-- ============================================================

CREATE TABLE IF NOT EXISTS ticket_attachments (
    id              UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id       UUID            NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    filename        VARCHAR(500)    NOT NULL CHECK (char_length(filename) >= 1),
    file_path       TEXT            NOT NULL,
    file_size       BIGINT          NOT NULL CHECK (file_size > 0 AND file_size <= 52428800), -- max 50MB
    mime_type       VARCHAR(255)    NOT NULL,
    uploaded_by     UUID            NOT NULL,
    uploader_type   author_type     NOT NULL DEFAULT 'customer',
    checksum        VARCHAR(128),
    is_deleted      BOOLEAN         NOT NULL DEFAULT FALSE,
    deleted_at      TIMESTAMPTZ,
    storage_backend VARCHAR(50)     NOT NULL DEFAULT 'local',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE ticket_attachments IS 'File attachments associated with support tickets';
COMMENT ON COLUMN ticket_attachments.file_size IS 'File size in bytes; max 50MB enforced by constraint';
COMMENT ON COLUMN ticket_attachments.checksum IS 'SHA-256 or MD5 checksum for file integrity verification';
COMMENT ON COLUMN ticket_attachments.storage_backend IS 'Storage provider: local, s3, gcs, azure';
COMMENT ON COLUMN ticket_attachments.is_deleted IS 'Soft delete; actual file removal handled by cleanup job';

-- ============================================================
-- Indexes: support_agents
-- ============================================================

CREATE INDEX idx_agents_user_id         ON support_agents(user_id);
CREATE INDEX idx_agents_email           ON support_agents(email);
CREATE INDEX idx_agents_is_active       ON support_agents(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_agents_department      ON support_agents(department);
CREATE INDEX idx_agents_role            ON support_agents(role);
CREATE INDEX idx_agents_capacity        ON support_agents(is_active, current_tickets, max_tickets)
    WHERE is_active = TRUE;

-- ============================================================
-- Indexes: support_tickets
-- ============================================================

CREATE INDEX idx_tickets_customer_id    ON support_tickets(customer_id);
CREATE INDEX idx_tickets_agent_id       ON support_tickets(agent_id);
CREATE INDEX idx_tickets_status         ON support_tickets(status);
CREATE INDEX idx_tickets_priority       ON support_tickets(priority);
CREATE INDEX idx_tickets_category       ON support_tickets(category);
CREATE INDEX idx_tickets_created_at     ON support_tickets(created_at DESC);
CREATE INDEX idx_tickets_updated_at     ON support_tickets(updated_at DESC);
CREATE INDEX idx_tickets_due_date       ON support_tickets(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_tickets_open_priority  ON support_tickets(priority, created_at DESC)
    WHERE status IN ('open', 'in_progress');
CREATE INDEX idx_tickets_agent_open     ON support_tickets(agent_id, status)
    WHERE agent_id IS NOT NULL AND status NOT IN ('resolved', 'closed', 'cancelled');
CREATE INDEX idx_tickets_tags           ON support_tickets USING GIN(tags);
CREATE INDEX idx_tickets_metadata       ON support_tickets USING GIN(metadata);

-- ============================================================
-- Indexes: ticket_comments
-- ============================================================

CREATE INDEX idx_comments_ticket_id     ON ticket_comments(ticket_id, created_at ASC);
CREATE INDEX idx_comments_author_id     ON ticket_comments(author_id);
CREATE INDEX idx_comments_is_internal   ON ticket_comments(ticket_id, is_internal)
    WHERE is_deleted = FALSE;
CREATE INDEX idx_comments_not_deleted   ON ticket_comments(ticket_id)
    WHERE is_deleted = FALSE;

-- ============================================================
-- Indexes: knowledge_base_articles
-- ============================================================

CREATE INDEX idx_kb_category            ON knowledge_base_articles(category);
CREATE INDEX idx_kb_language            ON knowledge_base_articles(language);
CREATE INDEX idx_kb_is_published        ON knowledge_base_articles(is_published)
    WHERE is_published = TRUE;
CREATE INDEX idx_kb_tags                ON knowledge_base_articles USING GIN(tags);
CREATE INDEX idx_kb_search_vector       ON knowledge_base_articles USING GIN(search_vector);
CREATE INDEX idx_kb_slug                ON knowledge_base_articles(slug) WHERE slug IS NOT NULL;
CREATE INDEX idx_kb_view_count          ON knowledge_base_articles(view_count DESC)
    WHERE is_published = TRUE;
CREATE INDEX idx_kb_author_id           ON knowledge_base_articles(author_id);
CREATE INDEX idx_kb_title_trgm          ON knowledge_base_articles USING GIN(title gin_trgm_ops);

-- ============================================================
-- Indexes: ticket_attachments
-- ============================================================

CREATE INDEX idx_attachments_ticket_id  ON ticket_attachments(ticket_id);
CREATE INDEX idx_attachments_uploaded_by ON ticket_attachments(uploaded_by);
CREATE INDEX idx_attachments_active     ON ticket_attachments(ticket_id)
    WHERE is_deleted = FALSE;
CREATE INDEX idx_attachments_mime_type  ON ticket_attachments(mime_type);

-- ============================================================
-- Functions
-- ============================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Update agent current_tickets count on ticket assignment changes
CREATE OR REPLACE FUNCTION fn_sync_agent_ticket_count()
RETURNS TRIGGER AS $$
BEGIN
    -- Decrement old agent count
    IF OLD.agent_id IS NOT NULL AND OLD.agent_id IS DISTINCT FROM NEW.agent_id THEN
        UPDATE support_agents
        SET current_tickets = GREATEST(0, current_tickets - 1)
        WHERE id = OLD.agent_id;
    END IF;

    -- Increment new agent count
    IF NEW.agent_id IS NOT NULL AND NEW.agent_id IS DISTINCT FROM OLD.agent_id THEN
        UPDATE support_agents
        SET current_tickets = current_tickets + 1
        WHERE id = NEW.agent_id;
    END IF;

    -- If ticket resolved/closed/cancelled, decrement current agent
    IF NEW.agent_id IS NOT NULL
        AND NEW.agent_id = OLD.agent_id
        AND OLD.status NOT IN ('resolved', 'closed', 'cancelled')
        AND NEW.status IN ('resolved', 'closed', 'cancelled') THEN
        UPDATE support_agents
        SET current_tickets = GREATEST(0, current_tickets - 1)
        WHERE id = NEW.agent_id;
    END IF;

    -- If ticket reopened from terminal state, increment agent count
    IF NEW.agent_id IS NOT NULL
        AND NEW.agent_id = OLD.agent_id
        AND OLD.status IN ('resolved', 'closed', 'cancelled')
        AND NEW.status NOT IN ('resolved', 'closed', 'cancelled') THEN
        UPDATE support_agents
        SET current_tickets = current_tickets + 1
        WHERE id = NEW.agent_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Update KB article search vector
CREATE OR REPLACE FUNCTION fn_update_kb_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector =
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.excerpt, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.content, '')), 'C') ||
        setweight(to_tsvector('english', array_to_string(NEW.tags, ' ')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Set first_response_at on first agent comment
CREATE OR REPLACE FUNCTION fn_set_first_response()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.author_type = 'agent' AND NEW.is_internal = FALSE THEN
        UPDATE support_tickets
        SET first_response_at = NEW.created_at
        WHERE id = NEW.ticket_id
          AND first_response_at IS NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- Triggers
-- ============================================================

CREATE TRIGGER trg_agents_updated_at
    BEFORE UPDATE ON support_agents
    FOR EACH ROW
    EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_tickets_updated_at
    BEFORE UPDATE ON support_tickets
    FOR EACH ROW
    EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_kb_articles_updated_at
    BEFORE UPDATE ON knowledge_base_articles
    FOR EACH ROW
    EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_sync_agent_ticket_count
    AFTER UPDATE OF agent_id, status ON support_tickets
    FOR EACH ROW
    EXECUTE FUNCTION fn_sync_agent_ticket_count();

CREATE TRIGGER trg_kb_search_vector_insert
    BEFORE INSERT ON knowledge_base_articles
    FOR EACH ROW
    EXECUTE FUNCTION fn_update_kb_search_vector();

CREATE TRIGGER trg_kb_search_vector_update
    BEFORE UPDATE OF title, content, excerpt, tags ON knowledge_base_articles
    FOR EACH ROW
    EXECUTE FUNCTION fn_update_kb_search_vector();

CREATE TRIGGER trg_set_first_response
    AFTER INSERT ON ticket_comments
    FOR EACH ROW
    EXECUTE FUNCTION fn_set_first_response();

-- ============================================================
-- Views
-- ============================================================

CREATE OR REPLACE VIEW v_open_tickets AS
    SELECT
        t.id,
        t.ticket_number,
        t.title,
        t.status,
        t.priority,
        t.category,
        t.customer_id,
        t.agent_id,
        a.name          AS agent_name,
        a.department    AS agent_department,
        t.due_date,
        t.created_at,
        t.updated_at,
        EXTRACT(EPOCH FROM (NOW() - t.created_at)) / 3600 AS age_hours,
        CASE
            WHEN t.due_date IS NOT NULL AND t.due_date < NOW() THEN TRUE
            ELSE FALSE
        END AS is_overdue
    FROM support_tickets t
    LEFT JOIN support_agents a ON a.id = t.agent_id
    WHERE t.status NOT IN ('resolved', 'closed', 'cancelled');

COMMENT ON VIEW v_open_tickets IS 'Active tickets with agent info and overdue flag';

CREATE OR REPLACE VIEW v_agent_workload AS
    SELECT
        a.id,
        a.name,
        a.email,
        a.department,
        a.role,
        a.max_tickets,
        a.current_tickets,
        a.max_tickets - a.current_tickets   AS available_slots,
        ROUND(
            (a.current_tickets::DECIMAL / NULLIF(a.max_tickets, 0)) * 100, 2
        )                                   AS utilization_pct,
        a.is_active,
        a.last_active_at
    FROM support_agents a
    WHERE a.is_active = TRUE
    ORDER BY utilization_pct ASC;

COMMENT ON VIEW v_agent_workload IS 'Real-time agent capacity and utilization for routing decisions';

CREATE OR REPLACE VIEW v_ticket_stats AS
    SELECT
        status,
        priority,
        category,
        COUNT(*)                                        AS total,
        AVG(EXTRACT(EPOCH FROM (COALESCE(resolved_at, NOW()) - created_at)) / 3600)
                                                        AS avg_resolution_hours,
        AVG(EXTRACT(EPOCH FROM (first_response_at - created_at)) / 60)
                                                        AS avg_first_response_mins,
        AVG(satisfaction_score)                         AS avg_satisfaction,
        COUNT(*) FILTER (WHERE due_date < NOW()
            AND status NOT IN ('resolved','closed','cancelled'))
                                                        AS overdue_count
    FROM support_tickets
    GROUP BY status, priority, category;

COMMENT ON VIEW v_ticket_stats IS 'Aggregated ticket statistics for dashboards and reporting';

-- ============================================================
-- Seed Data: Default Knowledge Base Categories
-- ============================================================

INSERT INTO knowledge_base_articles (
    id, title, content, category, tags, language,
    is_published, published_at, author_id, slug, excerpt
) VALUES (
    uuid_generate_v4(),
    'Getting Started with Support',
    'Welcome to our support center. This article explains how to submit a ticket, check ticket status, and use our knowledge base effectively.',
    'general',
    ARRAY['getting-started', 'tickets', 'support'],
    'en',
    TRUE,
    NOW(),
    '00000000-0000-0000-0000-000000000001',
    'getting-started-with-support',
    'Learn how to submit tickets, track status, and find answers in our knowledge base.'
);

INSERT INTO knowledge_base_articles (
    id, title, content, category, tags, language,
    is_published, published_at, author_id, slug, excerpt
) VALUES (
    uuid_generate_v4(),
    'Billing FAQ',
    'Find answers to common billing questions including payment methods, invoice requests, refund policies, and subscription management.',
    'billing',
    ARRAY['billing', 'payment', 'invoice', 'refund', 'subscription'],
    'en',
    TRUE,
    NOW(),
    '00000000-0000-0000-0000-000000000001',
    'billing-faq',
    'Answers to the most common billing and payment questions.'
);

-- ============================================================
-- Migration metadata table (for tracking applied migrations)
-- ============================================================

CREATE TABLE IF NOT EXISTS schema_migrations (
    version         VARCHAR(255)    PRIMARY KEY,
    applied_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    checksum        VARCHAR(64),
    execution_ms    INTEGER
);

INSERT INTO schema_migrations (version, checksum) VALUES ('001_initial_schema', md5('001_initial_schema_v1'));

COMMIT;

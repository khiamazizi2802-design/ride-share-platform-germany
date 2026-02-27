-- Voice Assistant Service Database Schema
-- Migration: 001_initial_schema.sql
-- Created: 2024-01-01

BEGIN;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- ============================================================
-- ENUM TYPES
-- ============================================================

CREATE TYPE session_status AS ENUM (
    'initializing',
    'active',
    'paused',
    'completed',
    'failed',
    'terminated'
);

CREATE TYPE transcript_status AS ENUM (
    'processing',
    'completed',
    'failed',
    'partial'
);

CREATE TYPE intent_category AS ENUM (
    'navigation',
    'booking',
    'ride_control',
    'payment',
    'support',
    'settings',
    'information',
    'emergency',
    'communication',
    'unknown'
);

CREATE TYPE command_status AS ENUM (
    'received',
    'processing',
    'executed',
    'failed',
    'cancelled',
    'requires_confirmation'
);

CREATE TYPE audio_encoding AS ENUM (
    'LINEAR16',
    'FLAC',
    'MP3',
    'OGG_OPUS',
    'WEBM_OPUS',
    'AMR',
    'AMR_WB'
);

CREATE TYPE speaker_role AS ENUM (
    'driver',
    'passenger',
    'system'
);

CREATE TYPE command_source AS ENUM (
    'voice',
    'text',
    'gesture',
    'touch'
);

-- ============================================================
-- VOICE SESSIONS TABLE
-- ============================================================

CREATE TABLE voice_sessions (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                 UUID NOT NULL,
    ride_id                 UUID,
    device_id               VARCHAR(255) NOT NULL,
    session_token           VARCHAR(512) NOT NULL UNIQUE,
    status                  session_status NOT NULL DEFAULT 'initializing',
    speaker_role            speaker_role NOT NULL DEFAULT 'driver',
    language_code           VARCHAR(10) NOT NULL DEFAULT 'en-US',
    audio_encoding          audio_encoding NOT NULL DEFAULT 'LINEAR16',
    sample_rate_hertz       INTEGER NOT NULL DEFAULT 16000,
    audio_channels          SMALLINT NOT NULL DEFAULT 1,
    websocket_id            VARCHAR(255),
    client_ip               INET,
    user_agent              TEXT,
    nlu_provider            VARCHAR(50) NOT NULL DEFAULT 'dialogflow',
    stt_provider            VARCHAR(50) NOT NULL DEFAULT 'google',
    confidence_threshold    DECIMAL(4,3) NOT NULL DEFAULT 0.750,
    total_commands          INTEGER NOT NULL DEFAULT 0,
    successful_commands     INTEGER NOT NULL DEFAULT 0,
    failed_commands         INTEGER NOT NULL DEFAULT 0,
    total_duration_ms       BIGINT NOT NULL DEFAULT 0,
    total_audio_bytes       BIGINT NOT NULL DEFAULT 0,
    metadata                JSONB NOT NULL DEFAULT '{}',
    error_message           TEXT,
    started_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paused_at               TIMESTAMPTZ,
    ended_at                TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_voice_sessions_user_id ON voice_sessions(user_id);
CREATE INDEX idx_voice_sessions_ride_id ON voice_sessions(ride_id) WHERE ride_id IS NOT NULL;
CREATE INDEX idx_voice_sessions_status ON voice_sessions(status);
CREATE INDEX idx_voice_sessions_device_id ON voice_sessions(device_id);
CREATE INDEX idx_voice_sessions_started_at ON voice_sessions(started_at DESC);
CREATE INDEX idx_voice_sessions_last_activity ON voice_sessions(last_activity_at DESC);
CREATE INDEX idx_voice_sessions_user_status ON voice_sessions(user_id, status);
CREATE INDEX idx_voice_sessions_metadata ON voice_sessions USING GIN(metadata);
CREATE INDEX idx_voice_sessions_active ON voice_sessions(status, last_activity_at) WHERE status IN ('active', 'paused');

-- ============================================================
-- TRANSCRIPTS TABLE
-- ============================================================

CREATE TABLE transcripts (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id              UUID NOT NULL REFERENCES voice_sessions(id) ON DELETE CASCADE,
    user_id                 UUID NOT NULL,
    ride_id                 UUID,
    sequence_number         INTEGER NOT NULL,
    speaker_role            speaker_role NOT NULL DEFAULT 'driver',
    raw_text                TEXT NOT NULL,
    normalized_text         TEXT,
    language_code           VARCHAR(10) NOT NULL DEFAULT 'en-US',
    detected_language       VARCHAR(10),
    confidence_score        DECIMAL(5,4),
    stt_provider            VARCHAR(50) NOT NULL DEFAULT 'google',
    stt_model               VARCHAR(100),
    is_final                BOOLEAN NOT NULL DEFAULT FALSE,
    is_interim              BOOLEAN NOT NULL DEFAULT FALSE,
    audio_start_ms          BIGINT NOT NULL DEFAULT 0,
    audio_end_ms            BIGINT NOT NULL DEFAULT 0,
    audio_duration_ms       BIGINT GENERATED ALWAYS AS (audio_end_ms - audio_start_ms) STORED,
    audio_bytes             INTEGER,
    word_count              INTEGER,
    words_data              JSONB NOT NULL DEFAULT '[]',
    alternatives            JSONB NOT NULL DEFAULT '[]',
    status                  transcript_status NOT NULL DEFAULT 'processing',
    processing_time_ms      INTEGER,
    error_message           TEXT,
    metadata                JSONB NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transcripts_session_id ON transcripts(session_id);
CREATE INDEX idx_transcripts_user_id ON transcripts(user_id);
CREATE INDEX idx_transcripts_ride_id ON transcripts(ride_id) WHERE ride_id IS NOT NULL;
CREATE INDEX idx_transcripts_sequence ON transcripts(session_id, sequence_number);
CREATE INDEX idx_transcripts_created_at ON transcripts(created_at DESC);
CREATE INDEX idx_transcripts_status ON transcripts(status);
CREATE INDEX idx_transcripts_is_final ON transcripts(session_id, is_final);
CREATE INDEX idx_transcripts_raw_text_gin ON transcripts USING GIN(to_tsvector('english', raw_text));
CREATE INDEX idx_transcripts_words_data ON transcripts USING GIN(words_data);

-- ============================================================
-- INTENTS TABLE
-- ============================================================

CREATE TABLE intents (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transcript_id           UUID NOT NULL REFERENCES transcripts(id) ON DELETE CASCADE,
    session_id              UUID NOT NULL REFERENCES voice_sessions(id) ON DELETE CASCADE,
    user_id                 UUID NOT NULL,
    ride_id                 UUID,
    intent_name             VARCHAR(255) NOT NULL,
    display_name            VARCHAR(255),
    category                intent_category NOT NULL DEFAULT 'unknown',
    confidence_score        DECIMAL(5,4) NOT NULL,
    is_fallback             BOOLEAN NOT NULL DEFAULT FALSE,
    requires_confirmation   BOOLEAN NOT NULL DEFAULT FALSE,
    is_confirmed            BOOLEAN,
    nlu_provider            VARCHAR(50) NOT NULL DEFAULT 'dialogflow',
    nlu_model               VARCHAR(100),
    nlu_response_time_ms    INTEGER,
    raw_nlu_response        JSONB NOT NULL DEFAULT '{}',
    entities                JSONB NOT NULL DEFAULT '[]',
    parameters              JSONB NOT NULL DEFAULT '{}',
    output_contexts         JSONB NOT NULL DEFAULT '[]',
    input_contexts          JSONB NOT NULL DEFAULT '[]',
    sentiment_score         DECIMAL(4,3),
    sentiment_magnitude     DECIMAL(6,3),
    fulfillment_text        TEXT,
    fulfillment_messages    JSONB NOT NULL DEFAULT '[]',
    webhook_used            BOOLEAN NOT NULL DEFAULT FALSE,
    webhook_payload         JSONB,
    alternatives            JSONB NOT NULL DEFAULT '[]',
    metadata                JSONB NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_intents_transcript_id ON intents(transcript_id);
CREATE INDEX idx_intents_session_id ON intents(session_id);
CREATE INDEX idx_intents_user_id ON intents(user_id);
CREATE INDEX idx_intents_ride_id ON intents(ride_id) WHERE ride_id IS NOT NULL;
CREATE INDEX idx_intents_intent_name ON intents(intent_name);
CREATE INDEX idx_intents_category ON intents(category);
CREATE INDEX idx_intents_confidence ON intents(confidence_score DESC);
CREATE INDEX idx_intents_created_at ON intents(created_at DESC);
CREATE INDEX idx_intents_entities ON intents USING GIN(entities);
CREATE INDEX idx_intents_parameters ON intents USING GIN(parameters);
CREATE INDEX idx_intents_user_category ON intents(user_id, category, created_at DESC);

-- ============================================================
-- VOICE COMMANDS TABLE
-- ============================================================

CREATE TABLE voice_commands (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    intent_id               UUID NOT NULL REFERENCES intents(id) ON DELETE CASCADE,
    session_id              UUID NOT NULL REFERENCES voice_sessions(id) ON DELETE CASCADE,
    transcript_id           UUID NOT NULL REFERENCES transcripts(id) ON DELETE CASCADE,
    user_id                 UUID NOT NULL,
    ride_id                 UUID,
    command_type            VARCHAR(100) NOT NULL,
    command_source          command_source NOT NULL DEFAULT 'voice',
    status                  command_status NOT NULL DEFAULT 'received',
    priority                SMALLINT NOT NULL DEFAULT 5 CHECK (priority BETWEEN 1 AND 10),
    is_emergency            BOOLEAN NOT NULL DEFAULT FALSE,
    requires_ride_active    BOOLEAN NOT NULL DEFAULT FALSE,
    raw_command             TEXT NOT NULL,
    normalized_command      TEXT,
    action                  VARCHAR(100) NOT NULL,
    target_service          VARCHAR(100),
    parameters              JSONB NOT NULL DEFAULT '{}',
    context                 JSONB NOT NULL DEFAULT '{}',
    response_text           TEXT,
    response_audio_url      VARCHAR(1024),
    response_data           JSONB NOT NULL DEFAULT '{}',
    response_time_ms        INTEGER,
    execution_time_ms       INTEGER,
    retry_count             SMALLINT NOT NULL DEFAULT 0,
    max_retries             SMALLINT NOT NULL DEFAULT 3,
    error_code              VARCHAR(50),
    error_message           TEXT,
    kafka_topic             VARCHAR(255),
    kafka_partition         INTEGER,
    kafka_offset            BIGINT,
    kafka_published_at      TIMESTAMPTZ,
    confirmation_required   BOOLEAN NOT NULL DEFAULT FALSE,
    confirmed_at            TIMESTAMPTZ,
    cancelled_at            TIMESTAMPTZ,
    executed_at             TIMESTAMPTZ,
    failed_at               TIMESTAMPTZ,
    metadata                JSONB NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_voice_commands_intent_id ON voice_commands(intent_id);
CREATE INDEX idx_voice_commands_session_id ON voice_commands(session_id);
CREATE INDEX idx_voice_commands_transcript_id ON voice_commands(transcript_id);
CREATE INDEX idx_voice_commands_user_id ON voice_commands(user_id);
CREATE INDEX idx_voice_commands_ride_id ON voice_commands(ride_id) WHERE ride_id IS NOT NULL;
CREATE INDEX idx_voice_commands_status ON voice_commands(status);
CREATE INDEX idx_voice_commands_command_type ON voice_commands(command_type);
CREATE INDEX idx_voice_commands_action ON voice_commands(action);
CREATE INDEX idx_voice_commands_is_emergency ON voice_commands(is_emergency) WHERE is_emergency = TRUE;
CREATE INDEX idx_voice_commands_created_at ON voice_commands(created_at DESC);
CREATE INDEX idx_voice_commands_priority ON voice_commands(priority DESC, created_at ASC);
CREATE INDEX idx_voice_commands_pending ON voice_commands(status, priority DESC) WHERE status IN ('received', 'processing');
CREATE INDEX idx_voice_commands_parameters ON voice_commands USING GIN(parameters);
CREATE INDEX idx_voice_commands_user_type ON voice_commands(user_id, command_type, created_at DESC);

-- ============================================================
-- VOICE COMMAND TEMPLATES TABLE
-- ============================================================

CREATE TABLE voice_command_templates (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    intent_name             VARCHAR(255) NOT NULL UNIQUE,
    display_name            VARCHAR(255) NOT NULL,
    category                intent_category NOT NULL,
    description             TEXT,
    example_phrases         JSONB NOT NULL DEFAULT '[]',
    action                  VARCHAR(100) NOT NULL,
    target_service          VARCHAR(100),
    required_parameters     JSONB NOT NULL DEFAULT '[]',
    optional_parameters     JSONB NOT NULL DEFAULT '[]',
    requires_confirmation   BOOLEAN NOT NULL DEFAULT FALSE,
    is_emergency            BOOLEAN NOT NULL DEFAULT FALSE,
    requires_ride_active    BOOLEAN NOT NULL DEFAULT FALSE,
    priority                SMALLINT NOT NULL DEFAULT 5,
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    supported_languages     JSONB NOT NULL DEFAULT '["en-US"]',
    response_template       TEXT,
    metadata                JSONB NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_command_templates_intent_name ON voice_command_templates(intent_name);
CREATE INDEX idx_command_templates_category ON voice_command_templates(category);
CREATE INDEX idx_command_templates_is_active ON voice_command_templates(is_active);
CREATE INDEX idx_command_templates_action ON voice_command_templates(action);

-- ============================================================
-- VOICE SESSION EVENTS TABLE
-- ============================================================

CREATE TABLE voice_session_events (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id              UUID NOT NULL REFERENCES voice_sessions(id) ON DELETE CASCADE,
    user_id                 UUID NOT NULL,
    event_type              VARCHAR(100) NOT NULL,
    event_data              JSONB NOT NULL DEFAULT '{}',
    sequence_number         INTEGER NOT NULL,
    occurred_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_events_session_id ON voice_session_events(session_id);
CREATE INDEX idx_session_events_user_id ON voice_session_events(user_id);
CREATE INDEX idx_session_events_event_type ON voice_session_events(event_type);
CREATE INDEX idx_session_events_occurred_at ON voice_session_events(occurred_at DESC);
CREATE INDEX idx_session_events_sequence ON voice_session_events(session_id, sequence_number);

-- ============================================================
-- VOICE USER PREFERENCES TABLE
-- ============================================================

CREATE TABLE voice_user_preferences (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                 UUID NOT NULL UNIQUE,
    preferred_language      VARCHAR(10) NOT NULL DEFAULT 'en-US',
    voice_activation_word   VARCHAR(100) DEFAULT 'hey rideshare',
    is_voice_enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    auto_execute_commands   BOOLEAN NOT NULL DEFAULT FALSE,
    confirmation_threshold  DECIMAL(4,3) NOT NULL DEFAULT 0.800,
    speech_rate             DECIMAL(3,1) NOT NULL DEFAULT 1.0,
    voice_gender            VARCHAR(10) DEFAULT 'neutral',
    noise_cancellation      BOOLEAN NOT NULL DEFAULT TRUE,
    wake_word_sensitivity   DECIMAL(3,2) NOT NULL DEFAULT 0.50,
    preferred_nlu_model     VARCHAR(100),
    custom_commands         JSONB NOT NULL DEFAULT '{}',
    blocked_intents         JSONB NOT NULL DEFAULT '[]',
    notification_voice      BOOLEAN NOT NULL DEFAULT TRUE,
    metadata                JSONB NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_preferences_user_id ON voice_user_preferences(user_id);

-- ============================================================
-- VOICE ANALYTICS TABLE
-- ============================================================

CREATE TABLE voice_analytics (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                 UUID NOT NULL,
    ride_id                 UUID,
    session_id              UUID REFERENCES voice_sessions(id) ON DELETE SET NULL,
    metric_date             DATE NOT NULL DEFAULT CURRENT_DATE,
    metric_hour             SMALLINT NOT NULL DEFAULT EXTRACT(HOUR FROM NOW()),
    total_sessions          INTEGER NOT NULL DEFAULT 0,
    successful_sessions     INTEGER NOT NULL DEFAULT 0,
    failed_sessions         INTEGER NOT NULL DEFAULT 0,
    total_commands          INTEGER NOT NULL DEFAULT 0,
    successful_commands     INTEGER NOT NULL DEFAULT 0,
    failed_commands         INTEGER NOT NULL DEFAULT 0,
    total_transcripts       INTEGER NOT NULL DEFAULT 0,
    avg_confidence_score    DECIMAL(5,4),
    avg_response_time_ms    INTEGER,
    avg_session_duration_ms BIGINT,
    most_used_intents       JSONB NOT NULL DEFAULT '{}',
    intent_distribution     JSONB NOT NULL DEFAULT '{}',
    error_distribution      JSONB NOT NULL DEFAULT '{}',
    language_distribution   JSONB NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, metric_date, metric_hour)
);

CREATE INDEX idx_voice_analytics_user_id ON voice_analytics(user_id);
CREATE INDEX idx_voice_analytics_metric_date ON voice_analytics(metric_date DESC);
CREATE INDEX idx_voice_analytics_user_date ON voice_analytics(user_id, metric_date DESC);

-- ============================================================
-- FUNCTIONS AND TRIGGERS
-- ============================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_voice_sessions_updated_at
    BEFORE UPDATE ON voice_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_transcripts_updated_at
    BEFORE UPDATE ON transcripts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_intents_updated_at
    BEFORE UPDATE ON intents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_voice_commands_updated_at
    BEFORE UPDATE ON voice_commands
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_command_templates_updated_at
    BEFORE UPDATE ON voice_command_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_user_preferences_updated_at
    BEFORE UPDATE ON voice_user_preferences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_voice_analytics_updated_at
    BEFORE UPDATE ON voice_analytics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update session statistics
CREATE OR REPLACE FUNCTION update_session_on_command()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'executed' AND (OLD.status IS NULL OR OLD.status != 'executed') THEN
        UPDATE voice_sessions
        SET
            total_commands = total_commands + 1,
            successful_commands = successful_commands + 1,
            last_activity_at = NOW()
        WHERE id = NEW.session_id;
    ELSIF NEW.status = 'failed' AND (OLD.status IS NULL OR OLD.status != 'failed') THEN
        UPDATE voice_sessions
        SET
            total_commands = total_commands + 1,
            failed_commands = failed_commands + 1,
            last_activity_at = NOW()
        WHERE id = NEW.session_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_session_on_command
    AFTER INSERT OR UPDATE ON voice_commands
    FOR EACH ROW EXECUTE FUNCTION update_session_on_command();

-- Function to set word count on transcript
CREATE OR REPLACE FUNCTION set_transcript_word_count()
RETURNS TRIGGER AS $$
BEGIN
    NEW.word_count = array_length(string_to_array(trim(NEW.raw_text), ' '), 1);
    IF NEW.normalized_text IS NULL THEN
        NEW.normalized_text = lower(trim(NEW.raw_text));
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_transcript_word_count
    BEFORE INSERT OR UPDATE ON transcripts
    FOR EACH ROW EXECUTE FUNCTION set_transcript_word_count();

-- ============================================================
-- SEED DATA: VOICE COMMAND TEMPLATES
-- ============================================================

INSERT INTO voice_command_templates (intent_name, display_name, category, description, example_phrases, action, target_service, required_parameters, optional_parameters, requires_confirmation, is_emergency, requires_ride_active, priority) VALUES
('navigation.go_to_destination', 'Navigate to Destination', 'navigation', 'Navigate to a specified destination', '["take me to {location}", "navigate to {location}", "go to {location}"]', 'NAVIGATE_TO', 'navigation-service', '["location"]', '["via", "avoid"]', FALSE, FALSE, TRUE, 8),
('navigation.find_nearby', 'Find Nearby Places', 'navigation', 'Find nearby points of interest', '["find nearby {place_type}", "show me {place_type} near me"]', 'FIND_NEARBY', 'navigation-service', '["place_type"]', '["radius", "rating"]', FALSE, FALSE, FALSE, 5),
('navigation.get_eta', 'Get ETA', 'navigation', 'Get estimated time of arrival', '["how long until we arrive", "what is the ETA", "when will we get there"]', 'GET_ETA', 'navigation-service', '[]', '["destination"]', FALSE, FALSE, TRUE, 6),
('booking.request_ride', 'Request Ride', 'booking', 'Request a new ride', '["book a ride to {destination}", "get me a car to {destination}", "request a ride"]', 'REQUEST_RIDE', 'booking-service', '["destination"]', '["ride_type", "scheduled_time"]', TRUE, FALSE, FALSE, 9),
('booking.cancel_ride', 'Cancel Ride', 'booking', 'Cancel an existing ride', '["cancel my ride", "cancel the trip", "I want to cancel"]', 'CANCEL_RIDE', 'booking-service', '[]', '["reason"]', TRUE, FALSE, TRUE, 9),
('booking.schedule_ride', 'Schedule Ride', 'booking', 'Schedule a future ride', '["schedule a ride for {time}", "book a ride for {date} at {time}"]', 'SCHEDULE_RIDE', 'booking-service', '["destination", "scheduled_time"]', '["ride_type"]', TRUE, FALSE, FALSE, 7),
('ride_control.adjust_temperature', 'Adjust Temperature', 'ride_control', 'Adjust vehicle temperature', '["set temperature to {temperature}", "make it warmer", "make it cooler", "turn up the heat"]', 'ADJUST_TEMPERATURE', 'vehicle-service', '["direction"]', '["temperature", "zone"]', FALSE, FALSE, TRUE, 4),
('ride_control.play_music', 'Play Music', 'ride_control', 'Control music playback', '["play {song_or_artist}", "play some music", "skip this song", "pause the music"]', 'CONTROL_MUSIC', 'media-service', '["action"]', '["song", "artist", "playlist", "genre"]', FALSE, FALSE, TRUE, 3),
('ride_control.call_driver', 'Call Driver', 'ride_control', 'Call the driver', '["call my driver", "contact the driver", "call driver"]', 'CALL_DRIVER', 'communication-service', '[]', '[]', FALSE, FALSE, TRUE, 7),
('payment.add_tip', 'Add Tip', 'payment', 'Add a tip for the driver', '["add a {amount} tip", "tip the driver {amount}", "give a {percent} percent tip"]', 'ADD_TIP', 'payment-service', '["amount"]', '["currency"]', TRUE, FALSE, TRUE, 6),
('payment.split_fare', 'Split Fare', 'payment', 'Split the fare with others', '["split the fare", "share the cost", "split the bill"]', 'SPLIT_FARE', 'payment-service', '[]', '["split_count", "contact_list"]', TRUE, FALSE, TRUE, 5),
('support.report_issue', 'Report Issue', 'support', 'Report an issue during the ride', '["report an issue", "there is a problem", "I have a complaint"]', 'REPORT_ISSUE', 'support-service', '["issue_type"]', '["description"]', FALSE, FALSE, FALSE, 7),
('emergency.sos', 'Emergency SOS', 'emergency', 'Trigger emergency SOS', '["emergency", "SOS", "help me", "I am in danger", "call police"]', 'TRIGGER_SOS', 'safety-service', '[]', '["location"]', FALSE, TRUE, FALSE, 10),
('emergency.share_location', 'Share Location', 'emergency', 'Share current location with emergency contact', '["share my location", "send my location to {contact}", "let {contact} know where I am"]', 'SHARE_LOCATION', 'safety-service', '[]', '["contact"]', FALSE, FALSE, FALSE, 8),
('information.ride_status', 'Ride Status', 'information', 'Get current ride status', '["what is my ride status", "where is my driver", "how far is my driver"]', 'GET_RIDE_STATUS', 'booking-service', '[]', '[]', FALSE, FALSE, FALSE, 5),
('settings.change_language', 'Change Language', 'settings', 'Change the assistant language', '["change language to {language}", "speak in {language}", "switch to {language}"]', 'CHANGE_LANGUAGE', 'voice-assistant-service', '["language"]', '[]', FALSE, FALSE, FALSE, 4),
('communication.send_message', 'Send Message', 'communication', 'Send a message to a contact', '["send a message to {contact}", "text {contact} that I am {message}"]', 'SEND_MESSAGE', 'communication-service', '["contact", "message"]', '[]', TRUE, FALSE, FALSE, 6);

-- ============================================================
-- VIEWS
-- ============================================================

CREATE VIEW active_voice_sessions AS
SELECT
    vs.id,
    vs.user_id,
    vs.ride_id,
    vs.device_id,
    vs.status,
    vs.speaker_role,
    vs.language_code,
    vs.total_commands,
    vs.successful_commands,
    vs.failed_commands,
    vs.started_at,
    vs.last_activity_at,
    EXTRACT(EPOCH FROM (NOW() - vs.started_at)) AS session_duration_seconds,
    EXTRACT(EPOCH FROM (NOW() - vs.last_activity_at)) AS idle_seconds
FROM voice_sessions vs
WHERE vs.status IN ('active', 'paused')
  AND vs.last_activity_at > NOW() - INTERVAL '10 minutes';

CREATE VIEW voice_command_summary AS
SELECT
    vc.id,
    vc.session_id,
    vc.user_id,
    vc.ride_id,
    vc.command_type,
    vc.action,
    vc.status,
    vc.priority,
    vc.is_emergency,
    vc.raw_command,
    vc.response_text,
    vc.response_time_ms,
    vc.execution_time_ms,
    vc.error_message,
    t.raw_text AS transcript_text,
    t.confidence_score AS transcript_confidence,
    i.intent_name,
    i.category AS intent_category,
    i.confidence_score AS intent_confidence,
    vc.created_at,
    vc.executed_at
FROM voice_commands vc
JOIN transcripts t ON t.id = vc.transcript_id
JOIN intents i ON i.id = vc.intent_id;

-- ============================================================
-- GRANTS
-- ============================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO voice_assistant_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO voice_assistant_app;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO voice_assistant_readonly;

COMMIT;
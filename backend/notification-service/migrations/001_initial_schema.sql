-- =============================================================================
-- Migration: Notification Service
-- Version: 1.0.0
-- Beschreibung: Erstellt alle Tabellen fuer den Benachrichtigungsdienst
-- DSGVO-Hinweis: Alle personenbezogenen Daten unterliegen der DSGVO.
--               Automatische Loeschung ueber expires_at-Felder vorgesehen.
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- Erweiterungen aktivieren
-- ---------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- fuer gen_random_uuid()

-- ---------------------------------------------------------------------------
-- Hilfsfunktion: updated_at automatisch aktualisieren
-- DSGVO: Aenderungszeitpunkt wird protokolliert fuer Nachvollziehbarkeit
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

COMMENT ON FUNCTION fn_set_updated_at() IS
    'Trigger-Funktion: Setzt updated_at auf den aktuellen Zeitstempel bei jeder Aenderung einer Zeile.';

-- =============================================================================
-- Tabelle: notification_templates
-- Beschreibung: Vorlagen fuer Benachrichtigungen (kanalspezifisch)
-- DSGVO: Enthaelt keine personenbezogenen Daten; keine Loeschfrist erforderlich
-- =============================================================================
CREATE TABLE IF NOT EXISTS notification_templates (
    -- Eindeutiger Bezeichner der Vorlage
    id                  UUID            NOT NULL DEFAULT gen_random_uuid(),

    -- Eindeutiger interner Name der Vorlage (z. B. 'ride_started_push')
    name                VARCHAR(100)    NOT NULL,

    -- Kategorie der Benachrichtigung (z. B. 'ride_update', 'promotion', 'safety')
    type                VARCHAR(50)     NOT NULL,

    -- Uebertragungskanal der Benachrichtigung: 'push', 'sms', 'email'
    channel             VARCHAR(20)     NOT NULL,

    -- Titelvorlage mit Platzhaltern (z. B. 'Hallo {{name}}')
    title_template      VARCHAR(255)    NOT NULL,

    -- Textvorlage des Benachrichtigungsinhalts mit Platzhaltern
    body_template       TEXT            NOT NULL,

    -- JSON-Liste der benoetigten Variablen fuer die Vorlage
    -- Beispiel: ["name", "ride_id", "pickup_address"]
    variables           JSONB,

    -- Gibt an, ob die Vorlage aktiv und verwendbar ist
    is_active           BOOLEAN         NOT NULL DEFAULT TRUE,

    -- Erstellungszeitpunkt des Eintrags
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Letzter Aenderungszeitpunkt des Eintrags
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_notification_templates PRIMARY KEY (id),
    CONSTRAINT uq_notification_templates_name UNIQUE (name),
    CONSTRAINT chk_notification_templates_channel
        CHECK (channel IN ('push', 'sms', 'email')),
    CONSTRAINT chk_notification_templates_type
        CHECK (type IN ('ride_update', 'promotion', 'safety', 'account', 'payment', 'system'))
);

COMMENT ON TABLE notification_templates IS
    'Benachrichtigungsvorlagen fuer verschiedene Typen und Kanaele. Enthaelt keine personenbezogenen Daten.';
COMMENT ON COLUMN notification_templates.id IS
    'Primaerschluessel der Vorlage (UUID v4).';
COMMENT ON COLUMN notification_templates.name IS
    'Eindeutiger interner Name der Vorlage.';
COMMENT ON COLUMN notification_templates.type IS
    'Fachliche Kategorie der Benachrichtigung.';
COMMENT ON COLUMN notification_templates.channel IS
    'Uebertragungskanal: push, sms oder email.';
COMMENT ON COLUMN notification_templates.title_template IS
    'Titelvorlage mit Platzhaltern im Format {{variable}}.';
COMMENT ON COLUMN notification_templates.body_template IS
    'Inhaltsvorlage mit Platzhaltern im Format {{variable}}.';
COMMENT ON COLUMN notification_templates.variables IS
    'Liste der Pflichtfelder fuer die Vorlagenausfuellung als JSON-Array.';
COMMENT ON COLUMN notification_templates.is_active IS
    'Gibt an, ob diese Vorlage aktuell in Benutzung ist.';

-- Index: beschleunigt Suche nach Typ und Kanal
CREATE INDEX IF NOT EXISTS idx_notification_templates_type_channel
    ON notification_templates (type, channel);

CREATE INDEX IF NOT EXISTS idx_notification_templates_is_active
    ON notification_templates (is_active)
    WHERE is_active = TRUE;

-- Trigger: updated_at automatisch setzen
CREATE TRIGGER trg_notification_templates_updated_at
    BEFORE UPDATE ON notification_templates
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- =============================================================================
-- Tabelle: notifications
-- Beschreibung: Einzelne Benachrichtigungen pro Nutzer
-- DSGVO: Enthaelt personenbezogene Daten (user_id, Nachrichteninhalt).
--        expires_at bestimmt die maximale Aufbewahrungsfrist.
--        Nach Ablauf muessen Eintraege geloescht oder anonymisiert werden.
-- =============================================================================
CREATE TABLE IF NOT EXISTS notifications (
    -- Eindeutiger Bezeichner der Benachrichtigung
    id                  UUID            NOT NULL DEFAULT gen_random_uuid(),

    -- Fremdschluessel zum Nutzer (aus dem Nutzerverwaltungsdienst)
    -- DSGVO: Personenbezogenes Datum; darf nur so lange gespeichert werden,
    --        wie es fuer den Verarbeitungszweck notwendig ist.
    user_id             UUID            NOT NULL,

    -- Kategorie der Benachrichtigung
    type                VARCHAR(50)     NOT NULL,

    -- Uebertragungskanal: 'push', 'sms', 'email'
    channel             VARCHAR(20)     NOT NULL,

    -- Titel der Benachrichtigung (ggf. aus Vorlage generiert)
    title               VARCHAR(255)    NOT NULL,

    -- Hauptinhalt der Benachrichtigung
    -- DSGVO: Kann personenbezogene Daten enthalten; Aufbewahrung begrenzen.
    body                TEXT            NOT NULL,

    -- Zusaetzliche strukturierte Nutzdaten (z. B. Fahrt-ID, Deep-Link-URL)
    data                JSONB,

    -- Aktueller Zustand der Benachrichtigung
    -- Gueltige Werte: 'pending', 'sent', 'delivered', 'failed', 'read'
    status              VARCHAR(20)     NOT NULL DEFAULT 'pending',

    -- Versandpriorit
    -- Gueltige Werte: 'low', 'normal', 'high', 'urgent'
    priority            VARCHAR(10)     NOT NULL DEFAULT 'normal',

    -- Zeitpunkt, ab dem die Benachrichtigung versendet werden soll (optional)
    scheduled_at        TIMESTAMP WITH TIME ZONE,

    -- Zeitpunkt, zu dem die Benachrichtigung abgeschickt wurde
    sent_at             TIMESTAMP WITH TIME ZONE,

    -- Zeitpunkt der bestaetigen Zustellung
    delivered_at        TIMESTAMP WITH TIME ZONE,

    -- Zeitpunkt, zu dem der Nutzer die Benachrichtigung geoeffnet hat
    read_at             TIMESTAMP WITH TIME ZONE,

    -- DSGVO: Ablaufzeitpunkt der Benachrichtigung.
    --        Nach diesem Zeitpunkt MUSS der Eintrag geloescht oder
    --        vollstaendig anonymisiert werden (Art. 5 Abs. 1 lit. e DSGVO).
    expires_at          TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Erstellungszeitpunkt des Eintrags
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Letzter Aenderungszeitpunkt des Eintrags
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_notifications PRIMARY KEY (id),
    CONSTRAINT chk_notifications_channel
        CHECK (channel IN ('push', 'sms', 'email')),
    CONSTRAINT chk_notifications_status
        CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'read')),
    CONSTRAINT chk_notifications_priority
        CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    CONSTRAINT chk_notifications_expires_after_created
        CHECK (expires_at > created_at)
);

COMMENT ON TABLE notifications IS
    'DSGVO-relevant: Enthaelt personenbezogene Benachrichtigungsdaten. '
    'Aufbewahrungsfrist wird ueber expires_at gesteuert. '
    'Abgelaufene Eintraege sind regelmaessig zu loeschen (Art. 5 DSGVO).';
COMMENT ON COLUMN notifications.id IS
    'Primaerschluessel der Benachrichtigung (UUID v4).';
COMMENT ON COLUMN notifications.user_id IS
    'DSGVO: Referenz auf den betroffenen Nutzer. Personenbezogenes Datum.';
COMMENT ON COLUMN notifications.type IS
    'Fachliche Kategorie der Benachrichtigung (z. B. ride_update, promotion).';
COMMENT ON COLUMN notifications.channel IS
    'Uebertragungskanal: push, sms oder email.';
COMMENT ON COLUMN notifications.title IS
    'Titel der Benachrichtigung. Kann personenbezogene Daten enthalten.';
COMMENT ON COLUMN notifications.body IS
    'DSGVO: Inhalt der Nachricht. Kann personenbezogene Daten enthalten.';
COMMENT ON COLUMN notifications.data IS
    'Optionale strukturierte Zusatzdaten (JSON). Kann personenbezogene Daten enthalten.';
COMMENT ON COLUMN notifications.status IS
    'Aktueller Zustellstatus: pending, sent, delivered, failed oder read.';
COMMENT ON COLUMN notifications.priority IS
    'Versandpriorit: low, normal, high oder urgent.';
COMMENT ON COLUMN notifications.scheduled_at IS
    'Optionaler geplanter Versandzeitpunkt fuer zeitverzoegerte Benachrichtigungen.';
COMMENT ON COLUMN notifications.sent_at IS
    'Zeitstempel des tatsaechlichen Versands.';
COMMENT ON COLUMN notifications.delivered_at IS
    'Zeitstempel der bestaetigen Zustellung.';
COMMENT ON COLUMN notifications.read_at IS
    'Zeitstempel des Lesens durch den Nutzer.';
COMMENT ON COLUMN notifications.expires_at IS
    'DSGVO: Pflichtfeld. Bestimmt die maximale Aufbewahrungsdauer gemaess '
    'Datensparsamkeitsprinzip (Art. 5 Abs. 1 lit. e DSGVO). '
    'Abgelaufene Datensaetze muessen durch einen Batch-Job entfernt werden.';

-- Index: Hauptzugriffsweg fuer nutzerbezogene Abfragen (z. B. Posteingang)
CREATE INDEX IF NOT EXISTS idx_notifications_user_id
    ON notifications (user_id);

-- Index: Filtert nach Status fuer Verarbeitungs-Queues
CREATE INDEX IF NOT EXISTS idx_notifications_status
    ON notifications (status);

-- Index: Unterstuetzt chronologische Sortierung und Zeitraumfilter
CREATE INDEX IF NOT EXISTS idx_notifications_created_at
    ON notifications (created_at DESC);

-- Index: DSGVO-Loeschlaeufe; identifiziert abgelaufene Eintraege effizient
CREATE INDEX IF NOT EXISTS idx_notifications_expires_at
    ON notifications (expires_at);

-- Index: Kombinierter Index fuer typische Dashboard-Abfrage (Nutzer + Status + Zeit)
CREATE INDEX IF NOT EXISTS idx_notifications_user_status_created
    ON notifications (user_id, status, created_at DESC);

-- Index: Geplante Benachrichtigungen zum Versand
CREATE INDEX IF NOT EXISTS idx_notifications_scheduled_at
    ON notifications (scheduled_at)
    WHERE scheduled_at IS NOT NULL AND status = 'pending';

-- Trigger: updated_at automatisch setzen
CREATE TRIGGER trg_notifications_updated_at
    BEFORE UPDATE ON notifications
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- =============================================================================
-- Tabelle: user_notification_preferences
-- Beschreibung: Nutzereinstellungen und DSGVO-Einwilligungen fuer Benachrichtigungen
-- DSGVO: Enthaelt explizite Einwilligungserklaerungen (Art. 7 DSGVO).
--        consent_given_at ist Pflicht fuer Nachweispflicht.
--        marketing_opt_in darf nur mit Einwilligung TRUE sein.
-- =============================================================================
CREATE TABLE IF NOT EXISTS user_notification_preferences (
    -- Primaerschluessel = Nutzer-ID (1:1 Beziehung)
    -- DSGVO: Personenbezogenes Datum
    user_id                 UUID        NOT NULL,

    -- Ob E-Mail-Benachrichtigungen zugestellt werden sollen
    email_enabled           BOOLEAN     NOT NULL DEFAULT TRUE,

    -- Ob SMS-Benachrichtigungen zugestellt werden sollen
    sms_enabled             BOOLEAN     NOT NULL DEFAULT TRUE,

    -- Ob Push-Benachrichtigungen zugestellt werden sollen
    push_enabled            BOOLEAN     NOT NULL DEFAULT TRUE,

    -- Beginn der Ruhezeit (keine Benachrichtigungen in dieser Periode)
    quiet_hours_start       TIME,

    -- Ende der Ruhezeit
    quiet_hours_end         TIME,

    -- DSGVO Art. 6 Abs. 1 lit. a: Einwilligung fuer Marketingkommunikation.
    --        Standard FALSE = kein Opt-in. Darf nur auf TRUE gesetzt werden,
    --        wenn eine explizite, dokumentierte Einwilligung vorliegt.
    marketing_opt_in        BOOLEAN     NOT NULL DEFAULT FALSE,

    -- DSGVO: Zeitstempel der erteilten Einwilligung (Nachweispflicht Art. 7 DSGVO).
    --        Muss gesetzt sein, wenn marketing_opt_in = TRUE.
    consent_given_at        TIMESTAMP WITH TIME ZONE,

    -- Erstellungszeitpunkt des Eintrags
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Letzter Aenderungszeitpunkt des Eintrags
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_user_notification_preferences PRIMARY KEY (user_id),
    -- DSGVO-Integritaetsregel: Einwilligung erfordert Zeitstempel
    CONSTRAINT chk_marketing_consent_timestamp
        CHECK (
            (marketing_opt_in = FALSE)
            OR (marketing_opt_in = TRUE AND consent_given_at IS NOT NULL)
        ),
    -- Ruhezeit: Entweder beide gesetzt oder keine
    CONSTRAINT chk_quiet_hours_both_or_none
        CHECK (
            (quiet_hours_start IS NULL AND quiet_hours_end IS NULL)
            OR (quiet_hours_start IS NOT NULL AND quiet_hours_end IS NOT NULL)
        )
);

COMMENT ON TABLE user_notification_preferences IS
    'DSGVO-relevant: Speichert Nutzereinstellungen und Einwilligungen fuer Benachrichtigungen. '
    'Einwilligungen gemaess Art. 6 Abs. 1 lit. a und Art. 7 DSGVO werden hier dokumentiert. '
    'Jede Aenderung von marketing_opt_in muss mit Zeitstempel versehen werden.';
COMMENT ON COLUMN user_notification_preferences.user_id IS
    'DSGVO: Primaerschluessel und Referenz auf den Nutzer. Personenbezogenes Datum.';
COMMENT ON COLUMN user_notification_preferences.email_enabled IS
    'Gibt an, ob der Nutzer E-Mail-Benachrichtigungen empfangen moechte.';
COMMENT ON COLUMN user_notification_preferences.sms_enabled IS
    'Gibt an, ob der Nutzer SMS-Benachrichtigungen empfangen moechte.';
COMMENT ON COLUMN user_notification_preferences.push_enabled IS
    'Gibt an, ob der Nutzer Push-Benachrichtigungen empfangen moechte.';
COMMENT ON COLUMN user_notification_preferences.quiet_hours_start IS
    'Beginn des Zeitfensters, in dem keine Benachrichtigungen gesendet werden sollen.';
COMMENT ON COLUMN user_notification_preferences.quiet_hours_end IS
    'Ende des Zeitfensters, in dem keine Benachrichtigungen gesendet werden sollen.';
COMMENT ON COLUMN user_notification_preferences.marketing_opt_in IS
    'DSGVO Art. 6 Abs. 1 lit. a: Explizite Einwilligung fuer Werbemitteilungen. '
    'Standard: FALSE. Darf nur mit dokumentierter Einwilligung auf TRUE gesetzt werden.';
COMMENT ON COLUMN user_notification_preferences.consent_given_at IS
    'DSGVO Art. 7 Abs. 1: Zeitstempel der Einwilligungserteilung fuer Marketingkommunikation. '
    'Pflichtfeld wenn marketing_opt_in = TRUE. Dient als Nachweis gegenueber Aufsichtsbehoerden.';

-- Index: DSGVO-Auskunftsanfragen und Loeschanfragen nach Nutzer
CREATE INDEX IF NOT EXISTS idx_user_notification_preferences_marketing_opt_in
    ON user_notification_preferences (marketing_opt_in)
    WHERE marketing_opt_in = TRUE;

-- Trigger: updated_at automatisch setzen
CREATE TRIGGER trg_user_notification_preferences_updated_at
    BEFORE UPDATE ON user_notification_preferences
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- =============================================================================
-- Tabelle: delivery_attempts
-- Beschreibung: Protokolliert jeden Zustellversuch einer Benachrichtigung
-- DSGVO: Protokolldaten koennen personenbezogene Daten enthalten (via notification_id).
--        expires_at sichert zeitlich begrenzte Aufbewahrung gemaess Art. 5 DSGVO.
-- =============================================================================
CREATE TABLE IF NOT EXISTS delivery_attempts (
    -- Eindeutiger Bezeichner des Zustellversuchs
    id                      UUID            NOT NULL DEFAULT gen_random_uuid(),

    -- Referenz auf die zugehoerige Benachrichtigung
    -- Kaskadierendes Loeschen: Wird die Benachrichtigung geloescht,
    -- werden auch alle Zustellversuche entfernt (DSGVO-konform)
    notification_id         UUID            NOT NULL,

    -- Name des verwendeten Drittanbieters (z. B. 'fcm', 'twilio', 'sendgrid')
    provider                VARCHAR(50)     NOT NULL,

    -- Nachrichten-ID des Drittanbieters fuer Rueckverfolgung
    provider_message_id     VARCHAR(255),

    -- Ergebnisstatus des Zustellversuchs
    status                  VARCHAR(20)     NOT NULL,

    -- Fehlercode des Anbieters bei Misserfolg
    error_code              VARCHAR(50),

    -- Detaillierte Fehlerbeschreibung des Anbieters
    -- DSGVO: Keine personenbezogenen Daten in Fehlermeldungen speichern!
    error_message           TEXT,

    -- Zeitpunkt des Zustellversuchs
    attempted_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- DSGVO: Ablaufzeitpunkt fuer diesen Protokolleintrag.
    --        Spiegelt idealerweise den expires_at-Wert der uebergeordneten
    --        Benachrichtigung wider. Pflichtfeld fuer Datensparsamkeit.
    expires_at              TIMESTAMP WITH TIME ZONE NOT NULL,

    CONSTRAINT pk_delivery_attempts PRIMARY KEY (id),
    CONSTRAINT fk_delivery_attempts_notification
        FOREIGN KEY (notification_id)
        REFERENCES notifications (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    CONSTRAINT chk_delivery_attempts_status
        CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'retrying'))
);

COMMENT ON TABLE delivery_attempts IS
    'DSGVO-relevant: Protokolliert Zustellversuche. Verknuepft mit Benachrichtigungen '
    '(personenbezogene Daten). Kaskadierendes Loeschen bei Entfernung der Benachrichtigung. '
    'expires_at begrenzt Aufbewahrungsdauer gemaess Datensparsamkeitsprinzip (Art. 5 DSGVO).';
COMMENT ON COLUMN delivery_attempts.id IS
    'Primaerschluessel des Zustellversuchs (UUID v4).';
COMMENT ON COLUMN delivery_attempts.notification_id IS
    'Fremdschluessel zur Benachrichtigung. ON DELETE CASCADE stellt DSGVO-konforme Loeschung sicher.';
COMMENT ON COLUMN delivery_attempts.provider IS
    'Name des Versanddienstleisters (z. B. fcm, twilio, sendgrid).';
COMMENT ON COLUMN delivery_attempts.provider_message_id IS
    'Eindeutige Nachrichten-ID des Drittanbieters fuer Statusabgleich und Debugging.';
COMMENT ON COLUMN delivery_attempts.status IS
    'Status des Versuchsergebnisses: pending, sent, delivered, failed oder retrying.';
COMMENT ON COLUMN delivery_attempts.error_code IS
    'Fehlercode des Anbieters. Darf keine personenbezogenen Daten enthalten.';
COMMENT ON COLUMN delivery_attempts.error_message IS
    'DSGVO-Hinweis: Fehlertexte muessen frei von personenbezogenen Daten sein. '
    'Nur technische Systemmeldungen speichern.';
COMMENT ON COLUMN delivery_attempts.attempted_at IS
    'Zeitstempel des Versuchszeitpunkts.';
COMMENT ON COLUMN delivery_attempts.expires_at IS
    'DSGVO: Pflichtfeld. Ablaufzeitpunkt des Protokolleintrags. '
    'Sollte mit expires_at der uebergeordneten Benachrichtigung uebereinstimmen.';

-- Index: Alle Versuche zu einer bestimmten Benachrichtigung abrufen
CREATE INDEX IF NOT EXISTS idx_delivery_attempts_notification_id
    ON delivery_attempts (notification_id);

-- Index: Fehleranalyse und Wiederholungslogik
CREATE INDEX IF NOT EXISTS idx_delivery_attempts_status
    ON delivery_attempts (status);

-- Index: Protokolleintraege nach Zeitraum durchsuchen
CREATE INDEX IF NOT EXISTS idx_delivery_attempts_attempted_at
    ON delivery_attempts (attempted_at DESC);

-- Index: DSGVO-Loeschlauf fuer abgelaufene Protokolleintraege
CREATE INDEX IF NOT EXISTS idx_delivery_attempts_expires_at
    ON delivery_attempts (expires_at);

-- Index: Anbieter-Nachrichten-ID fuer Status-Callbacks
CREATE INDEX IF NOT EXISTS idx_delivery_attempts_provider_message_id
    ON delivery_attempts (provider_message_id)
    WHERE provider_message_id IS NOT NULL;

-- =============================================================================
-- Tabelle: notification_webhooks
-- Beschreibung: Konfigurierte Webhook-Endpunkte externer Anbieter
-- DSGVO: Das secret-Feld muss verschluesselt gespeichert werden (ausserhalb DB).
--        Diese Tabelle enthaelt keine direkten personenbezogenen Nutzerdaten.
-- =============================================================================
CREATE TABLE IF NOT EXISTS notification_webhooks (
    -- Eindeutiger Bezeichner des Webhook-Eintrags
    id          UUID            NOT NULL DEFAULT gen_random_uuid(),

    -- Name des Anbieters, von dem Webhooks empfangen werden
    provider    VARCHAR(50)     NOT NULL,

    -- Ziel-URL, an die der Webhook-Request gesendet wird
    url         VARCHAR(500)    NOT NULL,

    -- HMAC-Secret fuer die Signaturpruefung eingehender Webhook-Anfragen
    -- DSGVO/Sicherheit: Dieses Feld ist sicherheitskritisch. Es wird empfohlen,
    -- den Wert verschluesselt zu speichern (z. B. via pgcrypto oder Vault).
    -- Im Klartext nur fuer Entwicklungsumgebungen akzeptabel.
    secret      VARCHAR(255),

    -- JSON-Array der abonnierten Ereignistypen
    -- Beispiel: ["delivered", "failed", "bounced"]
    events      JSONB,

    -- Gibt an, ob dieser Webhook-Eintrag aktiv ist
    is_active   BOOLEAN         NOT NULL DEFAULT TRUE,

    -- Erstellungszeitpunkt des Eintrags
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_notification_webhooks PRIMARY KEY (id),
    CONSTRAINT chk_notification_webhooks_url
        CHECK (url ~* '^https?://.+')
);

COMMENT ON TABLE notification_webhooks IS
    'Konfigurierte Webhook-Endpunkte fuer Statusmeldungen externer Anbieter. '
    'Enthaelt keine personenbezogenen Nutzerdaten. '
    'SICHERHEIT: Das secret-Feld sollte in Produktionsumgebungen verschluesselt abgelegt werden.';
COMMENT ON COLUMN notification_webhooks.id IS
    'Primaerschluessel des Webhook-Eintrags (UUID v4).';
COMMENT ON COLUMN notification_webhooks.provider IS
    'Name des Anbieters, der diesen Webhook ausloest (z. B. fcm, twilio, sendgrid).';
COMMENT ON COLUMN notification_webhooks.url IS
    'URL des Empfaengers. Muss ein gueltiges HTTP/HTTPS-Format haben.';
COMMENT ON COLUMN notification_webhooks.secret IS
    'SICHERHEITSHINWEIS: HMAC-Signaturgeheimnis. In Produktionsumgebungen verschluesselt '
    'speichern oder aus einem Secret-Management-System (z. B. HashiCorp Vault) laden.';
COMMENT ON COLUMN notification_webhooks.events IS
    'JSON-Array der Ereignistypen, auf die dieser Webhook reagiert.';
COMMENT ON COLUMN notification_webhooks.is_active IS
    'Gibt an, ob dieser Webhook-Endpunkt aktuell aktiv ist.';
COMMENT ON COLUMN notification_webhooks.created_at IS
    'Erstellungszeitpunkt des Eintrags.';

-- Index: Schneller Zugriff auf aktive Webhooks eines Anbieters
CREATE INDEX IF NOT EXISTS idx_notification_webhooks_provider_active
    ON notification_webhooks (provider, is_active)
    WHERE is_active = TRUE;

-- =============================================================================
-- DSGVO-Hilfsfunktion: Abgelaufene Benachrichtigungen loeschen
-- Empfehlung: Als regelmaessigen Cron-Job oder pg_cron-Task einplanen.
-- Beispiel: SELECT fn_purge_expired_notifications();
-- =============================================================================
CREATE OR REPLACE FUNCTION fn_purge_expired_notifications()
RETURNS TABLE(
    deleted_notifications   BIGINT,
    deleted_delivery_attempts BIGINT
)
LANGUAGE plpgsql
AS $$
DECLARE
    v_deleted_notifications     BIGINT := 0;
    v_deleted_delivery_attempts BIGINT := 0;
BEGIN
    -- Schritt 1: Abgelaufene Zustellversuche loeschen (ggf. bereits durch CASCADE geloescht)
    DELETE FROM delivery_attempts
    WHERE expires_at < NOW();
    GET DIAGNOSTICS v_deleted_delivery_attempts = ROW_COUNT;

    -- Schritt 2: Abgelaufene Benachrichtigungen loeschen
    -- Zugehoerige delivery_attempts werden durch ON DELETE CASCADE entfernt
    DELETE FROM notifications
    WHERE expires_at < NOW();
    GET DIAGNOSTICS v_deleted_notifications = ROW_COUNT;

    RETURN QUERY SELECT v_deleted_notifications, v_deleted_delivery_attempts;
END;
$$;

COMMENT ON FUNCTION fn_purge_expired_notifications() IS
    'DSGVO Art. 5 Abs. 1 lit. e: Loescht alle Benachrichtigungen und Zustellversuche, '
    'deren Aufbewahrungsfrist (expires_at) abgelaufen ist. '
    'Sollte regelmaessig als Hintergrundjob ausgefuehrt werden (z. B. taeglich via pg_cron). '
    'Gibt die Anzahl geloeschter Datensaetze zurueck.';

COMMIT;

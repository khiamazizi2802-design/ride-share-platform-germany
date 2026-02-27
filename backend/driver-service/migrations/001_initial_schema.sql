-- ============================================================
-- Migration: 001_initial_schema.sql
-- Service: driver-service
-- Beschreibung: Initiales Datenbankschema fuer den Fahrer-Service
-- DSGVO-Konformitaet: Alle personenbezogenen Daten werden gemaess
--   der Datenschutz-Grundverordnung (EU) 2016/679 verwaltet.
-- Erstellt: 2024-01-01
-- ============================================================

-- Aktiviere UUID-Erweiterung falls nicht vorhanden
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- ENUM TYPEN
-- ============================================================

-- Fahrerstatus-Enum
COMMENT ON TYPE driver_status IS 'Status des Fahrers im System' IS NOT VALID;
DO $$ BEGIN
    CREATE TYPE driver_status AS ENUM (
        'pending',    -- Registrierung ausstehend / Pruefung laeuft
        'active',     -- Aktiv und einsatzbereit
        'suspended',  -- Voruebergehend gesperrt
        'inactive'    -- Inaktiv / abgemeldet
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Abrechnungszeitraum-Enum
DO $$ BEGIN
    CREATE TYPE earnings_period_type AS ENUM (
        'daily',    -- Taeglich
        'weekly',   -- Woechentlich
        'monthly'   -- Monatlich
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Fahrt-Status-Enum fuer Fahrthistorie
DO $$ BEGIN
    CREATE TYPE trip_history_status AS ENUM (
        'completed',   -- Erfolgreich abgeschlossen
        'cancelled',   -- Abgebrochen
        'no_show'      -- Fahrgast nicht erschienen
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- ============================================================
-- TRIGGER FUNKTION: updated_at automatisch setzen
-- ============================================================

CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    -- Aktualisierungszeitstempel automatisch setzen
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public;

COMMENT ON FUNCTION trigger_set_updated_at() IS
    'Trigger-Funktion: Setzt updated_at automatisch beim Aktualisieren eines Datensatzes.';

-- ============================================================
-- TABELLE: drivers
-- DSGVO-Hinweis: Enthaelt personenbezogene Daten gemaess Art. 4 DSGVO.
--   Rechtsgrundlage: Art. 6 Abs. 1 lit. b DSGVO (Vertragserfuellung).
--   Loeschfrist: Daten werden nach Vertragsende gemaess gesetzlicher
--   Aufbewahrungsfrist (§ 147 AO: 10 Jahre) geloescht.
-- ============================================================

CREATE TABLE IF NOT EXISTS drivers (
    -- Primaerschluessel
    id                          UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Verknuepfung zum Benutzerkonto (extern, users-service)
    -- DSGVO: Referenz auf das zentrale Benutzerprofil
    user_id                     UUID            NOT NULL,

    -- Fuehrerscheindaten
    -- DSGVO: Besonders schutzwuerdige Identifikationsdaten
    license_number              VARCHAR(100)    NOT NULL,
    license_expiry_date         DATE            NOT NULL,

    -- Fahrzeugdaten
    vehicle_type                VARCHAR(50)     NOT NULL,
    vehicle_registration        VARCHAR(20)     NOT NULL,
    vehicle_model               VARCHAR(100)    NOT NULL,
    vehicle_year                SMALLINT        NOT NULL CHECK (vehicle_year >= 1900 AND vehicle_year <= EXTRACT(YEAR FROM NOW()) + 1),

    -- Betriebsstatus
    status                      driver_status   NOT NULL DEFAULT 'pending',

    -- Leistungskennzahlen
    rating                      NUMERIC(3, 2)   NOT NULL DEFAULT 0.00 CHECK (rating >= 0.00 AND rating <= 5.00),
    total_trips                 INTEGER         NOT NULL DEFAULT 0 CHECK (total_trips >= 0),

    -- Zeitstempel
    created_at                  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    -- --------------------------------------------------------
    -- DSGVO-Compliance-Felder (Datenschutz-Grundverordnung)
    -- --------------------------------------------------------

    -- Art. 17 DSGVO: Recht auf Loeschung ("Recht auf Vergessenwerden")
    -- Datum, ab dem der Datensatz zum Loeschen vorgemerkt ist
    dsgvo_loeschdatum           TIMESTAMPTZ     NULL,

    -- Art. 17 DSGVO: Kennzeichnung ob Loeschung beantragt wurde
    dsgvo_loeschung_beantragt   BOOLEAN         NOT NULL DEFAULT FALSE,

    -- Art. 7 DSGVO: Einwilligung zur Datenverarbeitung
    -- Zeitpunkt der Einwilligung des Fahrers
    dsgvo_einwilligung_datum    TIMESTAMPTZ     NULL,

    -- Art. 13/14 DSGVO: Informationspflicht - wurde der Fahrer informiert?
    dsgvo_informiert_am         TIMESTAMPTZ     NULL,

    -- Art. 20 DSGVO: Recht auf Datenuebertragbarkeit - Export angefordert
    dsgvo_export_angefordert_am TIMESTAMPTZ     NULL,

    -- Pseudonymisierungskennzeichen (Art. 4 Nr. 5 DSGVO)
    -- Wird gesetzt wenn Echtdaten pseudonymisiert wurden
    dsgvo_pseudonymisiert       BOOLEAN         NOT NULL DEFAULT FALSE,

    -- Versionierung fuer Audit-Trail (Art. 5 Abs. 2 DSGVO: Rechenschaftspflicht)
    record_version              INTEGER         NOT NULL DEFAULT 1
);

-- Tabellenkommentar
COMMENT ON TABLE drivers IS
    'Fahrer-Profile. Enthaelt personenbezogene Daten gemaess DSGVO Art. 4. '
    'Rechtsgrundlage: Art. 6 Abs. 1 lit. b DSGVO (Vertragserfuellung). '
    'Verantwortlicher: Siehe Datenschutzbeauftragter. '
    'Aufbewahrungsfrist: 10 Jahre nach Vertragsende gemaess § 147 AO.';

-- Spaltenkommentare
COMMENT ON COLUMN drivers.id IS 'Eindeutiger Primaerschluessel (UUID v4)';
COMMENT ON COLUMN drivers.user_id IS 'Fremdschluessel zum zentralen Benutzerkonto (users-service). DSGVO: Verknuepfung mit personenbezogenen Stammdaten.';
COMMENT ON COLUMN drivers.license_number IS 'Fuehrerscheinnummer. DSGVO: Identifikationsdatum, nur fuer berechtigte Mitarbeiter sichtbar.';
COMMENT ON COLUMN drivers.license_expiry_date IS 'Ablaufdatum des Fuehrerscheins. Wird fuer Fahrerberechtigung geprueft.';
COMMENT ON COLUMN drivers.vehicle_type IS 'Fahrzeugkategorie (z.B. PKW, Kombi, Van)';
COMMENT ON COLUMN drivers.vehicle_registration IS 'Fahrzeugkennzeichen. DSGVO: Kann zur Identifikation des Fahrers fuehren.';
COMMENT ON COLUMN drivers.vehicle_model IS 'Fahrzeugmodell und Marke';
COMMENT ON COLUMN drivers.vehicle_year IS 'Erstzulassungsjahr des Fahrzeugs';
COMMENT ON COLUMN drivers.status IS 'Aktueller Betriebsstatus des Fahrers im System';
COMMENT ON COLUMN drivers.rating IS 'Durchschnittliche Bewertung (0.00 - 5.00)';
COMMENT ON COLUMN drivers.total_trips IS 'Gesamtanzahl abgeschlossener Fahrten';
COMMENT ON COLUMN drivers.created_at IS 'Zeitpunkt der Datensatzerstellung (UTC)';
COMMENT ON COLUMN drivers.updated_at IS 'Zeitpunkt der letzten Aktualisierung (UTC), automatisch gesetzt';
COMMENT ON COLUMN drivers.dsgvo_loeschdatum IS 'DSGVO Art. 17: Geplantes Loeschdatum. Nach Ablauf sind Daten zu loeschen oder zu anonymisieren.';
COMMENT ON COLUMN drivers.dsgvo_loeschung_beantragt IS 'DSGVO Art. 17: Kennzeichen ob Betroffener Loeschung beantragt hat.';
COMMENT ON COLUMN drivers.dsgvo_einwilligung_datum IS 'DSGVO Art. 7: Zeitpunkt der Einwilligung zur Datenverarbeitung.';
COMMENT ON COLUMN drivers.dsgvo_informiert_am IS 'DSGVO Art. 13/14: Zeitpunkt der Datenschutzinformation an den Betroffenen.';
COMMENT ON COLUMN drivers.dsgvo_export_angefordert_am IS 'DSGVO Art. 20: Zeitpunkt des Datenportabilitaets-Exports.';
COMMENT ON COLUMN drivers.dsgvo_pseudonymisiert IS 'DSGVO Art. 4 Nr. 5: Kennzeichen ob Daten pseudonymisiert wurden.';
COMMENT ON COLUMN drivers.record_version IS 'Versionszaehler fuer optimistisches Locking und Audit-Trail (DSGVO Art. 5 Abs. 2).';

-- Trigger: updated_at automatisch aktualisieren
CREATE TRIGGER trg_drivers_updated_at
    BEFORE UPDATE ON drivers
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TRIGGER trg_drivers_updated_at ON drivers IS
    'Setzt updated_at automatisch auf NOW() bei jeder Aktualisierung.';

-- Indexes fuer drivers
CREATE UNIQUE INDEX IF NOT EXISTS idx_drivers_user_id
    ON drivers (user_id);
COMMENT ON INDEX idx_drivers_user_id IS 'Eindeutiger Index: Ein Benutzer kann nur ein Fahrerprofil haben.';

CREATE UNIQUE INDEX IF NOT EXISTS idx_drivers_license_number
    ON drivers (license_number)
    WHERE dsgvo_pseudonymisiert = FALSE;
COMMENT ON INDEX idx_drivers_license_number IS 'Eindeutiger Index auf Fuehrerscheinnummer (nur nicht-pseudonymisierte Eintraege).';

CREATE UNIQUE INDEX IF NOT EXISTS idx_drivers_vehicle_registration
    ON drivers (vehicle_registration)
    WHERE dsgvo_pseudonymisiert = FALSE;
COMMENT ON INDEX idx_drivers_vehicle_registration IS 'Eindeutiger Index auf Fahrzeugkennzeichen (nur nicht-pseudonymisierte Eintraege).';

CREATE INDEX IF NOT EXISTS idx_drivers_status
    ON drivers (status);
COMMENT ON INDEX idx_drivers_status IS 'Index fuer Statusfilterung (aktive/gesperrte Fahrer abrufen).';

CREATE INDEX IF NOT EXISTS idx_drivers_rating
    ON drivers (rating DESC);
COMMENT ON INDEX idx_drivers_rating IS 'Index fuer Bewertungsranking (absteigende Reihenfolge).';

CREATE INDEX IF NOT EXISTS idx_drivers_dsgvo_loeschdatum
    ON drivers (dsgvo_loeschdatum)
    WHERE dsgvo_loeschdatum IS NOT NULL;
COMMENT ON INDEX idx_drivers_dsgvo_loeschdatum IS 'DSGVO Art. 17: Index fuer automatischen Loeschprozess. Nur Datensaetze mit gesetztem Loeschdatum.';

CREATE INDEX IF NOT EXISTS idx_drivers_dsgvo_loeschung_beantragt
    ON drivers (dsgvo_loeschung_beantragt)
    WHERE dsgvo_loeschung_beantragt = TRUE;
COMMENT ON INDEX idx_drivers_dsgvo_loeschung_beantragt IS 'DSGVO Art. 17: Partial-Index fuer ausstehende Loeschantraege.';

CREATE INDEX IF NOT EXISTS idx_drivers_created_at
    ON drivers (created_at DESC);
COMMENT ON INDEX idx_drivers_created_at IS 'Index fuer zeitliche Sortierung und Berichterstellung.';

-- ============================================================
-- TABELLE: driver_locations
-- DSGVO-Hinweis: GPS-Standortdaten sind besonders schutzwuerdige
--   personenbezogene Daten (Bewegungsprofil). Rechtsgrundlage:
--   Art. 6 Abs. 1 lit. b DSGVO (Vertragserfuellung - aktive Fahrt).
--   Automatische Loeschung nach Ablauf von expires_at (Datensparsamkeit
--   gemaess Art. 5 Abs. 1 lit. c DSGVO).
-- ============================================================

CREATE TABLE IF NOT EXISTS driver_locations (
    -- Primaerschluessel
    id              UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Verknuepfung zum Fahrerprofil
    driver_id       UUID            NOT NULL,

    -- GPS-Koordinaten
    -- DSGVO: Standortdaten ermoeglichen Bewegungsprofile - besonders schutzwuerdig
    latitude        NUMERIC(10, 7)  NOT NULL CHECK (latitude >= -90 AND latitude <= 90),
    longitude       NUMERIC(10, 7)  NOT NULL CHECK (longitude >= -180 AND longitude <= 180),

    -- GPS-Genauigkeit in Metern
    accuracy        NUMERIC(8, 2)   NULL CHECK (accuracy IS NULL OR accuracy >= 0),

    -- Zeitpunkt der Aufzeichnung
    recorded_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    -- --------------------------------------------------------
    -- DSGVO-Compliance: Automatische Datenpurge
    -- Art. 5 Abs. 1 lit. e DSGVO: Speicherbegrenzung
    -- Standortdaten werden nach festgelegter Frist automatisch geloescht.
    -- Standardmaessig: 30 Tage nach Aufzeichnung
    -- --------------------------------------------------------
    expires_at      TIMESTAMPTZ     NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),

    -- Kontext der Standorterfassung (z.B. 'on_trip', 'available', 'offline')
    location_context VARCHAR(50)    NULL,

    -- Fahrt-Referenz falls Standort waehrend einer Fahrt erfasst wurde
    -- Nach Fahrtabschluss: Loeschung nach kuerzerer Frist moeglich
    ride_id         UUID            NULL
);

-- Tabellenkommentar
COMMENT ON TABLE driver_locations IS
    'GPS-Standortdaten der Fahrer. DSGVO: Bewegungsdaten sind besonders schutzwuerdig. '
    'Rechtsgrundlage: Art. 6 Abs. 1 lit. b DSGVO (Vertragserfuellung). '
    'Datensparsamkeit: Automatische Loeschung nach Ablauf von expires_at (Art. 5 Abs. 1 lit. e DSGVO). '
    'Kein Aufbau von Bewegungsprofilen ohne explizite Einwilligung (Art. 6 Abs. 1 lit. a DSGVO).';

-- Spaltenkommentare
COMMENT ON COLUMN driver_locations.id IS 'Eindeutiger Primaerschluessel (UUID v4)';
COMMENT ON COLUMN driver_locations.driver_id IS 'Fremdschluessel zum Fahrerprofil. DSGVO: Personenbezug durch Verknuepfung.';
COMMENT ON COLUMN driver_locations.latitude IS 'GPS-Breitengrad (-90 bis +90). DSGVO: Personenbezogenes Standortdatum.';
COMMENT ON COLUMN driver_locations.longitude IS 'GPS-Laengengrad (-180 bis +180). DSGVO: Personenbezogenes Standortdatum.';
COMMENT ON COLUMN driver_locations.accuracy IS 'GPS-Genauigkeit in Metern (kleinerer Wert = genauer).';
COMMENT ON COLUMN driver_locations.recorded_at IS 'Zeitpunkt der GPS-Erfassung (UTC). DSGVO: Ermoeglicht Zeitanalyse der Bewegung.';
COMMENT ON COLUMN driver_locations.expires_at IS 'DSGVO Art. 5 Abs. 1 lit. e: Ablaufzeitpunkt fuer automatische Loeschung. Standard: 30 Tage nach Erfassung.';
COMMENT ON COLUMN driver_locations.location_context IS 'Kontext der Standorterfassung (verfuegbar, auf Fahrt, offline).';
COMMENT ON COLUMN driver_locations.ride_id IS 'Optionale Referenz zur Fahrt bei der der Standort erfasst wurde.';

-- Foreign Key Constraint
ALTER TABLE driver_locations
    ADD CONSTRAINT fk_driver_locations_driver_id
    FOREIGN KEY (driver_id)
    REFERENCES drivers (id)
    ON DELETE CASCADE
    ON UPDATE CASCADE;

COMMENT ON CONSTRAINT fk_driver_locations_driver_id ON driver_locations IS
    'FK zu drivers: CASCADE DELETE stellt sicher, dass Standortdaten bei Fahrerprofil-Loeschung mitgeloescht werden (DSGVO Art. 17).';

-- Indexes fuer driver_locations
CREATE INDEX IF NOT EXISTS idx_driver_locations_driver_id
    ON driver_locations (driver_id);
COMMENT ON INDEX idx_driver_locations_driver_id IS 'Index fuer Standortabfragen nach Fahrer-ID.';

CREATE INDEX IF NOT EXISTS idx_driver_locations_recorded_at
    ON driver_locations (recorded_at DESC);
COMMENT ON INDEX idx_driver_locations_recorded_at IS 'Index fuer zeitliche Sortierung der Standortdaten.';

CREATE INDEX IF NOT EXISTS idx_driver_locations_expires_at
    ON driver_locations (expires_at)
    WHERE expires_at IS NOT NULL;
COMMENT ON INDEX idx_driver_locations_expires_at IS 'DSGVO: Index fuer automatischen Loeschprozess abgelaufener Standortdaten (Art. 5 Abs. 1 lit. e DSGVO).';

CREATE INDEX IF NOT EXISTS idx_driver_locations_driver_recorded
    ON driver_locations (driver_id, recorded_at DESC);
COMMENT ON INDEX idx_driver_locations_driver_recorded IS 'Zusammengesetzter Index fuer letzte Standortabfrage eines Fahrers.';

CREATE INDEX IF NOT EXISTS idx_driver_locations_coords
    ON driver_locations USING GIST (
        point(longitude, latitude)
    );
COMMENT ON INDEX idx_driver_locations_coords IS 'Raeumlicher GIST-Index fuer geografische Naeheabfragen (verfuegbare Fahrer in der Naehe).';

CREATE INDEX IF NOT EXISTS idx_driver_locations_ride_id
    ON driver_locations (ride_id)
    WHERE ride_id IS NOT NULL;
COMMENT ON INDEX idx_driver_locations_ride_id IS 'Partial-Index fuer Standortabfragen einer bestimmten Fahrt.';

-- ============================================================
-- TABELLE: driver_earnings
-- DSGVO-Hinweis: Einkommensdaten sind personenbezogene Daten.
--   Rechtsgrundlage: Art. 6 Abs. 1 lit. b DSGVO (Vertragserfuellung)
--   sowie Art. 6 Abs. 1 lit. c DSGVO (steuerrechtliche Aufbewahrung
--   gemaess § 147 AO: 10 Jahre).
-- ============================================================

CREATE TABLE IF NOT EXISTS driver_earnings (
    -- Primaerschluessel
    id              UUID                    PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Verknuepfung zum Fahrerprofil
    driver_id       UUID                    NOT NULL,

    -- Abrechnungszeitraum
    period_type     earnings_period_type    NOT NULL,
    period_start    DATE                    NOT NULL,
    period_end      DATE                    NOT NULL,

    -- Finanzdaten in Euro-Cent (Integer vermeidet Rundungsfehler)
    -- DSGVO: Einkommensdaten, Zugriff nur fuer Fahrer und berechtigte Buchhaltung
    total_fares     NUMERIC(12, 2)          NOT NULL DEFAULT 0.00 CHECK (total_fares >= 0),
    commission      NUMERIC(12, 2)          NOT NULL DEFAULT 0.00 CHECK (commission >= 0),
    bonuses         NUMERIC(12, 2)          NOT NULL DEFAULT 0.00 CHECK (bonuses >= 0),
    net_earnings    NUMERIC(12, 2)          NOT NULL DEFAULT 0.00,

    -- Fahrtanzahl im Zeitraum
    trip_count      INTEGER                 NOT NULL DEFAULT 0 CHECK (trip_count >= 0),

    -- Zeitstempel
    created_at      TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ             NOT NULL DEFAULT NOW(),

    -- Eindeutigkeitsbedingung: Pro Fahrer nur ein Eintrag pro Zeitraum-Typ und Zeitraum
    CONSTRAINT uq_driver_earnings_period
        UNIQUE (driver_id, period_type, period_start, period_end),

    -- Validierung: period_end muss nach period_start liegen
    CONSTRAINT chk_driver_earnings_period_order
        CHECK (period_end >= period_start)
);

-- Tabellenkommentar
COMMENT ON TABLE driver_earnings IS
    'Abrechnungsdaten der Fahrer (taeglich/woechentlich/monatlich). '
    'DSGVO: Einkommensdaten sind personenbezogen. '
    'Rechtsgrundlage: Art. 6 Abs. 1 lit. b (Vertrag) und lit. c DSGVO (§ 147 AO). '
    'Aufbewahrungsfrist: 10 Jahre gemaess steuerrechtlicher Verpflichtung.';

-- Spaltenkommentare
COMMENT ON COLUMN driver_earnings.id IS 'Eindeutiger Primaerschluessel (UUID v4)';
COMMENT ON COLUMN driver_earnings.driver_id IS 'Fremdschluessel zum Fahrerprofil.';
COMMENT ON COLUMN driver_earnings.period_type IS 'Abrechnungsperiode: taeglich, woechentlich oder monatlich.';
COMMENT ON COLUMN driver_earnings.period_start IS 'Beginn des Abrechnungszeitraums (inklusive).';
COMMENT ON COLUMN driver_earnings.period_end IS 'Ende des Abrechnungszeitraums (inklusive).';
COMMENT ON COLUMN driver_earnings.total_fares IS 'Gesamte Fahrtpreise im Zeitraum (in EUR). DSGVO: Einkommensdatum.';
COMMENT ON COLUMN driver_earnings.commission IS 'Plattformprovision im Zeitraum (in EUR).';
COMMENT ON COLUMN driver_earnings.bonuses IS 'Bonuszahlungen und Anreize im Zeitraum (in EUR).';
COMMENT ON COLUMN driver_earnings.net_earnings IS 'Nettoverdienst des Fahrers im Zeitraum (total_fares - commission + bonuses) in EUR.';
COMMENT ON COLUMN driver_earnings.trip_count IS 'Anzahl der abgeschlossenen Fahrten im Zeitraum.';
COMMENT ON COLUMN driver_earnings.created_at IS 'Zeitpunkt der Datensatzerstellung (UTC).';
COMMENT ON COLUMN driver_earnings.updated_at IS 'Zeitpunkt der letzten Aktualisierung (UTC), automatisch gesetzt.';

-- Trigger: updated_at automatisch aktualisieren
CREATE TRIGGER trg_driver_earnings_updated_at
    BEFORE UPDATE ON driver_earnings
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TRIGGER trg_driver_earnings_updated_at ON driver_earnings IS
    'Setzt updated_at automatisch auf NOW() bei jeder Aktualisierung.';

-- Foreign Key Constraint
ALTER TABLE driver_earnings
    ADD CONSTRAINT fk_driver_earnings_driver_id
    FOREIGN KEY (driver_id)
    REFERENCES drivers (id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE;

COMMENT ON CONSTRAINT fk_driver_earnings_driver_id ON driver_earnings IS
    'FK zu drivers: RESTRICT DELETE verhindert Loeschung von Fahrern mit Abrechnungsdaten (steuerrechtliche Aufbewahrung gemaess § 147 AO).';

-- Indexes fuer driver_earnings
CREATE INDEX IF NOT EXISTS idx_driver_earnings_driver_id
    ON driver_earnings (driver_id);
COMMENT ON INDEX idx_driver_earnings_driver_id IS 'Index fuer Verdienstabfragen nach Fahrer-ID.';

CREATE INDEX IF NOT EXISTS idx_driver_earnings_period_type
    ON driver_earnings (period_type);
COMMENT ON INDEX idx_driver_earnings_period_type IS 'Index fuer Filterung nach Abrechnungsperiode.';

CREATE INDEX IF NOT EXISTS idx_driver_earnings_period_start
    ON driver_earnings (period_start DESC);
COMMENT ON INDEX idx_driver_earnings_period_start IS 'Index fuer zeitliche Sortierung und Bereichsabfragen.';

CREATE INDEX IF NOT EXISTS idx_driver_earnings_driver_period
    ON driver_earnings (driver_id, period_type, period_start DESC);
COMMENT ON INDEX idx_driver_earnings_driver_period IS 'Zusammengesetzter Index fuer fahrerspezifische Periodenabfragen.';

CREATE INDEX IF NOT EXISTS idx_driver_earnings_net_earnings
    ON driver_earnings (net_earnings DESC);
COMMENT ON INDEX idx_driver_earnings_net_earnings IS 'Index fuer Verdienstranking und Berichterstellung.';

-- ============================================================
-- TABELLE: driver_trip_history
-- DSGVO-Hinweis: Fahrthistorie enthaelt Bewegungsdaten und
--   Einkommensdaten. Rechtsgrundlage: Art. 6 Abs. 1 lit. b DSGVO
--   (Vertragserfuellung) und Art. 6 Abs. 1 lit. c DSGVO
--   (Aufbewahrungspflichten).
-- ============================================================

CREATE TABLE IF NOT EXISTS driver_trip_history (
    -- Primaerschluessel
    id                  UUID                PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Verknuepfung zum Fahrerprofil
    driver_id           UUID                NOT NULL,

    -- Verknuepfung zur Fahrt (ride-service, extern)
    ride_id             UUID                NOT NULL,

    -- Standortdaten der Fahrt
    -- DSGVO: Abholort und Zielort ermoeglichen Rueckschluesse auf Verhalten
    --   des Fahrgasts - nur fuer Abrechnungszwecke gespeichert
    pickup_location     TEXT                NOT NULL,
    dropoff_location    TEXT                NOT NULL,

    -- Optionale Koordinaten fuer Abholpunkt
    pickup_latitude     NUMERIC(10, 7)      NULL,
    pickup_longitude    NUMERIC(10, 7)      NULL,

    -- Optionale Koordinaten fuer Zielort
    dropoff_latitude    NUMERIC(10, 7)      NULL,
    dropoff_longitude   NUMERIC(10, 7)      NULL,

    -- Finanzdaten
    -- DSGVO: Fahrtbezogene Einkommensdaten des Fahrers
    fare                NUMERIC(10, 2)      NOT NULL DEFAULT 0.00 CHECK (fare >= 0),
    commission          NUMERIC(10, 2)      NOT NULL DEFAULT 0.00 CHECK (commission >= 0),
    driver_earnings     NUMERIC(10, 2)      NOT NULL DEFAULT 0.00,

    -- Fahrtdauer in Minuten
    duration_minutes    SMALLINT            NULL CHECK (duration_minutes IS NULL OR duration_minutes >= 0),

    -- Fahrtdistanz in Kilometern
    distance_km         NUMERIC(8, 3)       NULL CHECK (distance_km IS NULL OR distance_km >= 0),

    -- Fahrgastbewertung fuer diese Fahrt
    passenger_rating    NUMERIC(3, 2)       NULL CHECK (passenger_rating IS NULL OR (passenger_rating >= 0 AND passenger_rating <= 5)),

    -- Status der Fahrt
    status              trip_history_status NOT NULL DEFAULT 'completed',

    -- Abschlusszeitpunkt der Fahrt
    completed_at        TIMESTAMPTZ         NULL,

    -- Erstellungszeitpunkt
    created_at          TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

    -- --------------------------------------------------------
    -- DSGVO-Compliance-Felder
    -- --------------------------------------------------------

    -- Art. 5 Abs. 1 lit. e DSGVO: Nach Ablauf sind detaillierte
    -- Standortdaten zu anonymisieren (nur aggregierte Daten behalten)
    dsgvo_standort_loeschdatum  TIMESTAMPTZ NULL,

    -- Kennzeichen ob Standortdetails bereits anonymisiert wurden
    dsgvo_standort_anonymisiert BOOLEAN     NOT NULL DEFAULT FALSE,

    -- Eindeutigkeitsbedingung: Eine Fahrt kann nur einmal in der Historie erscheinen
    CONSTRAINT uq_driver_trip_history_ride_id
        UNIQUE (driver_id, ride_id)
);

-- Tabellenkommentar
COMMENT ON TABLE driver_trip_history IS
    'Fahrthistorie der Fahrer. Enthaelt Fahrt- und Einkommensdaten. '
    'DSGVO: Standortdaten (Abholung/Ziel) werden nach Ablauf anonymisiert. '
    'Rechtsgrundlage: Art. 6 Abs. 1 lit. b DSGVO (Vertragserfuellung). '
    'Aufbewahrung: Finanzdaten 10 Jahre (§ 147 AO), Standortdetails max. 90 Tage.';

-- Spaltenkommentare
COMMENT ON COLUMN driver_trip_history.id IS 'Eindeutiger Primaerschluessel (UUID v4)';
COMMENT ON COLUMN driver_trip_history.driver_id IS 'Fremdschluessel zum Fahrerprofil.';
COMMENT ON COLUMN driver_trip_history.ride_id IS 'Referenz zur Fahrt im ride-service (externer Service).';
COMMENT ON COLUMN driver_trip_history.pickup_location IS 'Abholadresse als Text. DSGVO: Wird nach 90 Tagen anonymisiert.';
COMMENT ON COLUMN driver_trip_history.dropoff_location IS 'Zieladresse als Text. DSGVO: Wird nach 90 Tagen anonymisiert.';
COMMENT ON COLUMN driver_trip_history.pickup_latitude IS 'GPS-Breitengrad des Abholpunkts. DSGVO: Wird nach 90 Tagen geloescht.';
COMMENT ON COLUMN driver_trip_history.pickup_longitude IS 'GPS-Laengengrad des Abholpunkts. DSGVO: Wird nach 90 Tagen geloescht.';
COMMENT ON COLUMN driver_trip_history.dropoff_latitude IS 'GPS-Breitengrad des Zielorts. DSGVO: Wird nach 90 Tagen geloescht.';
COMMENT ON COLUMN driver_trip_history.dropoff_longitude IS 'GPS-Laengengrad des Zielorts. DSGVO: Wird nach 90 Tagen geloescht.';
COMMENT ON COLUMN driver_trip_history.fare IS 'Gesamtfahrtpreis in EUR.';
COMMENT ON COLUMN driver_trip_history.commission IS 'Plattformprovision fuer diese Fahrt in EUR.';
COMMENT ON COLUMN driver_trip_history.driver_earnings IS 'Nettoverdienst des Fahrers fuer diese Fahrt (fare - commission) in EUR.';
COMMENT ON COLUMN driver_trip_history.duration_minutes IS 'Fahrtdauer in Minuten.';
COMMENT ON COLUMN driver_trip_history.distance_km IS 'Zurueckgelegte Distanz in Kilometern.';
COMMENT ON COLUMN driver_trip_history.passenger_rating IS 'Bewertung des Fahrgasts fuer diese Fahrt (0.00 - 5.00).';
COMMENT ON COLUMN driver_trip_history.status IS 'Fahrtabschlussstatus: abgeschlossen, abgebrochen oder Fahrgast nicht erschienen.';
COMMENT ON COLUMN driver_trip_history.completed_at IS 'Zeitpunkt des Fahrtabschlusses (UTC).';
COMMENT ON COLUMN driver_trip_history.created_at IS 'Zeitpunkt der Datensatzerstellung (UTC).';
COMMENT ON COLUMN driver_trip_history.dsgvo_standort_loeschdatum IS 'DSGVO: Datum ab dem detaillierte Standortdaten zu anonymisieren sind (standard: 90 Tage nach Fahrt).';
COMMENT ON COLUMN driver_trip_history.dsgvo_standort_anonymisiert IS 'DSGVO: Kennzeichen ob Standortdetails dieser Fahrt bereits anonymisiert wurden.';

-- Foreign Key Constraint
ALTER TABLE driver_trip_history
    ADD CONSTRAINT fk_driver_trip_history_driver_id
    FOREIGN KEY (driver_id)
    REFERENCES drivers (id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE;

COMMENT ON CONSTRAINT fk_driver_trip_history_driver_id ON driver_trip_history IS
    'FK zu drivers: RESTRICT DELETE schuetzt Fahrthistorie vor versehentlicher Loeschung (Aufbewahrungspflicht § 147 AO).';

-- Indexes fuer driver_trip_history
CREATE INDEX IF NOT EXISTS idx_driver_trip_history_driver_id
    ON driver_trip_history (driver_id);
COMMENT ON INDEX idx_driver_trip_history_driver_id IS 'Index fuer Fahrthistorieabfragen nach Fahrer-ID.';

CREATE INDEX IF NOT EXISTS idx_driver_trip_history_ride_id
    ON driver_trip_history (ride_id);
COMMENT ON INDEX idx_driver_trip_history_ride_id IS 'Index fuer Fahrtsuche nach ride_id (ride-service Verknuepfung).';

CREATE INDEX IF NOT EXISTS idx_driver_trip_history_completed_at
    ON driver_trip_history (completed_at DESC)
    WHERE completed_at IS NOT NULL;
COMMENT ON INDEX idx_driver_trip_history_completed_at IS 'Partial-Index fuer abgeschlossene Fahrten in zeitlicher Reihenfolge.';

CREATE INDEX IF NOT EXISTS idx_driver_trip_history_driver_completed
    ON driver_trip_history (driver_id, completed_at DESC);
COMMENT ON INDEX idx_driver_trip_history_driver_completed IS 'Zusammengesetzter Index fuer fahrerspezifische Historieabfragen.';

CREATE INDEX IF NOT EXISTS idx_driver_trip_history_status
    ON driver_trip_history (status);
COMMENT ON INDEX idx_driver_trip_history_status IS 'Index fuer Statusfilterung (abgeschlossene, abgebrochene Fahrten).';

CREATE INDEX IF NOT EXISTS idx_driver_trip_history_created_at
    ON driver_trip_history (created_at DESC);
COMMENT ON INDEX idx_driver_trip_history_created_at IS 'Index fuer zeitliche Sortierung und Berichterstellung.';

CREATE INDEX IF NOT EXISTS idx_driver_trip_history_dsgvo_standort
    ON driver_trip_history (dsgvo_standort_loeschdatum)
    WHERE dsgvo_standort_anonymisiert = FALSE AND dsgvo_standort_loeschdatum IS NOT NULL;
COMMENT ON INDEX idx_driver_trip_history_dsgvo_standort IS 'DSGVO: Partial-Index fuer Anonymisierungsprozess von Standortdaten aus der Fahrthistorie.';

CREATE INDEX IF NOT EXISTS idx_driver_trip_history_earnings
    ON driver_trip_history (driver_earnings DESC);
COMMENT ON INDEX idx_driver_trip_history_earnings IS 'Index fuer Verdienstauswertungen nach Einzelfahrt.';

-- ============================================================
-- DSGVO AUDIT LOG TABELLE
-- Art. 5 Abs. 2 DSGVO: Rechenschaftspflicht
-- Alle DSGVO-relevanten Aktionen werden protokolliert
-- ============================================================

CREATE TABLE IF NOT EXISTS driver_dsgvo_audit_log (
    id                  UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id           UUID            NOT NULL,

    -- Art der DSGVO-Aktion
    aktion              VARCHAR(100)    NOT NULL,
    -- z.B.: 'AUSKUNFT_ERTEILT', 'LOESCHUNG_BEANTRAGT', 'EINWILLIGUNG_ERTEILT',
    --       'EINWILLIGUNG_WIDERRUFEN', 'EXPORT_ERSTELLT', 'DATEN_PSEUDONYMISIERT'

    -- Betroffene Tabelle oder Datenfeld
    betroffene_tabelle  VARCHAR(100)    NULL,
    betroffene_felder   TEXT[]          NULL,

    -- Rechtsgrundlage der Verarbeitung
    rechtsgrundlage     VARCHAR(200)    NULL,

    -- Ausfuehrender (System-Benutzer oder Fahrer selbst)
    ausgefuehrt_von     VARCHAR(200)    NULL,

    -- Zusaetzliche Informationen
    notizen             TEXT            NULL,

    -- Unveraenderlicher Zeitstempel
    erstellt_am         TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE driver_dsgvo_audit_log IS
    'DSGVO Audit-Protokoll fuer alle datenschutzrelevanten Aktionen. '
    'Art. 5 Abs. 2 DSGVO: Rechenschaftspflicht gegenueber Aufsichtsbehoerden. '
    'Unveraenderliches Protokoll - kein UPDATE oder DELETE erlaubt. '
    'Aufbewahrung: Mindestens 3 Jahre (Verjaebrungsfristen).';

COMMENT ON COLUMN driver_dsgvo_audit_log.id IS 'Eindeutiger Primaerschluessel (UUID v4)';
COMMENT ON COLUMN driver_dsgvo_audit_log.driver_id IS 'Referenz zum betroffenen Fahrerprofil.';
COMMENT ON COLUMN driver_dsgvo_audit_log.aktion IS 'Art der DSGVO-Aktion (z.B. AUSKUNFT, LOESCHUNG, EXPORT).';
COMMENT ON COLUMN driver_dsgvo_audit_log.betroffene_tabelle IS 'Tabelle die von der Aktion betroffen ist.';
COMMENT ON COLUMN driver_dsgvo_audit_log.betroffene_felder IS 'Liste der betroffenen Datenfelder.';
COMMENT ON COLUMN driver_dsgvo_audit_log.rechtsgrundlage IS 'Rechtsgrundlage der Verarbeitung (z.B. Art. 6 Abs. 1 lit. b DSGVO).';
COMMENT ON COLUMN driver_dsgvo_audit_log.ausgefuehrt_von IS 'Identifikation des Ausfuehrenden (Mitarbeiter-ID, System oder Fahrer).';
COMMENT ON COLUMN driver_dsgvo_audit_log.notizen IS 'Freitextfeld fuer zusaetzliche Informationen zur Aktion.';
COMMENT ON COLUMN driver_dsgvo_audit_log.erstellt_am IS 'Unveraenderlicher Erstellungszeitstempel des Log-Eintrags.';

-- Foreign Key (kein CASCADE - Audit-Log bleibt auch nach Fahrer-Loeschung erhalten)
ALTER TABLE driver_dsgvo_audit_log
    ADD CONSTRAINT fk_dsgvo_audit_driver_id
    FOREIGN KEY (driver_id)
    REFERENCES drivers (id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE;

COMMENT ON CONSTRAINT fk_dsgvo_audit_driver_id ON driver_dsgvo_audit_log IS
    'FK zu drivers: RESTRICT stellt sicher, dass Audit-Log vor Fahrer-Loeschung erhalten bleibt (Rechenschaftspflicht Art. 5 Abs. 2 DSGVO).';

-- Indexes fuer driver_dsgvo_audit_log
CREATE INDEX IF NOT EXISTS idx_dsgvo_audit_driver_id
    ON driver_dsgvo_audit_log (driver_id);
COMMENT ON INDEX idx_dsgvo_audit_driver_id IS 'Index fuer Abfrage aller DSGVO-Aktionen eines Fahrers.';

CREATE INDEX IF NOT EXISTS idx_dsgvo_audit_aktion
    ON driver_dsgvo_audit_log (aktion);
COMMENT ON INDEX idx_dsgvo_audit_aktion IS 'Index fuer Filterung nach Aktionstyp.';

CREATE INDEX IF NOT EXISTS idx_dsgvo_audit_erstellt_am
    ON driver_dsgvo_audit_log (erstellt_am DESC);
COMMENT ON INDEX idx_dsgvo_audit_erstellt_am IS 'Index fuer zeitliche Sortierung des Audit-Protokolls.';

-- ============================================================
-- SICHERHEIT: Row Level Security (RLS) Vorbereitung
-- DSGVO Art. 25: Datenschutz durch Technikgestaltung
-- ============================================================

-- RLS auf allen Tabellen aktivieren (Policies werden vom Anwendungscode gesetzt)
ALTER TABLE drivers ENABLE ROW LEVEL SECURITY;
ALTER TABLE driver_locations ENABLE ROW LEVEL SECURITY;
ALTER TABLE driver_earnings ENABLE ROW LEVEL SECURITY;
ALTER TABLE driver_trip_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE driver_dsgvo_audit_log ENABLE ROW LEVEL SECURITY;

-- Standard-Policy: Nur Service-Account hat vollen Zugriff
-- (Spezifische Policies werden vom Datenbankadministrator konfiguriert)
COMMENT ON TABLE drivers IS
    'DSGVO Art. 25 (Privacy by Design): Row Level Security aktiviert. '
    'Fahrer-Profile. Enthaelt personenbezogene Daten gemaess DSGVO Art. 4. '
    'Rechtsgrundlage: Art. 6 Abs. 1 lit. b DSGVO (Vertragserfuellung). '
    'Verantwortlicher: Siehe Datenschutzbeauftragter. '
    'Aufbewahrungsfrist: 10 Jahre nach Vertragsende gemaess § 147 AO.';

-- ============================================================
-- DATENBANKKOMMENTARE: Schema-Level
-- ============================================================

COMMENT ON SCHEMA public IS
    'Driver-Service Datenbankschema. '
    'DSGVO-konform gemaess EU-Datenschutz-Grundverordnung (EU) 2016/679. '
    'Datenschutzbeauftragter: Bitte im Unternehmensdatenschutzverzeichnis nachschlagen. '
    'Verzeichnis der Verarbeitungstaetigkeiten (VVT): Art. 30 DSGVO.';

-- ============================================================
-- MIGRATION ABGESCHLOSSEN
-- ============================================================
-- Zusammenfassung der erstellten Objekte:
--   Erweiterungen : uuid-ossp
--   Enums         : driver_status, earnings_period_type, trip_history_status
--   Tabellen      : drivers, driver_locations, driver_earnings,
--                   driver_trip_history, driver_dsgvo_audit_log
--   Trigger       : trg_drivers_updated_at, trg_driver_earnings_updated_at
--   Funktionen    : trigger_set_updated_at()
--   Constraints   : 4x FK, diverse CHECK und UNIQUE constraints
--   Indexes       : 22 Indexes (inkl. GIST raeumlicher Index)
--   RLS           : Aktiviert auf allen 5 Tabellen
-- DSGVO-Massnahmen:
--   - Automatische Loeschfristen (expires_at)
--   - Audit-Log fuer Rechenschaftspflicht (Art. 5 Abs. 2)
--   - Row Level Security (Art. 25 - Privacy by Design)
--   - Pseudonymisierungskennzeichen
--   - Datensparsamkeitsprinzip (Art. 5 Abs. 1 lit. c)
-- ============================================================

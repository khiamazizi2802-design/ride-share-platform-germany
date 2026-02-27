-- =============================================================================
-- Vehicle Management Service - Database Migrations
-- =============================================================================
-- Compliant with German vehicle registration and documentation requirements
-- Follows Fahrzeugzulassungsverordnung (FZV) and StVZO regulations
-- =============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- =============================================================================
-- ENUMS
-- =============================================================================

-- Vehicle operational status
CREATE TYPE vehicle_status AS ENUM (
    'pending_inspection',    -- Awaiting initial TÃV/DEKRA inspection
    'active',               -- Approved and operational
    'suspended',            -- Temporarily suspended (e.g., expired documents)
    'under_maintenance',    -- Currently in maintenance/repair
    'decommissioned',       -- Permanently removed from service
    'stolen',               -- Reported stolen
    'total_loss'            -- Insurance write-off
);

COMMENT ON TYPE vehicle_status IS 'Operational status of a vehicle in the fleet';

-- Vehicle categories per EU Directive 2007/46/EC and German StVZO
CREATE TYPE vehicle_category AS ENUM (
    'M1',    -- Passenger vehicles up to 8 seats
    'M2',    -- Passenger vehicles over 8 seats, max 5t
    'M3',    -- Passenger vehicles over 8 seats, over 5t (buses)
    'N1',    -- Light commercial vehicles up to 3.5t
    'N2',    -- Medium commercial vehicles 3.5t-12t
    'N3',    -- Heavy commercial vehicles over 12t
    'L3e',   -- Motorcycles
    'L5e'    -- Motor tricycles
);

COMMENT ON TYPE vehicle_category IS 'EU vehicle category classification per Directive 2007/46/EC';

-- Fuel types
CREATE TYPE fuel_type AS ENUM (
    'petrol',           -- Benzin
    'diesel',           -- Diesel
    'electric',         -- Elektrisch (BEV)
    'hybrid_petrol',    -- Hybrid mit Benzin
    'hybrid_diesel',    -- Hybrid mit Diesel
    'plug_in_hybrid',   -- Plug-in Hybrid (PHEV)
    'natural_gas',      -- Erdgas (CNG/LNG)
    'hydrogen',         -- Wasserstoff (FCEV)
    'lpg'               -- FlÃ¼ssiggas
);

COMMENT ON TYPE fuel_type IS 'Vehicle fuel/propulsion type';

-- Transmission types
CREATE TYPE transmission_type AS ENUM (
    'manual',       -- Schaltgetriebe
    'automatic',    -- Automatikgetriebe
    'semi_automatic', -- Halbautomatik
    'cvt'           -- Stufenloses Getriebe
);

-- Document types for vehicle compliance (German regulatory requirements)
CREATE TYPE vehicle_document_type AS ENUM (
    'fahrzeugschein',           -- Vehicle registration certificate Part I (Zulassungsbescheinigung Teil I)
    'fahrzeugbrief',            -- Vehicle registration certificate Part II (Zulassungsbescheinigung Teil II)
    'tuev_report',              -- TÃV/DEKRA technical inspection report (Â§29 StVZO)
    'hauptuntersuchung',        -- Main inspection (HU) sticker record
    'abgasuntersuchung',        -- Emissions inspection (AU) record
    'insurance_certificate',    -- Versicherungsnachweis (eVB-Nummer)
    'insurance_policy',         -- Versicherungspolice
    'purchase_contract',        -- Kaufvertrag
    'leasing_contract',         -- Leasingvertrag
    'service_history',          -- Serviceheft / Wartungsnachweis
    'accident_report',          -- Unfallbericht
    'damage_assessment',        -- Schadensgutachten
    'emission_badge',           -- Umweltplakette
    'toll_declaration',         -- Mautanmeldung
    'customs_document',         -- Zolldokument (for non-EU vehicles)
    'recall_notice',            -- RÃ¼ckrufbenachrichtigung
    'modification_approval',    -- ABE / Einzelabnahme fÃ¼r Umbauten
    'other'                     -- Sonstiges
);

COMMENT ON TYPE vehicle_document_type IS 'Vehicle document types per German regulatory requirements (StVZO, FZV)';

-- Document verification status
CREATE TYPE document_status AS ENUM (
    'pending',      -- Uploaded, awaiting review
    'under_review', -- Currently being reviewed
    'approved',     -- Verified and accepted
    'rejected',     -- Failed verification
    'expired',      -- Past expiry date
    'superseded'    -- Replaced by newer version
);

COMMENT ON TYPE document_status IS 'Verification status of a vehicle document';

-- Maintenance record types
CREATE TYPE maintenance_type AS ENUM (
    'routine_service',          -- RegelmÃ¤Ãige Wartung
    'oil_change',               -- Ãlwechsel
    'tire_replacement',         -- Reifenwechsel
    'brake_service',            -- Bremsenwartung/-reparatur
    'engine_repair',            -- Motorinstandsetzung
    'transmission_repair',      -- Getriebereparatur
    'electrical_repair',        -- Elektrische Reparatur
    'body_repair',              -- Karosseriereparatur
    'tuev_preparation',         -- TÃV-Vorbereitung
    'recall_fix',               -- RÃ¼ckrufaktion
    'accident_repair',          -- Unfallreparatur
    'inspection',               -- Inspektion
    'cleaning',                 -- Fahrzeugreinigung
    'parts_replacement',        -- Teileaustausch
    'software_update',          -- Software-Update (for EVs/modern vehicles)
    'battery_service',          -- Batterieservice (EV)
    'other'                     -- Sonstiges
);

COMMENT ON TYPE maintenance_type IS 'Types of vehicle maintenance and repair activities';

-- Maintenance status
CREATE TYPE maintenance_status AS ENUM (
    'scheduled',    -- Geplant
    'in_progress',  -- In Bearbeitung
    'completed',    -- Abgeschlossen
    'cancelled',    -- Storniert
    'deferred'      -- Verschoben
);

-- Audit action types
CREATE TYPE audit_action AS ENUM (
    'INSERT',
    'UPDATE',
    'DELETE',
    'SELECT',   -- For sensitive data access logging
    'EXPORT',
    'IMPORT',
    'APPROVE',
    'REJECT',
    'SUSPEND',
    'REACTIVATE',
    'DECOMMISSION'
);

COMMENT ON TYPE audit_action IS 'Types of auditable actions for DSGVO compliance';

-- =============================================================================
-- TABLE: vehicles
-- =============================================================================

CREATE TABLE vehicles (
    -- Primary identification
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id               UUID NOT NULL,  -- References driver-onboarding-service
    fleet_id                UUID,           -- Optional fleet/company grouping

    -- Vehicle identification (legally required in Germany)
    vin                     VARCHAR(17) NOT NULL,   -- Fahrzeug-Identifizierungsnummer (FIN) - ISO 3779
    license_plate           VARCHAR(20) NOT NULL,   -- Kennzeichen (e.g., "B-AB 1234")
    license_plate_region    VARCHAR(3),             -- Unterscheidungszeichen (e.g., "B", "M", "HH")
    internal_fleet_number   VARCHAR(50),            -- Internal fleet tracking number

    -- Vehicle specifications
    make                    VARCHAR(100) NOT NULL,  -- Hersteller (e.g., "Mercedes-Benz")
    model                   VARCHAR(100) NOT NULL,  -- Modell (e.g., "E-Klasse")
    variant                 VARCHAR(100),           -- AusfÃ¼hrung/Variante
    year                    SMALLINT NOT NULL,      -- Erstzulassungsjahr
    color                   VARCHAR(50) NOT NULL,   -- Farbe
    color_code              VARCHAR(20),            -- Herstellerfarbcode
    category                vehicle_category NOT NULL DEFAULT 'M1',
    fuel_type               fuel_type NOT NULL,
    transmission            transmission_type NOT NULL DEFAULT 'automatic',
    number_of_seats         SMALLINT NOT NULL DEFAULT 5,
    number_of_doors         SMALLINT,

    -- Engine specifications
    engine_displacement_cc  INTEGER,                -- Hubraum in ccm
    engine_power_kw         DECIMAL(8,2),           -- Motorleistung in kW
    engine_power_hp         DECIMAL(8,2),           -- Motorleistung in PS (auto-computed)
    co2_emissions_gkm       DECIMAL(8,2),           -- CO2-Emissionen in g/km
    euro_emission_standard  VARCHAR(10),            -- Euro-Abgasnorm (e.g., "Euro 6d")
    emission_badge_color    VARCHAR(10),            -- Umweltplakette: green, yellow, red

    -- Electric vehicle specifics
    battery_capacity_kwh    DECIMAL(8,2),           -- BatteriekapazitÃ¤t in kWh (EV only)
    electric_range_km       INTEGER,                -- Elektrische Reichweite in km (EV only)

    -- Registration details (Zulassungsdaten)
    first_registration_date DATE NOT NULL,          -- Datum der Erstzulassung
    registration_date       DATE,                   -- Datum der aktuellen Zulassung
    registration_authority  VARCHAR(100),           -- ZulassungsbehÃ¶rde
    registration_district   VARCHAR(100),           -- Zulassungskreis

    -- Technical inspection (TÃV/HU)
    last_inspection_date    DATE,                   -- Datum der letzten Hauptuntersuchung
    next_inspection_due     DATE,                   -- FÃ¤lligkeitsdatum nÃ¤chste HU (Â§29 StVZO)
    inspection_interval_months SMALLINT DEFAULT 24, -- PrÃ¼fintervall in Monaten

    -- Odometer
    current_mileage_km      INTEGER,                -- Aktueller Kilometerstand
    mileage_recorded_at     TIMESTAMPTZ,            -- Zeitstempel der Kilometerstandserfassung

    -- Insurance (Kfz-Versicherung - mandatory in Germany per PflVG)
    insurance_provider      VARCHAR(100),           -- Versicherungsgesellschaft
    insurance_policy_number VARCHAR(100),           -- Versicherungsscheinnummer
    insurance_type          VARCHAR(50),            -- Haftpflicht / Teilkasko / Vollkasko
    insurance_evb_number    VARCHAR(20),            -- eVB-Nummer (elektronische VersicherungsbestÃ¤tigung)
    insurance_expiry_date   DATE,                   -- Ablaufdatum der Versicherung

    -- Status
    status                  vehicle_status NOT NULL DEFAULT 'pending_inspection',
    status_reason           TEXT,                   -- BegrÃ¼ndung fÃ¼r StatusÃ¤nderung
    status_changed_at       TIMESTAMPTZ,
    status_changed_by       UUID,                   -- Staff member who changed status

    -- Metadata
    notes                   TEXT,                   -- Internal notes
    tags                    TEXT[],                 -- Flexible tagging
    metadata                JSONB DEFAULT '{}',     -- Extensible metadata

    -- Soft delete
    deleted_at              TIMESTAMPTZ,
    deleted_by              UUID,
    deletion_reason         TEXT,

    -- Audit timestamps
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL,
    updated_by              UUID NOT NULL,

    -- Constraints
    CONSTRAINT vehicles_vin_format CHECK (
        vin ~ '^[A-HJ-NPR-Z0-9]{17}$'
    ),
    CONSTRAINT vehicles_year_valid CHECK (
        year >= 1900 AND year <= EXTRACT(YEAR FROM NOW())::SMALLINT + 1
    ),
    CONSTRAINT vehicles_seats_positive CHECK (number_of_seats > 0),
    CONSTRAINT vehicles_doors_positive CHECK (number_of_doors IS NULL OR number_of_doors > 0),
    CONSTRAINT vehicles_mileage_non_negative CHECK (current_mileage_km IS NULL OR current_mileage_km >= 0),
    CONSTRAINT vehicles_engine_power_positive CHECK (
        engine_power_kw IS NULL OR engine_power_kw > 0
    ),
    CONSTRAINT vehicles_battery_capacity_positive CHECK (
        battery_capacity_kwh IS NULL OR battery_capacity_kwh > 0
    ),
    CONSTRAINT vehicles_emission_badge_valid CHECK (
        emission_badge_color IS NULL OR
        emission_badge_color IN ('green', 'yellow', 'red', 'none')
    ),
    CONSTRAINT vehicles_inspection_interval_valid CHECK (
        inspection_interval_months > 0 AND inspection_interval_months <= 48
    )
);

COMMENT ON TABLE vehicles IS 'Core vehicle registry - stores all fleet vehicle information per German StVZO/FZV requirements';
COMMENT ON COLUMN vehicles.vin IS 'Vehicle Identification Number per ISO 3779 (17 chars, no I, O, Q)';
COMMENT ON COLUMN vehicles.license_plate IS 'German Kraftfahrzeugkennzeichen (KFZ-Kennzeichen) per FZV';
COMMENT ON COLUMN vehicles.next_inspection_due IS 'HU-FÃ¤lligkeitsdatum per Â§29 StVZO - triggers compliance alerts';
COMMENT ON COLUMN vehicles.insurance_evb_number IS 'Elektronische VersicherungsbestÃ¤tigung - required for registration per PflVG';
COMMENT ON COLUMN vehicles.euro_emission_standard IS 'EU emission standard classification (Euro 1-6d) affects Umweltzone access';
COMMENT ON COLUMN vehicles.metadata IS 'Extensible JSON field for service-specific or partner-specific vehicle data';

-- Unique constraints
ALTER TABLE vehicles ADD CONSTRAINT vehicles_vin_unique
    UNIQUE (vin)
    WHERE deleted_at IS NULL;

ALTER TABLE vehicles ADD CONSTRAINT vehicles_license_plate_unique
    UNIQUE (license_plate)
    WHERE deleted_at IS NULL;

-- Indexes for vehicles
CREATE INDEX idx_vehicles_driver_id ON vehicles (driver_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicles_fleet_id ON vehicles (fleet_id) WHERE fleet_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_vehicles_status ON vehicles (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicles_license_plate ON vehicles (license_plate) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicles_vin ON vehicles (vin);
CREATE INDEX idx_vehicles_make_model ON vehicles (make, model) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicles_next_inspection ON vehicles (next_inspection_due) WHERE deleted_at IS NULL AND status != 'decommissioned';
CREATE INDEX idx_vehicles_insurance_expiry ON vehicles (insurance_expiry_date) WHERE deleted_at IS NULL AND status != 'decommissioned';
CREATE INDEX idx_vehicles_first_registration ON vehicles (first_registration_date);
CREATE INDEX idx_vehicles_created_at ON vehicles (created_at);
CREATE INDEX idx_vehicles_driver_status ON vehicles (driver_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicles_tags ON vehicles USING GIN (tags) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicles_metadata ON vehicles USING GIN (metadata) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicles_deleted_at ON vehicles (deleted_at) WHERE deleted_at IS NOT NULL;

-- =============================================================================
-- TABLE: vehicle_documents
-- =============================================================================

CREATE TABLE vehicle_documents (
    -- Primary identification
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id          UUID NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    driver_id           UUID NOT NULL,  -- Denormalized for query efficiency

    -- Document classification
    document_type       vehicle_document_type NOT NULL,
    document_number     VARCHAR(100),           -- Urkundennummer / Dokumentennummer
    issuing_authority   VARCHAR(200),           -- Ausstellende BehÃ¶rde / Institution
    issuing_country     CHAR(2) DEFAULT 'DE',   -- ISO 3166-1 alpha-2

    -- File storage
    file_url            TEXT NOT NULL,          -- Encrypted storage URL (S3/MinIO)
    file_name           VARCHAR(255),           -- Original filename
    file_size_bytes     INTEGER,                -- File size for storage management
    file_mime_type      VARCHAR(100),           -- MIME type
    file_hash           VARCHAR(64),            -- SHA-256 hash for integrity verification
    thumbnail_url       TEXT,                   -- Preview thumbnail URL

    -- Validity
    issue_date          DATE,                   -- Ausstellungsdatum
    expiry_date         DATE,                   -- Ablaufdatum / FÃ¤lligkeitsdatum
    is_mandatory        BOOLEAN NOT NULL DEFAULT FALSE, -- Pflichtdokument per Gesetz

    -- Verification
    status              document_status NOT NULL DEFAULT 'pending',
    reviewed_by         UUID,                   -- Staff member who reviewed
    reviewed_at         TIMESTAMPTZ,
    rejection_reason    TEXT,                   -- Ablehnungsgrund
    verification_notes  TEXT,                   -- Internal verification notes

    -- OCR / Auto-extraction results
    ocr_extracted_data  JSONB DEFAULT '{}',     -- Machine-readable extracted fields
    ocr_confidence      DECIMAL(5,4),           -- OCR confidence score (0.0000-1.0000)
    ocr_processed_at    TIMESTAMPTZ,

    -- Versioning
    version             INTEGER NOT NULL DEFAULT 1,
    superseded_by       UUID REFERENCES vehicle_documents(id),
    supersedes          UUID REFERENCES vehicle_documents(id),

    -- Reminder tracking
    reminder_sent_at    TIMESTAMPTZ[],          -- Timestamps of expiry reminders sent
    next_reminder_at    TIMESTAMPTZ,            -- Scheduled next reminder

    -- Metadata
    notes               TEXT,
    metadata            JSONB DEFAULT '{}',

    -- Soft delete
    deleted_at          TIMESTAMPTZ,
    deleted_by          UUID,

    -- Audit timestamps
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by         UUID NOT NULL,          -- Who uploaded the document
    updated_by          UUID NOT NULL,

    -- Constraints
    CONSTRAINT vehicle_documents_file_size_positive CHECK (
        file_size_bytes IS NULL OR file_size_bytes > 0
    ),
    CONSTRAINT vehicle_documents_expiry_after_issue CHECK (
        expiry_date IS NULL OR issue_date IS NULL OR expiry_date >= issue_date
    ),
    CONSTRAINT vehicle_documents_ocr_confidence_range CHECK (
        ocr_confidence IS NULL OR (ocr_confidence >= 0 AND ocr_confidence <= 1)
    ),
    CONSTRAINT vehicle_documents_version_positive CHECK (version > 0),
    CONSTRAINT vehicle_documents_no_self_supersede CHECK (
        superseded_by IS NULL OR superseded_by != id
    )
);

COMMENT ON TABLE vehicle_documents IS 'Vehicle document storage - manages all regulatory and compliance documents per German law';
COMMENT ON COLUMN vehicle_documents.file_hash IS 'SHA-256 hash for document integrity verification and deduplication';
COMMENT ON COLUMN vehicle_documents.is_mandatory IS 'Indicates legally required documents per StVZO/FZV/PflVG';
COMMENT ON COLUMN vehicle_documents.ocr_extracted_data IS 'Structured data extracted via OCR for automated validation';
COMMENT ON COLUMN vehicle_documents.superseded_by IS 'Reference to newer document version that replaced this one';
COMMENT ON COLUMN vehicle_documents.reminder_sent_at IS 'Array of timestamps when expiry reminders were dispatched';

-- Indexes for vehicle_documents
CREATE INDEX idx_vehicle_docs_vehicle_id ON vehicle_documents (vehicle_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicle_docs_driver_id ON vehicle_documents (driver_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicle_docs_type ON vehicle_documents (document_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicle_docs_status ON vehicle_documents (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicle_docs_expiry ON vehicle_documents (expiry_date) WHERE deleted_at IS NULL AND status NOT IN ('expired', 'superseded');
CREATE INDEX idx_vehicle_docs_vehicle_type ON vehicle_documents (vehicle_id, document_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicle_docs_next_reminder ON vehicle_documents (next_reminder_at) WHERE next_reminder_at IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_vehicle_docs_reviewed_by ON vehicle_documents (reviewed_by) WHERE reviewed_by IS NOT NULL;
CREATE INDEX idx_vehicle_docs_mandatory ON vehicle_documents (vehicle_id, is_mandatory, status) WHERE is_mandatory = TRUE AND deleted_at IS NULL;
CREATE INDEX idx_vehicle_docs_ocr_data ON vehicle_documents USING GIN (ocr_extracted_data) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicle_docs_metadata ON vehicle_documents USING GIN (metadata);
CREATE INDEX idx_vehicle_docs_created_at ON vehicle_documents (created_at);
CREATE INDEX idx_vehicle_docs_file_hash ON vehicle_documents (file_hash) WHERE file_hash IS NOT NULL;

-- =============================================================================
-- TABLE: maintenance_records
-- =============================================================================

CREATE TABLE maintenance_records (
    -- Primary identification
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id              UUID NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    driver_id               UUID NOT NULL,  -- Denormalized for query efficiency

    -- Maintenance classification
    maintenance_type        maintenance_type NOT NULL,
    status                  maintenance_status NOT NULL DEFAULT 'scheduled',
    priority                SMALLINT NOT NULL DEFAULT 2, -- 1=critical, 2=normal, 3=low

    -- Service provider details
    service_provider_name   VARCHAR(200),           -- Werkstattname
    service_provider_id     UUID,                   -- Reference to service provider registry
    service_provider_address TEXT,                  -- Werkstattadresse
    technician_name         VARCHAR(100),           -- Name des Technikers
    work_order_number       VARCHAR(100),           -- Werkstattauftragsnummer

    -- Service details
    title                   VARCHAR(200) NOT NULL,  -- Brief title of the maintenance
    description             TEXT,                   -- Detailed description of work performed
    parts_replaced          JSONB DEFAULT '[]',     -- Array of replaced parts with details
    labor_hours             DECIMAL(6,2),           -- Arbeitsstunden

    -- Scheduling
    scheduled_date          DATE,                   -- Geplantes Datum
    scheduled_at            TIMESTAMPTZ,            -- Geplanter Zeitpunkt (mit Uhrzeit)
    started_at              TIMESTAMPTZ,            -- TatsÃ¤chlicher Beginn
    completed_at            TIMESTAMPTZ,            -- TatsÃ¤chlicher Abschluss

    -- Odometer readings
    mileage_at_service_km   INTEGER,                -- Kilometerstand bei Wartung
    next_service_mileage_km INTEGER,                -- Kilometerstand fÃ¼r nÃ¤chste Wartung
    next_service_date       DATE,                   -- Datum der nÃ¤chsten Wartung

    -- Cost tracking (in EUR, German market)
    labor_cost_eur          DECIMAL(10,2),          -- Lohnkosten in EUR (net)
    parts_cost_eur          DECIMAL(10,2),          -- Teilekosten in EUR (net)
    total_cost_eur          DECIMAL(10,2),          -- Gesamtkosten in EUR (net)
    vat_rate                DECIMAL(5,4) DEFAULT 0.19, -- MwSt-Satz (19% standard in Germany)
    vat_amount_eur          DECIMAL(10,2),          -- MwSt-Betrag in EUR
    total_cost_incl_vat_eur DECIMAL(10,2),          -- Gesamtkosten brutto in EUR
    invoice_number          VARCHAR(100),           -- Rechnungsnummer
    invoice_url             TEXT,                   -- URL zur Rechnung
    payment_method          VARCHAR(50),            -- Zahlungsart
    paid_at                 TIMESTAMPTZ,            -- Zahlungszeitpunkt

    -- Coverage
    covered_by_warranty     BOOLEAN DEFAULT FALSE,  -- Unter Garantie/GewÃ¤hrleistung
    warranty_claim_number   VARCHAR(100),           -- Garantienummer
    covered_by_insurance    BOOLEAN DEFAULT FALSE,  -- Durch Versicherung abgedeckt
    insurance_claim_number  VARCHAR(100),           -- Schadennummer

    -- Recall tracking
    is_recall_related       BOOLEAN DEFAULT FALSE,
    recall_number           VARCHAR(100),           -- KBA-RÃ¼ckrufnummer

    -- Inspection results
    inspection_passed       BOOLEAN,                -- TÃV/HU bestanden?
    defects_found           JSONB DEFAULT '[]',     -- Array of defects found
    defects_resolved        JSONB DEFAULT '[]',     -- Array of defects resolved

    -- Documents
    document_ids            UUID[],                 -- References to vehicle_documents
    receipt_url             TEXT,                   -- Kassenbon/Quittung URL

    -- Metadata
    internal_notes          TEXT,                   -- Internal staff notes
    tags                    TEXT[],
    metadata                JSONB DEFAULT '{}',

    -- Soft delete
    deleted_at              TIMESTAMPTZ,
    deleted_by              UUID,
    deletion_reason         TEXT,

    -- Audit timestamps
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL,
    updated_by              UUID NOT NULL,

    -- Constraints
    CONSTRAINT maintenance_priority_range CHECK (priority BETWEEN 1 AND 3),
    CONSTRAINT maintenance_labor_hours_positive CHECK (
        labor_hours IS NULL OR labor_hours > 0
    ),
    CONSTRAINT maintenance_mileage_positive CHECK (
        mileage_at_service_km IS NULL OR mileage_at_service_km >= 0
    ),
    CONSTRAINT maintenance_next_mileage_greater CHECK (
        next_service_mileage_km IS NULL OR
        mileage_at_service_km IS NULL OR
        next_service_mileage_km > mileage_at_service_km
    ),
    CONSTRAINT maintenance_costs_non_negative CHECK (
        (labor_cost_eur IS NULL OR labor_cost_eur >= 0) AND
        (parts_cost_eur IS NULL OR parts_cost_eur >= 0) AND
        (total_cost_eur IS NULL OR total_cost_eur >= 0)
    ),
    CONSTRAINT maintenance_vat_rate_valid CHECK (
        vat_rate IS NULL OR (vat_rate >= 0 AND vat_rate <= 1)
    ),
    CONSTRAINT maintenance_completed_after_started CHECK (
        completed_at IS NULL OR started_at IS NULL OR completed_at >= started_at
    ),
    CONSTRAINT maintenance_warranty_requires_number CHECK (
        NOT covered_by_warranty OR warranty_claim_number IS NOT NULL
    ),
    CONSTRAINT maintenance_recall_requires_number CHECK (
        NOT is_recall_related OR recall_number IS NOT NULL
    )
);

COMMENT ON TABLE maintenance_records IS 'Vehicle maintenance and repair history - tracks all service activities';
COMMENT ON COLUMN maintenance_records.priority IS '1=Critical (safety-relevant), 2=Normal, 3=Low priority';
COMMENT ON COLUMN maintenance_records.parts_replaced IS 'JSON array: [{"part_number": "...", "description": "...", "quantity": 1, "unit_cost_eur": 0.00}]';
COMMENT ON COLUMN maintenance_records.defects_found IS 'JSON array of defects identified during service';
COMMENT ON COLUMN maintenance_records.vat_rate IS 'German MwSt rate (0.19 standard, 0.07 reduced)';
COMMENT ON COLUMN maintenance_records.recall_number IS 'KBA (Kraftfahrtbundesamt) official recall reference number';

-- Indexes for maintenance_records
CREATE INDEX idx_maintenance_vehicle_id ON maintenance_records (vehicle_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_maintenance_driver_id ON maintenance_records (driver_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_maintenance_type ON maintenance_records (maintenance_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_maintenance_status ON maintenance_records (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_maintenance_scheduled_date ON maintenance_records (scheduled_date) WHERE deleted_at IS NULL AND status IN ('scheduled', 'in_progress');
CREATE INDEX idx_maintenance_completed_at ON maintenance_records (completed_at) WHERE completed_at IS NOT NULL;
CREATE INDEX idx_maintenance_vehicle_status ON maintenance_records (vehicle_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_maintenance_priority ON maintenance_records (priority, status) WHERE deleted_at IS NULL AND status != 'completed';
CREATE INDEX idx_maintenance_service_provider ON maintenance_records (service_provider_id) WHERE service_provider_id IS NOT NULL;
CREATE INDEX idx_maintenance_next_service ON maintenance_records (next_service_date) WHERE next_service_date IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_maintenance_recall ON maintenance_records (recall_number) WHERE is_recall_related = TRUE;
CREATE INDEX idx_maintenance_cost ON maintenance_records (total_cost_incl_vat_eur) WHERE total_cost_incl_vat_eur IS NOT NULL;
CREATE INDEX idx_maintenance_tags ON maintenance_records USING GIN (tags) WHERE deleted_at IS NULL;
CREATE INDEX idx_maintenance_parts ON maintenance_records USING GIN (parts_replaced);
CREATE INDEX idx_maintenance_defects ON maintenance_records USING GIN (defects_found);
CREATE INDEX idx_maintenance_created_at ON maintenance_records (created_at);

-- =============================================================================
-- TABLE: audit_logs
-- =============================================================================

CREATE TABLE audit_logs (
    -- Primary identification
    id                  BIGSERIAL PRIMARY KEY,  -- Sequential for ordering
    correlation_id      UUID DEFAULT uuid_generate_v4(), -- Request/transaction correlation
    causation_id        UUID,                   -- ID of event that caused this action

    -- Target entity
    entity_type         VARCHAR(100) NOT NULL,  -- Table/entity name (e.g., 'vehicles')
    entity_id           UUID NOT NULL,          -- ID of affected record
    entity_version      INTEGER,                -- Version/revision of the entity

    -- Action details
    action              audit_action NOT NULL,
    action_description  TEXT,                   -- Human-readable description

    -- Change tracking
    old_values          JSONB,                  -- Previous state (for UPDATE/DELETE)
    new_values          JSONB,                  -- New state (for INSERT/UPDATE)
    changed_fields      TEXT[],                 -- Array of modified field names

    -- Actor information
    actor_id            UUID,                   -- User/service who performed the action
    actor_type          VARCHAR(50),            -- 'user', 'driver', 'admin', 'system', 'service'
    actor_email         VARCHAR(255),           -- Snapshot of actor email at time of action
    actor_role          VARCHAR(100),           -- Role at time of action
    on_behalf_of        UUID,                   -- If admin acted on behalf of driver

    -- Request context
    ip_address          INET,                   -- Client IP address
    user_agent          TEXT,                   -- Client user agent string
    request_id          VARCHAR(100),           -- HTTP request ID
    session_id          VARCHAR(100),           -- Session identifier
    service_name        VARCHAR(100),           -- Originating microservice
    service_version     VARCHAR(50),            -- Service version for debugging
    api_endpoint        TEXT,                   -- API endpoint that triggered action
    http_method         VARCHAR(10),            -- HTTP method (GET, POST, PUT, etc.)

    -- DSGVO / Compliance fields
    legal_basis         VARCHAR(100),           -- DSGVO Rechtsgrundlage (e.g., 'Art. 6(1)(b) DSGVO')
    data_classification VARCHAR(50),            -- 'public', 'internal', 'confidential', 'restricted'
    retention_until     DATE,                   -- Aufbewahrungsfrist per HGB/GoB (10 years)
    gdpr_relevant       BOOLEAN DEFAULT FALSE,  -- Flags DSGVO-relevant actions

    -- Outcome
    success             BOOLEAN NOT NULL DEFAULT TRUE,
    error_code          VARCHAR(50),
    error_message       TEXT,
    duration_ms         INTEGER,                -- Operation duration in milliseconds

    -- Metadata
    tags                TEXT[],
    metadata            JSONB DEFAULT '{}',

    -- Immutable timestamp (no updated_at - audit logs are write-once)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT audit_logs_actor_type_valid CHECK (
        actor_type IS NULL OR
        actor_type IN ('user', 'driver', 'admin', 'system', 'service', 'scheduler')
    ),
    CONSTRAINT audit_logs_data_classification_valid CHECK (
        data_classification IS NULL OR
        data_classification IN ('public', 'internal', 'confidential', 'restricted')
    ),
    CONSTRAINT audit_logs_http_method_valid CHECK (
        http_method IS NULL OR
        http_method IN ('GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS')
    ),
    CONSTRAINT audit_logs_duration_positive CHECK (
        duration_ms IS NULL OR duration_ms >= 0
    )
);

COMMENT ON TABLE audit_logs IS 'Immutable compliance audit trail - DSGVO Art. 5(2) Rechenschaftspflicht. Retention: 10 years per HGB Â§257';
COMMENT ON COLUMN audit_logs.id IS 'Sequential BIGSERIAL for guaranteed ordering of audit events';
COMMENT ON COLUMN audit_logs.correlation_id IS 'Links related events within a single request/transaction';
COMMENT ON COLUMN audit_logs.causation_id IS 'References the event that triggered this audit entry';
COMMENT ON COLUMN audit_logs.old_values IS 'JSON snapshot of entity state before modification (UPDATE/DELETE)';
COMMENT ON COLUMN audit_logs.new_values IS 'JSON snapshot of entity state after modification (INSERT/UPDATE)';
COMMENT ON COLUMN audit_logs.legal_basis IS 'DSGVO processing legal basis (Art. 6(1) a-f) for traceability';
COMMENT ON COLUMN audit_logs.retention_until IS 'Calculated retention date per Â§257 HGB (10yr commercial records)';
COMMENT ON COLUMN audit_logs.gdpr_relevant IS 'Flag for DSGVO-relevant operations requiring special handling';

-- Indexes for audit_logs (optimized for compliance queries, NOT for high-frequency writes)
CREATE INDEX idx_audit_entity ON audit_logs (entity_type, entity_id);
CREATE INDEX idx_audit_entity_id ON audit_logs (entity_id);
CREATE INDEX idx_audit_actor_id ON audit_logs (actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_audit_action ON audit_logs (action);
CREATE INDEX idx_audit_created_at ON audit_logs (created_at);
CREATE INDEX idx_audit_created_at_brin ON audit_logs USING BRIN (created_at); -- Efficient for time-range queries on large tables
CREATE INDEX idx_audit_correlation_id ON audit_logs (correlation_id);
CREATE INDEX idx_audit_service ON audit_logs (service_name);
CREATE INDEX idx_audit_gdpr ON audit_logs (gdpr_relevant, created_at) WHERE gdpr_relevant = TRUE;
CREATE INDEX idx_audit_entity_action ON audit_logs (entity_type, action, created_at);
CREATE INDEX idx_audit_retention ON audit_logs (retention_until) WHERE retention_until IS NOT NULL;
CREATE INDEX idx_audit_success ON audit_logs (success, action) WHERE success = FALSE; -- Quick access to failures
CREATE INDEX idx_audit_ip_address ON audit_logs (ip_address) WHERE ip_address IS NOT NULL;
CREATE INDEX idx_audit_metadata ON audit_logs USING GIN (metadata);
CREATE INDEX idx_audit_changed_fields ON audit_logs USING GIN (changed_fields);

-- Prevent modifications to audit log (security measure)
REVOKE UPDATE, DELETE, TRUNCATE ON audit_logs FROM PUBLIC;

-- =============================================================================
-- TABLE: vehicle_status_history
-- =============================================================================
-- Tracks all status transitions for compliance and dispute resolution

CREATE TABLE vehicle_status_history (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id      UUID NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    from_status     vehicle_status,             -- NULL for initial status
    to_status       vehicle_status NOT NULL,
    reason          TEXT,
    changed_by      UUID NOT NULL,
    changed_by_type VARCHAR(50) DEFAULT 'user',
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE vehicle_status_history IS 'Immutable vehicle status transition log for compliance and dispute resolution';

CREATE INDEX idx_status_history_vehicle ON vehicle_status_history (vehicle_id, created_at);
CREATE INDEX idx_status_history_to_status ON vehicle_status_history (to_status);
CREATE INDEX idx_status_history_created_at ON vehicle_status_history (created_at);

-- =============================================================================
-- TABLE: maintenance_schedules
-- =============================================================================
-- Predictive/recurring maintenance planning

CREATE TABLE maintenance_schedules (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id              UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    maintenance_type        maintenance_type NOT NULL,
    title                   VARCHAR(200) NOT NULL,
    description             TEXT,

    -- Recurrence (interval-based)
    interval_days           INTEGER,            -- Repeat every N days
    interval_mileage_km     INTEGER,            -- Repeat every N km

    -- Next due
    next_due_date           DATE,
    next_due_mileage_km     INTEGER,

    -- Notification settings
    notify_days_before      INTEGER DEFAULT 14, -- Alert N days before due date
    notify_km_before        INTEGER DEFAULT 500, -- Alert N km before due mileage

    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL,

    CONSTRAINT schedules_interval_defined CHECK (
        interval_days IS NOT NULL OR interval_mileage_km IS NOT NULL
    ),
    CONSTRAINT schedules_interval_positive CHECK (
        (interval_days IS NULL OR interval_days > 0) AND
        (interval_mileage_km IS NULL OR interval_mileage_km > 0)
    )
);

COMMENT ON TABLE maintenance_schedules IS 'Recurring maintenance schedule templates for proactive fleet management';

CREATE INDEX idx_maint_schedules_vehicle ON maintenance_schedules (vehicle_id) WHERE is_active = TRUE;
CREATE INDEX idx_maint_schedules_due_date ON maintenance_schedules (next_due_date) WHERE is_active = TRUE;

-- =============================================================================
-- FUNCTIONS AND TRIGGERS
-- =============================================================================

-- Function: auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION fn_update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION fn_update_timestamp() IS 'Automatically updates the updated_at column on row modification';

-- Apply updated_at trigger to all relevant tables
CREATE TRIGGER trg_vehicles_updated_at
    BEFORE UPDATE ON vehicles
    FOR EACH ROW EXECUTE FUNCTION fn_update_timestamp();

CREATE TRIGGER trg_vehicle_documents_updated_at
    BEFORE UPDATE ON vehicle_documents
    FOR EACH ROW EXECUTE FUNCTION fn_update_timestamp();

CREATE TRIGGER trg_maintenance_records_updated_at
    BEFORE UPDATE ON maintenance_records
    FOR EACH ROW EXECUTE FUNCTION fn_update_timestamp();

CREATE TRIGGER trg_maintenance_schedules_updated_at
    BEFORE UPDATE ON maintenance_schedules
    FOR EACH ROW EXECUTE FUNCTION fn_update_timestamp();

-- Function: auto-compute engine_power_hp from engine_power_kw
CREATE OR REPLACE FUNCTION fn_compute_engine_power_hp()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.engine_power_kw IS NOT NULL THEN
        NEW.engine_power_hp = ROUND(NEW.engine_power_kw * 1.35962::DECIMAL, 2);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION fn_compute_engine_power_hp() IS 'Automatically computes horsepower from kW (1 kW = 1.35962 PS)';

CREATE TRIGGER trg_vehicles_compute_hp
    BEFORE INSERT OR UPDATE OF engine_power_kw ON vehicles
    FOR EACH ROW EXECUTE FUNCTION fn_compute_engine_power_hp();

-- Function: auto-compute total maintenance cost including VAT
CREATE OR REPLACE FUNCTION fn_compute_maintenance_totals()
RETURNS TRIGGER AS $$
BEGIN
    -- Calculate total net cost
    IF NEW.labor_cost_eur IS NOT NULL OR NEW.parts_cost_eur IS NOT NULL THEN
        NEW.total_cost_eur = COALESCE(NEW.labor_cost_eur, 0) + COALESCE(NEW.parts_cost_eur, 0);
    END IF;

    -- Calculate VAT and gross total
    IF NEW.total_cost_eur IS NOT NULL AND NEW.vat_rate IS NOT NULL THEN
        NEW.vat_amount_eur = ROUND(NEW.total_cost_eur * NEW.vat_rate, 2);
        NEW.total_cost_incl_vat_eur = ROUND(NEW.total_cost_eur + NEW.vat_amount_eur, 2);
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION fn_compute_maintenance_totals() IS 'Auto-computes total costs and VAT amounts for maintenance records';

CREATE TRIGGER trg_maintenance_compute_costs
    BEFORE INSERT OR UPDATE OF labor_cost_eur, parts_cost_eur, vat_rate ON maintenance_records
    FOR EACH ROW EXECUTE FUNCTION fn_compute_maintenance_totals();

-- Function: record vehicle status transitions
CREATE OR REPLACE FUNCTION fn_track_vehicle_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR (TG_OP = 'UPDATE' AND OLD.status IS DISTINCT FROM NEW.status) THEN
        INSERT INTO vehicle_status_history (
            vehicle_id,
            from_status,
            to_status,
            reason,
            changed_by,
            metadata
        ) VALUES (
            NEW.id,
            CASE WHEN TG_OP = 'UPDATE' THEN OLD.status ELSE NULL END,
            NEW.status,
            NEW.status_reason,
            NEW.status_changed_by,
            jsonb_build_object(
                'trigger', 'fn_track_vehicle_status_change',
                'operation', TG_OP
            )
        );

        -- Update status change metadata on the vehicle itself
        IF TG_OP = 'UPDATE' THEN
            NEW.status_changed_at = NOW();
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION fn_track_vehicle_status_change() IS 'Automatically records all vehicle status transitions to vehicle_status_history';

CREATE TRIGGER trg_vehicle_status_history
    BEFORE INSERT OR UPDATE OF status ON vehicles
    FOR EACH ROW EXECUTE FUNCTION fn_track_vehicle_status_change();

-- Function: auto-mark documents as expired
CREATE OR REPLACE FUNCTION fn_mark_expired_documents()
RETURNS INTEGER AS $$
DECLARE
    updated_count INTEGER;
BEGIN
    UPDATE vehicle_documents
    SET
        status = 'expired',
        updated_at = NOW()
    WHERE
        status NOT IN ('expired', 'superseded', 'rejected') AND
        expiry_date IS NOT NULL AND
        expiry_date < CURRENT_DATE AND
        deleted_at IS NULL;

    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RETURN updated_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION fn_mark_expired_documents() IS 'Marks vehicle documents as expired based on expiry_date. Should be called via scheduled job daily';

-- Function: validate VIN checksum (North American standard - optional for EU)
CREATE OR REPLACE FUNCTION fn_validate_vin_format(vin_input TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    -- ISO 3779 format: 17 chars, no I, O, Q
    RETURN vin_input ~ '^[A-HJ-NPR-Z0-9]{17}$';
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION fn_validate_vin_format(TEXT) IS 'Validates VIN format per ISO 3779 (17 chars, excludes I, O, Q)';

-- Function: compute next HU inspection date
CREATE OR REPLACE FUNCTION fn_compute_next_inspection_date(
    p_last_inspection_date DATE,
    p_interval_months INTEGER DEFAULT 24
)
RETURNS DATE AS $$
BEGIN
    IF p_last_inspection_date IS NULL THEN
        RETURN NULL;
    END IF;
    RETURN p_last_inspection_date + (p_interval_months || ' months')::INTERVAL;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION fn_compute_next_inspection_date(DATE, INTEGER) IS 'Computes HU due date per Â§29 StVZO based on last inspection and interval';

-- Trigger: auto-update next_inspection_due when last_inspection_date changes
CREATE OR REPLACE FUNCTION fn_update_inspection_due_date()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.last_inspection_date IS NOT NULL AND (
        OLD.last_inspection_date IS DISTINCT FROM NEW.last_inspection_date OR
        OLD.inspection_interval_months IS DISTINCT FROM NEW.inspection_interval_months
    ) THEN
        NEW.next_inspection_due = fn_compute_next_inspection_date(
            NEW.last_inspection_date,
            NEW.inspection_interval_months
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_vehicles_inspection_due
    BEFORE INSERT OR UPDATE OF last_inspection_date, inspection_interval_months ON vehicles
    FOR EACH ROW EXECUTE FUNCTION fn_update_inspection_due_date();

-- =============================================================================
-- VIEWS
-- =============================================================================

-- View: Active vehicles with compliance summary
CREATE OR REPLACE VIEW v_vehicle_compliance_summary AS
SELECT
    v.id,
    v.driver_id,
    v.license_plate,
    v.make,
    v.model,
    v.year,
    v.status,
    v.next_inspection_due,
    v.insurance_expiry_date,
    -- Days until inspection due
    (v.next_inspection_due - CURRENT_DATE)::INTEGER AS days_until_inspection,
    -- Days until insurance expires
    (v.insurance_expiry_date - CURRENT_DATE)::INTEGER AS days_until_insurance_expiry,
    -- Compliance alerts
    CASE
        WHEN v.next_inspection_due < CURRENT_DATE THEN 'OVERDUE'
        WHEN v.next_inspection_due <= CURRENT_DATE + INTERVAL '30 days' THEN 'DUE_SOON'
        ELSE 'OK'
    END AS inspection_alert,
    CASE
        WHEN v.insurance_expiry_date < CURRENT_DATE THEN 'EXPIRED'
        WHEN v.insurance_expiry_date <= CURRENT_DATE + INTERVAL '30 days' THEN 'EXPIRING_SOON'
        ELSE 'OK'
    END AS insurance_alert,
    -- Document counts
    COUNT(DISTINCT vd.id) FILTER (WHERE vd.status = 'approved') AS approved_documents,
    COUNT(DISTINCT vd.id) FILTER (WHERE vd.status = 'pending') AS pending_documents,
    COUNT(DISTINCT vd.id) FILTER (WHERE vd.status = 'expired') AS expired_documents,
    COUNT(DISTINCT vd.id) FILTER (WHERE vd.status = 'rejected') AS rejected_documents,
    -- Pending maintenance
    COUNT(DISTINCT mr.id) FILTER (WHERE mr.status = 'scheduled' AND mr.priority = 1) AS critical_maintenance_count,
    v.created_at,
    v.updated_at
FROM vehicles v
LEFT JOIN vehicle_documents vd ON vd.vehicle_id = v.id AND vd.deleted_at IS NULL
LEFT JOIN maintenance_records mr ON mr.vehicle_id = v.id AND mr.deleted_at IS NULL
WHERE v.deleted_at IS NULL
GROUP BY v.id;

COMMENT ON VIEW v_vehicle_compliance_summary IS 'Aggregated compliance overview per vehicle - used for fleet management dashboard';

-- View: Documents expiring within 60 days
CREATE OR REPLACE VIEW v_expiring_documents AS
SELECT
    vd.id,
    vd.vehicle_id,
    vd.driver_id,
    v.license_plate,
    v.make,
    v.model,
    vd.document_type,
    vd.document_number,
    vd.expiry_date,
    (vd.expiry_date - CURRENT_DATE)::INTEGER AS days_until_expiry,
    vd.is_mandatory,
    vd.status,
    vd.next_reminder_at
FROM vehicle_documents vd
JOIN vehicles v ON v.id = vd.vehicle_id AND v.deleted_at IS NULL
WHERE
    vd.deleted_at IS NULL AND
    vd.expiry_date IS NOT NULL AND
    vd.expiry_date >= CURRENT_DATE AND
    vd.expiry_date <= CURRENT_DATE + INTERVAL '60 days' AND
    vd.status NOT IN ('expired', 'superseded', 'rejected')
ORDER BY vd.expiry_date ASC;

COMMENT ON VIEW v_expiring_documents IS 'Documents expiring within 60 days - used for proactive renewal reminders';

-- View: Vehicles requiring maintenance
CREATE OR REPLACE VIEW v_pending_maintenance AS
SELECT
    mr.id,
    mr.vehicle_id,
    mr.driver_id,
    v.license_plate,
    v.make,
    v.model,
    mr.maintenance_type,
    mr.title,
    mr.priority,
    mr.status,
    mr.scheduled_date,
    (mr.scheduled_date - CURRENT_DATE)::INTEGER AS days_until_scheduled,
    mr.next_service_date,
    mr.service_provider_name
FROM maintenance_records mr
JOIN vehicles v ON v.id = mr.vehicle_id AND v.deleted_at IS NULL
WHERE
    mr.deleted_at IS NULL AND
    mr.status IN ('scheduled', 'in_progress')
ORDER BY mr.priority ASC, mr.scheduled_date ASC;

COMMENT ON VIEW v_pending_maintenance IS 'Active maintenance tasks ordered by priority and schedule';

-- =============================================================================
-- ROW LEVEL SECURITY (RLS)
-- =============================================================================

-- Enable RLS on sensitive tables
ALTER TABLE vehicles ENABLE ROW LEVEL SECURITY;
ALTER TABLE vehicle_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE maintenance_records ENABLE ROW LEVEL SECURITY;

-- Policy: Service role has full access (for backend services)
CREATE POLICY vehicles_service_full_access ON vehicles
    FOR ALL
    TO service_role
    USING (TRUE);

CREATE POLICY vehicle_documents_service_full_access ON vehicle_documents
    FOR ALL
    TO service_role
    USING (TRUE);

CREATE POLICY maintenance_service_full_access ON maintenance_records
    FOR ALL
    TO service_role
    USING (TRUE);

-- Policy: Drivers can only see their own vehicles
CREATE POLICY vehicles_driver_own ON vehicles
    FOR SELECT
    TO driver_role
    USING (driver_id = current_setting('app.current_driver_id', TRUE)::UUID);

CREATE POLICY vehicle_documents_driver_own ON vehicle_documents
    FOR SELECT
    TO driver_role
    USING (driver_id = current_setting('app.current_driver_id', TRUE)::UUID);

CREATE POLICY maintenance_driver_own ON maintenance_records
    FOR SELECT
    TO driver_role
    USING (driver_id = current_setting('app.current_driver_id', TRUE)::UUID);

-- =============================================================================
-- INITIAL SEED DATA
-- =============================================================================

-- Insert mandatory document types reference (for validation purposes)
CREATE TABLE vehicle_document_requirements (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_category    vehicle_category,           -- NULL means applies to all
    document_type       vehicle_document_type NOT NULL,
    is_mandatory        BOOLEAN NOT NULL DEFAULT TRUE,
    description         TEXT,
    legal_basis         VARCHAR(200),               -- Gesetzliche Grundlage
    renewal_interval_months INTEGER,                -- Null = no fixed renewal
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE vehicle_document_requirements IS 'Reference table defining mandatory documents per vehicle category and German law';

INSERT INTO vehicle_document_requirements
    (vehicle_category, document_type, is_mandatory, description, legal_basis, renewal_interval_months)
VALUES
    (NULL, 'fahrzeugschein', TRUE,
     'Zulassungsbescheinigung Teil I - must be carried in vehicle at all times',
     'Â§11 FZV', NULL),
    (NULL, 'fahrzeugbrief', TRUE,
     'Zulassungsbescheinigung Teil II - vehicle ownership document',
     'Â§11 FZV', NULL),
    (NULL, 'insurance_certificate', TRUE,
     'Haftpflichtversicherungsnachweis (eVB) - mandatory third-party liability insurance',
     'Â§1 PflVG, Â§29a StVZO', 12),
    (NULL, 'hauptuntersuchung', TRUE,
     'Hauptuntersuchung (HU) per Â§29 StVZO - mandatory technical inspection',
     'Â§29 StVZO, Anlage VIII', 24),
    (NULL, 'abgasuntersuchung', TRUE,
     'Abgasuntersuchung (AU) - emissions inspection, combined with HU since 2010',
     'Â§29 StVZO', 24),
    ('M1', 'tuev_report', FALSE,
     'TÃV/DEKRA inspection report - recommended to keep for service records',
     'Â§29 StVZO', NULL);

-- =============================================================================
-- GRANTS AND PERMISSIONS
-- =============================================================================

-- Note: Roles must exist in your PostgreSQL instance
-- These are examples - adjust to your actual role structure

-- GRANT USAGE ON SCHEMA public TO service_role;
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO service_role;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO service_role;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO service_role;

-- GRANT USAGE ON SCHEMA public TO driver_role;
-- GRANT SELECT ON vehicles, vehicle_documents, maintenance_records TO driver_role;
-- GRANT SELECT ON v_vehicle_compliance_summary, v_expiring_documents, v_pending_maintenance TO driver_role;

-- GRANT USAGE ON SCHEMA public TO readonly_role;
-- GRANT SELECT ON ALL TABLES IN SCHEMA public TO readonly_role;

-- =============================================================================
-- SCHEMA VERSION TRACKING
-- =============================================================================

CREATE TABLE IF NOT EXISTS schema_migrations (
    version         VARCHAR(50) PRIMARY KEY,
    description     TEXT,
    applied_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_by      VARCHAR(100) DEFAULT CURRENT_USER,
    checksum        VARCHAR(64)  -- SHA-256 of migration file
);

COMMENT ON TABLE schema_migrations IS 'Tracks applied database migrations for version control';

INSERT INTO schema_migrations (version, description) VALUES
    ('001', 'Initial schema: vehicles, vehicle_documents, maintenance_records, audit_logs - vehicle-management-service');

-- =============================================================================
-- END OF MIGRATION
-- =============================================================================

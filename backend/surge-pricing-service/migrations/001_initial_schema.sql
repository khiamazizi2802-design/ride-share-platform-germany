-- Migration: Initial schema for Surge Pricing & Dynamic Pricing Service
-- Service: surge-pricing-service
-- Platform: German ride-sharing platform
-- Compliance: PBefG (PersonenbefÃ¶rderungsgesetz)

-- ============================================================
-- Extensions
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis"; -- Geographic zone boundary support

-- ============================================================
-- Enums
-- ============================================================

CREATE TYPE vehicle_type AS ENUM (
    'economy',
    'comfort',
    'premium',
    'van',
    'electric',
    'accessible' -- Behindertengerechte Fahrzeuge (accessible vehicles)
);

CREATE TYPE pricing_rule_status AS ENUM (
    'draft',
    'active',
    'inactive',
    'archived'
);

CREATE TYPE zone_type AS ENUM (
    'city_center',       -- Innenstadtbereich
    'airport',           -- Flughafenbereich
    'train_station',     -- Bahnhofsbereich
    'suburban',          -- Vorortbereich
    'rural',             -- LÃ¤ndlicher Bereich
    'event_venue',       -- Veranstaltungsort
    'special_economic'   -- Sonderwirtschaftszone
);

CREATE TYPE surge_event_type AS ENUM (
    'surge_activated',
    'surge_deactivated',
    'surge_multiplier_updated',
    'zone_override_applied',
    'emergency_cap_applied'  -- Applied when PBefG pricing limits are hit
);

CREATE TYPE demand_level AS ENUM (
    'very_low',
    'low',
    'normal',
    'high',
    'very_high',
    'critical'
);

CREATE TYPE price_calculation_status AS ENUM (
    'estimated',
    'confirmed',
    'adjusted',   -- Post-ride adjustment applied
    'disputed',   -- Customer dispute raised
    'refunded'
);

-- ============================================================
-- Table: pricing_configs
-- Global configuration parameters for the pricing engine
-- ============================================================

CREATE TABLE pricing_configs (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    config_key                  VARCHAR(128) NOT NULL UNIQUE,
    config_value                JSONB NOT NULL,
    description                 TEXT,

    -- PBefG Compliance: German law mandates transparent pricing caps
    -- Ref: PBefG Â§39 - BefÃ¶rderungsentgelte (transport fares)
    is_pbefg_regulated          BOOLEAN NOT NULL DEFAULT FALSE,
    pbefg_reference             VARCHAR(64),  -- e.g. 'PBefG Â§39 Abs. 1'

    effective_from              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_until             TIMESTAMPTZ,

    created_by                  UUID NOT NULL,
    updated_by                  UUID,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pricing_configs_effective_range_check
        CHECK (effective_until IS NULL OR effective_until > effective_from)
);

COMMENT ON TABLE pricing_configs IS
    'Global configuration parameters for the dynamic pricing engine. '
    'PBefG-regulated entries define hard limits imposed by German transport law.';

COMMENT ON COLUMN pricing_configs.config_key IS
    'Unique identifier for this configuration parameter, e.g. ''global.max_surge_multiplier''';
COMMENT ON COLUMN pricing_configs.config_value IS
    'JSON value for flexible typing: numeric, boolean, object or array depending on parameter';
COMMENT ON COLUMN pricing_configs.is_pbefg_regulated IS
    'TRUE if this config is constrained by PersonenbefÃ¶rderungsgesetz regulations';
COMMENT ON COLUMN pricing_configs.pbefg_reference IS
    'Specific PBefG section governing this parameter, e.g. PBefG Â§39 Abs. 1';

CREATE INDEX idx_pricing_configs_key ON pricing_configs (config_key);
CREATE INDEX idx_pricing_configs_pbefg ON pricing_configs (is_pbefg_regulated) WHERE is_pbefg_regulated = TRUE;
CREATE INDEX idx_pricing_configs_effective ON pricing_configs (effective_from, effective_until);

-- ============================================================
-- Table: pricing_rules
-- Configurable pricing strategies per vehicle type and context
-- ============================================================

CREATE TABLE pricing_rules (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    name                        VARCHAR(256) NOT NULL,
    description                 TEXT,
    status                      pricing_rule_status NOT NULL DEFAULT 'draft',
    vehicle_type                vehicle_type NOT NULL,

    -- Base fare components (all in EUR, stored as NUMERIC for financial precision)
    base_fare_eur               NUMERIC(10, 4) NOT NULL,
    per_km_rate_eur             NUMERIC(10, 4) NOT NULL,
    per_minute_rate_eur         NUMERIC(10, 4) NOT NULL,
    minimum_fare_eur            NUMERIC(10, 4) NOT NULL,

    -- PBefG compliance: maximum permissible fare before regulatory cap triggers
    -- Ref: PBefG Â§39 - platforms must not exceed approved BefÃ¶rderungsentgelte
    pbefg_fare_cap_eur          NUMERIC(10, 4),
    pbefg_reference             VARCHAR(64),

    -- Time-based multipliers stored as JSONB for flexibility
    -- Structure: { "monday": [{"start": "07:00", "end": "09:00", "multiplier": 1.5}], ... }
    time_based_multipliers      JSONB NOT NULL DEFAULT '{}',

    -- Day-of-week applicability (bit mask: Mon=1, Tue=2, Wed=4 ... Sun=64)
    applicable_days_mask        SMALLINT NOT NULL DEFAULT 127, -- All days

    -- Surge configuration for this rule
    surge_enabled               BOOLEAN NOT NULL DEFAULT TRUE,
    max_surge_multiplier        NUMERIC(5, 2) NOT NULL DEFAULT 3.00,
    surge_step_size             NUMERIC(4, 2) NOT NULL DEFAULT 0.10, -- Increment per demand unit

    -- Cancellation fee (in EUR)
    cancellation_fee_eur        NUMERIC(10, 4) NOT NULL DEFAULT 0.00,

    -- Validity window
    valid_from                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until                 TIMESTAMPTZ,

    -- Audit
    created_by                  UUID NOT NULL,
    updated_by                  UUID,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pricing_rules_base_fare_positive
        CHECK (base_fare_eur >= 0),
    CONSTRAINT pricing_rules_per_km_positive
        CHECK (per_km_rate_eur >= 0),
    CONSTRAINT pricing_rules_per_min_positive
        CHECK (per_minute_rate_eur >= 0),
    CONSTRAINT pricing_rules_minimum_fare_positive
        CHECK (minimum_fare_eur >= 0),
    CONSTRAINT pricing_rules_min_fare_gte_base
        CHECK (minimum_fare_eur >= base_fare_eur),
    CONSTRAINT pricing_rules_max_surge_bounds
        CHECK (max_surge_multiplier BETWEEN 1.00 AND 10.00),
    CONSTRAINT pricing_rules_surge_step_positive
        CHECK (surge_step_size > 0),
    CONSTRAINT pricing_rules_valid_range
        CHECK (valid_until IS NULL OR valid_until > valid_from),
    CONSTRAINT pricing_rules_days_mask_bounds
        CHECK (applicable_days_mask BETWEEN 1 AND 127)
);

COMMENT ON TABLE pricing_rules IS
    'Configurable pricing strategies per vehicle type. Each rule defines base rates, '
    'per-km/per-minute charges, surge limits, and optional PBefG fare caps. '
    'Time-based multipliers allow peak/off-peak pricing within a single rule.';

COMMENT ON COLUMN pricing_rules.base_fare_eur IS
    'Fixed flag-fall charge in EUR applied at the start of every ride';
COMMENT ON COLUMN pricing_rules.per_km_rate_eur IS
    'Variable per-kilometre rate in EUR';
COMMENT ON COLUMN pricing_rules.per_minute_rate_eur IS
    'Variable per-minute rate in EUR applied to the total ride duration';
COMMENT ON COLUMN pricing_rules.minimum_fare_eur IS
    'Floor price in EUR; a ride can never be charged less than this amount';
COMMENT ON COLUMN pricing_rules.pbefg_fare_cap_eur IS
    'Maximum permissible fare ceiling enforced by PBefG Â§39; NULL if no statutory cap applies';
COMMENT ON COLUMN pricing_rules.time_based_multipliers IS
    'JSONB map of day-of-week to time-window multipliers for scheduled peak pricing';
COMMENT ON COLUMN pricing_rules.applicable_days_mask IS
    'Bitmask of active days: Mon=1, Tue=2, Wed=4, Thu=8, Fri=16, Sat=32, Sun=64; 127 = all days';
COMMENT ON COLUMN pricing_rules.max_surge_multiplier IS
    'Hard ceiling on surge; overrides zone-level multiplier if lower. Bounded 1.00â10.00.';
COMMENT ON COLUMN pricing_rules.surge_step_size IS
    'Granularity of each surge increment applied per unit increase in demand index';

CREATE INDEX idx_pricing_rules_vehicle_type ON pricing_rules (vehicle_type);
CREATE INDEX idx_pricing_rules_status ON pricing_rules (status);
CREATE INDEX idx_pricing_rules_valid_range ON pricing_rules (valid_from, valid_until);
CREATE INDEX idx_pricing_rules_vehicle_status ON pricing_rules (vehicle_type, status)
    WHERE status = 'active';
CREATE INDEX idx_pricing_rules_time_multipliers ON pricing_rules USING GIN (time_based_multipliers);

-- ============================================================
-- Table: surge_zones
-- Geographic zones with surge boundaries and multiplier caps
-- ============================================================

CREATE TABLE surge_zones (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    name                        VARCHAR(256) NOT NULL,
    description                 TEXT,
    zone_type                   zone_type NOT NULL,

    -- Geographic boundary using PostGIS polygon
    boundary                    GEOMETRY(POLYGON, 4326) NOT NULL,
    center_point                GEOMETRY(POINT, 4326),     -- Centroid for quick proximity checks

    -- Zone-level surge configuration
    base_multiplier             NUMERIC(5, 2) NOT NULL DEFAULT 1.00,
    max_multiplier              NUMERIC(5, 2) NOT NULL DEFAULT 2.50,
    current_multiplier          NUMERIC(5, 2) NOT NULL DEFAULT 1.00,

    -- Demand thresholds that trigger surge changes
    surge_threshold_low         NUMERIC(5, 2) NOT NULL DEFAULT 1.20,  -- Demand/supply ratio
    surge_threshold_medium      NUMERIC(5, 2) NOT NULL DEFAULT 1.50,
    surge_threshold_high        NUMERIC(5, 2) NOT NULL DEFAULT 2.00,
    surge_threshold_critical    NUMERIC(5, 2) NOT NULL DEFAULT 3.00,

    -- Cooldown to prevent rapid oscillation
    surge_cooldown_seconds      INT NOT NULL DEFAULT 300,  -- 5 minutes default

    -- Override: PBefG regulated zones may have regulatory multiplier ceilings
    pbefg_max_multiplier_cap    NUMERIC(5, 2),
    pbefg_reference             VARCHAR(64),

    is_active                   BOOLEAN NOT NULL DEFAULT TRUE,

    -- Priority: higher priority zones win in overlap scenarios
    priority                    INT NOT NULL DEFAULT 0,

    -- Audit
    created_by                  UUID NOT NULL,
    updated_by                  UUID,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT surge_zones_base_multiplier_bounds
        CHECK (base_multiplier BETWEEN 0.50 AND 5.00),
    CONSTRAINT surge_zones_max_multiplier_bounds
        CHECK (max_multiplier BETWEEN 1.00 AND 10.00),
    CONSTRAINT surge_zones_current_multiplier_bounds
        CHECK (current_multiplier BETWEEN 0.50 AND 10.00),
    CONSTRAINT surge_zones_max_gte_base
        CHECK (max_multiplier >= base_multiplier),
    CONSTRAINT surge_zones_current_lte_max
        CHECK (current_multiplier <= max_multiplier),
    CONSTRAINT surge_zones_cooldown_positive
        CHECK (surge_cooldown_seconds > 0),
    CONSTRAINT surge_zones_thresholds_ordered
        CHECK (
            surge_threshold_low < surge_threshold_medium
            AND surge_threshold_medium < surge_threshold_high
            AND surge_threshold_high < surge_threshold_critical
        ),
    CONSTRAINT surge_zones_pbefg_cap_bounds
        CHECK (pbefg_max_multiplier_cap IS NULL OR pbefg_max_multiplier_cap BETWEEN 1.00 AND 10.00)
);

COMMENT ON TABLE surge_zones IS
    'Geographic pricing zones defined by PostGIS polygon boundaries. '
    'Each zone maintains its own surge multiplier range and demand thresholds. '
    'Zones with pbefg_max_multiplier_cap enforce PBefG Â§39 regulatory ceilings.';

COMMENT ON COLUMN surge_zones.boundary IS
    'PostGIS POLYGON (SRID 4326 / WGS-84) defining the exact geographic extent of this zone';
COMMENT ON COLUMN surge_zones.center_point IS
    'Pre-computed centroid for fast proximity queries; should be kept in sync with boundary';
COMMENT ON COLUMN surge_zones.current_multiplier IS
    'Live surge multiplier currently applied to rides within this zone; updated by the pricing engine';
COMMENT ON COLUMN surge_zones.surge_cooldown_seconds IS
    'Minimum seconds between multiplier adjustments to prevent oscillation';
COMMENT ON COLUMN surge_zones.priority IS
    'When zones overlap, the zone with the highest priority value takes precedence';
COMMENT ON COLUMN surge_zones.pbefg_max_multiplier_cap IS
    'Regulatory ceiling imposed by PBefG; engine must not exceed this value regardless of demand';

CREATE INDEX idx_surge_zones_boundary ON surge_zones USING GIST (boundary);
CREATE INDEX idx_surge_zones_center_point ON surge_zones USING GIST (center_point);
CREATE INDEX idx_surge_zones_zone_type ON surge_zones (zone_type);
CREATE INDEX idx_surge_zones_active ON surge_zones (is_active) WHERE is_active = TRUE;
CREATE INDEX idx_surge_zones_priority ON surge_zones (priority DESC);
CREATE INDEX idx_surge_zones_current_multiplier ON surge_zones (current_multiplier);

-- ============================================================
-- Table: demand_metrics
-- Real-time demand/supply tracking per zone
-- ============================================================

CREATE TABLE demand_metrics (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    zone_id                     UUID NOT NULL REFERENCES surge_zones (id) ON DELETE CASCADE,

    -- Snapshot timestamp (bucketed to minute for aggregation)
    recorded_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    bucket_minute               TIMESTAMPTZ NOT NULL, -- Truncated to minute: DATE_TRUNC('minute', recorded_at)

    -- Supply metrics
    available_drivers           INT NOT NULL DEFAULT 0,
    active_drivers              INT NOT NULL DEFAULT 0,   -- Currently on a ride
    drivers_en_route            INT NOT NULL DEFAULT 0,   -- Heading to pickup

    -- Demand metrics
    open_ride_requests          INT NOT NULL DEFAULT 0,
    completed_rides_last_5min   INT NOT NULL DEFAULT 0,
    cancelled_rides_last_5min   INT NOT NULL DEFAULT 0,
    estimated_demand_next_15min INT NOT NULL DEFAULT 0,   -- ML forecast

    -- Computed ratios
    demand_supply_ratio         NUMERIC(8, 4),  -- open_requests / max(available_drivers, 1)
    demand_level                demand_level NOT NULL DEFAULT 'normal',

    -- Average wait time in seconds
    avg_wait_time_seconds       INT,
    p95_wait_time_seconds       INT,  -- 95th percentile

    -- Surge state at time of snapshot
    surge_multiplier_at_record  NUMERIC(5, 2) NOT NULL DEFAULT 1.00,
    surge_was_active            BOOLEAN NOT NULL DEFAULT FALSE,

    -- Vehicle-type breakdown stored as JSONB
    -- Structure: {"economy": {"available": 5, "requests": 8}, "premium": {...}}
    vehicle_breakdown           JSONB NOT NULL DEFAULT '{}',

    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT demand_metrics_available_drivers_non_negative
        CHECK (available_drivers >= 0),
    CONSTRAINT demand_metrics_active_drivers_non_negative
        CHECK (active_drivers >= 0),
    CONSTRAINT demand_metrics_en_route_non_negative
        CHECK (drivers_en_route >= 0),
    CONSTRAINT demand_metrics_open_requests_non_negative
        CHECK (open_ride_requests >= 0),
    CONSTRAINT demand_metrics_demand_supply_ratio_positive
        CHECK (demand_supply_ratio IS NULL OR demand_supply_ratio >= 0),
    CONSTRAINT demand_metrics_surge_multiplier_bounds
        CHECK (surge_multiplier_at_record BETWEEN 0.50 AND 10.00)
);

COMMENT ON TABLE demand_metrics IS
    'Per-minute demand and supply snapshots per surge zone. '
    'Powers the pricing engine decision loop and provides historical analytics. '
    'Rows are append-only; historical data should be archived to cold storage after 90 days.';

COMMENT ON COLUMN demand_metrics.bucket_minute IS
    'Minute-truncated timestamp for time-series bucketing and aggregation queries';
COMMENT ON COLUMN demand_metrics.demand_supply_ratio IS
    'Core signal: open_ride_requests / max(available_drivers, 1); drives surge multiplier selection';
COMMENT ON COLUMN demand_metrics.estimated_demand_next_15min IS
    'ML model forecast of ride requests in the next 15 minutes; used for predictive surge';
COMMENT ON COLUMN demand_metrics.vehicle_breakdown IS
    'Per-vehicle-type demand and supply snapshot for granular surge decisions';

CREATE INDEX idx_demand_metrics_zone_id ON demand_metrics (zone_id);
CREATE INDEX idx_demand_metrics_bucket_minute ON demand_metrics (bucket_minute DESC);
CREATE INDEX idx_demand_metrics_zone_bucket ON demand_metrics (zone_id, bucket_minute DESC);
CREATE INDEX idx_demand_metrics_demand_level ON demand_metrics (demand_level);
CREATE INDEX idx_demand_metrics_surge_active ON demand_metrics (surge_was_active, bucket_minute DESC)
    WHERE surge_was_active = TRUE;
CREATE INDEX idx_demand_metrics_high_ratio ON demand_metrics (demand_supply_ratio DESC)
    WHERE demand_supply_ratio IS NOT NULL;

-- ============================================================
-- Table: surge_events
-- Immutable log of surge activation/deactivation/update events
-- ============================================================

CREATE TABLE surge_events (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    zone_id                     UUID NOT NULL REFERENCES surge_zones (id) ON DELETE RESTRICT,
    event_type                  surge_event_type NOT NULL,

    -- Multiplier state transition
    previous_multiplier         NUMERIC(5, 2) NOT NULL,
    new_multiplier              NUMERIC(5, 2) NOT NULL,

    -- Demand context at the time of this event
    demand_metric_id            UUID REFERENCES demand_metrics (id) ON DELETE SET NULL,
    demand_supply_ratio_at_event NUMERIC(8, 4),
    demand_level_at_event       demand_level,

    -- Trigger source
    triggered_by                VARCHAR(64) NOT NULL DEFAULT 'pricing_engine', -- 'pricing_engine' | 'manual_override' | 'emergency_cap'
    triggered_by_user_id        UUID,    -- Populated for manual_override events
    trigger_reason              TEXT,    -- Human-readable explanation

    -- PBefG compliance: was a regulatory cap applied?
    pbefg_cap_applied           BOOLEAN NOT NULL DEFAULT FALSE,
    pbefg_cap_value             NUMERIC(5, 2),
    pbefg_reference             VARCHAR(64),

    -- Duration the event was in effect (populated on deactivation)
    duration_seconds            INT,

    event_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT surge_events_previous_multiplier_bounds
        CHECK (previous_multiplier BETWEEN 0.50 AND 10.00),
    CONSTRAINT surge_events_new_multiplier_bounds
        CHECK (new_multiplier BETWEEN 0.50 AND 10.00),
    CONSTRAINT surge_events_duration_positive
        CHECK (duration_seconds IS NULL OR duration_seconds >= 0),
    CONSTRAINT surge_events_manual_override_requires_user
        CHECK (
            triggered_by != 'manual_override'
            OR triggered_by_user_id IS NOT NULL
        )
);

COMMENT ON TABLE surge_events IS
    'Immutable audit log of all surge state transitions. '
    'Each row captures a before/after multiplier state with the demand context that caused it. '
    'Manual overrides must reference the user who performed the action for PBefG audit compliance.';

COMMENT ON COLUMN surge_events.triggered_by IS
    'Source of the surge change: ''pricing_engine'' (automated), ''manual_override'' (ops team), '
    'or ''emergency_cap'' (PBefG limit enforcement)';
COMMENT ON COLUMN surge_events.pbefg_cap_applied IS
    'TRUE when the new_multiplier was clamped to the PBefG regulatory ceiling';
COMMENT ON COLUMN surge_events.duration_seconds IS
    'Elapsed seconds this surge level was in effect; populated when a subsequent event closes it';

CREATE INDEX idx_surge_events_zone_id ON surge_events (zone_id);
CREATE INDEX idx_surge_events_event_type ON surge_events (event_type);
CREATE INDEX idx_surge_events_event_at ON surge_events (event_at DESC);
CREATE INDEX idx_surge_events_zone_event_at ON surge_events (zone_id, event_at DESC);
CREATE INDEX idx_surge_events_pbefg_caps ON surge_events (pbefg_cap_applied, event_at DESC)
    WHERE pbefg_cap_applied = TRUE;
CREATE INDEX idx_surge_events_manual_overrides ON surge_events (triggered_by_user_id, event_at DESC)
    WHERE triggered_by = 'manual_override';

-- ============================================================
-- Table: price_history
-- Full audit trail of every price calculation with breakdown
-- ============================================================

CREATE TABLE price_history (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Ride and user references (foreign keys managed externally by ride service)
    ride_id                     UUID NOT NULL,
    rider_id                    UUID NOT NULL,
    driver_id                   UUID,

    -- Applied rule and zone
    pricing_rule_id             UUID NOT NULL REFERENCES pricing_rules (id) ON DELETE RESTRICT,
    surge_zone_id               UUID REFERENCES surge_zones (id) ON DELETE SET NULL,

    vehicle_type                vehicle_type NOT NULL,
    calculation_status          price_calculation_status NOT NULL DEFAULT 'estimated',

    -- Ride dimensions
    distance_km                 NUMERIC(10, 4) NOT NULL,
    duration_minutes            NUMERIC(10, 4) NOT NULL,

    -- Price breakdown (all EUR, 4 decimal places for financial precision)
    base_fare_eur               NUMERIC(10, 4) NOT NULL,
    distance_charge_eur         NUMERIC(10, 4) NOT NULL,  -- per_km_rate * distance_km
    time_charge_eur             NUMERIC(10, 4) NOT NULL,  -- per_min_rate * duration_minutes
    subtotal_eur                NUMERIC(10, 4) NOT NULL,  -- base + distance + time

    -- Surge
    surge_multiplier_applied    NUMERIC(5, 2) NOT NULL DEFAULT 1.00,
    surge_amount_eur            NUMERIC(10, 4) NOT NULL DEFAULT 0.00,  -- subtotal * (multiplier - 1)

    -- Time-based multiplier applied (if any scheduled peak pricing)
    time_multiplier_applied     NUMERIC(5, 2) NOT NULL DEFAULT 1.00,
    time_multiplier_amount_eur  NUMERIC(10, 4) NOT NULL DEFAULT 0.00,

    -- Promotions / discounts
    discount_code               VARCHAR(64),
    discount_amount_eur         NUMERIC(10, 4) NOT NULL DEFAULT 0.00,

    -- Tolls and extras
    toll_charges_eur            NUMERIC(10, 4) NOT NULL DEFAULT 0.00,
    extras_eur                  NUMERIC(10, 4) NOT NULL DEFAULT 0.00,

    -- Minimum fare enforcement
    minimum_fare_applied        BOOLEAN NOT NULL DEFAULT FALSE,

    -- Final amounts
    total_before_tax_eur        NUMERIC(10, 4) NOT NULL,
    vat_rate_percent            NUMERIC(5, 2) NOT NULL DEFAULT 19.00,  -- German MwSt (Mehrwertsteuer)
    vat_amount_eur              NUMERIC(10, 4) NOT NULL,
    total_eur                   NUMERIC(10, 4) NOT NULL,

    -- PBefG compliance
    pbefg_cap_applied           BOOLEAN NOT NULL DEFAULT FALSE,
    pbefg_cap_value_eur         NUMERIC(10, 4),
    pbefg_reference             VARCHAR(64),

    -- Full calculation context for reproducibility / audits
    calculation_input_snapshot  JSONB NOT NULL DEFAULT '{}',
    calculation_breakdown       JSONB NOT NULL DEFAULT '{}',

    -- Pickup location for zone verification
    pickup_point                GEOMETRY(POINT, 4326),
    dropoff_point               GEOMETRY(POINT, 4326),

    -- Timestamps
    calculated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at                TIMESTAMPTZ,
    adjusted_at                 TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT price_history_distance_positive
        CHECK (distance_km > 0),
    CONSTRAINT price_history_duration_positive
        CHECK (duration_minutes > 0),
    CONSTRAINT price_history_base_fare_non_negative
        CHECK (base_fare_eur >= 0),
    CONSTRAINT price_history_distance_charge_non_negative
        CHECK (distance_charge_eur >= 0),
    CONSTRAINT price_history_time_charge_non_negative
        CHECK (time_charge_eur >= 0),
    CONSTRAINT price_history_surge_multiplier_bounds
        CHECK (surge_multiplier_applied BETWEEN 1.00 AND 10.00),
    CONSTRAINT price_history_surge_amount_non_negative
        CHECK (surge_amount_eur >= 0),
    CONSTRAINT price_history_discount_non_negative
        CHECK (discount_amount_eur >= 0),
    CONSTRAINT price_history_total_non_negative
        CHECK (total_eur >= 0),
    CONSTRAINT price_history_vat_rate_valid
        CHECK (vat_rate_percent BETWEEN 0 AND 100),
    CONSTRAINT price_history_vat_amount_non_negative
        CHECK (vat_amount_eur >= 0),
    CONSTRAINT price_history_total_consistency
        CHECK (total_eur = total_before_tax_eur + vat_amount_eur)
);

COMMENT ON TABLE price_history IS
    'Complete audit trail of every price calculation including the full cost breakdown. '
    'Records are append-only once confirmed; adjustments create a new row with status=adjusted. '
    'Retained permanently for PBefG Â§39 and German commercial law (HGB Â§257) compliance.';

COMMENT ON COLUMN price_history.subtotal_eur IS
    'Pre-surge, pre-discount sum: base_fare + distance_charge + time_charge';
COMMENT ON COLUMN price_history.surge_amount_eur IS
    'Monetary amount added purely due to surge: subtotal * (surge_multiplier - 1)';
COMMENT ON COLUMN price_history.vat_rate_percent IS
    'German Mehrwertsteuer (MwSt); standard rate 19%, reduced rate 7% for certain journeys';
COMMENT ON COLUMN price_history.calculation_input_snapshot IS
    'Full snapshot of inputs (demand_metrics, zone state, rule params) for exact audit reproducibility';
COMMENT ON COLUMN price_history.calculation_breakdown IS
    'Step-by-step calculation trace enabling customer-facing fare explanation (PBefG transparency)';
COMMENT ON COLUMN price_history.pbefg_cap_applied IS
    'TRUE if the final total was clamped to the PBefG Â§39 regulatory fare ceiling';

CREATE INDEX idx_price_history_ride_id ON price_history (ride_id);
CREATE INDEX idx_price_history_rider_id ON price_history (rider_id);
CREATE INDEX idx_price_history_driver_id ON price_history (driver_id) WHERE driver_id IS NOT NULL;
CREATE INDEX idx_price_history_pricing_rule_id ON price_history (pricing_rule_id);
CREATE INDEX idx_price_history_surge_zone_id ON price_history (surge_zone_id) WHERE surge_zone_id IS NOT NULL;
CREATE INDEX idx_price_history_vehicle_type ON price_history (vehicle_type);
CREATE INDEX idx_price_history_calculated_at ON price_history (calculated_at DESC);
CREATE INDEX idx_price_history_status ON price_history (calculation_status);
CREATE INDEX idx_price_history_surge_active ON price_history (surge_multiplier_applied, calculated_at DESC)
    WHERE surge_multiplier_applied > 1.00;
CREATE INDEX idx_price_history_pbefg_caps ON price_history (pbefg_cap_applied, calculated_at DESC)
    WHERE pbefg_cap_applied = TRUE;
CREATE INDEX idx_price_history_pickup_point ON price_history USING GIST (pickup_point)
    WHERE pickup_point IS NOT NULL;
CREATE INDEX idx_price_history_calculation_breakdown ON price_history USING GIN (calculation_breakdown);

-- ============================================================
-- Trigger function: auto-update updated_at
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

COMMENT ON FUNCTION set_updated_at IS
    'Generic trigger function to automatically refresh the updated_at column on every UPDATE';

CREATE TRIGGER trg_pricing_configs_updated_at
    BEFORE UPDATE ON pricing_configs
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_pricing_rules_updated_at
    BEFORE UPDATE ON pricing_rules
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_surge_zones_updated_at
    BEFORE UPDATE ON surge_zones
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_price_history_updated_at
    BEFORE UPDATE ON price_history
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================
-- Seed: PBefG-mandated global pricing config defaults
-- ============================================================

INSERT INTO pricing_configs (id, config_key, config_value, description, is_pbefg_regulated, pbefg_reference, created_by)
VALUES
(
    uuid_generate_v4(),
    'global.max_surge_multiplier',
    '3.00',
    'Absolute maximum surge multiplier permitted platform-wide before PBefG emergency cap triggers',
    TRUE,
    'PBefG Â§39 Abs. 1',
    '00000000-0000-0000-0000-000000000000'
),
(
    uuid_generate_v4(),
    'global.vat_rate_standard',
    '19.00',
    'Standard German Mehrwertsteuer (VAT) rate applied to ride fares',
    TRUE,
    'UStG Â§12 Abs. 1',
    '00000000-0000-0000-0000-000000000000'
),
(
    uuid_generate_v4(),
    'global.vat_rate_reduced',
    '7.00',
    'Reduced German MwSt rate applicable to certain public transport-equivalent journeys',
    TRUE,
    'UStG Â§12 Abs. 2',
    '00000000-0000-0000-0000-000000000000'
),
(
    uuid_generate_v4(),
    'global.surge_cooldown_default_seconds',
    '300',
    'Default cooldown period (in seconds) between surge multiplier adjustments to prevent oscillation',
    FALSE,
    NULL,
    '00000000-0000-0000-0000-000000000000'
),
(
    uuid_generate_v4(),
    'global.demand_metric_retention_days',
    '90',
    'Number of days demand_metrics rows are retained in hot storage before archival',
    FALSE,
    NULL,
    '00000000-0000-0000-0000-000000000000'
),
(
    uuid_generate_v4(),
    'global.price_history_retention_years',
    '10',
    'Retention period for price_history in years per HGB Â§257 (German commercial record-keeping law)',
    TRUE,
    'HGB Â§257 Abs. 4',
    '00000000-0000-0000-0000-000000000000'
);

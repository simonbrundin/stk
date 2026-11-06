-- Schema for repo-template

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id          BIGSERIAL PRIMARY KEY,
    email       VARCHAR(255) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    password    VARCHAR(255) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- The application must update this value on row updates.
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for email lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_log (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action      VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id   BIGINT,
    details     JSONB,
    ip_address  INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for audit log queries
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log(created_at DESC);

-- Koppling mellan resurser i kommunens bokningssystem och banor i MATCHi.
-- Varje rad kopplar en kommunresurs till en bana eller sal i MATCHi.
CREATE TABLE IF NOT EXISTS court_mappings (
    id                   BIGSERIAL PRIMARY KEY,
    kommun_resource_id   BIGINT NOT NULL,
    kommun_resource_name TEXT NOT NULL DEFAULT '',
    matchi_court_id      BIGINT NOT NULL,
    matchi_court_name    TEXT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (kommun_resource_id, matchi_court_id)
);

-- Bokningar som speglas mellan kommunens bokningssystem och MATCHi.
CREATE TABLE IF NOT EXISTS bookings (
    id                     BIGSERIAL PRIMARY KEY,
    kommun_appointment_id  BIGINT NOT NULL,
    kommun_resource_id     BIGINT NOT NULL,
    matchi_court_id        BIGINT NOT NULL,
    matchi_booking_id      TEXT NOT NULL DEFAULT '',
    matchi_slot_id         TEXT NOT NULL DEFAULT '',
    start_time             TIMESTAMPTZ NOT NULL,
    end_time               TIMESTAMPTZ NOT NULL,
    status                 TEXT NOT NULL DEFAULT 'synced',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (kommun_appointment_id, matchi_court_id)
);

CREATE INDEX IF NOT EXISTS idx_bookings_start_time ON bookings (start_time);
CREATE INDEX IF NOT EXISTS idx_bookings_matchi_slot_id ON bookings (matchi_slot_id);

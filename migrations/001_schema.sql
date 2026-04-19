CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    username      VARCHAR(100) UNIQUE NOT NULL,
    email         VARCHAR(255),
    password_hash VARCHAR(255) NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id         VARCHAR(64) PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS symptoms (
    id          SERIAL PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    source      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS symptom_tables (
    id         SERIAL PRIMARY KEY,
    symptom_id INTEGER NOT NULL REFERENCES symptoms(id) ON DELETE CASCADE,
    title      VARCHAR(255) NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS symptom_table_rows (
    id               SERIAL PRIMARY KEY,
    symptom_table_id INTEGER NOT NULL REFERENCES symptom_tables(id) ON DELETE CASCADE,
    medication       TEXT NOT NULL DEFAULT '',
    right_col        TEXT NOT NULL DEFAULT '',
    sort_order       INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS medications (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    source      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS medication_entries (
    id            SERIAL PRIMARY KEY,
    medication_id INTEGER NOT NULL REFERENCES medications(id) ON DELETE CASCADE,
    left_col      TEXT NOT NULL DEFAULT '',
    right_col     TEXT NOT NULL DEFAULT '',
    sort_order    INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS symptom_medications (
    symptom_id    INTEGER NOT NULL REFERENCES symptoms(id) ON DELETE CASCADE,
    medication_id INTEGER NOT NULL REFERENCES medications(id) ON DELETE CASCADE,
    PRIMARY KEY (symptom_id, medication_id)
);

CREATE INDEX IF NOT EXISTS idx_symptoms_title ON symptoms USING gin(to_tsvector('german', title));
CREATE INDEX IF NOT EXISTS idx_medications_name ON medications USING gin(to_tsvector('german', name));
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_symptom_tables_symptom ON symptom_tables(symptom_id);
CREATE INDEX IF NOT EXISTS idx_symptom_table_rows_table ON symptom_table_rows(symptom_table_id);

CREATE OR REPLACE FUNCTION cleanup_sessions() RETURNS void AS $$
    DELETE FROM sessions WHERE expires_at < NOW();
$$ LANGUAGE SQL;


-- Leitsymptome bekommen mehrere benannte Tabellen statt flacher Eintrags-Liste

CREATE TABLE IF NOT EXISTS symptom_tables (
    id          SERIAL PRIMARY KEY,
    symptom_id  INTEGER NOT NULL REFERENCES symptoms(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL DEFAULT '',   -- optionale Überschrift
    sort_order  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS symptom_table_rows (
    id               SERIAL PRIMARY KEY,
    symptom_table_id INTEGER NOT NULL REFERENCES symptom_tables(id) ON DELETE CASCADE,
    medication       TEXT NOT NULL DEFAULT '',   -- linke Spalte: Medikament
    right_col        TEXT NOT NULL DEFAULT '',   -- rechte Spalte: Dosierung / Info
    sort_order       INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_symptom_tables_symptom ON symptom_tables(symptom_id);
CREATE INDEX IF NOT EXISTS idx_symptom_table_rows_table ON symptom_table_rows(symptom_table_id);

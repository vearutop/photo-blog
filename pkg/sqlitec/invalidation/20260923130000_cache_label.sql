-- +goose Up
-- Re-creates cache_label for databases that skipped 20260506120000_init.sql:
-- invalidation and sqlitec migrations share goose version table, so on a fresh DB
-- the init migration is older than the latest sqlitec one and goose never applies it.
-- Keep invalidation migration versions newer than sqlitec ones.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS cache_label
(
    cache_name TEXT    NOT NULL,
    cache_key  TEXT    NOT NULL,
    label      TEXT    NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (cache_name, cache_key, label)
);

CREATE INDEX IF NOT EXISTS idx_cache_label_label ON cache_label (label);
CREATE INDEX IF NOT EXISTS idx_cache_label_cache_key ON cache_label (cache_name, cache_key);
-- +goose StatementEnd

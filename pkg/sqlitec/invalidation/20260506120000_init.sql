-- +goose Up
-- +goose StatementBegin
CREATE TABLE cache_label
(
    cache_name TEXT    NOT NULL,
    cache_key  TEXT    NOT NULL,
    label      TEXT    NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (cache_name, cache_key, label)
);

CREATE INDEX idx_cache_label_label ON cache_label (label);
CREATE INDEX idx_cache_label_cache_key ON cache_label (cache_name, cache_key);
-- +goose StatementEnd

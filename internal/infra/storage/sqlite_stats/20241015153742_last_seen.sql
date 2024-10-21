-- +goose Up
-- +goose StatementBegin
    ALTER TABLE visitor ADD COLUMN `last_seen` DATETIME NOT NULL DEFAULT '1970-01-01 00:00:00';
-- +goose StatementEnd

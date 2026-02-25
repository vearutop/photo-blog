-- +goose Up
-- +goose StatementBegin
ALTER TABLE album
    ADD COLUMN `updated_at` DATETIME NOT NULL DEFAULT '1970-01-01 00:00:00';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE album SET updated_at = created_at;
-- +goose StatementEnd

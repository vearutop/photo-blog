-- +goose Up
-- +goose StatementBegin
ALTER TABLE image ADD COLUMN `taken_at` DATETIME DEFAULT NULL;
-- +goose StatementEnd


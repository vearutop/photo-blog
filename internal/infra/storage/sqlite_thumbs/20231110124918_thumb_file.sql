-- +goose Up
-- +goose StatementBegin
ALTER TABLE thumb ADD COLUMN file_path VARCHAR(255) DEFAULT '';
-- +goose StatementEnd


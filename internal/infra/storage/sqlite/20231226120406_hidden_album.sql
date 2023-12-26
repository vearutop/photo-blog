-- +goose Up
-- +goose StatementBegin
ALTER TABLE album ADD COLUMN hidden INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd


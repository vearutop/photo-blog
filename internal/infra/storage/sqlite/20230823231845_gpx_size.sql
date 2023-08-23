-- +goose Up
-- +goose StatementBegin
ALTER TABLE gpx ADD COLUMN size int default 0;
-- +goose StatementEnd


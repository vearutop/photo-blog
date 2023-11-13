-- +goose Up
-- +goose StatementBegin
ALTER TABLE image
    ADD COLUMN settings TEXT default null;
-- +goose StatementEnd


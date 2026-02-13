-- +goose Up
-- +goose StatementBegin
ALTER TABLE image
    ADD COLUMN `is_hdr` INTEGER DEFAULT NULL;
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin
ALTER TABLE image ADD COLUMN `sharpness` integer DEFAULT NULL;
-- +goose StatementEnd

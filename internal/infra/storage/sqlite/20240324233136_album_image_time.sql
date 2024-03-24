-- +goose Up
-- +goose StatementBegin
ALTER TABLE album_image ADD COLUMN `timestamp` DATETIME DEFAULT NULL;
-- +goose StatementEnd

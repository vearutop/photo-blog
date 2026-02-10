-- +goose Up
-- +goose StatementBegin
ALTER TABLE album_image DROP COLUMN `timestamp`;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE album_image ADD COLUMN `utime` integer DEFAULT NULL;
-- +goose StatementEnd

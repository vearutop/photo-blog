-- +goose Up
-- +goose StatementBegin
ALTER TABLE image ADD COLUMN `utime` integer DEFAULT 0;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE image SET `utime` = UNIXEPOCH(COALESCE(`taken_at`, `created_at`));
-- +goose StatementEnd
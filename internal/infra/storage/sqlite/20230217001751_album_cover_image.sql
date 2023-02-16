-- +goose Up
-- +goose StatementBegin
ALTER TABLE `album` ADD COLUMN `cover_image` INTEGER not null DEFAULT 0;
-- +goose StatementEnd

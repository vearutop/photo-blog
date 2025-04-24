-- +goose Up
-- +goose StatementBegin
ALTER TABLE `thumb` ADD COLUMN `size` INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd


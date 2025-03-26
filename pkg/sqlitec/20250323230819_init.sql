-- +goose Up
-- +goose StatementBegin
CREATE TABLE record
(
    `key`        VARCHAR(255) NOT NULL PRIMARY KEY,
    `created_at` INTEGER      NOT NULL DEFAULT 0,
    `updated_at` INTEGER      NOT NULL DEFAULT 0,
    `expire_at`  INTEGER      NOT NULL DEFAULT 0,
    `val`        TEXT                  DEFAULT NULL
);
-- +goose StatementEnd

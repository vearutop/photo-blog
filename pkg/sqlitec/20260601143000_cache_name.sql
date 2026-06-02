-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS record;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE record
(
    `cache_name` VARCHAR(255) NOT NULL,
    `key`        VARCHAR(255) NOT NULL,
    `created_at` INTEGER      NOT NULL DEFAULT 0,
    `updated_at` INTEGER      NOT NULL DEFAULT 0,
    `expire_at`  INTEGER      NOT NULL DEFAULT 0,
    `val`        TEXT                  DEFAULT NULL,
    PRIMARY KEY (`cache_name`, `key`)
);
-- +goose StatementEnd

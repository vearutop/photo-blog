-- +goose Up
-- +goose StatementBegin
CREATE TABLE message
(
    `id`           INTEGER      NOT NULL PRIMARY KEY,
    `created_at`   INTEGER      NOT NULL DEFAULT 0,
    `try_after`    INTEGER      NOT NULL DEFAULT 0,
    `started_at`   INTEGER      NOT NULL DEFAULT 0,
    `processed_at` INTEGER      NOT NULL DEFAULT 0,
    `elapsed`      REAL         NOT NULL DEFAULT 0,
    `topic`        VARCHAR(255) NOT NULL DEFAULT '',
    `header`       TEXT                  DEFAULT NULL,
    `payload`      TEXT                  DEFAULT NULL,
    `error`        VARCHAR(255)          DEFAULT NULL,
    `tries`        INTEGER      NOT NULL DEFAULT 0,
    `on_success`   TEXT                  DEFAULT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE message_archive
(
    `id`           INTEGER      NOT NULL PRIMARY KEY,
    `created_at`   INTEGER      NOT NULL DEFAULT 0,
    `try_after`    INTEGER      NOT NULL DEFAULT 0,
    `started_at`   INTEGER      NOT NULL DEFAULT 0,
    `processed_at` INTEGER      NOT NULL DEFAULT 0,
    `elapsed`      REAL         NOT NULL DEFAULT 0,
    `topic`        VARCHAR(255) NOT NULL DEFAULT '',
    `header`       TEXT                  DEFAULT NULL,
    `payload`      TEXT                  DEFAULT NULL,
    `error`        VARCHAR(255)          DEFAULT NULL,
    `tries`        INTEGER      NOT NULL DEFAULT 0,
    `on_success`   TEXT                  DEFAULT NULL
);
-- +goose StatementEnd

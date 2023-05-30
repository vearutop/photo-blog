-- +goose Up
-- +goose StatementBegin
CREATE TABLE gpx
(
    `hash`       INTEGER      NOT NULL DEFAULT 0,
    `created_at` DATETIME     NOT NULL DEFAULT current_timestamp,
    `settings`   TEXT                  default null,
    `stats`      TEXT                  default null,
    `path`       VARCHAR(255) NOT NULL,
    PRIMARY KEY (`hash`)
);
-- +goose StatementEnd


-- +goose Up
-- +goose StatementBegin
CREATE TABLE `visitor`
(
    `hash`        INTEGER  NOT NULL PRIMARY KEY,
    `created_at`  DATETIME NOT NULL DEFAULT current_timestamp,
    `latest`      DATETIME NOT NULL DEFAULT current_timestamp,
    `user_agent`  varchar,
    `referrer`    varchar,
    `destination` varchar,
    `remote_addr` varchar,
    `hits`        integer
)
-- +goose StatementEnd


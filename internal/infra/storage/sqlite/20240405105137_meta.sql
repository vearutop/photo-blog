-- +goose Up
-- +goose StatementBegin
CREATE TABLE meta
(
    `hash`       INTEGER  NOT NULL PRIMARY KEY,
    `created_at` DATETIME NOT NULL DEFAULT current_timestamp,
    `data`       TEXT              default null
);
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin
CREATE TABLE names
(
    `hash` INTEGER      NOT NULL PRIMARY KEY,
    `name` VARCHAR(255) NOT NULL default ''
);
-- +goose StatementEnd

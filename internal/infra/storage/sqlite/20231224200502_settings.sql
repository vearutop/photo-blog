-- +goose Up
-- +goose StatementBegin
CREATE TABLE settings
(
    `name`  varchar(255) not null primary key,
    `value` TEXT default null
);
-- +goose StatementEnd

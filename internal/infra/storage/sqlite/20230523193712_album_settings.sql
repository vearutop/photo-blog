-- +goose Up
-- +goose StatementBegin
ALTER TABLE album ADD settings TEXT default null;
-- +goose StatementEnd

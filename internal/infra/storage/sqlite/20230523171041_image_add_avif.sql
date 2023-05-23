-- +goose Up
-- +goose StatementBegin
ALTER TABLE image ADD COLUMN has_avif INTEGER not null default 0;
-- +goose StatementEnd


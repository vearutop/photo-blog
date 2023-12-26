-- +goose Up
-- +goose StatementBegin
DROP TABLE app;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE image DROP COLUMN has_avif;
-- +goose StatementEnd

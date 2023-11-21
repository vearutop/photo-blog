-- +goose Up
-- +goose StatementBegin
ALTER TABLE image ADD column phash INTEGER DEFAULT 0;
-- +goose StatementEnd

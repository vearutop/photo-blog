-- +goose Up
-- +goose StatementBegin
ALTER TABLE message
    ADD COLUMN result TEXT DEFAULT NULL;

-- +goose StatementEnd


-- +goose StatementBegin
ALTER TABLE message_archive
    ADD COLUMN result TEXT DEFAULT NULL;

-- +goose StatementEnd

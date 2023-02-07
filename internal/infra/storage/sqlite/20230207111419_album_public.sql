-- +goose Up
-- +goose StatementBegin
ALTER TABLE albums ADD COLUMN public INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE albums DROP COLUMN public;
-- +goose StatementEnd

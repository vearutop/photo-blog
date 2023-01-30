


-- +goose Up

-- +goose StatementBegin
ALTER TABLE images ADD COLUMN `size` INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
ALTER TABLE images DROP COLUMN `size`;
-- +goose StatementEnd

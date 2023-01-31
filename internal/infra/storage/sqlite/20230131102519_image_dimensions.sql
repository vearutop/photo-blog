-- +goose Up

-- +goose StatementBegin
ALTER TABLE images
    ADD COLUMN `width` INTEGER NOT NULL DEFAULT 0;
ALTER TABLE images
    ADD COLUMN `height` INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
ALTER TABLE images
    DROP COLUMN `width`;
ALTER TABLE images
    DROP COLUMN `height`;
-- +goose StatementEnd

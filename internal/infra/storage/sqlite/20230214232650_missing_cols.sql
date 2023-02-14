-- +goose Up
-- +goose StatementBegin
ALTER TABLE image ADD COLUMN `size` INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE image
    ADD COLUMN `width` INTEGER NOT NULL DEFAULT 0;
ALTER TABLE image
    ADD COLUMN `height` INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE album ADD COLUMN public INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

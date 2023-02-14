-- +goose Up
-- +goose StatementBegin
INSERT INTO `image` (`hash`, `created_at`, `path`, `size`, `width`, `height`)
SELECT`hash`, `created_at`, `path`, `size`, `width`, `height` FROM images WHERE hash != 0
-- +goose StatementEnd


-- +goose Up
-- +goose StatementBegin
INSERT INTO `thumb` (`hash`, `created_at`, `width`, `height`, `data`)
SELECT `hash`, `created_at`, `width`, `height`, `data` FROM thumbs WHERE hash != 0
-- +goose StatementEnd



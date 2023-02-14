-- +goose Up
-- +goose StatementBegin
ALTER TABLE thumbs ADD COLUMN hash INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE thumbs SET hash = COALESCE((SELECT hash FROM images WHERE images.id = thumbs.id), 0);
-- +goose StatementEnd
-- +goose StatementBegin
DELETE FROM thumbs WHERE hash = 0;
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
ALTER TABLE thumbs DROP COLUMN hash;
-- +goose StatementEnd

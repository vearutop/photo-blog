-- +goose Up
-- +goose StatementBegin
ALTER TABLE exif ADD COLUMN projection_type VARCHAR NOT NULL DEFAULT "";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE exif DROP COLUMN projection_type;
-- +goose StatementEnd

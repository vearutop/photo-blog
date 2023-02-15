-- +goose Up
-- +goose StatementBegin
DROP TABLE albums;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE images;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE album_images;
-- +goose StatementEnd

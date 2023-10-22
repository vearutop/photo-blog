-- +goose Up
-- +goose StatementBegin
DROP TABLE label;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE user;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE visitor;
-- +goose StatementEnd

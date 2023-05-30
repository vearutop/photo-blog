-- +goose Up
-- +goose StatementBegin
CREATE TABLE app
(
    `settings` TEXT default null
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO `app` (settings) VALUES ('{}');
-- +goose StatementEnd

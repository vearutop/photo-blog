-- +goose Up
-- +goose StatementBegin
CREATE TABLE favorite_image
(
    `visitor_hash` INTEGER not null,
    `image_hash` integer not null,
    `created_at` DATETIME NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY (`visitor_hash`, `image_hash`)
);
-- +goose StatementEnd


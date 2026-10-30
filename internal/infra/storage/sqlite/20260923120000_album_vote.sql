-- +goose Up
-- +goose StatementBegin
CREATE TABLE album_vote
(
    `album_hash` INTEGER not null,
    `visitor_hash` INTEGER not null,
    `image_hash` INTEGER not null,
    `created_at` DATETIME NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY (`album_hash`, `visitor_hash`, `image_hash`)
);
-- +goose StatementEnd

-- +goose Up

-- +goose StatementBegin
CREATE TABLE albums
(
    `id`         INTEGER PRIMARY KEY,
    `created_at` DATETIME     NOT NULL DEFAULT current_timestamp,
    `title`      VARCHAR(255) NOT NULL,
    `name`       VARCHAR(255) NOT NULL UNIQUE
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE album_images
(
    `album_id` INTEGER,
    `image_id` integer,
    PRIMARY KEY (album_id, image_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE images
(
    `id`         INTEGER PRIMARY KEY,
    `created_at` DATETIME     NOT NULL DEFAULT current_timestamp,
    `hash`       integer      not null unique,
    `path`       VARCHAR(255) NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE thumbs
(
    `id`         INTEGER PRIMARY KEY,
    `image_id`   integer,
    `created_at` DATETIME NOT NULL DEFAULT current_timestamp,
    `width`      integer,
    `height`     integer,
    `data`       BLOB
);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE `albums`;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE `album_images`;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE `images`;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE `thumbs`;
-- +goose StatementEnd

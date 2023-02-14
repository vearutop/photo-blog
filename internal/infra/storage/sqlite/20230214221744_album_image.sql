-- +goose Up
-- +goose StatementBegin
CREATE TABLE album_image
(
    `album_hash` INTEGER not null,
    `image_hash` integer not null,
    `weight`     integer not null default 1,
    PRIMARY KEY (album_hash, album_hash)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE image
(
    `hash`       INTEGER      NOT NULL PRIMARY KEY,
    `created_at` DATETIME     NOT NULL DEFAULT current_timestamp,
    `path`       VARCHAR(255) NOT NULL,
    `blurhash`   varchar(255) not null DEFAULT ''
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE album
(
    `hash`       INTEGER      NOT NULL PRIMARY KEY,
    `created_at` DATETIME     NOT NULL DEFAULT current_timestamp,
    `title`      VARCHAR(255) NOT NULL,
    `name`       VARCHAR(255) NOT NULL UNIQUE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE album_image;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE album;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE image;
-- +goose StatementEnd
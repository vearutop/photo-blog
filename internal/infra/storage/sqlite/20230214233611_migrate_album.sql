-- +goose Up

-- +goose StatementBegin
DROP TABLE album_image;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE album_image
(
    `album_hash` INTEGER not null,
    `image_hash` integer not null,
    `weight`     integer not null default 1,
    PRIMARY KEY (album_hash, image_hash)
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO `album` (`hash`, `created_at`, `title`, `name`, `public`)
SELECT `hash`, `created_at`, `title`, `name`, `public` FROM albums WHERE hash != 0;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO `album_image` (album_hash, image_hash)
SELECT a.hash, i.hash
FROM album_images
JOIN albums a on album_images.album_id = a.id
JOIN images i on album_images.image_id = i.id
WHERE a.hash != 0 and i.hash != 0;
-- +goose StatementEnd

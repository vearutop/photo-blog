-- +goose Up
-- +goose StatementBegin
CREATE TABLE thumb
(
    `hash`       INTEGER  NOT NULL DEFAULT 0,
    `width`      integer NOT NULL DEFAULT 0,
    `height`     integer NOT NULL DEFAULT 0,

    `created_at` DATETIME NOT NULL DEFAULT current_timestamp,

    `data`       BLOB,
    PRIMARY KEY (`hash`, `width`, `height`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE thumb;
-- +goose StatementEnd

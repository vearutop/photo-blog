-- +goose Up
-- +goose StatementBegin
CREATE TABLE thumb
(
    `hash`       INTEGER  NOT NULL PRIMARY KEY,
    `created_at` DATETIME NOT NULL DEFAULT current_timestamp,

    `width`      integer,
    `height`     integer,
    `data`       BLOB
);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE thumb;
-- +goose StatementEnd

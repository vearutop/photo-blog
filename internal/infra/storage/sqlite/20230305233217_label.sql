-- +goose Up
-- +goose StatementBegin
CREATE TABLE label
(
    `hash`       INTEGER    NOT NULL,
    `created_at` DATETIME   NOT NULL DEFAULT current_timestamp,
    `locale`     varchar(5) NOT NULL default '',
    `text`       text       not null default '',
    PRIMARY KEY (`hash`, `locale`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE label;
-- +goose StatementEnd

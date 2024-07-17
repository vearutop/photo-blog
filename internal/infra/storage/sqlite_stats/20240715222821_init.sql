-- +goose Up
-- +goose StatementBegin
CREATE TABLE visitor
(
    `hash`       INTEGER  NOT NULL PRIMARY KEY,
    `created_at` DATETIME NOT NULL DEFAULT current_timestamp,
    `device`     VARCHAR(255)      DEFAULT '',
    `lang`       VARCHAR(255)      DEFAULT '',
    `ip_addr`    VARCHAR(255)      DEFAULT '',
    `user_agent` VARCHAR(500)      DEFAULT '',
    `is_bot`     integer  NOT NULL DEFAULT 0,
    `is_admin`   integer  NOT NULL DEFAULT 0,
    `referer`    VARCHAR(255)      DEFAULT '',
    `scr_h`      integer  not null default 0,
    `scr_w`      integer  not null default 0,
    `px_r`       real     not null default 0
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE page_visitors
(
    `visitor` integer NOT NULL,
    `page`    integer not null, -- album/image hash or 0 for main page
    PRIMARY KEY (`visitor`, `page`)
);
-- +goose StatementEnd

CREATE TABLE image_views
(
    `hash` INTEGER NOT NULL PRIMARY KEY,
    `time_ms` integer not null,
    `cnt` integer not null
);
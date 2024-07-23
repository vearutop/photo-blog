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
    `page`    integer not null, -- album hash or 0 for main page
    `date`    integer not null, -- visit date as truncated unix timestamp
    PRIMARY KEY (`page`, `date`, `visitor`)
);
-- +goose StatementEnd

CREATE TABLE daily_page_stats
(
    `hash`   integer not null, -- album hash or 0 for main page
    `date`   integer not null, -- visit date as truncated unix timestamp
    `views`  integer not null default 0,
    `refers` integer not null default 0,
    `uniq`   integer not null default 0,
    PRIMARY KEY (`hash`, `date`)
);

-- +goose StatementBegin
CREATE TABLE image_visitors
(
    `visitor` integer NOT NULL,
    `image`   integer not null, -- image hash
    PRIMARY KEY (`image`, `visitor`)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE image_stats
(
    `hash`         INTEGER NOT NULL PRIMARY KEY,
    `view_ms`      integer not null default 0,
    `thumb_ms`     integer not null default 0,
    `thumb_prt_ms` integer not null default 0, -- thumb on screen time for mobile in portrait mode
    `views`        integer not null default 0,
    `zooms`        integer not null default 0,
    `uniq`         integer not null default 0
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE page_stats
(
    `hash`   INTEGER NOT NULL PRIMARY KEY, -- album hash or 0 for main page
    `views`  integer not null default 0,
    `uniq`   integer not null default 0,
    `refers` integer not null default 0
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE refers
(
    `ts`      integer,
    `visitor` integer      NOT NULL,
    `referer` VARCHAR(255) NOT NULL default '',
    `url`     VARCHAR(255) NOT NULL default ''
);
-- +goose StatementEnd

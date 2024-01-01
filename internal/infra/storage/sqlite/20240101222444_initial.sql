-- +goose Up
-- +goose StatementBegin
CREATE TABLE exif
(
    `hash`              INTEGER  NOT NULL PRIMARY KEY,
    `created_at`        DATETIME NOT NULL DEFAULT current_timestamp,
    `rating`            integer  not null default 0,
    `exposure_time`     varchar  not null default '',
    `exposure_time_sec` float    not null default 0,
    `f_number`          float    not null default 0,
    `focal_length`      float    not null default 0,
    `iso_speed`         integer  not null default 0,
    `lens_model`        varchar  not null default '',
    `camera_make`       varchar  not null default '',
    `camera_model`      varchar  not null default '',
    `software`          varchar  not null default '',
    `digitized`         DATETIME,
    `projection_type`   VARCHAR  NOT NULL DEFAULT ''
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE gps
(
    `hash`       INTEGER  NOT NULL PRIMARY KEY,
    `created_at` DATETIME NOT NULL DEFAULT current_timestamp,
    `altitude`   float    not null default 0,
    `longitude`  float    not null default 0,
    `latitude`   float    not null default 0,
    `time`       DATETIME
);
-- +goose StatementEnd


-- +goose StatementBegin
CREATE TABLE image
(
    `hash`       INTEGER      NOT NULL PRIMARY KEY,
    `created_at` DATETIME     NOT NULL DEFAULT current_timestamp,
    `path`       VARCHAR(255) NOT NULL,
    `blurhash`   varchar(255) not null DEFAULT '',
    `size`       INTEGER      NOT NULL DEFAULT 0,
    `width`      INTEGER      NOT NULL DEFAULT 0,
    `height`     INTEGER      NOT NULL DEFAULT 0,
    `taken_at`   DATETIME              DEFAULT NULL,
    `settings`   TEXT                  default null,
    `phash`      INTEGER               DEFAULT 0
);
-- +goose StatementEnd


-- +goose StatementBegin
CREATE TABLE album
(
    `hash`        INTEGER      NOT NULL PRIMARY KEY,
    `created_at`  DATETIME     NOT NULL DEFAULT current_timestamp,
    `title`       VARCHAR(255) NOT NULL,
    `name`        VARCHAR(255) NOT NULL UNIQUE,
    `public`      INTEGER      NOT NULL DEFAULT 0,
    `cover_image` INTEGER      not null DEFAULT 0,
    `settings`    TEXT                  default null,
    `hidden`      INTEGER      NOT NULL DEFAULT 0
);
-- +goose StatementEnd


-- +goose StatementBegin
CREATE TABLE album_image
(
    `album_hash` INTEGER not null,
    `image_hash` integer not null,
    `weight`     integer not null default 1,
    PRIMARY KEY (`album_hash`, `image_hash`)
);
-- +goose StatementEnd


-- +goose StatementBegin
CREATE TABLE gpx
(
    `hash`       INTEGER      NOT NULL DEFAULT 0,
    `created_at` DATETIME     NOT NULL DEFAULT current_timestamp,
    `settings`   TEXT                  default null,
    `stats`      TEXT                  default null,
    `path`       VARCHAR(255) NOT NULL,
    `size`       int                   default 0,
    PRIMARY KEY (`hash`)
);
-- +goose StatementEnd


-- +goose StatementBegin
CREATE TABLE settings
(
    `name`  varchar(255) not null primary key,
    `value` TEXT default null
);
-- +goose StatementEnd

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
    `digitized`         DATETIME
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

-- +goose Down
-- +goose StatementBegin
DROP TABLE exif;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE gps;
-- +goose StatementEnd

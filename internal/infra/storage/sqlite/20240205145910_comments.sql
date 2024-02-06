-- +goose Up
-- +goose StatementBegin
CREATE TABLE thread
(
    `hash`         INTEGER     NOT NULL DEFAULT 0,  -- hash of (type, related_hash, related_at)
    `type`         VARCHAR(30) NOT NULL DEFAULT '', -- image, album, album/chrono
    `created_at`   DATETIME    NOT NULL DEFAULT current_timestamp,
    `related_hash` INTEGER     NOT NULL DEFAULT 0,
    `related_at`   DATETIME             DEFAULT NULL,
    PRIMARY KEY (`hash`)
);
-- +goose StatementEnd

-- +goose StatementBegin
create index thread_of_related_idx on thread (related_hash);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE message
(
    `hash`         INTEGER  NOT NULL DEFAULT 0, -- hash of (thread_hash, message)
    `thread_hash`  INTEGER  NOT NULL DEFAULT 0,
    `visitor_hash` INTEGER  NOT NULL DEFAULT 0,
    `created_at`   DATETIME NOT NULL DEFAULT current_timestamp,
    `approved`     INTEGER  NOT NULL DEFAULT 0,
    `text`         TEXT     NOT NULL DEFAULT '',
    PRIMARY KEY (`hash`)
);
-- +goose StatementEnd

-- +goose StatementBegin
create index messages_of_thread_idx on message (thread_hash);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE visitor
(
    `hash`       INTEGER  NOT NULL DEFAULT 0,
    `created_at` DATETIME NOT NULL DEFAULT current_timestamp,
    `name`       VARCHAR(255)      DEFAULT '',
    `approved`   INTEGER  NOT NULL DEFAULT 0,
    PRIMARY KEY (`hash`)
);
-- +goose StatementEnd
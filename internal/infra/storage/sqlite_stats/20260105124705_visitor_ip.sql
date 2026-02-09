-- +goose Up
-- +goose StatementBegin
ALTER TABLE `visitor`
    ADD COLUMN `ip` VARCHAR(255) not null DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE `visitor`
    ADD COLUMN `longitude` float not null default 0;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE `visitor`
    ADD COLUMN `latitude` float not null default 0;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE `visitor`
    ADD COLUMN `label` VARCHAR(255) not null default '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE `visitor`
    ADD COLUMN `city` VARCHAR(255) not null default '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE `visitor`
    ADD COLUMN `country` CHAR(2) not null default '';
-- +goose StatementEnd
-- +goose Up
-- Clears album caches, needed when their JSON shape changes incompatibly (add a new migration like this one).
-- Also removes entries of former versioned cache names.
-- +goose StatementBegin
DELETE FROM record WHERE cache_name IN ('album-data', 'album-page', 'album-data-2', 'album-page-2');
DELETE FROM cache_label WHERE cache_name IN ('album-data', 'album-page', 'album-data-2', 'album-page-2');
-- +goose StatementEnd

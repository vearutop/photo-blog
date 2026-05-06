package invalidation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/bool64/cache"
	"github.com/bool64/sqluct"
)

// Index is a SQLite-backed sibling of cache.InvalidationIndex.
type Index struct {
	st *sqluct.Storage

	mu       sync.RWMutex
	deleters map[string][]cache.Deleter
}

// NewIndex creates a persistent invalidation index.
func NewIndex(st *sqluct.Storage, deleters ...cache.Deleter) *Index {
	idx := &Index{
		st:       st,
		deleters: make(map[string][]cache.Deleter),
	}

	if len(deleters) > 0 {
		idx.deleters["default"] = append([]cache.Deleter(nil), deleters...)
	}

	return idx
}

// AddCache adds a named instance of cache with deletable entries.
func (i *Index) AddCache(name string, deleter cache.Deleter) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.deleters[name] = append(i.deleters[name], deleter)
}

// AddLabels registers invalidation labels to a cache key.
func (i *Index) AddLabels(cacheName string, key []byte, labels ...string) {
	if len(labels) == 0 {
		return
	}

	for _, label := range labels {
		_, err := i.st.DB().DB.Exec(
			`INSERT OR IGNORE INTO cache_label(cache_name, cache_key, label) VALUES (?, ?, ?)`,
			cacheName, string(key), label)
		if err != nil {
			println(fmt.Errorf("insert invalidation label: %w", err).Error())
			return
		}
	}
}

// ResetKey removes all invalidation labels attached to a cache key.
func (i *Index) ResetKey(ctx context.Context, cacheName string, key []byte) error {
	_, err := i.st.DB().DB.ExecContext(ctx,
		`DELETE FROM cache_label WHERE cache_name = ? AND cache_key = ?`,
		cacheName, string(key))
	if err != nil {
		return fmt.Errorf("reset invalidation labels: %w", err)
	}

	return nil
}

// InvalidateByLabels deletes keys from cache that have any of provided labels and returns number of deleted entries.
func (i *Index) InvalidateByLabels(ctx context.Context, labels ...string) (int, error) {
	if len(labels) == 0 {
		return 0, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(labels)), ",")
	args := make([]any, len(labels))
	for j, label := range labels {
		args[j] = label
	}

	rows, err := i.st.DB().DB.QueryContext(ctx,
		`SELECT DISTINCT cache_name, cache_key FROM cache_label WHERE label IN (`+placeholders+`)`,
		args...)
	if err != nil {
		return 0, fmt.Errorf("query invalidation labels: %w", err)
	}
	defer rows.Close()

	type entry struct {
		cacheName string
		cacheKey  string
	}

	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.cacheName, &e.cacheKey); err != nil {
			return 0, fmt.Errorf("scan invalidation label: %w", err)
		}

		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate invalidation labels: %w", err)
	}

	cnt := 0
	for _, e := range entries {
		for _, d := range i.cacheDeleters(e.cacheName) {
			err := d.Delete(ctx, []byte(e.cacheKey))
			if err != nil && !errors.Is(err, cache.ErrNotFound) {
				return cnt, err
			}

			if err == nil {
				cnt++
			}
		}

		if _, err := i.st.DB().DB.ExecContext(ctx,
			`DELETE FROM cache_label WHERE cache_name = ? AND cache_key = ?`,
			e.cacheName, e.cacheKey); err != nil {
			return cnt, fmt.Errorf("delete invalidated labels: %w", err)
		}
	}

	return cnt, nil
}

func (i *Index) cacheDeleters(cacheName string) []cache.Deleter {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return append([]cache.Deleter(nil), i.deleters[cacheName]...)
}

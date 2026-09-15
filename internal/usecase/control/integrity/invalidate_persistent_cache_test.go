package integrity

import (
	"context"
	"testing"

	"github.com/bool64/sqluct"
	"github.com/stretchr/testify/require"
)

func TestInvalidatePersistentCache(t *testing.T) {
	ctx := context.Background()
	deps := newCleanupTestDeps(t)

	insertJSONRecord(t, ctx, deps.st, "album-page", "page:yachting/false/false/en", `{}`)
	insertJSONRecord(t, ctx, deps.st, "album-data", "yachting/false/en/false", `{}`)
	insertJSONRecord(t, ctx, deps.st, "main-page", "mainfalseen", `{}`)
	insertJSONRecord(t, ctx, deps.st, "thumb-grid", "grid/yachting/6/2/-1/300/200", `[]`)
	insertJSONRecord(t, ctx, deps.st, "unrelated", "key", `{}`)

	insertCacheLabel(t, ctx, deps.st, "album-page", "page:yachting/false/false/en", "album/yachting")
	insertCacheLabel(t, ctx, deps.st, "album-data", "yachting/false/en/false", "album/yachting")
	insertCacheLabel(t, ctx, deps.st, "main-page", "mainfalseen", "album-list")
	insertCacheLabel(t, ctx, deps.st, "thumb-grid", "grid/yachting/6/2/-1/300/200", "album/yachting")
	insertCacheLabel(t, ctx, deps.st, "unrelated", "key", "album/yachting")

	report, err := invalidatePersistentCache(ctx, deps.st, "album-page")
	require.NoError(t, err)

	require.Equal(t, "album-page", report.CacheName)
	require.Equal(t, 1, report.DeletedRecordCount)
	require.Equal(t, 1, report.DeletedLabelCount)

	requireRecordMissing(t, ctx, deps.st, "album-page", "page:yachting/false/false/en")
	requireCacheLabelMissing(t, ctx, deps.st, "album-page", "page:yachting/false/false/en")

	requireRecordPresent(t, ctx, deps.st, "album-data", "yachting/false/en/false")
	requireRecordPresent(t, ctx, deps.st, "main-page", "mainfalseen")
	requireRecordPresent(t, ctx, deps.st, "thumb-grid", "grid/yachting/6/2/-1/300/200")
	requireRecordPresent(t, ctx, deps.st, "unrelated", "key")

	requireCacheLabelPresent(t, ctx, deps.st, "album-data", "yachting/false/en/false")
	requireCacheLabelPresent(t, ctx, deps.st, "main-page", "mainfalseen")
	requireCacheLabelPresent(t, ctx, deps.st, "thumb-grid", "grid/yachting/6/2/-1/300/200")
	requireCacheLabelPresent(t, ctx, deps.st, "unrelated", "key")
}

func insertJSONRecord(t *testing.T, ctx context.Context, st *sqluct.Storage, cacheName, key, val string) {
	t.Helper()

	_, err := st.DB().DB.ExecContext(ctx,
		`INSERT INTO record(cache_name, key, val, created_at, updated_at, expire_at) VALUES (?, ?, ?, unixepoch(), unixepoch(), 0)`,
		cacheName, key, val)
	require.NoError(t, err)
}

func insertCacheLabel(t *testing.T, ctx context.Context, st *sqluct.Storage, cacheName, cacheKey, label string) {
	t.Helper()

	_, err := st.DB().DB.ExecContext(ctx,
		`INSERT INTO cache_label(cache_name, cache_key, label) VALUES (?, ?, ?)`,
		cacheName, cacheKey, label)
	require.NoError(t, err)
}

func requireRecordMissing(t *testing.T, ctx context.Context, st *sqluct.Storage, cacheName, key string) {
	t.Helper()

	var count int
	err := st.DB().DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM record WHERE cache_name = ? AND key = ?`, cacheName, key).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count, cacheName+"/"+key)
}

func requireRecordPresent(t *testing.T, ctx context.Context, st *sqluct.Storage, cacheName, key string) {
	t.Helper()

	var count int
	err := st.DB().DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM record WHERE cache_name = ? AND key = ?`, cacheName, key).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, cacheName+"/"+key)
}

func requireCacheLabelMissing(t *testing.T, ctx context.Context, st *sqluct.Storage, cacheName, cacheKey string) {
	t.Helper()

	var count int
	err := st.DB().DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM cache_label WHERE cache_name = ? AND cache_key = ?`,
		cacheName, cacheKey).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count, cacheName+"/"+cacheKey)
}

func requireCacheLabelPresent(t *testing.T, ctx context.Context, st *sqluct.Storage, cacheName, cacheKey string) {
	t.Helper()

	var count int
	err := st.DB().DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM cache_label WHERE cache_name = ? AND cache_key = ?`,
		cacheName, cacheKey).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, cacheName+"/"+cacheKey)
}

package integrity

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/bool64/brick/database"
	"github.com/bool64/cache"
	"github.com/bool64/cache/blob"
	"github.com/bool64/cache/filecache"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/stats"
	"github.com/stretchr/testify/require"
	"github.com/vearutop/gooselite"
	"github.com/vearutop/gooselite/iofs"
	"github.com/vearutop/photo-blog/internal/infra/image/sprite"
	"github.com/vearutop/photo-blog/pkg/sqlitec"
	sqlitecinvalidation "github.com/vearutop/photo-blog/pkg/sqlitec/invalidation"
	_ "modernc.org/sqlite"
)

func TestCleanupAlbumSprites_ShouldKeepManifestReachableForPendingPageCache(t *testing.T) {
	ctx := context.Background()
	deps := newCleanupTestDeps(t)

	manifestKey := "album-sprite-manifest:stale-pending:v1"
	manifest := sprite.Manifest{
		Revision: "stale-pending",
		Version:  "v1",
		Images: map[string]sprite.ImageThumb{
			"img-1": {Chunk1x: "album-sprite:stale-1x", Chunk2x: "album-sprite:stale-2x", Width: 300, Height: 200, BackgroundWidth: 300, BackgroundHeight: 200},
		},
	}

	writeManifest(t, ctx, deps, manifestKey, manifest)
	writeBlob(t, ctx, deps, "album-sprite:stale-1x", 128)
	writeBlob(t, ctx, deps, "album-sprite:stale-2x", 256)

	// This page cache models the observed production race: the cache row survived in a pending state,
	// so it remembers the manifest key but still has no sprite sheets.
	writeAlbumPageCache(t, ctx, deps.PersistentCacheStorage(), "album-page:page:berlin/false/false/en", cachedAlbumPageData{
		AlbumData:         cachedAlbumOutput{},
		SpriteManifestKey: manifestKey,
		SpriteSheets:      nil,
	})

	report, err := cleanupAlbumSprites(ctx, deps, true)
	require.NoError(t, err)

	require.Equal(t, 1, report.AlbumPageCaches)
	require.Equal(t, 0, report.PageCachesWithSprites)
	require.Equal(t, 1, report.PageCachesWithoutSprites)

	// Expected behavior: a pending page cache that explicitly remembers a manifest key should still
	// protect that manifest and its chunks from cleanup until the page cache is refreshed.
	require.Equal(t, 1, report.ReferencedManifestCount)
	require.Contains(t, report.ReferencedManifestKeys, manifestKey)
	require.NotContains(t, report.WouldDeleteManifestKeys, manifestKey)
	require.NotContains(t, report.WouldDeleteBlobKeys, "album-sprite:stale-1x")
	require.NotContains(t, report.WouldDeleteBlobKeys, "album-sprite:stale-2x")
}

func TestCleanupAlbumSprites_ShouldDeleteBlobsOnlyWhenTheirManifestIsAlsoDeleted(t *testing.T) {
	ctx := context.Background()
	deps := newCleanupTestDeps(t)

	reachableManifestKey := "album-sprite-manifest:reachable:v1"
	coldManifestKey := "album-sprite-manifest:cold:v1"

	reachableManifest := sprite.Manifest{
		Revision: "reachable",
		Version:  "v1",
		Images: map[string]sprite.ImageThumb{
			"img-r": {Chunk1x: "album-sprite:reachable-1x", Chunk2x: "album-sprite:reachable-2x", Width: 300, Height: 200, BackgroundWidth: 300, BackgroundHeight: 200},
		},
	}

	coldManifest := sprite.Manifest{
		Revision: "cold",
		Version:  "v1",
		Images: map[string]sprite.ImageThumb{
			"img-c": {Chunk1x: "album-sprite:cold-1x", Chunk2x: "album-sprite:cold-2x", Width: 300, Height: 200, BackgroundWidth: 300, BackgroundHeight: 200},
		},
	}

	writeManifest(t, ctx, deps, reachableManifestKey, reachableManifest)
	writeManifest(t, ctx, deps, coldManifestKey, coldManifest)
	writeBlob(t, ctx, deps, "album-sprite:reachable-1x", 128)
	writeBlob(t, ctx, deps, "album-sprite:reachable-2x", 256)
	writeBlob(t, ctx, deps, "album-sprite:cold-1x", 512)
	writeBlob(t, ctx, deps, "album-sprite:cold-2x", 1024)

	// Only one page cache is rooted, so the second manifest is still a stored manifest but invisible to page-rooted cleanup.
	writeAlbumPageCache(t, ctx, deps.PersistentCacheStorage(), "album-page:page:people/false/false/en", cachedAlbumPageData{
		AlbumData: cachedAlbumOutput{
			Images: []cachedImage{
				{Hash: "a", Width: 1200, Height: 800},
			},
		},
		SpriteManifestKey: reachableManifestKey,
		SpriteSheets: map[string]sprite.Sheet{
			"s0": {Chunk1x: "album-sprite:reachable-1x", Chunk2x: "album-sprite:reachable-2x"},
		},
	})

	report, err := cleanupAlbumSprites(ctx, deps, true)
	require.NoError(t, err)

	require.Equal(t, 1, report.ReferencedManifestCount)
	require.Equal(t, []string{reachableManifestKey}, report.ReferencedManifestKeys)
	require.Contains(t, report.WouldDeleteManifestKeys, coldManifestKey)

	// Expected behavior: once a manifest is itself slated for deletion, its chunks should be
	// deletable in the same cleanup run.
	require.Contains(t, report.WouldDeleteBlobKeys, "album-sprite:cold-1x")
	require.Contains(t, report.WouldDeleteBlobKeys, "album-sprite:cold-2x")
	require.NotContains(t, report.WouldDeleteBlobKeys, "album-sprite:reachable-1x")
	require.NotContains(t, report.WouldDeleteBlobKeys, "album-sprite:reachable-2x")
}

type cleanupTestDeps struct {
	logger    ctxd.Logger
	st        *sqluct.Storage
	svc       *sprite.Service
	blobStore *filecache.Storage[string]
}

func (d cleanupTestDeps) CtxdLogger() ctxd.Logger {
	return d.logger
}

func (d cleanupTestDeps) PersistentCacheStorage() *sqluct.Storage {
	return d.st
}

func (d cleanupTestDeps) AlbumSprites() *sprite.Service {
	return d.svc
}

func newCleanupTestDeps(t *testing.T) cleanupTestDeps {
	t.Helper()

	st := testCleanupStorage(t)

	blobs, err := filecache.NewStorage[string](t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, blobs.Close())
	})

	svc := sprite.NewService(
		ctxd.NoOpLogger{},
		stats.NoOp{},
		nil,
		sqlitec.NewDBMapOf[sprite.Manifest](st, func(cfg *cache.ConfigOf[sprite.Manifest]) {
			cfg.TimeToLive = cache.UnlimitedTTL
		}),
		blobs,
	)

	return cleanupTestDeps{
		logger:    ctxd.NoOpLogger{},
		st:        st,
		svc:       svc,
		blobStore: blobs,
	}
}

func testCleanupStorage(t *testing.T) *sqluct.Storage {
	t.Helper()

	cfg := database.Config{
		DriverName:      "sqlite",
		DSN:             filepath.Join(t.TempDir(), "persistent-cache.sqlite") + "?_time_format=sqlite",
		ApplyMigrations: true,
		MaxOpen:         1,
		MaxIdle:         1,
	}

	st, err := database.SetupStorageDSN(cfg, ctxd.NoOpLogger{}, stats.NoOp{}, sqlitec.Migrations)
	require.NoError(t, err)
	require.NoError(t, applyTestStorageMigrations(st, sqlitecinvalidation.Migrations))

	t.Cleanup(func() {
		require.NoError(t, st.DB().DB.Close())
	})

	return st
}

func applyTestStorageMigrations(st *sqluct.Storage, migrations fs.FS) error {
	if migrations == nil {
		return nil
	}

	gooselite.SetLogger(noOpGooseLogger{})

	if err := gooselite.SetDialect("sqlite3"); err != nil {
		return err
	}

	return iofs.Up(st.DB().DB, migrations, ".")
}

type noOpGooseLogger struct{}

func (noOpGooseLogger) Fatal(v ...any)                 { panic(v) }
func (noOpGooseLogger) Fatalf(format string, v ...any) { panic(v) }
func (noOpGooseLogger) Print(v ...any)                 {}
func (noOpGooseLogger) Println(v ...any)               {}
func (noOpGooseLogger) Printf(string, ...any)          {}

func writeManifest(t *testing.T, ctx context.Context, deps cleanupTestDeps, key string, manifest sprite.Manifest) {
	t.Helper()

	require.NoError(t, deps.AlbumSprites().DeleteManifestRecord(ctx, key))
	require.NoError(t, sqlitec.NewDBMapOf[sprite.Manifest](deps.st).Write(ctx, []byte(key), manifest))
}

func writeBlob(t *testing.T, ctx context.Context, deps cleanupTestDeps, key string, size int) {
	t.Helper()

	payload := bytes.Repeat([]byte{'x'}, size)
	entry := blob.FromReader(bytes.NewReader(payload), blob.Meta{
		Name:    key + ".jpg",
		Size:    int64(len(payload)),
		ModTime: time.Now(),
	})

	require.NoError(t, deps.AlbumSprites().DeleteBlob(ctx, key))
	err := deps.blobStore.Write(ctx, key, entry)
	require.NoError(t, err)
}

func writeAlbumPageCache(t *testing.T, ctx context.Context, st *sqluct.Storage, key string, page cachedAlbumPageData) {
	t.Helper()

	j, err := json.Marshal(page)
	require.NoError(t, err)

	_, err = st.DB().DB.ExecContext(ctx,
		`INSERT INTO record(key, val, created_at) VALUES (?, ?, CURRENT_TIMESTAMP)`,
		key, string(j))
	require.NoError(t, err)
}

func TestAllManifestChunkRefs_SortsManifestKeys(t *testing.T) {
	refs := allManifestChunkRefs(map[string]storedManifestRecord{
		"m2": {Manifest: sprite.Manifest{Images: map[string]sprite.ImageThumb{"b": {Chunk1x: "c1"}}}},
		"m1": {Manifest: sprite.Manifest{Images: map[string]sprite.ImageThumb{"a": {Chunk1x: "c1"}}}},
	})

	require.True(t, slices.Equal([]string{"m1", "m2"}, refs["c1"]))
}

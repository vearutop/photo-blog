package integrity_test

import (
	"context"
	"encoding/json"
	"image"
	_ "image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"testing"
	"time"
	"unsafe"

	"github.com/bool64/cache"
	"github.com/bool64/brick"
	"github.com/stretchr/testify/require"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra"
	apphttp "github.com/vearutop/photo-blog/internal/infra/nethttp"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/image/sprite"
	integrity "github.com/vearutop/photo-blog/internal/usecase/control/integrity"
)

func TestCleanupAlbumSprites_EndToEnd(t *testing.T) {
	ctx := context.Background()
	loc := newTestLocator(t)
	router := apphttp.NewRouter(loc)

	// Create one album with three ready images that will produce an initial sprite manifest.
	album := createAlbum(t, ctx, loc, "sprite-e2e")
	images := []photo.Image{
		ensureImage(t, ctx, loc, "internal/infra/image/testdata/20240919_144111.jpg", time.Date(2024, time.September, 19, 14, 41, 11, 0, time.UTC)),
		ensureImage(t, ctx, loc, "internal/infra/image/testdata/20240919_144116.jpg", time.Date(2024, time.September, 19, 14, 41, 16, 0, time.UTC)),
		ensureImage(t, ctx, loc, "internal/infra/image/testdata/20240919_144120.jpg", time.Date(2024, time.September, 19, 14, 41, 20, 0, time.UTC)),
	}

	require.NoError(t, loc.PhotoAlbumImageAdder().AddImages(ctx, album.Hash, images[0].Hash, images[1].Hash, images[2].Hash))

	// First request kicks off async sprite build, second request uses the warmed sprite-backed page cache.
	getAlbumPage(t, router, album.Name)
	oldManifestKey, oldManifest := waitForAlbumManifest(t, ctx, loc, album.Name)
	getAlbumPage(t, router, album.Name)

	// With only one live page cache and one live manifest, cleanup should find nothing to delete.
	initial := cleanupAlbumSprites(t, router, true)
	require.Equal(t, 1, initial.AlbumPageCaches)
	require.Equal(t, 1, initial.PageCachesWithSprites)
	require.Equal(t, 0, initial.BrokenPageCacheCount)
	require.Equal(t, []string{oldManifestKey}, initial.ReferencedManifestKeys)
	require.Zero(t, initial.WouldDeleteManifestCount)
	require.Zero(t, initial.WouldDeleteBlobCount)

	// Remove one image so the album needs a different manifest on the next page render.
	removeImageFromAlbum(t, router, album.Name, images[2].Hash)
	getAlbumPage(t, router, album.Name)
	newManifestKey, newManifest := waitForAlbumManifest(t, ctx, loc, album.Name)
	require.NotEqual(t, oldManifestKey, newManifestKey)
	getAlbumPage(t, router, album.Name)

	// After the change, the old manifest and its chunks should become collectible.
	afterChange := cleanupAlbumSprites(t, router, true)
	require.Equal(t, 1, afterChange.PageCachesWithSprites)
	require.Equal(t, []string{newManifestKey}, afterChange.ReferencedManifestKeys)
	require.Contains(t, afterChange.WouldDeleteManifestKeys, oldManifestKey)

	oldChunks := sortedChunks(oldManifest)
	require.NotEmpty(t, oldChunks)
	require.ElementsMatch(t, oldChunks, afterChange.WouldDeleteBlobKeys)
	require.Equal(t, len(oldChunks), afterChange.WouldDeleteBlobCount)

	// Real cleanup should remove only the old assets and keep the new manifest alive.
	realRun := cleanupAlbumSprites(t, router, false)
	require.Contains(t, realRun.DeletedManifestKeys, oldManifestKey)
	require.ElementsMatch(t, oldChunks, realRun.DeletedBlobKeys)

	// A final dry run confirms the system has converged and only the new manifest remains reachable.
	final := cleanupAlbumSprites(t, router, true)
	require.Equal(t, []string{newManifestKey}, final.ReferencedManifestKeys)
	require.Zero(t, final.WouldDeleteManifestCount)
	require.Zero(t, final.WouldDeleteBlobCount)

	_, err := loc.AlbumSprites().ManifestReady(ctx, oldManifestKey)
	require.Error(t, err)
	require.ErrorIs(t, err, cache.ErrNotFound)

	for _, key := range oldChunks {
		_, err := loc.AlbumSprites().Open(ctx, key)
		require.Error(t, err)
		require.ErrorIs(t, err, cache.ErrNotFound)
	}

	for _, key := range sortedChunks(newManifest) {
		_, err := loc.AlbumSprites().Open(ctx, key)
		require.NoError(t, err)
	}
}

func TestCleanupAlbumSprites_SharedManifestAcrossAlbums(t *testing.T) {
	ctx := context.Background()
	loc := newTestLocator(t)
	router := apphttp.NewRouter(loc)

	// Prepare one shared image set that should produce the same manifest for two different albums.
	images := []photo.Image{
		ensureImage(t, ctx, loc, "internal/infra/image/testdata/20240919_144111.jpg", time.Date(2024, time.September, 19, 14, 41, 11, 0, time.UTC)),
		ensureImage(t, ctx, loc, "internal/infra/image/testdata/20240919_144116.jpg", time.Date(2024, time.September, 19, 14, 41, 16, 0, time.UTC)),
		ensureImage(t, ctx, loc, "internal/infra/image/testdata/20240919_144120.jpg", time.Date(2024, time.September, 19, 14, 41, 20, 0, time.UTC)),
	}

	albumA := createAlbum(t, ctx, loc, "sprite-shared-a")
	albumB := createAlbum(t, ctx, loc, "sprite-shared-b")

	require.NoError(t, loc.PhotoAlbumImageAdder().AddImages(ctx, albumA.Hash, images[0].Hash, images[1].Hash, images[2].Hash))
	require.NoError(t, loc.PhotoAlbumImageAdder().AddImages(ctx, albumB.Hash, images[0].Hash, images[1].Hash, images[2].Hash))

	// Warm both albums until their page caches use the same shared manifest.
	getAlbumPage(t, router, albumA.Name)
	getAlbumPage(t, router, albumB.Name)

	sharedManifestKeyA, sharedManifest := waitForAlbumManifest(t, ctx, loc, albumA.Name)
	sharedManifestKeyB, _ := waitForAlbumManifest(t, ctx, loc, albumB.Name)
	require.Equal(t, sharedManifestKeyA, sharedManifestKeyB)
	getAlbumPage(t, router, albumA.Name)
	getAlbumPage(t, router, albumB.Name)

	// While both albums still point at the shared manifest, there should be nothing collectible.
	sharedBefore := cleanupAlbumSprites(t, router, true)
	require.Equal(t, []string{sharedManifestKeyA}, sharedBefore.ReferencedManifestKeys)
	require.Zero(t, sharedBefore.WouldDeleteManifestCount)
	require.Zero(t, sharedBefore.WouldDeleteBlobCount)

	// Change only album A so it moves to a new manifest, while album B still keeps the old one alive.
	removeImageFromAlbum(t, router, albumA.Name, images[2].Hash)
	getAlbumPage(t, router, albumA.Name)

	albumANewManifestKey, albumANewManifest := waitForAlbumManifest(t, ctx, loc, albumA.Name)
	require.NotEqual(t, sharedManifestKeyA, albumANewManifestKey)
	getAlbumPage(t, router, albumA.Name)

	afterFirstChange := cleanupAlbumSprites(t, router, true)
	require.ElementsMatch(t, []string{sharedManifestKeyA, albumANewManifestKey}, afterFirstChange.ReferencedManifestKeys)
	require.Zero(t, afterFirstChange.WouldDeleteManifestCount)
	require.Zero(t, afterFirstChange.WouldDeleteBlobCount)

	// A real cleanup at this point must not delete anything because album B still references the shared manifest.
	realAfterFirstChange := cleanupAlbumSprites(t, router, false)
	require.Zero(t, realAfterFirstChange.DeletedManifestCount)
	require.Zero(t, realAfterFirstChange.DeletedBlobCount)

	_, err := loc.AlbumSprites().ManifestReady(ctx, sharedManifestKeyA)
	require.NoError(t, err)
	for _, key := range sortedChunks(sharedManifest) {
		_, err := loc.AlbumSprites().Open(ctx, key)
		require.NoError(t, err)
	}

	// Once album B makes the same change, both albums converge on the new manifest and the old shared assets become garbage.
	removeImageFromAlbum(t, router, albumB.Name, images[2].Hash)
	getAlbumPage(t, router, albumB.Name)

	albumBNewManifestKey, _ := waitForAlbumManifest(t, ctx, loc, albumB.Name)
	require.Equal(t, albumANewManifestKey, albumBNewManifestKey)
	getAlbumPage(t, router, albumB.Name)

	afterSecondChange := cleanupAlbumSprites(t, router, true)
	require.Equal(t, []string{albumANewManifestKey}, afterSecondChange.ReferencedManifestKeys)
	require.Equal(t, []string{sharedManifestKeyA}, afterSecondChange.WouldDeleteManifestKeys)

	sharedChunks := sortedChunks(sharedManifest)
	require.NotEmpty(t, sharedChunks)
	require.ElementsMatch(t, sharedChunks, afterSecondChange.WouldDeleteBlobKeys)

	// Final cleanup should now retire the shared manifest and its chunks, while keeping the replacement manifest intact.
	finalRun := cleanupAlbumSprites(t, router, false)
	require.Equal(t, []string{sharedManifestKeyA}, finalRun.DeletedManifestKeys)
	require.ElementsMatch(t, sharedChunks, finalRun.DeletedBlobKeys)

	final := cleanupAlbumSprites(t, router, true)
	require.Equal(t, []string{albumANewManifestKey}, final.ReferencedManifestKeys)
	require.Zero(t, final.WouldDeleteManifestCount)
	require.Zero(t, final.WouldDeleteBlobCount)

	_, err = loc.AlbumSprites().ManifestReady(ctx, sharedManifestKeyA)
	require.Error(t, err)
	require.ErrorIs(t, err, cache.ErrNotFound)

	for _, key := range sharedChunks {
		_, err := loc.AlbumSprites().Open(ctx, key)
		require.Error(t, err)
		require.ErrorIs(t, err, cache.ErrNotFound)
	}

	for _, key := range sortedChunks(albumANewManifest) {
		_, err := loc.AlbumSprites().Open(ctx, key)
		require.NoError(t, err)
	}
}

func TestCleanupAlbumSprites_AutoRetiresAfterAlbumChange(t *testing.T) {
	ctx := context.Background()
	loc := newTestLocator(t)
	setSpriteRetirementDelay(t, loc, 20*time.Millisecond)

	router := apphttp.NewRouter(loc)

	// Create one album with an initial manifest that will become ownerless after a content change.
	album := createAlbum(t, ctx, loc, "sprite-auto-retire")
	images := []photo.Image{
		ensureImage(t, ctx, loc, "internal/infra/image/testdata/20240919_144111.jpg", time.Date(2024, time.September, 19, 14, 41, 11, 0, time.UTC)),
		ensureImage(t, ctx, loc, "internal/infra/image/testdata/20240919_144116.jpg", time.Date(2024, time.September, 19, 14, 41, 16, 0, time.UTC)),
		ensureImage(t, ctx, loc, "internal/infra/image/testdata/20240919_144120.jpg", time.Date(2024, time.September, 19, 14, 41, 20, 0, time.UTC)),
	}
	require.NoError(t, loc.PhotoAlbumImageAdder().AddImages(ctx, album.Hash, images[0].Hash, images[1].Hash, images[2].Hash))

	// Warm the initial sprite-backed page cache and capture the original manifest and chunk set.
	getAlbumPage(t, router, album.Name)
	oldManifestKey, oldManifest := waitForAlbumManifest(t, ctx, loc, album.Name)
	getAlbumPage(t, router, album.Name)

	// Mutating the album should retire the old manifest owner and move the next render to a new manifest.
	removeImageFromAlbum(t, router, album.Name, images[2].Hash)
	getAlbumPage(t, router, album.Name)
	newManifestKey, newManifest := waitForAlbumManifest(t, ctx, loc, album.Name)
	require.NotEqual(t, oldManifestKey, newManifestKey)
	getAlbumPage(t, router, album.Name)

	// Without calling cleanup, the old manifest and chunks should disappear after the short retirement delay.
	waitForManifestDeleted(t, ctx, loc, oldManifestKey)
	for _, key := range sortedChunks(oldManifest) {
		waitForBlobDeleted(t, ctx, loc, key)
	}

	// The replacement manifest and its chunks must stay intact after automatic retirement finishes.
	_, err := loc.AlbumSprites().ManifestReady(ctx, newManifestKey)
	require.NoError(t, err)
	for _, key := range sortedChunks(newManifest) {
		_, err := loc.AlbumSprites().Open(ctx, key)
		require.NoError(t, err)
	}

	// Dry-run cleanup should now be a no-op because the automatic retirement already converged.
	report := cleanupAlbumSprites(t, router, true)
	require.Equal(t, []string{newManifestKey}, report.ReferencedManifestKeys)
	require.Zero(t, report.WouldDeleteManifestCount)
	require.Zero(t, report.WouldDeleteBlobCount)
}

func newTestLocator(t *testing.T) *service.Locator {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err)

	storagePath := t.TempDir()
	loc, err := infra.NewServiceLocator(service.Config{
		BaseConfig: brick.BaseConfig{
			Initialized:     true,
			ServiceName:     "photo-blog-test",
			ShutdownTimeout: time.Second,
		},
		StoragePath: storagePath,
	}, false)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, os.Chdir(wd))
	})

	return loc
}

func createAlbum(t *testing.T, ctx context.Context, loc *service.Locator, name string) photo.Album {
	t.Helper()

	album, err := loc.PhotoAlbumEnsurer().Ensure(ctx, photo.Album{
		Head: uniq.Head{Hash: photo.AlbumHash(name)},
		Name: name,
		Title: name,
		Public: true,
	})
	require.NoError(t, err)

	return album
}

func ensureImage(t *testing.T, ctx context.Context, loc *service.Locator, relPath string, takenAt time.Time) photo.Image {
	t.Helper()

	srcPath := repoFilePath(t, relPath)

	cfg, err := decodeConfig(srcPath)
	require.NoError(t, err)

	var img photo.Image
	require.NoError(t, img.SetPath(ctx, srcPath))

	img.Width = int64(cfg.Width)
	img.Height = int64(cfg.Height)
	img.BlurHash = "test-blurhash"
	img.TakenAt = &takenAt
	img.RefreshUTime()

	img, err = loc.PhotoImageEnsurer().Ensure(ctx, img)
	require.NoError(t, err)

	_, err = loc.PhotoThumbnailer().Thumbnail(ctx, img, "300w")
	require.NoError(t, err)
	_, err = loc.PhotoThumbnailer().Thumbnail(ctx, img, "600w")
	require.NoError(t, err)

	return img
}

func decodeConfig(path string) (image.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return image.Config{}, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)

	return cfg, err
}

func repoFilePath(t *testing.T, relPath string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)

	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", relPath)
}

func waitForAlbumManifest(t *testing.T, ctx context.Context, loc *service.Locator, albumName string) (string, sprite.Manifest) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		key := currentAlbumManifestKey(t, ctx, loc, albumName)
		manifest, err := loc.AlbumSprites().ManifestReady(ctx, key)
		if err == nil {
			return key, manifest
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for sprite manifest for album %s", albumName)
	return "", sprite.Manifest{}
}

func waitForManifestDeleted(t *testing.T, ctx context.Context, loc *service.Locator, manifestKey string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, err := loc.AlbumSprites().ManifestReady(ctx, manifestKey)
		if err != nil {
			require.ErrorIs(t, err, cache.ErrNotFound)
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for manifest deletion: %s", manifestKey)
}

func waitForBlobDeleted(t *testing.T, ctx context.Context, loc *service.Locator, blobKey string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, err := loc.AlbumSprites().Open(ctx, blobKey)
		if err != nil {
			require.ErrorIs(t, err, cache.ErrNotFound)
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for blob deletion: %s", blobKey)
}

func currentAlbumManifestKey(t *testing.T, ctx context.Context, loc *service.Locator, albumName string) string {
	t.Helper()

	images, err := loc.ImageSelector().Select().ByAlbumName(albumName).Find(ctx)
	require.NoError(t, err)

	spriteImages := make([]sprite.Image, 0, len(images))
	for _, img := range images {
		spriteImages = append(spriteImages, sprite.Image{
			Hash:   img.Hash,
			Width:  img.Width,
			Height: img.Height,
		})
	}

	return loc.AlbumSprites().ManifestKey(spriteImages)
}

func getAlbumPage(t *testing.T, handler http.Handler, albumName string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/"+albumName+"/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	require.Contains(t, []int{http.StatusOK, http.StatusNoContent}, rec.Code, rec.Body.String())
}

func removeImageFromAlbum(t *testing.T, handler http.Handler, albumName string, imageHash uniq.Hash) {
	t.Helper()

	req := httptest.NewRequest(http.MethodDelete, "/album/"+albumName+"/"+imageHash.String(), nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	require.Contains(t, []int{http.StatusOK, http.StatusNoContent}, rec.Code, rec.Body.String())
}

func cleanupAlbumSprites(t *testing.T, handler http.Handler, dryRun bool) integrity.AlbumSpriteCleanupReport {
	t.Helper()

	url := "/cleanup-album-sprites"
	if dryRun {
		url += "?dry_run=true"
	}

	req := httptest.NewRequest(http.MethodPost, url, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var report integrity.AlbumSpriteCleanupReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))

	return report
}

func sortedChunks(manifest sprite.Manifest) []string {
	keys := make([]string, 0)
	seen := make(map[string]struct{})

	for _, item := range manifest.Images {
		if item.Chunk1x != "" {
			seen[item.Chunk1x] = struct{}{}
		}
		if item.Chunk2x != "" {
			seen[item.Chunk2x] = struct{}{}
		}
	}

	for _, item := range manifest.Markers {
		if item.Chunk1x != "" {
			seen[item.Chunk1x] = struct{}{}
		}
		if item.Chunk2x != "" {
			seen[item.Chunk2x] = struct{}{}
		}
	}

	for key := range seen {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	return keys
}

func setSpriteRetirementDelay(t *testing.T, loc *service.Locator, delay time.Duration) {
	t.Helper()

	v := reflect.ValueOf(loc.AlbumSprites()).Elem().FieldByName("retirementDelay")
	require.True(t, v.IsValid())

	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().SetInt(int64(delay))
}

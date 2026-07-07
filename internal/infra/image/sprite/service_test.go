package sprite

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bool64/brick/database"
	"github.com/bool64/cache"
	"github.com/bool64/cache/blob"
	"github.com/bool64/cache/filecache"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/stats"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/sqlitec"
	_ "modernc.org/sqlite"
)

func TestServiceBuild_ReusesUnchangedChunk(t *testing.T) {
	ctx := context.Background()

	dir := t.TempDir()
	blobs, err := filecache.NewStorage[string](dir)
	if err != nil {
		t.Fatalf("new blob storage: %v", err)
	}
	defer func() {
		_ = blobs.Close()
	}()

	images := []photo.Image{
		newPhotoImage("a", 3000, 2000),
		newPhotoImage("b", 2400, 1600),
		newPhotoImage("c", 1800, 1200),
	}

	s := &Service{
		logger:      ctxd.NoOpLogger{},
		stats:       stats.NoOp{},
		thumbnailer: stubThumbnailer{},
		blobStore:   blobs,
		boxWidth:    300,
		boxHeight:   200,
		chunkSize:   2,
		version:     "test",
	}

	manifest1, _, err := s.build(ctx, []Image{
		{Hash: images[0].Hash, Width: images[0].Width, Height: images[0].Height},
		{Hash: images[1].Hash, Width: images[1].Width, Height: images[1].Height},
	})
	if err != nil {
		t.Fatalf("build manifest 1: %v", err)
	}

	manifest2, _, err := s.build(ctx, []Image{
		{Hash: images[0].Hash, Width: images[0].Width, Height: images[0].Height},
		{Hash: images[1].Hash, Width: images[1].Width, Height: images[1].Height},
		{Hash: images[2].Hash, Width: images[2].Width, Height: images[2].Height},
	})
	if err != nil {
		t.Fatalf("build manifest 2: %v", err)
	}

	a1 := manifest1.Images[images[0].Hash.String()]
	a2 := manifest2.Images[images[0].Hash.String()]
	b1 := manifest1.Images[images[1].Hash.String()]
	b2 := manifest2.Images[images[1].Hash.String()]
	c2 := manifest2.Images[images[2].Hash.String()]

	if a1.Chunk1x != a2.Chunk1x || a1.Chunk2x != a2.Chunk2x {
		t.Fatalf("first chunk was rebuilt unexpectedly: %#v %#v", a1, a2)
	}

	if b1.Chunk1x != b2.Chunk1x || b1.Chunk2x != b2.Chunk2x {
		t.Fatalf("second image chunk changed unexpectedly: %#v %#v", b1, b2)
	}

	if c2.Chunk1x == "" || c2.Chunk2x == "" {
		t.Fatalf("new image chunk was not created: %#v", c2)
	}

	if c2.OffsetY != 0 || c2.BackgroundHeight != 200 || c2.Width != 300 || c2.Height != 200 {
		t.Fatalf("unexpected new chunk placement: %#v", c2)
	}

	if _, err := s.blobStore.Read(ctx, c2.Chunk1x); err != nil {
		t.Fatalf("new chunk 1x blob missing: %v", err)
	}
	if _, err := s.blobStore.Read(ctx, c2.Chunk2x); err != nil {
		t.Fatalf("new chunk 2x blob missing: %v", err)
	}
}

func TestServiceBuild_GroupsSameChunkDifferentShapes(t *testing.T) {
	ctx := context.Background()

	dir := t.TempDir()
	blobs, err := filecache.NewStorage[string](dir)
	if err != nil {
		t.Fatalf("new blob storage: %v", err)
	}
	defer func() {
		_ = blobs.Close()
	}()

	images := []Image{
		{Hash: mustHash("a"), Width: 6000, Height: 4000}, // display 300x200, source 300x200
		{Hash: mustHash("b"), Width: 3000, Height: 4000}, // display 150x200, source 300x400
		{Hash: mustHash("c"), Width: 4000, Height: 4000}, // display 200x200, source 300x300
	}

	s := &Service{
		logger:      ctxd.NoOpLogger{},
		stats:       stats.NoOp{},
		thumbnailer: stubThumbnailer{},
		blobStore:   blobs,
		boxWidth:    300,
		boxHeight:   200,
		chunkSize:   10,
		version:     "test",
	}

	manifest, _, err := s.build(ctx, images)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}

	a := manifest.Images[images[0].Hash.String()]
	b := manifest.Images[images[1].Hash.String()]
	c := manifest.Images[images[2].Hash.String()]

	if a.Chunk1x != b.Chunk1x || a.Chunk1x != c.Chunk1x || a.Chunk2x != b.Chunk2x || a.Chunk2x != c.Chunk2x {
		t.Fatalf("images with same physical sprite width should share sprite: %#v %#v %#v", a, b, c)
	}

	if a.Width != 300 || a.Height != 200 || a.OffsetY != 0 {
		t.Fatalf("unexpected first image placement: %#v", a)
	}

	if b.Width != 150 || b.Height != 200 || b.OffsetY != 200 || b.BackgroundHeight != 900 {
		t.Fatalf("unexpected second image placement: %#v", b)
	}

	if c.Width != 200 || c.Height != 200 || c.OffsetY != 600 || c.BackgroundWidth != 300 || c.BackgroundHeight != 900 {
		t.Fatalf("unexpected third image placement: %#v", c)
	}
}

func TestServiceManifestKey_ReusesBySpriteInput(t *testing.T) {
	s := &Service{version: "test"}

	images := []Image{
		{Hash: mustHash("a"), Width: 1000, Height: 500, HasGPS: false},
		{Hash: mustHash("b"), Width: 800, Height: 600, HasGPS: true},
	}

	key1 := string(s.ManifestKey(images))
	key2 := string(s.ManifestKey([]Image{
		{Hash: mustHash("a"), Width: 1000, Height: 500, HasGPS: false},
		{Hash: mustHash("b"), Width: 800, Height: 600, HasGPS: true},
	}))
	key3 := string(s.ManifestKey([]Image{
		{Hash: mustHash("a"), Width: 1000, Height: 500, HasGPS: false},
		{Hash: mustHash("b"), Width: 800, Height: 600, HasGPS: false},
	}))

	if key1 != key2 {
		t.Fatalf("same sprite input should reuse manifest key: %s != %s", key1, key2)
	}

	if key1 == key3 {
		t.Fatalf("gps-affecting sprite input should change manifest key: %s", key1)
	}
}

func TestServiceTrackAlbumAndRetire(t *testing.T) {
	ctx := context.Background()
	st := testManifestStorage(t)

	blobs, err := filecache.NewStorage[string](t.TempDir())
	if err != nil {
		t.Fatalf("new blob storage: %v", err)
	}
	defer func() {
		_ = blobs.Close()
	}()

	s := &Service{
		logger:          ctxd.NoOpLogger{},
		stats:           stats.NoOp{},
		manifestBackend: sqlitec.NewDBMapOf[Manifest](st, "album-sprite-manifest"),
		blobStore:       blobs,
		version:         "test",
		retirementDelay: 20 * time.Millisecond,
	}

	images := []Image{{Hash: mustHash("a"), Width: 1000, Height: 500, HasGPS: true}}
	ownerA := mustHash("oa")
	ownerB := mustHash("ob")
	manifestKey := s.ManifestKey(images)

	manifest := Manifest{
		Revision: s.revision(images),
		Version:  s.version,
		Images: map[string]ImageThumb{
			images[0].Hash.String(): {
				Chunk1x:          "chunk-1x",
				Chunk2x:          "chunk-2x",
				Width:            300,
				Height:           150,
				BackgroundWidth:  300,
				BackgroundHeight: 150,
			},
		},
	}

	if err := s.manifestBackend.Write(ctx, []byte(manifestKey), manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	writeBlob(t, ctx, blobs, "chunk-1x")
	writeBlob(t, ctx, blobs, "chunk-2x")

	keyA, err := s.TrackAlbum(ctx, images, ownerA)
	if err != nil {
		t.Fatalf("track owner A: %v", err)
	}

	keyB, err := s.TrackAlbum(ctx, images, ownerB)
	if err != nil {
		t.Fatalf("track owner B: %v", err)
	}

	updated, err := s.manifestBackend.Read(ctx, []byte(manifestKey))
	if err != nil {
		t.Fatalf("read tracked manifest: %v", err)
	}

	if len(updated.Albums) != 2 || updated.Albums[0] != ownerA || updated.Albums[1] != ownerB {
		t.Fatalf("unexpected manifest owners: %#v", updated.Albums)
	}

	if err := s.Delete(ctx, keyA); err != nil {
		t.Fatalf("retire owner A: %v", err)
	}

	updated, err = s.manifestBackend.Read(ctx, []byte(manifestKey))
	if err != nil {
		t.Fatalf("read partially retired manifest: %v", err)
	}

	if len(updated.Albums) != 1 || updated.Albums[0] != ownerB {
		t.Fatalf("unexpected owners after first retirement: %#v", updated.Albums)
	}

	if err := s.Delete(ctx, keyB); err != nil {
		t.Fatalf("retire owner B: %v", err)
	}

	updated, err = s.manifestBackend.Read(ctx, []byte(manifestKey))
	if err != nil {
		t.Fatalf("read ownerless manifest: %v", err)
	}

	if len(updated.Albums) != 0 {
		t.Fatalf("manifest should become ownerless before delayed retirement: %#v", updated.Albums)
	}

	if _, err := blobs.Read(ctx, "chunk-1x"); err != nil {
		t.Fatalf("chunk-1x should still exist before delayed retirement: %v", err)
	}

	if _, err := blobs.Read(ctx, "chunk-2x"); err != nil {
		t.Fatalf("chunk-2x should still exist before delayed retirement: %v", err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, err := s.manifestBackend.Read(ctx, []byte(manifestKey)); err != nil {
			if _, err := blobs.Read(ctx, "chunk-1x"); err == nil {
				t.Fatalf("chunk-1x should be deleted after delayed retirement")
			}
			if _, err := blobs.Read(ctx, "chunk-2x"); err == nil {
				t.Fatalf("chunk-2x should be deleted after delayed retirement")
			}

			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("manifest should be deleted after delayed retirement")
}

func TestServiceDeleteManifest_KeepsSharedChunks(t *testing.T) {
	ctx := context.Background()
	st := testManifestStorage(t)

	blobs, err := filecache.NewStorage[string](t.TempDir())
	if err != nil {
		t.Fatalf("new blob storage: %v", err)
	}
	defer func() {
		_ = blobs.Close()
	}()

	s := &Service{
		logger:          ctxd.NoOpLogger{},
		stats:           stats.NoOp{},
		manifestBackend: sqlitec.NewDBMapOf[Manifest](st, "album-sprite-manifest"),
		blobStore:       blobs,
		version:         "test",
		retirementDelay: 0,
	}

	imagesA := []Image{{Hash: mustHash("a"), Width: 1000, Height: 500}}
	imagesB := []Image{{Hash: mustHash("b"), Width: 1000, Height: 500}}
	ownerA := mustHash("oa")
	ownerB := mustHash("ob")
	shared1x := "shared-1x"
	shared2x := "shared-2x"

	manifestKeyA := s.ManifestKey(imagesA)
	manifestKeyB := s.ManifestKey(imagesB)

	manifestA := Manifest{
		Revision: s.revision(imagesA),
		Version:  s.version,
		Images: map[string]ImageThumb{
			imagesA[0].Hash.String(): {
				Chunk1x:          shared1x,
				Chunk2x:          shared2x,
				Width:            300,
				Height:           150,
				BackgroundWidth:  300,
				BackgroundHeight: 150,
			},
		},
	}

	manifestB := Manifest{
		Revision: s.revision(imagesB),
		Version:  s.version,
		Images: map[string]ImageThumb{
			imagesB[0].Hash.String(): {
				Chunk1x:          shared1x,
				Chunk2x:          shared2x,
				Width:            300,
				Height:           150,
				BackgroundWidth:  300,
				BackgroundHeight: 150,
			},
		},
	}

	if err := s.manifestBackend.Write(ctx, []byte(manifestKeyA), manifestA); err != nil {
		t.Fatalf("write manifest A: %v", err)
	}
	if err := s.manifestBackend.Write(ctx, []byte(manifestKeyB), manifestB); err != nil {
		t.Fatalf("write manifest B: %v", err)
	}

	writeBlob(t, ctx, blobs, shared1x)
	writeBlob(t, ctx, blobs, shared2x)

	keyA, err := s.TrackAlbum(ctx, imagesA, ownerA)
	if err != nil {
		t.Fatalf("track owner A: %v", err)
	}
	if _, err := s.TrackAlbum(ctx, imagesB, ownerB); err != nil {
		t.Fatalf("track owner B: %v", err)
	}

	if err := s.Delete(ctx, keyA); err != nil {
		t.Fatalf("retire owner A: %v", err)
	}

	if _, err := s.manifestBackend.Read(ctx, []byte(manifestKeyA)); err == nil {
		t.Fatalf("manifest A should be deleted")
	}

	if _, err := s.manifestBackend.Read(ctx, []byte(manifestKeyB)); err != nil {
		t.Fatalf("manifest B should remain: %v", err)
	}

	if _, err := blobs.Read(ctx, shared1x); err != nil {
		t.Fatalf("shared chunk 1x should remain for manifest B: %v", err)
	}

	if _, err := blobs.Read(ctx, shared2x); err != nil {
		t.Fatalf("shared chunk 2x should remain for manifest B: %v", err)
	}
}

func TestServiceRegenerateChunk_RebuildsMissingBlob(t *testing.T) {
	ctx := context.Background()
	st := testManifestStorage(t)

	blobs, err := filecache.NewStorage[string](t.TempDir())
	if err != nil {
		t.Fatalf("new blob storage: %v", err)
	}
	defer func() {
		_ = blobs.Close()
	}()

	s := &Service{
		logger:          ctxd.NoOpLogger{},
		stats:           stats.NoOp{},
		thumbnailer:     stubThumbnailer{},
		manifestBackend: sqlitec.NewDBMapOf[Manifest](st, "album-sprite-manifest"),
		blobStore:       blobs,
		boxWidth:        300,
		boxHeight:       200,
		chunkSize:       2,
		version:         "test",
	}

	images := []Image{
		{Hash: mustHash("a"), Width: 3000, Height: 2000},
		{Hash: mustHash("b"), Width: 2400, Height: 1600},
	}

	manifest, _, err := s.build(ctx, images)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}

	manifestKey := s.ManifestKey(images)
	if err := s.manifestBackend.Write(ctx, []byte(manifestKey), manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	chunk1x := manifest.Images[images[0].Hash.String()].Chunk1x
	if chunk1x == "" {
		t.Fatalf("expected chunk1x to be set")
	}

	original, err := blobs.Read(ctx, chunk1x)
	if err != nil {
		t.Fatalf("chunk should exist after build: %v", err)
	}
	originalBytes := readEntry(t, original)

	if err := blobs.Delete(ctx, chunk1x); err != nil {
		t.Fatalf("delete chunk to simulate loss: %v", err)
	}

	if _, err := blobs.Read(ctx, chunk1x); err == nil {
		t.Fatalf("chunk should be gone before regeneration")
	}

	regenerated, err := s.RegenerateChunk(ctx, chunk1x)
	if err != nil {
		t.Fatalf("regenerate chunk: %v", err)
	}

	regeneratedBytes := readEntry(t, regenerated)
	if !bytes.Equal(originalBytes, regeneratedBytes) {
		t.Fatalf("regenerated chunk differs from original: content-derived key should reproduce identical bytes")
	}

	if _, err := blobs.Read(ctx, chunk1x); err != nil {
		t.Fatalf("chunk should be persisted after regeneration: %v", err)
	}
}

func TestServiceRegenerateChunk_UnknownKeyNotFound(t *testing.T) {
	ctx := context.Background()
	st := testManifestStorage(t)

	blobs, err := filecache.NewStorage[string](t.TempDir())
	if err != nil {
		t.Fatalf("new blob storage: %v", err)
	}
	defer func() {
		_ = blobs.Close()
	}()

	s := &Service{
		logger:          ctxd.NoOpLogger{},
		stats:           stats.NoOp{},
		manifestBackend: sqlitec.NewDBMapOf[Manifest](st, "album-sprite-manifest"),
		blobStore:       blobs,
		version:         "test",
	}

	if _, err := s.RegenerateChunk(ctx, "does-not-exist"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("expected cache.ErrNotFound, got: %v", err)
	}
}

func TestServiceRegenerateChunk_DedupsConcurrentCalls(t *testing.T) {
	ctx := context.Background()
	st := testManifestStorage(t)

	blobs, err := filecache.NewStorage[string](t.TempDir())
	if err != nil {
		t.Fatalf("new blob storage: %v", err)
	}
	defer func() {
		_ = blobs.Close()
	}()

	var calls int64
	thumbnailer := countingThumbnailer{calls: &calls, delay: 50 * time.Millisecond, inner: stubThumbnailer{}}

	s := &Service{
		logger:          ctxd.NoOpLogger{},
		stats:           stats.NoOp{},
		thumbnailer:     thumbnailer,
		manifestBackend: sqlitec.NewDBMapOf[Manifest](st, "album-sprite-manifest"),
		blobStore:       blobs,
		boxWidth:        300,
		boxHeight:       200,
		chunkSize:       2,
		version:         "test",
	}

	images := []Image{
		{Hash: mustHash("a"), Width: 3000, Height: 2000},
		{Hash: mustHash("b"), Width: 2400, Height: 1600},
	}

	manifest, _, err := s.build(ctx, images)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}

	manifestKey := s.ManifestKey(images)
	if err := s.manifestBackend.Write(ctx, []byte(manifestKey), manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	chunk1x := manifest.Images[images[0].Hash.String()].Chunk1x
	if chunk1x == "" {
		t.Fatalf("expected chunk1x to be set")
	}

	if err := blobs.Delete(ctx, chunk1x); err != nil {
		t.Fatalf("delete chunk to simulate loss: %v", err)
	}

	callsBeforeRegen := atomic.LoadInt64(&calls)

	const concurrency = 20

	var start, done sync.WaitGroup
	start.Add(1)
	done.Add(concurrency)

	errs := make([]error, concurrency)
	for i := 0; i < concurrency; i++ {
		go func(i int) {
			defer done.Done()
			start.Wait()
			_, regenErr := s.RegenerateChunk(ctx, chunk1x)
			errs[i] = regenErr
		}(i)
	}

	start.Done()
	done.Wait()

	for i, regenErr := range errs {
		if regenErr != nil {
			t.Fatalf("regenerate chunk (goroutine %d): %v", i, regenErr)
		}
	}

	// A real rebuild calls the thumbnailer once per image (2 here). Without dedup, 20
	// concurrent callers would each trigger their own rebuild, up to 20x that count.
	gotCalls := atomic.LoadInt64(&calls) - callsBeforeRegen
	if gotCalls > int64(len(images))*2 {
		t.Fatalf("expected concurrent regenerations to be deduplicated via singleflight, "+
			"thumbnailer was called %d times for %d concurrent callers", gotCalls, concurrency)
	}

	if _, err := blobs.Read(ctx, chunk1x); err != nil {
		t.Fatalf("chunk should exist after concurrent regeneration: %v", err)
	}
}

type countingThumbnailer struct {
	calls *int64
	delay time.Duration
	inner photo.Thumbnailer
}

func (c countingThumbnailer) Thumbnail(ctx context.Context, img photo.Image, size photo.ThumbSize) (photo.Thumb, error) {
	atomic.AddInt64(c.calls, 1)

	if c.delay > 0 {
		time.Sleep(c.delay)
	}

	return c.inner.Thumbnail(ctx, img, size)
}

type stubThumbnailer struct{}

func (stubThumbnailer) Thumbnail(_ context.Context, img photo.Image, size photo.ThumbSize) (photo.Thumb, error) {
	w, h, err := size.Resize(uint(img.Width), uint(img.Height))
	if err != nil {
		return photo.Thumb{}, err
	}

	canvas := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	fill := color.RGBA{R: byte(w % 255), G: byte(h % 255), B: 0x80, A: 0xff}
	for y := 0; y < int(h); y++ {
		for x := 0; x < int(w); x++ {
			canvas.SetRGBA(x, y, fill)
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, canvas, &jpeg.Options{Quality: 90}); err != nil {
		return photo.Thumb{}, err
	}

	return photo.Thumb{Data: buf.Bytes(), Width: w, Height: h, Format: size}, nil
}

func newPhotoImage(seed string, width, height int64) photo.Image {
	img := photo.Image{Width: width, Height: height}
	img.Hash = mustHash(seed)

	return img
}

func mustHash(seed string) uniq.Hash {
	var h uniq.Hash
	if err := h.UnmarshalText([]byte(seed)); err != nil {
		panic(err)
	}

	return h
}

func testManifestStorage(t *testing.T) *sqluct.Storage {
	t.Helper()

	cfg := database.Config{
		DriverName:      "sqlite",
		DSN:             filepath.Join(t.TempDir(), "sprite.sqlite") + "?_time_format=sqlite",
		ApplyMigrations: true,
		MaxOpen:         1,
		MaxIdle:         1,
	}

	st, err := database.SetupStorageDSN(cfg, ctxd.NoOpLogger{}, stats.NoOp{}, sqlitec.Migrations)
	if err != nil {
		t.Fatalf("setup storage: %v", err)
	}

	t.Cleanup(func() {
		if err := st.DB().DB.Close(); err != nil {
			t.Fatalf("close storage: %v", err)
		}
	})

	return st
}

func writeBlob(t *testing.T, ctx context.Context, blobs *filecache.Storage[string], key string) {
	t.Helper()

	entry := blob.FromReader(bytes.NewReader([]byte("x")), blob.Meta{
		Name:    key + ".jpg",
		Size:    1,
		ModTime: time.Now(),
	})

	if err := blobs.Write(ctx, key, entry); err != nil {
		t.Fatalf("write blob %s: %v", key, err)
	}
}

func readEntry(t *testing.T, entry blob.Entry) []byte {
	t.Helper()

	rc, err := entry.Open()
	if err != nil {
		t.Fatalf("open blob entry: %v", err)
	}
	defer func() {
		_ = rc.Close()
	}()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read blob entry: %v", err)
	}

	return data
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

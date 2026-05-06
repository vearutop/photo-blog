package sprite

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"

	"github.com/bool64/cache/filecache"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
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

	manifest1, err := s.build(ctx, photo.Album{}, []Image{
		{Hash: images[0].Hash, Width: images[0].Width, Height: images[0].Height},
		{Hash: images[1].Hash, Width: images[1].Width, Height: images[1].Height},
	})
	if err != nil {
		t.Fatalf("build manifest 1: %v", err)
	}

	manifest2, err := s.build(ctx, photo.Album{}, []Image{
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

	manifest, err := s.build(ctx, photo.Album{}, images)
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

	key1 := string(s.manifestKey(images))
	key2 := string(s.manifestKey([]Image{
		{Hash: mustHash("a"), Width: 1000, Height: 500, HasGPS: false},
		{Hash: mustHash("b"), Width: 800, Height: 600, HasGPS: true},
	}))
	key3 := string(s.manifestKey([]Image{
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

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

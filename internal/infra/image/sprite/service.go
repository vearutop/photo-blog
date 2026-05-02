package sprite

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/bool64/cache"
	"github.com/bool64/cache/blob"
	"github.com/bool64/cache/filecache"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/sqlitec"
	xdraw "golang.org/x/image/draw"
)

const (
	defaultBoxWidth  = 300
	defaultBoxHeight = 200
	defaultChunkSize = 20
	defaultVersion   = "v1"
)

type Image struct {
	Hash   uniq.Hash
	Width  int64
	Height int64
}

type Manifest struct {
	AlbumHash string                `json:"album_hash"`
	Revision  string                `json:"revision"`
	Version   string                `json:"version"`
	Images    map[string]ImageThumb `json:"images"`
}

type ImageThumb struct {
	Chunk1x          string `json:"chunk_1x"`
	Chunk2x          string `json:"chunk_2x"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	OffsetY          int    `json:"offset_y"`
	BackgroundWidth  int    `json:"background_width"`
	BackgroundHeight int    `json:"background_height"`
}

type ViewItem struct {
	Style            template.CSS
	BoxWidth         int
	BoxHeight        int
	OffsetY          int
	BackgroundWidth  int
	BackgroundHeight int
}

type imageFinder interface {
	FindByHashes(ctx context.Context, hashes ...uniq.Hash) ([]photo.Image, error)
}

type bucketKey struct {
	Width  int
	Height int
}

type Service struct {
	logger          ctxd.Logger
	stats           stats.Tracker
	imageFinder     imageFinder
	thumbnailer     photo.Thumbnailer
	manifestBackend *sqlitec.DBMapOf[Manifest]
	manifestCache   *cache.FailoverOf[Manifest]
	blobStore       *filecache.Storage

	boxWidth  int
	boxHeight int
	chunkSize int
	version   string
	building  sync.Map
}

func NewService(
	logger ctxd.Logger,
	stats stats.Tracker,
	imageFinder imageFinder,
	thumbnailer photo.Thumbnailer,
	manifestBackend *sqlitec.DBMapOf[Manifest],
	blobStore *filecache.Storage,
) *Service {
	s := &Service{
		logger:          logger,
		stats:           stats,
		imageFinder:     imageFinder,
		thumbnailer:     thumbnailer,
		manifestBackend: manifestBackend,
		blobStore:       blobStore,
		boxWidth:        defaultBoxWidth,
		boxHeight:       defaultBoxHeight,
		chunkSize:       defaultChunkSize,
		version:         defaultVersion,
	}

	s.manifestCache = cache.NewFailoverOf[Manifest](func(cfg *cache.FailoverConfigOf[Manifest]) {
		cfg.Backend = manifestBackend
	})

	return s
}

func (s *Service) Ready(ctx context.Context, album photo.Album, images []Image) (Manifest, bool, error) {
	if len(images) == 0 {
		return Manifest{}, false, nil
	}

	key := s.manifestKey(album)

	manifest, err := s.manifestBackend.Read(ctx, key)
	if err == nil {
		if !validManifest(manifest) {
			s.ensureBuild(album, images)

			return Manifest{}, false, nil
		}

		return manifest, true, nil
	}
	var expired cache.ErrWithExpiredItemOf[Manifest]
	if errors.As(err, &expired) {
		s.ensureBuild(album, images)

		return Manifest{}, false, nil
	}

	if err != nil && err != cache.ErrNotFound {
		return Manifest{}, false, fmt.Errorf("read sprite manifest: %w", err)
	}

	s.ensureBuild(album, images)

	return Manifest{}, false, nil
}

func (s *Service) ensureBuild(album photo.Album, images []Image) {
	key := string(s.manifestKey(album))
	if _, loaded := s.building.LoadOrStore(key, struct{}{}); loaded {
		return
	}

	s.stats.Add(context.Background(), "album_sprite_build", 1, "result", "started")

	go func() {
		ctx := context.Background()
		defer s.building.Delete(key)
		started := time.Now()

		_, err := s.manifestCache.Get(ctx, []byte(key), func(ctx context.Context) (Manifest, error) {
			return s.build(ctx, album, images)
		})
		s.stats.Add(ctx, "album_sprite_build_ms", float64(time.Since(started).Milliseconds()))
		if err != nil {
			s.stats.Add(ctx, "album_sprite_build", 1, "result", "error")
			s.logger.Error(ctx, "failed to build album sprite manifest", "album", album.Name, "error", err)

			return
		}

		s.stats.Add(ctx, "album_sprite_build", 1, "result", "success")
	}()
}

func (s *Service) View(manifest Manifest) map[string]*ViewItem {
	items := make(map[string]*ViewItem, len(manifest.Images))

	for hash, img := range manifest.Images {
		oneX := "/thumb-sprite/" + img.Chunk1x + ".jpg"
		twoX := "/thumb-sprite/" + img.Chunk2x + ".jpg"
		items[hash] = &ViewItem{
			BoxWidth:         img.Width,
			BoxHeight:        img.Height,
			OffsetY:          img.OffsetY,
			BackgroundWidth:  img.BackgroundWidth,
			BackgroundHeight: img.BackgroundHeight,
			Style: template.CSS(fmt.Sprintf(
				"width:%dpx;height:%dpx;background-image:url('%s');background-image:-webkit-image-set(url('%s') 1x, url('%s') 2x);background-image:image-set(url('%s') 1x, url('%s') 2x);background-position:0 -%dpx;background-repeat:no-repeat;background-size:%dpx %dpx;",
				img.Width, img.Height, oneX, oneX, twoX, oneX, twoX, img.OffsetY, img.BackgroundWidth, img.BackgroundHeight,
			)),
		}
	}

	return items
}

func (s *Service) Open(ctx context.Context, key string) (blob.Entry, error) {
	return s.blobStore.Read(ctx, []byte(key))
}

func (s *Service) Close() error {
	return s.blobStore.Close()
}

func (s *Service) build(ctx context.Context, album photo.Album, images []Image) (Manifest, error) {
	fullImages, err := s.imageFinder.FindByHashes(ctx, imageHashes(images)...)
	if err != nil {
		return Manifest{}, fmt.Errorf("find sprite source images: %w", err)
	}

	byHash := make(map[uniq.Hash]photo.Image, len(fullImages))
	for _, img := range fullImages {
		byHash[img.Hash] = img
	}

	manifest := Manifest{
		AlbumHash: album.Hash.String(),
		Revision:  s.revision(album),
		Version:   s.version,
		Images:    make(map[string]ImageThumb, len(images)),
	}

	buckets := make(map[bucketKey][]Image)
	bucketOrder := make([]bucketKey, 0)
	for _, img := range images {
		bw, bh := s.renderedBox(img)
		key := bucketKey{Width: bw, Height: bh}
		if _, ok := buckets[key]; !ok {
			bucketOrder = append(bucketOrder, key)
		}

		buckets[key] = append(buckets[key], img)
	}

	for _, key := range bucketOrder {
		bucketImages := buckets[key]
		for start := 0; start < len(bucketImages); start += s.chunkSize {
			end := start + s.chunkSize
			if end > len(bucketImages) {
				end = len(bucketImages)
			}

			chunk := bucketImages[start:end]
			chunk1x := s.chunkKey(1, key, chunk)
			chunk2x := s.chunkKey(2, key, chunk)

			if err := s.ensureChunk(ctx, chunk1x, key, 1, chunk, byHash); err != nil {
				return Manifest{}, fmt.Errorf("build sprite chunk 1x: %w", err)
			}
			if err := s.ensureChunk(ctx, chunk2x, key, 2, chunk, byHash); err != nil {
				return Manifest{}, fmt.Errorf("build sprite chunk 2x: %w", err)
			}

			bgHeight := len(chunk) * key.Height
			for idx, img := range chunk {
				manifest.Images[img.Hash.String()] = ImageThumb{
					Chunk1x:          chunk1x,
					Chunk2x:          chunk2x,
					Width:            key.Width,
					Height:           key.Height,
					OffsetY:          idx * key.Height,
					BackgroundWidth:  key.Width,
					BackgroundHeight: bgHeight,
				}
			}
		}
	}

	return manifest, nil
}

func (s *Service) ensureChunk(ctx context.Context, key string, bucket bucketKey, scale int, chunk []Image, byHash map[uniq.Hash]photo.Image) error {
	if _, err := s.blobStore.Read(ctx, []byte(key)); err == nil {
		return nil
	} else if err != cache.ErrNotFound {
		return fmt.Errorf("read sprite blob: %w", err)
	}

	rasterScale := spriteRasterScale(bucket, scale)
	cellW := int(math.Round(float64(bucket.Width) * rasterScale))
	cellH := int(math.Round(float64(bucket.Height) * rasterScale))
	bg := image.NewRGBA(image.Rect(0, 0, cellW, cellH*len(chunk)))

	for idx, item := range chunk {
		src, ok := byHash[item.Hash]
		if !ok {
			return fmt.Errorf("missing source image %s", item.Hash)
		}

		size := spriteThumbSize(bucket, scale)
		th, err := s.thumbnailer.Thumbnail(ctx, src, size)
		if err != nil {
			return fmt.Errorf("get thumbnail %s %s: %w", item.Hash, size, err)
		}

		j, err := decodeThumb(th)
		if err != nil {
			return fmt.Errorf("decode thumbnail %s %s: %w", item.Hash, size, err)
		}

		bounds := j.Bounds()
		y := idx * cellH
		xdraw.CatmullRom.Scale(bg, image.Rect(0, y, cellW, y+cellH), j, bounds, xdraw.Src, nil)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, bg, &jpeg.Options{Quality: 90}); err != nil {
		return fmt.Errorf("encode sprite jpeg: %w", err)
	}

	entry := blob.FromReader(bytes.NewReader(buf.Bytes()), blob.Meta{
		Name:    key + ".jpg",
		Size:    int64(buf.Len()),
		ModTime: time.Now(),
	})

	if err := s.blobStore.Write(ctx, []byte(key), entry); err != nil {
		return fmt.Errorf("store sprite blob: %w", err)
	}

	return nil
}

func (s *Service) manifestKey(album photo.Album) []byte {
	return []byte("album-sprite-manifest:" + album.Hash.String() + ":" + s.revision(album) + ":" + s.version)
}

func (s *Service) revision(album photo.Album) string {
	return fmt.Sprintf("%x", album.UpdatedAt.UTC().UnixNano())
}

func (s *Service) chunkKey(scale int, bucket bucketKey, chunk []Image) string {
	h := sha1.New()
	_, _ = io.WriteString(h, s.version)
	_, _ = io.WriteString(h, fmt.Sprintf(":%d:%d:%d", scale, bucket.Width, bucket.Height))

	for _, img := range chunk {
		_, _ = io.WriteString(h, ":"+img.Hash.String())
		_, _ = io.WriteString(h, fmt.Sprintf(":%dx%d", img.Width, img.Height))
	}

	return "album-sprite:" + hex.EncodeToString(h.Sum(nil))
}

func (s *Service) renderedBox(img Image) (int, int) {
	bw, bh := fitBox(uint(img.Width), uint(img.Height), uint(s.boxWidth), uint(s.boxHeight))

	return int(bw), int(bh)
}

func fitBox(origW, origH, maxW, maxH uint) (uint, uint) {
	if origW == 0 || origH == 0 {
		return maxW, maxH
	}

	scaleByWidth := float64(maxW) / float64(origW)
	scaleByHeight := float64(maxH) / float64(origH)
	scale := scaleByWidth
	if scaleByHeight < scale {
		scale = scaleByHeight
	}

	w := uint(math.Round(float64(origW) * scale))
	h := uint(math.Round(float64(origH) * scale))
	if w == 0 {
		w = 1
	}
	if h == 0 {
		h = 1
	}

	return w, h
}

func imageHashes(images []Image) []uniq.Hash {
	hashes := make([]uniq.Hash, 0, len(images))
	for _, img := range images {
		hashes = append(hashes, img.Hash)
	}

	return hashes
}

func spriteThumbSize(bucket bucketKey, scale int) photo.ThumbSize {
	if bucket.Width < defaultBoxWidth {
		if scale == 2 {
			return "600w"
		}

		return "300w"
	}

	if scale == 2 {
		return "600w"
	}

	return "300w"
}

func spriteRasterScale(bucket bucketKey, scale int) float64 {
	size := spriteThumbSize(bucket, scale)
	w, h, err := size.WidthHeight()
	if err != nil {
		return float64(scale)
	}

	if w > 0 && bucket.Width > 0 {
		return float64(w) / float64(bucket.Width)
	}

	if h > 0 && bucket.Height > 0 {
		return float64(h) / float64(bucket.Height)
	}

	return float64(scale)
}

func decodeThumb(th photo.Thumb) (image.Image, error) {
	r, err := th.Reader()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	img, err := jpeg.Decode(r)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func KeyFromPath(path string) string {
	return strings.TrimSuffix(path, ".jpg")
}

func validManifest(m Manifest) bool {
	if m.Version == "" || len(m.Images) == 0 {
		return false
	}

	for _, img := range m.Images {
		if img.Chunk1x == "" || img.Chunk2x == "" || img.Width <= 0 || img.Height <= 0 || img.BackgroundWidth <= 0 || img.BackgroundHeight <= 0 {
			return false
		}
	}

	return true
}

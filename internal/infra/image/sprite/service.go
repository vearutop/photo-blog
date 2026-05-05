package sprite

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"
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
	markerChunkSize  = 100
	defaultVersion   = "v1"
	markerBoxSize    = 40
)

type Image struct {
	Hash   uniq.Hash
	Width  int64
	Height int64
	HasGPS bool
}

type Manifest struct {
	AlbumHash string                `json:"album_hash"`
	Revision  string                `json:"revision"`
	Version   string                `json:"version"`
	Images    map[string]ImageThumb `json:"images"`
	Markers   map[string]ImageThumb `json:"markers,omitempty"`
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
	Chunk1x          string `json:"chunk_1x"`
	Chunk2x          string `json:"chunk_2x"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	OffsetY          int    `json:"offset_y"`
	BackgroundWidth  int    `json:"background_width"`
	BackgroundHeight int    `json:"background_height"`
}

type bucketKey struct {
	Width  int
	Height int
}

type Service struct {
	logger          ctxd.Logger
	stats           stats.Tracker
	thumbnailer     photo.Thumbnailer
	manifestBackend *sqlitec.DBMapOf[Manifest]
	manifestCache   *cache.FailoverOf[Manifest]
	blobStore       *filecache.Storage[string]

	boxWidth  int
	boxHeight int
	chunkSize int
	version   string
}

func NewService(
	logger ctxd.Logger,
	stats stats.Tracker,
	thumbnailer photo.Thumbnailer,
	manifestBackend *sqlitec.DBMapOf[Manifest],
	blobStore *filecache.Storage[string],
) *Service {
	s := &Service{
		logger:          logger,
		stats:           stats,
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
		if !validManifest(manifest, images) {
			s.ensureBuild(ctx, key, album, images)

			return Manifest{}, false, nil
		}

		return manifest, true, nil
	}
	var expired cache.ErrWithExpiredItemOf[Manifest]
	if errors.As(err, &expired) {
		s.ensureBuild(ctx, key, album, images)

		return Manifest{}, false, nil
	}

	if err != nil && err != cache.ErrNotFound {
		return Manifest{}, false, fmt.Errorf("read sprite manifest: %w", err)
	}

	s.ensureBuild(ctx, key, album, images)

	return Manifest{}, false, nil
}

func (s *Service) ensureBuild(ctx context.Context, key []byte, album photo.Album, images []Image) {
	s.stats.Add(ctx, "album_sprite_build", 1)

	go func() {
		ctx = context.WithoutCancel(ctx)

		_, err := s.manifestCache.Get(ctx, key, func(ctx context.Context) (Manifest, error) {
			st := time.Now()
			m, err := s.build(ctx, album, images)

			s.logger.Info(ctx, "build album sprite", "album", album.Name,
				"duration", time.Since(st).String())

			return m, err
		})
		if err != nil {
			s.logger.Error(ctx, "build album sprite", "album", album.Name,
				"error", err.Error())
		}
	}()
}

func (s *Service) View(manifest Manifest) map[string]*ViewItem {
	items := make(map[string]*ViewItem, len(manifest.Images))

	for hash, img := range manifest.Images {
		items[hash] = &ViewItem{
			Chunk1x:          img.Chunk1x,
			Chunk2x:          img.Chunk2x,
			Width:            img.Width,
			Height:           img.Height,
			OffsetY:          img.OffsetY,
			BackgroundWidth:  img.BackgroundWidth,
			BackgroundHeight: img.BackgroundHeight,
		}
	}

	return items
}

func (s *Service) MarkerData(manifest Manifest) map[string]ImageThumb {
	if len(manifest.Markers) == 0 {
		return nil
	}

	res := make(map[string]ImageThumb, len(manifest.Markers))
	for k, v := range manifest.Markers {
		res[k] = v
	}

	return res
}

func (s *Service) Open(ctx context.Context, key string) (blob.Entry, error) {
	return s.blobStore.Read(ctx, key)
}

func (s *Service) Close() error {
	return s.blobStore.Close()
}

func (s *Service) build(ctx context.Context, album photo.Album, images []Image) (Manifest, error) {
	manifest := Manifest{
		AlbumHash: album.Hash.String(),
		Revision:  s.revision(album),
		Version:   s.version,
		Images:    make(map[string]ImageThumb, len(images)),
		Markers:   make(map[string]ImageThumb),
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
			chunk1x := s.chunkKey(1, key, chunk, composeFit)
			chunk2x := s.chunkKey(2, key, chunk, composeFit)

			if err := s.ensureChunk(ctx, chunk1x, key, 1, chunk, composeFit); err != nil {
				return Manifest{}, fmt.Errorf("build sprite chunk 1x: %w", err)
			}
			if err := s.ensureChunk(ctx, chunk2x, key, 2, chunk, composeFit); err != nil {
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

	markerImages := make([]Image, 0)
	for _, img := range images {
		if img.HasGPS {
			markerImages = append(markerImages, img)
		}
	}

	if err := s.buildMarkerSprites(ctx, markerImages, &manifest); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func (s *Service) ensureChunk(ctx context.Context, key string, bucket bucketKey, scale int, chunk []Image, mode composeMode) error {
	if _, err := s.blobStore.Read(ctx, key); err == nil {
		return nil
	} else if err != cache.ErrNotFound {
		return fmt.Errorf("read sprite blob: %w", err)
	}

	rasterScale := spriteRasterScale(bucket, scale, mode)
	cellW := int(math.Round(float64(bucket.Width) * rasterScale))
	cellH := int(math.Round(float64(bucket.Height) * rasterScale))
	bg := image.NewRGBA(image.Rect(0, 0, cellW, cellH*len(chunk)))

	for idx, item := range chunk {
		src := photo.Image{}
		src.Hash = item.Hash
		src.Width = item.Width
		src.Height = item.Height

		size := spriteThumbSize(bucket, scale, mode)
		th, err := s.thumbnailer.Thumbnail(ctx, src, size)
		if err != nil {
			return fmt.Errorf("get thumbnail %s %s: %w", item.Hash, size, err)
		}

		j, err := decodeThumb(th)
		if err != nil {
			return fmt.Errorf("decode thumbnail %s %s: %w", item.Hash, size, err)
		}

		y := idx * cellH
		drawThumb(bg, image.Rect(0, y, cellW, y+cellH), j, mode)
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

	if err := s.blobStore.Write(ctx, key, entry); err != nil {
		return fmt.Errorf("store sprite blob: %w", err)
	}

	return nil
}

func (s *Service) buildMarkerSprites(ctx context.Context, images []Image, manifest *Manifest) error {
	if len(images) == 0 {
		return nil
	}

	bucket := bucketKey{Width: markerBoxSize, Height: markerBoxSize}
	for start := 0; start < len(images); start += markerChunkSize {
		end := start + markerChunkSize
		if end > len(images) {
			end = len(images)
		}

		chunk := images[start:end]
		chunk1x := s.chunkKey(1, bucket, chunk, composeCover)
		chunk2x := chunk1x

		if err := s.ensureChunk(ctx, chunk1x, bucket, 1, chunk, composeCover); err != nil {
			return fmt.Errorf("build marker sprite chunk 1x: %w", err)
		}

		bgHeight := len(chunk) * bucket.Height
		for idx, img := range chunk {
			manifest.Markers[img.Hash.String()] = ImageThumb{
				Chunk1x:          chunk1x,
				Chunk2x:          chunk2x,
				Width:            bucket.Width,
				Height:           bucket.Height,
				OffsetY:          idx * bucket.Height,
				BackgroundWidth:  bucket.Width,
				BackgroundHeight: bgHeight,
			}
		}
	}

	return nil
}

func (s *Service) manifestKey(album photo.Album) []byte {
	return []byte("album-sprite-manifest:" + album.Hash.String() + ":" + s.revision(album) + ":" + s.version)
}

func (s *Service) revision(album photo.Album) string {
	return fmt.Sprintf("%x", album.UpdatedAt.UTC().UnixNano())
}

func (s *Service) chunkKey(scale int, bucket bucketKey, chunk []Image, mode composeMode) string {
	h := sha1.New()
	_, _ = io.WriteString(h, s.version)
	_, _ = io.WriteString(h, fmt.Sprintf(":%d:%d:%d", scale, bucket.Width, bucket.Height))
	_, _ = io.WriteString(h, ":"+string(spriteThumbSize(bucket, scale, mode)))
	_, _ = io.WriteString(h, fmt.Sprintf(":rs%.2f:m%d", spriteRasterScale(bucket, scale, mode), mode))

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

func spriteThumbSize(bucket bucketKey, scale int, mode composeMode) photo.ThumbSize {
	if mode == composeCover && bucket.Width == markerBoxSize && bucket.Height == markerBoxSize {
		return "200h"
	}

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

func spriteRasterScale(bucket bucketKey, scale int, mode composeMode) float64 {
	if mode == composeCover && bucket.Width == markerBoxSize && bucket.Height == markerBoxSize {
		return 2
	}

	size := spriteThumbSize(bucket, scale, mode)
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

func drawThumb(dst *image.RGBA, dr image.Rectangle, src image.Image, mode composeMode) {
	sr := src.Bounds()

	if mode == composeCover {
		sr = cropRectForAspect(sr, float64(dr.Dx())/float64(dr.Dy()))
	}

	xdraw.CatmullRom.Scale(dst, dr, src, sr, xdraw.Src, nil)
}

func cropRectForAspect(bounds image.Rectangle, targetAspect float64) image.Rectangle {
	if targetAspect <= 0 {
		return bounds
	}

	w := float64(bounds.Dx())
	h := float64(bounds.Dy())
	srcAspect := w / h

	if math.Abs(srcAspect-targetAspect) < 0.0001 {
		return bounds
	}

	if srcAspect > targetAspect {
		newW := int(math.Round(h * targetAspect))
		if newW <= 0 || newW > bounds.Dx() {
			return bounds
		}

		left := bounds.Min.X + (bounds.Dx()-newW)/2

		return image.Rect(left, bounds.Min.Y, left+newW, bounds.Max.Y)
	}

	newH := int(math.Round(w / targetAspect))
	if newH <= 0 || newH > bounds.Dy() {
		return bounds
	}

	top := bounds.Min.Y + (bounds.Dy()-newH)/2

	return image.Rect(bounds.Min.X, top, bounds.Max.X, top+newH)
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

func validManifest(m Manifest, images []Image) bool {
	if m.Version == "" || len(m.Images) == 0 {
		return false
	}

	for _, img := range m.Images {
		if img.Chunk1x == "" || img.Chunk2x == "" || img.Width <= 0 || img.Height <= 0 || img.BackgroundWidth <= 0 || img.BackgroundHeight <= 0 {
			return false
		}
	}

	needMarkers := false
	for _, img := range images {
		if img.HasGPS {
			needMarkers = true
			marker, ok := m.Markers[img.Hash.String()]
			if !ok || marker.Chunk1x == "" || marker.Chunk2x == "" || marker.Width <= 0 || marker.Height <= 0 || marker.BackgroundWidth <= 0 || marker.BackgroundHeight <= 0 {
				return false
			}
		}
	}

	if needMarkers && len(m.Markers) == 0 {
		return false
	}

	return true
}

type composeMode int

const (
	composeFit composeMode = iota
	composeCover
)

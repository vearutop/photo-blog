package image

import (
	"context"
	"image/jpeg"
	"os"

	"github.com/bool64/ctxd"
	blurhash "github.com/buckket/go-blurhash"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type indexerDeps interface {
	CtxdLogger() ctxd.Logger

	PhotoThumbnailer() photo.Thumbnailer

	PhotoImageUpdater() uniq.Updater[photo.Image]

	PhotoExifEnsurer() uniq.Ensurer[photo.Exif]
	PhotoExifFinder() uniq.Finder[photo.Exif]

	PhotoGpsEnsurer() uniq.Ensurer[photo.Gps]
	PhotoGpsFinder() uniq.Finder[photo.Gps]
}

func NewIndexer(deps indexerDeps) *Indexer {
	return &Indexer{
		deps: deps,
	}
}

type Indexer struct {
	deps indexerDeps
}

func (i *Indexer) PhotoImageIndexer() photo.ImageIndexer {
	return i
}

func (i *Indexer) closeFile(ctx context.Context, f *os.File) {
	if f == nil {
		return
	}

	if err := f.Close(); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to close file", "error", err.Error())
	}
}

func (i *Indexer) Index(ctx context.Context, img photo.Image, flags photo.IndexingFlags) (err error) {
	ctx = ctxd.AddFields(ctx, "img", img)

	if img.Width == 0 {
		f, err := os.Open(img.Path)
		if err != nil {
			return ctxd.WrapError(ctx, err, "open image file")
		}
		c, err := jpeg.DecodeConfig(f)
		i.closeFile(ctx, f)

		if err != nil {
			return ctxd.WrapError(ctx, err, "image dimensions")
		}

		img.Width = int64(c.Width)
		img.Height = int64(c.Height)

		if err := i.deps.PhotoImageUpdater().Update(ctx, img); err != nil {
			return ctxd.WrapError(ctx, err, "update image")
		}
	}

	if err := i.ensureExif(ctx, img, flags); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to ensure exif", "error", err)
	}

	i.ensureThumbs(ctx, img)
	i.ensureBlurHash(ctx, img)

	return nil
}

func (i *Indexer) ensureBlurHash(ctx context.Context, img photo.Image) {
	if img.BlurHash != "" {
		return
	}

	th, err := i.deps.PhotoThumbnailer().Thumbnail(ctx, img, "300w")
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to get thumbnail",
			"error", err.Error(), "size", "300w")
		return
	}

	j, err := jpeg.Decode(th.ReadSeeker())
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed decode thumbnail",
			"error", err.Error())
		return
	}

	bh, err := blurhash.Encode(5, 5, j)
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to encode blurhash",
			"error", err.Error())
		return
	}

	img.BlurHash = bh

	if err := i.deps.PhotoImageUpdater().Update(ctx, img); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to save image",
			"error", err.Error())
	}
}

func (i *Indexer) ensureThumbs(ctx context.Context, img photo.Image) {
	for _, size := range photo.ThumbSizes {
		_, err := i.deps.PhotoThumbnailer().Thumbnail(ctx, img, size)
		if err != nil {
			i.deps.CtxdLogger().Error(ctx, "failed to get thumbnail",
				"error", err.Error(), "size", size)
		}
	}
}

func (i *Indexer) ensureExif(ctx context.Context, img photo.Image, flags photo.IndexingFlags) error {
	exifExists, gpsExists := false, false

	if _, err := i.deps.PhotoExifFinder().FindByHash(ctx, img.Hash); err == nil {
		exifExists = true
		if _, err := i.deps.PhotoGpsFinder().FindByHash(ctx, img.Hash); err == nil {
			gpsExists = true
		}
	}

	if exifExists && gpsExists && !flags.RebuildExif && !flags.RebuildGps {
		return nil
	}

	f, err := os.Open(img.Path)
	if err != nil {
		return ctxd.WrapError(ctx, err, "open image file")
	}

	m, err := ReadMeta(f)
	i.closeFile(ctx, f)
	if err != nil {
		return ctxd.WrapError(ctx, err, "read image meta")
	}

	m.Exif.Hash = img.Hash
	if _, err := i.deps.PhotoExifEnsurer().Ensure(ctx, m.Exif); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to store image meta",
			"error", err.Error(), "exif", m.Exif)
	}

	if m.GpsInfo != nil {
		m.GpsInfo.Hash = img.Hash
		if _, err := i.deps.PhotoGpsEnsurer().Ensure(ctx, *m.GpsInfo); err != nil {
			i.deps.CtxdLogger().Error(ctx, "failed to store image gps",
				"error", err.Error(), "gps", m.GpsInfo)
		}
	}

	return nil
}

package image

import (
	"context"
	"image/jpeg"
	"os"

	"github.com/bool64/ctxd"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type indexerDeps interface {
	CtxdLogger() ctxd.Logger

	PhotoThumbnailer() photo.Thumbnailer

	PhotoImageUpdater() photo.ImageUpdater

	PhotoExifEnsurer() photo.ExifEnsurer
	PhotoExifFinder() photo.ExifFinder

	PhotoGpsEnsurer() photo.GpsEnsurer
	PhotoGpsFinder() photo.GpsFinder
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

func (i *Indexer) Index(ctx context.Context, img photo.Image) (err error) {
	ctx = ctxd.AddFields(ctx, "img", img)

	var f *os.File

	if img.Width == 0 {
		f, err = os.Open(img.Path)
		if err != nil {
			return ctxd.WrapError(ctx, err, "open image file")
		}
		c, err := jpeg.DecodeConfig(f)
		defer i.closeFile(ctx, f)

		if err != nil {
			return ctxd.WrapError(ctx, err, "image dimensions")
		}

		img.Width = int64(c.Width)
		img.Height = int64(c.Height)

		if err := i.deps.PhotoImageUpdater().Update(ctx, img.ImageData); err != nil {
			return ctxd.WrapError(ctx, err, "update image")
		}
	}

	if err := i.ensureExif(ctx, img, f); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to ensure exif", "error", err)
	}

	i.ensureThumbs(ctx, img)

	return nil
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

func (i *Indexer) ensureExif(ctx context.Context, img photo.Image, f *os.File) error {
	if _, err := i.deps.PhotoExifFinder().FindByHash(ctx, img.Hash); err == nil {
		if _, err := i.deps.PhotoGpsFinder().FindByHash(ctx, img.Hash); err == nil {
			return nil
		}
	}

	if f == nil {
		f, err := os.Open(img.Path)
		if err != nil {
			return ctxd.WrapError(ctx, err, "open image file")
		}
		i.closeFile(ctx, f)
	}

	m, err := ReadMeta(f)
	if err != nil {
		return ctxd.WrapError(ctx, err, "read image meta")
	}

	m.Exif.Hash = img.Hash
	if err := i.deps.PhotoExifEnsurer().Ensure(ctx, m.Exif); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to store image meta",
			"error", err.Error(), "exif", m.Exif)
	}

	if m.GpsInfo != nil {
		m.GpsInfo.Hash = img.Hash
		if err := i.deps.PhotoGpsEnsurer().Ensure(ctx, *m.GpsInfo); err != nil {
			i.deps.CtxdLogger().Error(ctx, "failed to store image gps",
				"error", err.Error(), "gps", m.GpsInfo)
		}
	}

	return nil
}

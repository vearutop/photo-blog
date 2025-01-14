package image

import (
	"bytes"
	"context"
	"errors"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bool64/brick/opencensus"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	blurhash "github.com/buckket/go-blurhash"
	"github.com/corona10/goimagehash"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/geo/ors"
	"github.com/vearutop/photo-blog/internal/infra/image/cloudflare"
	"github.com/vearutop/photo-blog/internal/infra/image/faces"
	"go.opencensus.io/trace"
)

type indexerDeps interface {
	CtxdLogger() ctxd.Logger
	StatsTracker() stats.Tracker

	PhotoThumbnailer() photo.Thumbnailer

	PhotoImageUpdater() uniq.Updater[photo.Image]

	PhotoExifEnsurer() uniq.Ensurer[photo.Exif]
	PhotoExifFinder() uniq.Finder[photo.Exif]

	PhotoGpsEnsurer() uniq.Ensurer[photo.Gps]
	PhotoGpsFinder() uniq.Finder[photo.Gps]

	PhotoMetaEnsurer() uniq.Ensurer[photo.Meta]
	PhotoMetaFinder() uniq.Finder[photo.Meta]

	CloudflareImageClassifier() *cloudflare.ImageClassifier
	CloudflareImageDescriber() *cloudflare.ImageDescriber
	FacesRecognizer() *faces.Recognizer
	OpenRouteService() *ors.Client
}

func NewIndexer(deps indexerDeps) *Indexer {
	i := &Indexer{
		deps:  deps,
		queue: make(chan indexJob, 10000),
	}

	go i.consume()

	return i
}

type Indexer struct {
	deps      indexerDeps
	queue     chan indexJob
	queueSize int64
}

func (i *Indexer) PhotoImageIndexer() photo.ImageIndexer {
	return i
}

func (i *Indexer) consume() {
	for j := range i.queue {
		qs := atomic.AddInt64(&i.queueSize, -1)
		i.deps.StatsTracker().Set(context.Background(), "indexing_images_pending", float64(qs))

		if j.cb != nil {
			j.cb(j.ctx)
			continue
		}

		if err := i.Index(j.ctx, j.img, j.flags); err != nil {
			i.deps.CtxdLogger().Error(j.ctx, "failed to index image", "img", j.img, "error", err, "flags", j.flags)
		}
	}
}

func (i *Indexer) closeFile(ctx context.Context, f *os.File) {
	if f == nil {
		return
	}

	if err := f.Close(); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to close file", "error", err.Error())
	}
}

type indexJob struct {
	ctx   context.Context
	img   photo.Image
	flags photo.IndexingFlags
	cb    func(ctx context.Context)
}

func (i *Indexer) QueueCallback(ctx context.Context, cb func(ctx context.Context)) {
	atomic.AddInt64(&i.queueSize, 1)
	i.queue <- indexJob{
		ctx: detachedContext{parent: ctx},
		cb:  cb,
	}
}

func (i *Indexer) QueueIndex(ctx context.Context, img photo.Image, flags photo.IndexingFlags) {
	i.deps.CtxdLogger().Info(ctx, "queueing image indexing", "img", img, "flags", flags)

	atomic.AddInt64(&i.queueSize, 1)
	i.queue <- indexJob{
		ctx:   detachedContext{parent: ctx},
		img:   img,
		flags: flags,
	}
}

func ensureImageDimensions(ctx context.Context, img *photo.Image, flags photo.IndexingFlags) (updated bool, err error) {
	if img.Height > 0 && img.Width > 0 && !flags.RebuildImageSize {
		return false, nil
	}

	f, err := os.Open(img.Path)
	if err != nil {
		return false, ctxd.WrapError(ctx, err, "open image file")
	}
	defer func() {
		if clErr := f.Close(); clErr != nil && err == nil {
			err = clErr
		}
	}()

	c, err := jpeg.DecodeConfig(f)
	if err != nil {
		return false, ctxd.WrapError(ctx, err, "image dimensions")
	}

	img.Width = int64(c.Width)
	img.Height = int64(c.Height)

	// Swap dimensions of rotated image.
	if img.Settings.Rotate == 90 || img.Settings.Rotate == 270 {
		img.Width = int64(c.Height)
		img.Height = int64(c.Width)
	}

	return true, nil
}

func (i *Indexer) Index(ctx context.Context, img photo.Image, flags photo.IndexingFlags) (err error) {
	ctx, done := opencensus.AddSpan(ctx, trace.StringAttribute("path", img.Path))
	defer done(&err)

	ctx = ctxd.AddFields(ctx, "img", img)

	if len(img.Settings.HTTPSources) > 1 {
		dup := map[string]bool{}
		res := make([]string, 0, 1)
		for _, src := range img.Settings.HTTPSources {
			if dup[src] {
				continue
			}

			res = append(res, src)
		}

		img.Settings.HTTPSources = res
		if err := i.deps.PhotoImageUpdater().Update(ctx, img); err != nil {
			return ctxd.WrapError(ctx, err, "dedup image sources")
		}
	} else if len(img.Settings.HTTPSources) == 0 {
		fi, err := os.Stat(img.Path)
		if err != nil {
			return ctxd.WrapError(ctx, err, "stat image file")
		}

		if fi.Size() != img.Size {
			img.Size = fi.Size()

			if err := i.deps.PhotoImageUpdater().Update(ctx, img); err != nil {
				return ctxd.WrapError(ctx, err, "update image size")
			}
		}
	}

	if err := i.ensureExif(ctx, &img, flags); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to ensure exif", "error", err)
	}

	if updated, err := ensureImageDimensions(ctx, &img, flags); err != nil {
		return err
	} else if updated {
		if err := i.deps.PhotoImageUpdater().Update(ctx, img); err != nil {
			return ctxd.WrapError(ctx, err, "update image to ensure dimensions")
		}
	}

	if img.TakenAt == nil {
		if exif, err := i.deps.PhotoExifFinder().FindByHash(ctx, img.Hash); err != nil {
			i.deps.CtxdLogger().Error(ctx, "failed to find exif", "error", err)
		} else {
			img.TakenAt = exif.Digitized
			if err := i.deps.PhotoImageUpdater().Update(ctx, img); err != nil {
				return ctxd.WrapError(ctx, err, "update image")
			}
		}
	}

	if !flags.SkipThumbnail {
		i.ensureThumbs(ctx, img)
		i.ensureBlurHash(ctx, &img)
		i.ensurePHash(ctx, &img)

		go i.ensureFacesRecognized(ctx, img)
		go i.ensureCFClassification(ctx, img)
		go i.ensureCFDescription(ctx, img)
	}

	go i.ensureGeoLabel(ctx, img.Hash)

	return nil
}

func (i *Indexer) ensureGeoLabel(ctx context.Context, hash uniq.Hash) {
	g, err := i.deps.PhotoGpsFinder().FindByHash(ctx, hash)
	if err != nil {
		if !errors.Is(err, status.NotFound) {
			i.deps.CtxdLogger().Error(ctx, "failed to find photo metadata", "error", err)
		}

		return
	}

	m, err := i.deps.PhotoMetaFinder().FindByHash(ctx, hash)
	if err != nil && !errors.Is(err, status.NotFound) {
		i.deps.CtxdLogger().Error(ctx, "failed to find photo metadata", "error", err)

		return
	}

	if m.Data.Val.GeoLabel != nil {
		return
	}

	label, err := i.deps.OpenRouteService().ReverseGeocode(ctx, g.Latitude, g.Longitude)
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to reverse geocode", "error", err, "gps", g)

		return
	}

	if _, err := i.deps.PhotoMetaEnsurer().Ensure(ctx, m, uniq.EnsureOption[photo.Meta]{
		Prepare: func(candidate, existing *photo.Meta) bool {
			if existing != nil {
				*candidate = *existing
			}
			candidate.Data.Val.GeoLabel = &label

			return false
		},
	}); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to ensure photo metadata", "error", err)
	}
}

func (i *Indexer) ensureCFClassification(ctx context.Context, img photo.Image) {
	ctx = ctxd.AddFields(ctx, "action", "cf_classify")

	m, err := i.deps.PhotoMetaFinder().FindByHash(ctx, img.Hash)
	if err != nil && !errors.Is(err, status.NotFound) {
		i.deps.CtxdLogger().Error(ctx, "failed to find photo metadata", "error", err)

		return
	}

	m.Hash = img.Hash

	// Check if already classified.
	if m.Data.Val.ImageClassification != nil {
		for _, l := range m.Data.Val.ImageClassification {
			if l.Model == cloudflare.ResNet50 {
				return
			}
		}
	}

	i.deps.CloudflareImageClassifier().Classify(ctx, img.Hash, func(labels []photo.ImageLabel) {
		if _, err := i.deps.PhotoMetaEnsurer().Ensure(ctx, m, uniq.EnsureOption[photo.Meta]{
			Prepare: func(candidate, existing *photo.Meta) bool {
				if existing != nil {
					*candidate = *existing
				}
				candidate.Data.Val.ImageClassification = append(candidate.Data.Val.ImageClassification, labels...)

				return false
			},
		}); err != nil {
			i.deps.CtxdLogger().Error(ctx, "failed to ensure photo metadata", "error", err)
		}
	})
}

func (i *Indexer) ensureCFDescription(ctx context.Context, img photo.Image) {
	ctx = ctxd.AddFields(ctx, "action", "cf_describe")

	m, err := i.deps.PhotoMetaFinder().FindByHash(ctx, img.Hash)
	if err != nil && !errors.Is(err, status.NotFound) {
		i.deps.CtxdLogger().Error(ctx, "failed to find photo metadata", "error", err)

		return
	}

	m.Hash = img.Hash

	// Check if already classified.
	if m.Data.Val.ImageClassification != nil {
		for _, l := range m.Data.Val.ImageClassification {
			if l.Model == cloudflare.UformGen2 {
				return
			}
		}
	}

	i.deps.CloudflareImageDescriber().Describe(ctx, img.Hash, func(label photo.ImageLabel) {
		if _, err := i.deps.PhotoMetaEnsurer().Ensure(ctx, m, uniq.EnsureOption[photo.Meta]{
			Prepare: func(candidate, existing *photo.Meta) bool {
				if existing != nil {
					*candidate = *existing
				}
				candidate.Data.Val.ImageClassification = append(candidate.Data.Val.ImageClassification, label)

				return false
			},
		}); err != nil {
			i.deps.CtxdLogger().Error(ctx, "failed to ensure photo metadata", "error", err)
		}
	})
}

func (i *Indexer) ensureFacesRecognized(ctx context.Context, img photo.Image) {
	ctx = ctxd.AddFields(ctx, "action", "faces")

	m, err := i.deps.PhotoMetaFinder().FindByHash(ctx, img.Hash)
	if err != nil && !errors.Is(err, status.NotFound) {
		i.deps.CtxdLogger().Error(ctx, "failed to find photo metadata", "error", err)

		return
	}

	m.Hash = img.Hash

	// Already recognized.
	if m.Data.Val.Faces != nil {
		return
	}

	th, err := i.deps.PhotoThumbnailer().Thumbnail(ctx, img, "2400w")
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to find thumbnail", "error", err)

		return
	}

	fn := th.FilePath
	var fr io.ReadCloser

	if strings.HasPrefix("https://", fn) || strings.HasPrefix("http://", fn) {
		resp, err := http.Get(fn)
		if err != nil {
			i.deps.CtxdLogger().Error(ctx, "failed to fetch image", "error", err)

			return
		}

		fr = resp.Body
	} else if fn == "" {
		fr = io.NopCloser(bytes.NewReader(th.Data))
	} else {
		f, err := os.Open(fn)
		if err != nil {
			i.deps.CtxdLogger().Error(ctx, "failed to open thumb file", "error", err)

			return
		}

		fr = f
	}

	f, err := i.deps.FacesRecognizer().Recognize(ctx, fr)
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to recognize faces", "error", err)

		return
	}

	if f == nil {
		return
	}

	if _, err := i.deps.PhotoMetaEnsurer().Ensure(ctx, m, uniq.EnsureOption[photo.Meta]{
		Prepare: func(candidate, existing *photo.Meta) bool {
			if existing != nil {
				*candidate = *existing
			}
			candidate.Data.Val.Faces = &f

			return false
		},
	}); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to ensure photo metadata", "error", err)
	}
}

func (i *Indexer) ensurePHash(ctx context.Context, img *photo.Image) {
	if img.PHash != 0 {
		return
	}

	th, err := i.deps.PhotoThumbnailer().Thumbnail(ctx, *img, "300w")
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to get thumbnail",
			"error", err.Error(), "size", "300w")
		return
	}

	j, err := thumbJPEG(ctx, th)
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to decode thumbnail",
			"error", err.Error())
		return
	}

	h, err := goimagehash.PerceptionHash(j)
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to encode perception hash",
			"error", err.Error())
		return
	}

	img.PHash = uniq.Hash(h.GetHash())

	if err := i.deps.PhotoImageUpdater().Update(ctx, *img); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to save image",
			"error", err.Error())
	}
}

func (i *Indexer) ensureBlurHash(ctx context.Context, img *photo.Image) {
	if img.BlurHash != "" {
		return
	}

	th, err := i.deps.PhotoThumbnailer().Thumbnail(ctx, *img, "300w")
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to get thumbnail",
			"error", err.Error(), "size", "300w")
		return
	}

	j, err := thumbJPEG(ctx, th)
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to decode thumbnail",
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

	if err := i.deps.PhotoImageUpdater().Update(ctx, *img); err != nil {
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

func readMeta(ctx context.Context, img *photo.Image) (m Meta, err error) {
	f, err := os.Open(img.Path)
	if err != nil {
		return m, ctxd.WrapError(ctx, err, "open image file")
	}

	defer func() {
		if clErr := f.Close(); clErr != nil && err == nil {
			err = clErr
		}
	}()

	m, err = ReadMeta(f)
	if err != nil {
		return m, ctxd.WrapError(ctx, err, "read image meta")
	}

	img.Settings.Rotate = m.Rotate

	exifQuirks(&m.Exif)

	return m, nil
}

func (i *Indexer) ensureExif(ctx context.Context, img *photo.Image, flags photo.IndexingFlags) error {
	exifExists, err := i.deps.PhotoExifFinder().Exists(ctx, img.Hash)
	if err != nil {
		return ctxd.WrapError(ctx, err, "check existing exif")
	}

	gpsExists, err := i.deps.PhotoGpsFinder().Exists(ctx, img.Hash)
	if err != nil {
		return ctxd.WrapError(ctx, err, "check existing gps")
	}

	if exifExists && !flags.RebuildExif && !flags.RebuildGps {
		return nil
	}

	ctx = ctxd.AddFields(ctx, "exifExists", exifExists, "gpsExists", gpsExists,
		"rebuildExif", flags.RebuildExif, "rebuildGps", flags.RebuildGps)

	m, err := readMeta(ctx, img)
	if err != nil {
		return err
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

func exifQuirks(exif *photo.Exif) {
	if exif.CameraModel == "SM-C200" {
		exif.ProjectionType = "equirectangular"
	}
}

// detachedContext exposes parent values, but suppresses parent cancellation.
type detachedContext struct {
	parent context.Context //nolint:containedctx // This wrapping is here on purpose.
}

func (d detachedContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (d detachedContext) Done() <-chan struct{} {
	return nil
}

func (d detachedContext) Err() error {
	return nil
}

func (d detachedContext) Value(key interface{}) interface{} {
	return d.parent.Value(key)
}

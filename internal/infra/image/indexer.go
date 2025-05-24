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
	"time"

	"github.com/bool64/brick/opencensus"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	blurhash "github.com/buckket/go-blurhash"
	"github.com/corona10/goimagehash"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/image-prompt/imageprompt"
	"github.com/vearutop/image-prompt/multi"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/topic"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/geo/ors"
	"github.com/vearutop/photo-blog/internal/infra/image/cloudflare"
	"github.com/vearutop/photo-blog/internal/infra/image/faces"
	"github.com/vearutop/photo-blog/internal/infra/image/sharpness"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/pkg/qlite"
	"go.opencensus.io/trace"
)

type indexerDeps interface {
	CtxdLogger() ctxd.Logger
	StatsTracker() stats.Tracker
	QueueBroker() *qlite.Broker

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
	ImagePrompter() *multi.ImagePrompter

	Settings() settings.Values
}

const (
	TopicSharpness = "index_sharpness"
)

func StartIndexer(deps indexerDeps) {
	i := &indexer{
		deps: deps,
	}

	b := deps.QueueBroker()

	must(
		qlite.AddConsumer[IndexJob](b, topic.IndexImage, i.index, func(o *qlite.ConsumerOptions) {
			o.Concurrency = 10
		}),
		qlite.AddConsumer[photo.Image](b, TopicSharpness, i.ensureSharpness),
	)
}

func must(errs ...error) {
	for _, err := range errs {
		if err != nil {
			panic(err)
		}
	}
}

type indexer struct {
	deps indexerDeps
}

type IndexJob struct {
	Image photo.Image         `json:"image"`
	Flags photo.IndexingFlags `json:"flags,omitzero"`
}

func (i *indexer) index(ctx context.Context, job IndexJob) error {
	return i.Index(ctx, job.Image, job.Flags)
}

func (i *indexer) closeFile(ctx context.Context, f *os.File) {
	if f == nil {
		return
	}

	if err := f.Close(); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to close file", "error", err.Error())
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

func (i *indexer) Index(ctx context.Context, img photo.Image, flags photo.IndexingFlags) (err error) {
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

	s := i.deps.Settings().Indexing()

	i.ensureThumbs(ctx, img, flags)
	i.ensureBlurHash(ctx, &img)

	if s.Phash {
		i.ensurePHash(ctx, &img)
	}

	if s.SharpnessV0 {
		i.ensureSharpness(ctx, img)
	}

	if s.Faces {
		go i.ensureFacesRecognized(ctx, img)
	}

	if s.CFClassification {
		go i.ensureCFClassification(ctx, img)
	}

	if s.CFDescription {
		go i.ensureCFDescription(ctx, img)
	}

	if s.GeoLabel {
		go i.ensureGeoLabel(ctx, img.Hash)
	}

	if s.LLMDescription {
		go i.ensureLLMDescription(ctx, img)
	}

	return nil
}

func (i *indexer) ensureGeoLabel(ctx context.Context, hash uniq.Hash) {
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

func (i *indexer) ensureCFClassification(ctx context.Context, img photo.Image) {
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

// Generate a detailed caption for this image, up to 100 words. Don't name the places, items or people unless you're sure.
func (i *indexer) ensureLLMDescription(ctx context.Context, img photo.Image) {
	ctx = ctxd.AddFields(ctx, "action", "llm_describe")

	m, err := i.deps.PhotoMetaFinder().FindByHash(ctx, img.Hash)
	if err != nil && !errors.Is(err, status.NotFound) {
		i.deps.CtxdLogger().Error(ctx, "failed to find photo metadata", "error", err)

		return
	}

	m.Hash = img.Hash

	if len(m.Data.Val.ImageDescriptions) > 0 {
		return
	}

	th, err := i.deps.PhotoThumbnailer().Thumbnail(ctx, img, photo.ThumbMid)
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to get thumb", "error", err)
	}

	for {
		rd, err := th.Reader()
		if err != nil {
			i.deps.CtxdLogger().Error(ctx, "failed to read thumb", "error", err)
			return
		}

		st := time.Now()

		res, err := i.deps.ImagePrompter().PromptImage(ctx, rd)
		rd.Close()

		if err != nil {
			if !errors.Is(err, imageprompt.ErrResourceExhausted) {
				i.deps.CtxdLogger().Warn(ctx, "image prompter failed", "error", err)
				return
			}

			time.Sleep(time.Minute)
			continue
		}

		if _, err := i.deps.PhotoMetaEnsurer().Ensure(ctx, m, uniq.EnsureOption[photo.Meta]{
			Prepare: func(candidate, existing *photo.Meta) bool {
				if existing != nil {
					*candidate = *existing
				}
				candidate.Data.Val.ImageDescriptions = append(candidate.Data.Val.ImageDescriptions, res)

				return false
			},
		}); err != nil {
			i.deps.CtxdLogger().Error(ctx, "failed to ensure photo metadata", "error", err)
		}

		i.deps.CtxdLogger().Info(ctx, "added LLM description", "st", st, "ela", time.Since(st).String(), "result", res)

		return
	}
}

func (i *indexer) ensureCFDescription(ctx context.Context, img photo.Image) {
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

func (i *indexer) ensureFacesRecognized(ctx context.Context, img photo.Image) {
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

	th, err := i.deps.PhotoThumbnailer().Thumbnail(ctx, img, "1200w")
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

func (i *indexer) ensurePHash(ctx context.Context, img *photo.Image) {
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

func (i *indexer) ensureSharpness(ctx context.Context, img photo.Image) error {
	if img.Sharpness != nil {
		return nil
	}

	jpg, err := loadImage(ctx, img, 10000, 10000)
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to load image",
			"error", err.Error())
	}

	sh, err := sharpness.FirstPercentile(Gray(jpg))
	if err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to calc sharpness",
			"error", err.Error())
	}

	img.Sharpness = &sh

	if err := i.deps.PhotoImageUpdater().Update(ctx, img); err != nil {
		i.deps.CtxdLogger().Error(ctx, "failed to save image",
			"error", err.Error())
	}

	return nil
}

func (i *indexer) ensureBlurHash(ctx context.Context, img *photo.Image) {
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

func (i *indexer) ensureThumbs(ctx context.Context, img photo.Image, flags photo.IndexingFlags) {
	s := i.deps.Settings().Indexing()

	for _, size := range photo.ThumbSizes {
		if size == "2400w" && s.Skip2400wThumb {
			continue
		}

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

func (i *indexer) ensureExif(ctx context.Context, img *photo.Image, flags photo.IndexingFlags) error {
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
	if err != nil && err.Error() != "read image meta: search exif: no exif data" {
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

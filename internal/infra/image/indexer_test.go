package image

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/stats"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/image-prompt/multi"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/geo/ors"
	"github.com/vearutop/photo-blog/internal/infra/image/cloudflare"
	faceinfra "github.com/vearutop/photo-blog/internal/infra/image/faces"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/pkg/qlite"
)

func TestIndexer_ReloadsCurrentImageBeforeUpdating(t *testing.T) {
	ctx := context.Background()
	filePath := "./testdata/20240919_144111.jpg"
	fi, err := os.Stat(filePath)
	require.NoError(t, err)

	current := photo.Image{
		File: uniq.File{
			Head: uniq.Head{Hash: 123},
			Path: filePath,
			Size: fi.Size(),
		},
		Settings: photo.ImageSettings{
			Description: "keep me",
		},
	}

	deps := testIndexerDeps{
		imageFinder: &stubImageFinder{current: current},
		imageUpdater: &recordingImageUpdater{},
		thumbs:      &stubThumbnailer{path: filePath},
	}

	staleJob := current
	staleJob.Settings.Description = ""

	err = (&indexer{deps: deps}).Index(ctx, staleJob, photo.IndexingFlags{})
	require.NoError(t, err)
	require.NotEmpty(t, deps.imageUpdater.updates)

	for _, updated := range deps.imageUpdater.updates {
		require.Equal(t, "keep me", updated.Settings.Description)
	}
}

type testIndexerDeps struct {
	imageFinder  *stubImageFinder
	imageUpdater *recordingImageUpdater
	thumbs       *stubThumbnailer
}

func (t testIndexerDeps) CtxdLogger() ctxd.Logger { return ctxd.NoOpLogger{} }
func (t testIndexerDeps) StatsTracker() stats.Tracker { return noopStats{} }
func (t testIndexerDeps) QueueBroker() *qlite.Broker { return nil }
func (t testIndexerDeps) PhotoThumbnailer() photo.Thumbnailer { return t.thumbs }
func (t testIndexerDeps) PhotoImageFinder() uniq.Finder[photo.Image] { return t.imageFinder }
func (t testIndexerDeps) PhotoImageUpdater() uniq.Updater[photo.Image] { return t.imageUpdater }
func (t testIndexerDeps) PhotoExifEnsurer() uniq.Ensurer[photo.Exif] { return noopEnsurer[photo.Exif]{} }
func (t testIndexerDeps) PhotoExifFinder() uniq.Finder[photo.Exif] { return noopFinder[photo.Exif]{} }
func (t testIndexerDeps) PhotoGpsEnsurer() uniq.Ensurer[photo.Gps] { return noopEnsurer[photo.Gps]{} }
func (t testIndexerDeps) PhotoGpsFinder() uniq.Finder[photo.Gps] { return noopFinder[photo.Gps]{} }
func (t testIndexerDeps) PhotoMetaEnsurer() uniq.Ensurer[photo.Meta] { return noopEnsurer[photo.Meta]{} }
func (t testIndexerDeps) PhotoMetaFinder() uniq.Finder[photo.Meta] { return noopFinder[photo.Meta]{} }
func (t testIndexerDeps) CloudflareImageClassifier() *cloudflare.ImageClassifier { return nil }
func (t testIndexerDeps) CloudflareImageDescriber() *cloudflare.ImageDescriber { return nil }
func (t testIndexerDeps) FacesRecognizer() *faceinfra.Recognizer { return nil }
func (t testIndexerDeps) OpenRouteService() *ors.Client { return nil }
func (t testIndexerDeps) ImagePrompter() *multi.ImagePrompter { return nil }
func (t testIndexerDeps) Settings() settings.Values { return testSettings{} }

type stubImageFinder struct {
	current photo.Image
}

func (s *stubImageFinder) FindByHash(context.Context, uniq.Hash) (photo.Image, error) { return s.current, nil }
func (s *stubImageFinder) FindByHashes(context.Context, ...uniq.Hash) ([]photo.Image, error) {
	return []photo.Image{s.current}, nil
}
func (s *stubImageFinder) Exists(context.Context, uniq.Hash) (bool, error) { return true, nil }
func (s *stubImageFinder) FindAll(context.Context) ([]photo.Image, error) { return []photo.Image{s.current}, nil }

type recordingImageUpdater struct {
	updates []photo.Image
}

func (r *recordingImageUpdater) Update(_ context.Context, value photo.Image, _ ...func(o *sqluct.Options)) error {
	r.updates = append(r.updates, value)
	return nil
}

type stubThumbnailer struct {
	path string
}

func (s *stubThumbnailer) Thumbnail(_ context.Context, image photo.Image, size photo.ThumbSize) (photo.Thumb, error) {
	return photo.Thumb{
		Head:   uniq.Head{Hash: image.Hash},
		Format: size,
		Data:   mustReadFile(s.path),
	}, nil
}

type noopEnsurer[V any] struct{}

func (noopEnsurer[V]) Ensure(_ context.Context, value V, _ ...uniq.EnsureOption[V]) (V, error) {
	return value, nil
}

type noopFinder[V any] struct{}

func (noopFinder[V]) FindByHash(context.Context, uniq.Hash) (V, error) {
	var zero V
	return zero, status.Wrap(errors.New("not found"), status.NotFound)
}
func (noopFinder[V]) FindByHashes(context.Context, ...uniq.Hash) ([]V, error) { return nil, nil }
func (noopFinder[V]) Exists(context.Context, uniq.Hash) (bool, error) { return false, nil }
func (noopFinder[V]) FindAll(context.Context) ([]V, error) { return nil, nil }

type noopStats struct{}

func (noopStats) Add(context.Context, string, float64, ...string) {}
func (noopStats) Set(context.Context, string, float64, ...string) {}

type testSettings struct{}

func (testSettings) Security() settings.Security { return settings.Security{} }
func (testSettings) Appearance() settings.Appearance { return settings.Appearance{} }
func (testSettings) Maps() settings.Maps { return settings.Maps{} }
func (testSettings) Visitors() settings.Visitors { return settings.Visitors{} }
func (testSettings) Storage() settings.Storage { return settings.Storage{} }
func (testSettings) Privacy() settings.Privacy { return settings.Privacy{} }
func (testSettings) ExternalAPI() settings.ExternalAPI { return settings.ExternalAPI{} }
func (testSettings) CFImageClassifier() cloudflare.ImageWorkerConfig { return cloudflare.ImageWorkerConfig{} }
func (testSettings) CFImageDescriber() cloudflare.ImageWorkerConfig { return cloudflare.ImageWorkerConfig{} }
func (testSettings) ORSConfig() ors.Config { return ors.Config{} }
func (testSettings) ImagePrompt() multi.Config { return multi.Config{} }
func (testSettings) Indexing() settings.Indexing { return settings.Indexing{} }

func mustReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	return data
}

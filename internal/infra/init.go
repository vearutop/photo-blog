package infra

import (
	"context"
	"io/fs"
	"net/http"
	"time"

	"github.com/bool64/brick"
	"github.com/bool64/brick/database"
	"github.com/bool64/brick/jaeger"
	"github.com/bool64/sqluct"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/swgui"
	"github.com/vearutop/photo-blog/internal/infra/image"
	"github.com/vearutop/photo-blog/internal/infra/schema"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/storage"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite_thumbs"
	_ "modernc.org/sqlite" // SQLite3 driver.
)

// NewServiceLocator creates application service locator.
func NewServiceLocator(cfg service.Config) (loc *service.Locator, err error) {
	l := &service.Locator{}

	defer func() {
		if err != nil && l != nil && l.LoggerProvider != nil {
			l.CtxdLogger().Error(context.Background(), err.Error())
		}
	}()

	l.BaseLocator, err = brick.NewBaseLocator(cfg.BaseConfig)
	if err != nil {
		return nil, err
	}

	if err = jaeger.Setup(cfg.Jaeger, l.BaseLocator); err != nil {
		return nil, err
	}

	l.SwaggerUIOptions = append(l.SwaggerUIOptions, func(cfg *swgui.Config) {
		cfg.HideCurl = true
	})

	schema.SetupOpenapiCollector(l.OpenAPI)

	l.HTTPServerMiddlewares = append(l.HTTPServerMiddlewares,
		gzip.Middleware,
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				h.ServeHTTP(writer, request)
			})
		},
	)

	if err = setupStorage(l, cfg.Database); err != nil {
		return nil, err
	}

	ir := storage.NewImageRepository(l.Storage)

	ar := storage.NewAlbumRepository(l.Storage, ir)
	l.PhotoAlbumEnsurerProvider = ar
	l.PhotoAlbumImageAdderProvider = ar
	l.PhotoAlbumImageDeleterProvider = ar

	albumRepo := storage.NewAlbumsRepository(l.Storage)
	l.PhotoAlbumAdderProvider = albumRepo
	l.PhotoAlbumUpdaterProvider = albumRepo
	l.PhotoAlbumFinderOldProvider = albumRepo
	l.PhotoAlbumDeleterProvider = albumRepo

	imageRepo := storage.NewImagesRepository(l.Storage)
	l.PhotoImageEnsurerProvider = image.NewHasher(imageRepo, l.CtxdLogger())
	l.PhotoImageUpdaterProvider = imageRepo
	l.PhotoImageFinderProvider = imageRepo

	thumbStorage, err := setupThumbStorage(l, cfg.ThumbStorage)
	if err != nil {
		return nil, err
	}
	l.PhotoThumbnailerProvider = storage.NewThumbRepository(thumbStorage, image.NewThumbnailer(l.CtxdLogger()))

	exifRepo := storage.NewExifRepository(l.Storage)
	l.PhotoExifFinderProvider = exifRepo
	l.PhotoExifEnsurerProvider = exifRepo

	gpsRepo := storage.NewGpsRepository(l.Storage)
	l.PhotoGpsFinderProvider = gpsRepo
	l.PhotoGpsEnsurerProvider = gpsRepo

	l.PhotoImageIndexerProvider = image.NewIndexer(l)

	return l, nil
}

func setupThumbStorage(l *service.Locator, filepath string) (*sqluct.Storage, error) {
	cfg := database.Config{}
	cfg.DriverName = "sqlite"
	cfg.MaxOpen = 1
	cfg.DSN = filepath
	cfg.ApplyMigrations = true

	l.CtxdLogger().Info(context.Background(), "setting up thumb storage")
	start := time.Now()
	st, err := database.SetupStorageDSN(cfg, l.CtxdLogger(), l.StatsTracker(), sqlite_thumbs.Migrations)
	if err != nil {
		return nil, err
	}
	l.CtxdLogger().Info(context.Background(), "thumb storage setup complete", "elapsed", time.Since(start).String())

	return st, nil
}

func setupStorage(l *service.Locator, cfg database.Config) error {
	if cfg.DriverName == "" {
		cfg.DriverName = "sqlite"
	}

	var (
		err        error
		migrations fs.FS
	)

	switch cfg.DriverName {
	case "sqlite":
		migrations = sqlite.Migrations
	}

	l.CtxdLogger().Info(context.Background(), "setting up storage")
	start := time.Now()
	l.Storage, err = database.SetupStorageDSN(cfg, l.CtxdLogger(), l.StatsTracker(), migrations)
	if err != nil {
		return err
	}
	l.CtxdLogger().Info(context.Background(), "storage setup complete", "elapsed", time.Since(start).String())

	return nil
}

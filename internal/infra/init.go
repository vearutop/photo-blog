package infra

import (
	"context"
	"github.com/bool64/brick"
	"github.com/bool64/brick/database"
	"github.com/bool64/brick/jaeger"
	"github.com/swaggest/rest/response/gzip"
	"github.com/vearutop/photo-blog/internal/infra/image"
	"github.com/vearutop/photo-blog/internal/infra/schema"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/storage"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite"
	"io/fs"
	_ "modernc.org/sqlite" // SQLite3 driver.
	"net/http"
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

	ar := storage.NewAlbumRepository(l.Storage)
	l.PhotoAlbumAdderProvider = ar
	l.PhotoAlbumFinderProvider = ar

	ir := storage.NewImageRepository(l.Storage)
	l.PhotoImageEnsurerProvider = image.NewHasher(ir, l.CtxdLogger())
	l.PhotoImageUpdaterProvider = ir
	l.PhotoThumbnailerProvider = storage.NewThumbRepository(l.Storage, image.NewThumbnailer())
	l.PhotoImageFinderProvider = ir

	return l, nil
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

	l.Storage, err = database.SetupStorageDSN(cfg, l.CtxdLogger(), l.StatsTracker(), migrations)
	if err != nil {
		return err
	}

	return nil
}

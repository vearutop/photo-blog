package infra

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/bool64/brick"
	"github.com/bool64/brick/database"
	"github.com/bool64/brick/jaeger"
	"github.com/bool64/sqluct"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/swgui"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/infra/image"
	"github.com/vearutop/photo-blog/internal/infra/schema"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/storage"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite_thumbs"
	"github.com/vearutop/photo-blog/internal/usecase/control"
	_ "modernc.org/sqlite" // SQLite3 driver.
)

// NewServiceLocator creates application service locator.
func NewServiceLocator(cfg service.Config, docsMode bool) (loc *service.Locator, err error) {
	l := &service.Locator{}
	l.Config = cfg

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
	l.SchemaRepo = schema.NewRepository(&l.OpenAPI.Reflector().Reflector)
	if err := setupSchemaRepo(l.SchemaRepo); err != nil {
		return nil, err
	}

	if docsMode {
		return l, nil
	}

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
	l.PhotoImageEnsurerProvider = image.NewHasher(ir, l.CtxdLogger())
	l.PhotoImageUpdaterProvider = ir
	l.PhotoImageFinderProvider = ir

	ar := storage.NewAlbumRepository(l.Storage, ir)
	l.PhotoAlbumEnsurerProvider = ar
	l.PhotoAlbumUpdaterProvider = ar
	l.PhotoAlbumFinderProvider = ar
	l.PhotoAlbumImageAdderProvider = ar
	l.PhotoAlbumImageFinderProvider = ar
	l.PhotoAlbumImageDeleterProvider = ar

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

	labelRepo := storage.NewLabelRepository(l.Storage)
	l.TextLabelFinderProvider = labelRepo
	l.TextLabelEnsurerProvider = labelRepo
	l.TextLabelEnsurerProvider = labelRepo

	visitorRepo := storage.NewVisitorRepository(l.Storage)
	l.AuthVisitorEnsurerProvider = visitorRepo
	l.AuthVisitorFinderProvider = visitorRepo

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
		if !strings.Contains(cfg.DSN, "?") {
			cfg.DSN += "?_time_format=sqlite"
		}
	}

	l.CtxdLogger().Info(context.Background(), "setting up storage")
	start := time.Now()
	l.Storage, err = database.SetupStorageDSN(cfg, l.CtxdLogger(), l.StatsTracker(), migrations)
	if err != nil {
		return err
	}
	l.CtxdLogger().Info(context.Background(), "storage setup complete", "elapsed", time.Since(start).String())

	l.Storage.Trace = func(ctx context.Context, stmt string, args []interface{}) (newCtx context.Context, onFinish func(error)) {
		return ctx, func(err error) {
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				l.CtxdLogger().Warn(ctx, "sql failed",
					"stmt", stmt, "args", args, "error", err.Error())
			}
		}
	}

	return nil
}

func setupSchemaRepo(r *schema.Repository) error {
	return firstFail(
		r.AddSchema("update-image-input", control.UpdateImageInput{}),
		r.AddSchema("album", photo.Album{}),
		r.AddSchema("image", photo.Image{}),
		r.AddSchema("gps", photo.Gps{}),
		r.AddSchema("exif", photo.Gps{}),
	)
}

func firstFail(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

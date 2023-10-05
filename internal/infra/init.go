package infra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bool64/brick"
	"github.com/bool64/brick/database"
	"github.com/bool64/brick/jaeger"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/zapctxd"
	"github.com/swaggest/jsonform-go"
	"github.com/swaggest/refl"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/swgui"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/infra/image"
	"github.com/vearutop/photo-blog/internal/infra/schema"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/storage"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite_thumbs"
	"go.uber.org/zap"
	_ "modernc.org/sqlite" // SQLite3 driver.
)

// NewServiceLocator creates application service locator.
func NewServiceLocator(cfg service.Config, docsMode bool) (loc *service.Locator, err error) {
	l := &service.Locator{}
	l.Config = cfg

	ctx := context.Background()

	defer func() {
		if err != nil && l != nil && l.LoggerProvider != nil {
			l.CtxdLogger().Error(ctx, err.Error())
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
	l.SchemaRepo = jsonform.NewRepository(&l.OpenAPI.Reflector().Reflector)
	if err := l.SchemaRepo.Add(
		service.Settings{},
		photo.Album{},
		photo.Image{},
	); err != nil {
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

	if err = l.Storage.DB().Get(&l.Config, "SELECT settings FROM app"); err != nil {
		return nil, err
	}

	if err = setupAccessLog(l); err != nil {
		return nil, err
	}

	ir := storage.NewImageRepository(l.Storage)
	l.PhotoImageEnsurerProvider = ir
	l.PhotoImageUpdaterProvider = ir
	l.PhotoImageFinderProvider = ir

	ar := storage.NewAlbumRepository(l.Storage, ir)
	l.PhotoAlbumEnsurerProvider = ar
	l.PhotoAlbumUpdaterProvider = ar
	l.PhotoAlbumFinderProvider = ar
	l.PhotoAlbumDeleterProvider = ar
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

	gpxRepo := storage.NewGpxRepository(l.Storage)
	l.PhotoGpxFinderProvider = gpxRepo
	l.PhotoGpxEnsurerProvider = gpxRepo

	labelRepo := storage.NewLabelRepository(l.Storage)
	l.TextLabelFinderProvider = labelRepo
	l.TextLabelEnsurerProvider = labelRepo
	l.TextLabelDeleterProvider = labelRepo

	visitorRepo := storage.NewVisitorRepository(l.Storage)
	l.AuthVisitorEnsurerProvider = visitorRepo
	l.AuthVisitorFinderProvider = visitorRepo

	l.PhotoImageIndexerProvider = image.NewIndexer(l)

	if err := refl.NoEmptyFields(l); err != nil {
		return nil, err
	}

	l.CtxdLogger().Important(ctx, "service locator initialized successfully")

	return l, nil
}

func setupAccessLog(l *service.Locator) error {
	cfg := l.Config

	if cfg.Settings.AccessLogFile == "" {
		l.AccessLogger = ctxd.NoOpLogger{}
	} else {
		f, err := os.OpenFile(cfg.Settings.AccessLogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
		if err != nil {
			return fmt.Errorf("access log: %w", err)
		}

		al := zapctxd.New(zapctxd.Config{
			Level:  zap.InfoLevel,
			Output: f,
		})

		l.OnShutdown("access-log", func() {
			if err := al.ZapLogger().Sync(); err != nil {
				l.CtxdLogger().Error(context.Background(), "failed to sync access log", "error", err)
			}

			if err := f.Close(); err != nil {
				l.CtxdLogger().Error(context.Background(), "failed to close access log file", "error", err)
			}
		})

		l.AccessLogger = al
	}

	return nil
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

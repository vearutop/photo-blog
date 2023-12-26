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
	"github.com/bool64/sqluct"
	"github.com/bool64/zapctxd"
	"github.com/swaggest/jsonform-go"
	"github.com/swaggest/refl"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/swgui"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/files"
	"github.com/vearutop/photo-blog/internal/infra/image"
	"github.com/vearutop/photo-blog/internal/infra/schema"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/storage"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite_thumbs"
	"github.com/vearutop/photo-blog/pkg/txt"
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

	cfg.Debug.Middlewares = append(cfg.Debug.Middlewares, auth.BasicAuth("Admin Access", l.Settings))

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

	l.DepCacheInstance = dep.NewCache(l.CacheInvalidationIndex())

	if err = setupUploadStorage(cfg.StoragePath); err != nil {
		return nil, err
	}

	if err = setupStorage(l, cfg.StoragePath+"db.sqlite"); err != nil {
		return nil, err
	}

	if l.SettingsManagerInstance, err = settings.NewManager(storage.NewSettingsRepository(l.Storage), l.DepCache()); err != nil {
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

	thumbStorage, err := setupThumbStorage(l, cfg.StoragePath+"thumbs.sqlite")
	if err != nil {
		return nil, err
	}
	l.PhotoThumbnailerProvider = storage.NewThumbRepository(thumbStorage, image.NewThumbnailer(l))

	exifRepo := storage.NewExifRepository(l.Storage)
	l.PhotoExifFinderProvider = exifRepo
	l.PhotoExifEnsurerProvider = exifRepo

	gpsRepo := storage.NewGpsRepository(l.Storage)
	l.PhotoGpsFinderProvider = gpsRepo
	l.PhotoGpsEnsurerProvider = gpsRepo

	gpxRepo := storage.NewGpxRepository(l.Storage)
	l.PhotoGpxFinderProvider = gpxRepo
	l.PhotoGpxEnsurerProvider = gpxRepo

	l.PhotoImageIndexerProvider = image.NewIndexer(l)
	l.TxtRendererProvider = txt.NewRenderer()

	l.FilesProcessorInstance = files.NewProcessor(l)

	if err := refl.NoEmptyFields(l); err != nil {
		return nil, err
	}

	if err := ensureFeaturedAlbum(l); err != nil {
		return nil, err
	}

	l.CtxdLogger().Important(ctx, "service locator initialized successfully")

	return l, nil
}

func setupAccessLog(l *service.Locator) error {
	cfg := l.Config

	f, err := os.OpenFile(cfg.StoragePath+"/access.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
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

	return nil
}

func setupThumbStorage(l *service.Locator, filepath string) (*sqluct.Storage, error) {
	cfg := database.Config{}
	cfg.DriverName = "sqlite"
	cfg.MaxOpen = 1
	cfg.DSN = filepath + "?_time_format=sqlite"
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

func setupStorage(l *service.Locator, filepath string) error {
	cfg := database.Config{}
	cfg.DriverName = "sqlite"
	cfg.MaxOpen = 1
	cfg.DSN = filepath + "?_time_format=sqlite"
	cfg.ApplyMigrations = true

	var err error

	l.CtxdLogger().Info(context.Background(), "setting up storage")
	start := time.Now()
	l.Storage, err = database.SetupStorageDSN(cfg, l.CtxdLogger(), l.StatsTracker(), sqlite.Migrations)
	if err != nil {
		return fmt.Errorf("main db: %w", err)
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

func setupUploadStorage(p string) error {
	if p == "" {
		return nil
	}

	// Create temporary directory for TUS uploads.
	p = strings.TrimSuffix(p, "/") + "/temp"

	if _, err := os.Stat(p); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Try to create upload directory if it does not exist.
			err = os.MkdirAll(p, 0o700)
		}

		if err != nil {
			return fmt.Errorf("init upload storage %s: %w", p, err)
		}
	}

	return nil
}

func ensureFeaturedAlbum(l *service.Locator) error {
	featured := l.Settings().Appearance().FeaturedAlbumName
	if featured == "" {
		return nil
	}

	ctx := context.Background()

	exists, err := l.PhotoAlbumFinder().Exists(ctx, uniq.StringHash(featured))
	if err != nil {
		return fmt.Errorf("featured exists: %w", err)
	}

	if exists {
		return nil
	}

	a := photo.Album{}
	a.Title = "Featured Photos"
	a.Name = featured
	a.Hash = uniq.StringHash(featured)
	a.Hidden = true

	if _, err = l.PhotoAlbumEnsurer().Ensure(ctx, a); err != nil {
		return fmt.Errorf("create featured: %w", err)
	}

	return nil
}

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
	"github.com/bool64/cache"
	"github.com/bool64/sqluct"
	"github.com/bool64/zapctxd"
	"github.com/swaggest/jsonform-go"
	"github.com/swaggest/refl"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/swgui"
	"github.com/vearutop/dbcon/dbcon"
	"github.com/vearutop/image-prompt/multi"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/files"
	"github.com/vearutop/photo-blog/internal/infra/geo/ors"
	"github.com/vearutop/photo-blog/internal/infra/image"
	"github.com/vearutop/photo-blog/internal/infra/image/cloudflare"
	"github.com/vearutop/photo-blog/internal/infra/image/faces"
	"github.com/vearutop/photo-blog/internal/infra/queue"
	"github.com/vearutop/photo-blog/internal/infra/schema"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/storage"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite_stats"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite_thumbs"
	"github.com/vearutop/photo-blog/internal/infra/storage/visitor"
	"github.com/vearutop/photo-blog/pkg/qlite"
	"github.com/vearutop/photo-blog/pkg/sqlitec"
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

	if err = setupUploadStorage(cfg.StoragePath); err != nil {
		return nil, err
	}

	if err = os.Chdir(cfg.StoragePath); err != nil {
		return nil, fmt.Errorf("change dir to storage path: %w", err)
	}

	if l.Storage, err = setupStorage(l, "db", sqlite.Migrations); err != nil {
		return nil, err
	}

	queueStorage, err := setupStorage(l, "queue", qlite.Migrations)
	if err != nil {
		return nil, err
	}

	l.QueueBrokerInstance = qlite.NewBroker(queueStorage)
	l.QueueBroker().Logger = l.CtxdLogger()

	l.DepCacheInstance = dep.NewCache(l)

	mapTilesStorage, err := setupStorage(l, "map-tiles", sqlitec.Migrations)
	if err != nil {
		return nil, err
	}

	l.MapTilesCacheInstance = brick.MakeCacheOf[[]byte](l, "map-tiles", 7*24*time.Hour,
		func(cfg *cache.FailoverConfigOf[[]byte]) {
			cfg.Backend = sqlitec.NewDBMapOf[[]byte](mapTilesStorage, func(cfg *cache.Config) {
				cfg.CountSoftLimit = 1000
				cfg.DeleteExpiredJobInterval = time.Hour
				cfg.DeleteExpiredAfter = 2 * time.Hour
			})
		},
	)

	if l.SettingsManagerInstance, err = settings.NewManager(storage.NewSettingsRepository(l.Storage), l.DepCache()); err != nil {
		return nil, err
	}

	l.CloudflareImageClassifierInstance = cloudflare.NewImageClassifier(l.CtxdLogger(), l.Settings().CFImageClassifier)
	l.CloudflareImageDescriberInstance = cloudflare.NewImageDescriber(l.CtxdLogger(), l.Settings().CFImageDescriber)
	l.FacesRecognizerInstance = faces.NewRecognizer(l.CtxdLogger(), l.Settings().ExternalAPI().FacesRecognizer)
	l.ORS = ors.NewORS(l, l.Settings().ORSConfig)
	l.ImagePrompterInstance = multi.NewImagePrompter(l.Settings().ImagePrompt)

	if err = setupAccessLog(l); err != nil {
		return nil, err
	}

	ir := storage.NewImageRepository(l.Storage)
	l.PhotoImageEnsurerProvider = ir
	l.PhotoImageUpdaterProvider = ir
	l.PhotoImageFinderProvider = ir

	metaRepo := storage.NewMetaRepository(l.Storage)
	l.PhotoMetaFinderProvider = metaRepo
	l.PhotoMetaEnsurerProvider = metaRepo

	ar := storage.NewAlbumRepository(l.Storage, ir, metaRepo)
	l.PhotoAlbumEnsurerProvider = ar
	l.PhotoAlbumUpdaterProvider = ar
	l.PhotoAlbumFinderProvider = ar
	l.PhotoAlbumDeleterProvider = ar
	l.PhotoAlbumImageAdderProvider = ar
	l.PhotoAlbumImageFinderProvider = ar
	l.PhotoAlbumImageDeleterProvider = ar

	fr := storage.NewFavoriteRepository(l.Storage)
	l.FavoriteRepositoryProvider = fr

	statsStorage, err := setupStorage(l, "stats", sqlite_stats.Migrations)
	if err != nil {
		return nil, err
	}

	l.VisitorStatsInstance, err = visitor.NewStats(statsStorage, l.CtxdLogger())
	if err != nil {
		return nil, err
	}

	thumbStorage, err := setupStorage(l, "thumbs", sqlite_thumbs.Migrations)
	if err != nil {
		return nil, err
	}
	l.PhotoThumbnailerProvider = storage.NewThumbRepository(thumbStorage, image.NewThumbnailer(l), l.CtxdLogger())

	exifRepo := storage.NewExifRepository(l.Storage)
	l.PhotoExifFinderProvider = exifRepo
	l.PhotoExifEnsurerProvider = exifRepo

	gpsRepo := storage.NewGpsRepository(l.Storage)
	l.PhotoGpsFinderProvider = gpsRepo
	l.PhotoGpsEnsurerProvider = gpsRepo

	gpxRepo := storage.NewGpxRepository(l.Storage)
	l.PhotoGpxFinderProvider = gpxRepo
	l.PhotoGpxEnsurerProvider = gpxRepo

	visitorRepo := storage.NewVisitorRepository(l.Storage)
	l.SiteVisitorFinderProvider = visitorRepo
	l.SiteVisitorEnsurerProvider = visitorRepo

	messageRepo := storage.NewMessageRepository(l.Storage)
	l.CommentMessageEnsurerProvider = messageRepo
	l.CommentMessageFinderProvider = messageRepo

	threadRepo := storage.NewThreadRepository(l.Storage)
	l.CommentThreadEnsurerProvider = threadRepo
	l.CommentThreadFinderProvider = threadRepo

	image.StartIndexer(l)
	l.TxtRendererProvider = txt.NewRenderer()

	l.FilesProcessorInstance = files.NewProcessor(l)

	if err := refl.NoEmptyFields(l); err != nil {
		return nil, err
	}

	if err := ensureFeaturedAlbum(l); err != nil {
		return nil, err
	}

	l.CtxdLogger().Important(ctx, "service locator initialized successfully")

	if err := queue.SetupBroker(l); err != nil {
		return nil, err
	}

	go func() {
		for {
			l.QueueBroker().Poll()
			<-time.Tick(time.Minute)
		}
	}()

	return l, nil
}

func setupAccessLog(l *service.Locator) error {
	f, err := os.OpenFile("access.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
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

func setupStorage(l *service.Locator, name string, migrations fs.FS) (*sqluct.Storage, error) {
	cfg := database.Config{}
	cfg.DriverName = "sqlite"
	cfg.MaxOpen = 1
	cfg.MaxIdle = 1
	cfg.DSN = name + ".sqlite?_time_format=sqlite"
	cfg.ApplyMigrations = true
	cfg.MethodSkipPackages = []string{"github.com/vearutop/photo-blog/internal/infra/storage/hashed"}

	var err error

	l.CtxdLogger().Info(context.Background(), "setting up storage", "name", name)
	start := time.Now()
	st, err := database.SetupStorageDSN(cfg, l.CtxdLogger(), l.StatsTracker(), migrations)
	if err != nil {
		return nil, fmt.Errorf("%s db: %w", name, err)
	}
	l.CtxdLogger().Info(context.Background(), "storage setup complete", "name", name, "elapsed", time.Since(start).String())

	st.Trace = func(ctx context.Context, stmt string, args []interface{}) (newCtx context.Context, onFinish func(error)) {
		return context.WithoutCancel(ctx), func(err error) {
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				l.CtxdLogger().Warn(ctx, "sql failed",
					"name", name, "stmt", stmt, "args", args, "error", err.Error())
			}
		}
	}

	l.AddDBConInstance(dbcon.DBInstance{
		Name:     name,
		Dialect:  sqluct.DialectSQLite3,
		Instance: st.DB().DB,
	})

	return st, nil
}

func setupUploadStorage(p string) error {
	if p == "" {
		return errors.New("storage path is empty")
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

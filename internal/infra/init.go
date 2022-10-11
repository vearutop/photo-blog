package infra

import (
	"context"
	"io/fs"
	"time"

	"github.com/bool64/brick"
	"github.com/bool64/brick/database"
	"github.com/bool64/brick/jaeger"
	"github.com/swaggest/rest/response/gzip"
	"github.com/vearutop/photo-blog/internal/domain/greeting"
	"github.com/vearutop/photo-blog/internal/infra/cached"
	"github.com/vearutop/photo-blog/internal/infra/schema"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/storage"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite"
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

	schema.SetupOpenapiCollector(l.OpenAPI)

	l.HTTPServerMiddlewares = append(l.HTTPServerMiddlewares, gzip.Middleware)

	if err = setupStorage(l, cfg.Database); err != nil {
		return nil, err
	}

	gs := &storage.GreetingSaver{
		Upstream: &greeting.SimpleMaker{},
		Storage:  l.Storage,
		Stats:    l.StatsTracker(),
	}

	l.GreetingMakerProvider = gs
	l.GreetingClearerProvider = gs

	greetingsCache := brick.MakeCacheOf[string](l.BaseLocator, "greetings", 3*time.Minute)
	l.GreetingMakerProvider = cached.NewGreetingMaker(l.GreetingMaker(), greetingsCache)

	if err := l.TransferCache(context.Background()); err != nil {
		l.CtxdLogger().Warn(context.Background(), "failed to transfer cache", "error", err)
	}

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

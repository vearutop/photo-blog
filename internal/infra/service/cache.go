package service

import (
	"time"

	"github.com/bool64/cache"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/stats"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/pkg/sqlitec"
)

// MakePersistentCacheOf creates a SQLite-backed failover cache and registers it for persistent invalidation.
func MakePersistentCacheOf[V any](l interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PersistentCacheStorage() *sqluct.Storage
	DepCache() *dep.Cache
}, name string, ttl time.Duration, options ...func(cfg *cache.FailoverConfigOf[V])) *cache.FailoverOf[V] {
	backend := sqlitec.NewDBMapOf[V](l.PersistentCacheStorage(), name, func(cfg *cache.ConfigOf[V]) {
			cfg.Name = name
			cfg.Logger = l.CtxdLogger()
			cfg.Stats = l.StatsTracker()
			cfg.TimeToLive = ttl
			cfg.DeleteExpiredAfter = 2 * time.Hour
			cfg.DeleteExpiredJobInterval = time.Hour
		})

	l.DepCache().PersistentInvalidationIndex().AddCache(name, backend)

	cfg := cache.FailoverConfigOf[V]{}
	cfg.Name = name
	cfg.Logger = l.CtxdLogger()
	cfg.Stats = l.StatsTracker()
	cfg.Backend = backend

	for _, option := range options {
		option(&cfg)
	}

	return cache.NewFailoverOf[V](func(c *cache.FailoverConfigOf[V]) {
		*c = cfg
	})
}

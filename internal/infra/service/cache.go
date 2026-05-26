package service

import (
	"context"
	"time"

	"github.com/bool64/cache"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/stats"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/pkg/sqlitec"
)

type namedBackendOf[V any] struct {
	name    string
	backend *sqlitec.DBMapOf[V]
}

func (n namedBackendOf[V]) Read(ctx context.Context, key []byte) (V, error) {
	return n.backend.Read(ctx, n.key(key))
}

func (n namedBackendOf[V]) Write(ctx context.Context, key []byte, val V) error {
	return n.backend.Write(ctx, n.key(key), val)
}

func (n namedBackendOf[V]) Delete(ctx context.Context, key []byte) error {
	return n.backend.Delete(ctx, n.key(key))
}

func (n namedBackendOf[V]) key(key []byte) []byte {
	buf := make([]byte, 0, len(n.name)+1+len(key))
	buf = append(buf, n.name...)
	buf = append(buf, ':')
	buf = append(buf, key...)

	return buf
}

// MakePersistentCacheOf creates a SQLite-backed failover cache and registers it for persistent invalidation.
func MakePersistentCacheOf[V any](l interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PersistentCacheStorage() *sqluct.Storage
	DepCache() *dep.Cache
}, name string, ttl time.Duration, options ...func(cfg *cache.FailoverConfigOf[V])) *cache.FailoverOf[V] {
	backend := namedBackendOf[V]{
		name: name,
		backend: sqlitec.NewDBMapOf[V](l.PersistentCacheStorage(), func(cfg *cache.ConfigOf[V]) {
			cfg.Name = name
			cfg.Logger = l.CtxdLogger()
			cfg.Stats = l.StatsTracker()
			cfg.TimeToLive = ttl
			cfg.DeleteExpiredAfter = 2 * time.Hour
			cfg.DeleteExpiredJobInterval = time.Hour
		}),
	}

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

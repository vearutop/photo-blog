package dep

import (
	"context"
	"fmt"

	"github.com/bool64/cache"
	"github.com/bool64/ctxd"
	"github.com/vearutop/photo-blog/internal/domain/topic"
	"github.com/vearutop/photo-blog/pkg/qlite"
)

type Deps interface {
	CacheInvalidationIndex() *cache.InvalidationIndex
	CtxdLogger() ctxd.Logger
	QueueBroker() *qlite.Broker
}

func NewCache(deps Deps) *Cache {
	c := &Cache{
		logger: deps.CtxdLogger(),
		index:  deps.CacheInvalidationIndex(),
	}

	if err := qlite.AddConsumer[string](deps.QueueBroker(), topic.AlbumChanged, func(ctx context.Context, v string) error {
		return c.AlbumChanged(ctx, v)
	}, func(o *qlite.ConsumerOptions) {
		o.Concurrency = 10
	}); err != nil {
		panic(err)
	}

	return c
}

type Cache struct {
	index  *cache.InvalidationIndex
	logger ctxd.Logger
}

type labelsCtxKey struct{}

// WithLabels adds cache labels to context.
func WithLabels(ctx context.Context, name string) context.Context {
	labels, _ := ctx.Value(labelsCtxKey{}).([]string)

	return context.WithValue(ctx, labelsCtxKey{}, append(labels[0:len(labels):len(labels)], name))
}

func (n *Cache) AddLabel(ctx context.Context, cacheName string, cacheKey []byte) {
	labels, _ := ctx.Value(labelsCtxKey{}).([]string)

	n.index.AddLabels(cacheName, cacheKey, labels...)
}

func (n *Cache) AlbumListDependency(cacheName string, cacheKey []byte) {
	n.index.AddLabels(cacheName, cacheKey, "album-list")
}

func (n *Cache) AlbumDependency(cacheName string, cacheKey []byte, albumNames ...string) {
	for i, s := range albumNames {
		albumNames[i] = "album/" + s
	}

	n.index.AddLabels(cacheName, cacheKey, albumNames...)
}

func (n *Cache) AlbumListChanged(ctx context.Context) error {
	_, err := n.index.InvalidateByLabels(ctx, "album-list")
	if err != nil {
		err = fmt.Errorf("album list changed: %w", err)
	}

	return err
}

func (n *Cache) AlbumChanged(ctx context.Context, name string) error {
	n.logger.Debug(ctx, "album changed", "name", name)

	_, err := n.index.InvalidateByLabels(ctx, "album/"+name)
	if err != nil {
		err = fmt.Errorf("album %s changed: %w", name, err)
	}

	return err
}

func (n *Cache) ServiceSettingsDependency(cacheName string, cacheKey []byte) {
	n.index.AddLabels(cacheName, cacheKey, "service-settings")
}

func (n *Cache) ServiceSettingsChanged(ctx context.Context) error {
	_, err := n.index.InvalidateByLabels(ctx, "service-settings")
	if err != nil {
		err = fmt.Errorf("service settings changed: %w", err)
	}

	return err
}

func (n *Cache) DepCache() *Cache {
	return n
}

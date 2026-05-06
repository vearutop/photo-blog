package dep

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/topic"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/qlite"
	"github.com/vearutop/photo-blog/pkg/sqlitec/invalidation"
)

type Deps interface {
	CtxdLogger() ctxd.Logger
	QueueBroker() *qlite.Broker
	PhotoAlbumUpdater() uniq.Updater[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
}

func NewCache(deps Deps, depStorage *sqluct.Storage) *Cache {
	c := &Cache{
		logger:           deps.CtxdLogger(),
		albumUpdater:     deps.PhotoAlbumUpdater(),
		albumImageFinder: deps.PhotoAlbumImageFinder(),
		index:            invalidation.NewIndex(depStorage),
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
	logger           ctxd.Logger
	albumUpdater     uniq.Updater[photo.Album]
	albumImageFinder photo.AlbumImageFinder
	index            *invalidation.Index
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
		return err
	}

	a := photo.Album{UpdatedAt: time.Now()}
	a.Hash = photo.AlbumHash(name)

	err = n.albumUpdater.Update(ctx, a, sqluct.Columns("updated_at"))
	if err != nil {
		err = fmt.Errorf("album %s changed: %w", name, err)
		return err
	}

	return nil
}

func (n *Cache) ImageChanged(ctx context.Context, hash uniq.Hash) error {
	albumsByImage, err := n.albumImageFinder.FindImageAlbums(ctx, 0, hash)
	if err != nil {
		return fmt.Errorf("find image albums: %w", err)
	}

	var errs []error
	for _, album := range albumsByImage[hash] {
		if album.Name == "" {
			continue
		}

		errs = append(errs, n.AlbumChanged(ctx, album.Name))
	}

	return errors.Join(errs...)
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

func (n *Cache) PersistentInvalidationIndex() *invalidation.Index {
	return n.index
}

func (n *Cache) ResetKey(ctx context.Context, cacheName string, cacheKey []byte) error {
	return n.index.ResetKey(ctx, cacheName, cacheKey)
}

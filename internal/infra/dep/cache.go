package dep

import (
	"context"
	"fmt"

	"github.com/bool64/cache"
	"github.com/bool64/ctxd"
)

func NewCache(index *cache.InvalidationIndex, logger ctxd.Logger) *Cache {
	return &Cache{
		logger: logger,
		index:  index,
	}
}

type Cache struct {
	index  *cache.InvalidationIndex
	logger ctxd.Logger
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

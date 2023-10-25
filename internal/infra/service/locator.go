package service

import (
	"github.com/bool64/brick"
	"github.com/bool64/cache"
	"github.com/bool64/ctxd"
	"github.com/swaggest/jsonform-go"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

// Locator defines application resources.
type Locator struct {
	*brick.BaseLocator

	SchemaRepo   *jsonform.Repository
	AccessLogger ctxd.Logger
	MapCache     *cache.FailoverOf[photo.MapTile]

	Config Config

	PhotoAlbumEnsurerProvider
	PhotoAlbumUpdaterProvider
	PhotoAlbumDeleterProvider
	PhotoAlbumFinderProvider

	PhotoAlbumImageAdderProvider
	PhotoAlbumImageFinderProvider
	PhotoAlbumImageDeleterProvider

	PhotoImageEnsurerProvider
	PhotoImageUpdaterProvider
	PhotoImageFinderProvider
	PhotoImageIndexerProvider

	PhotoThumbnailerProvider

	PhotoExifEnsurerProvider
	PhotoExifFinderProvider

	PhotoGpsEnsurerProvider
	PhotoGpsFinderProvider

	PhotoGpxEnsurerProvider
	PhotoGpxFinderProvider

	TxtRendererProvider
}

// ServiceConfig gives access to service configuration.
func (l *Locator) ServiceConfig() Config {
	return l.Config
}

func (l *Locator) SchemaRepository() *jsonform.Repository {
	return l.SchemaRepo
}

// ServiceSettings give access to dynamic service settings.
func (l *Locator) ServiceSettings() Settings {
	return l.Config.Settings
}

func (l *Locator) AccessLog() ctxd.Logger {
	return l.AccessLogger
}

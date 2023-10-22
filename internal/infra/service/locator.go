package service

import (
	"github.com/bool64/brick"
	"github.com/bool64/ctxd"
	"github.com/swaggest/jsonform-go"
)

// Locator defines application resources.
type Locator struct {
	*brick.BaseLocator

	SchemaRepo   *jsonform.Repository
	AccessLogger ctxd.Logger

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
}

// ServiceConfig gives access to service configuration.
func (l *Locator) ServiceConfig() Config {
	return l.Config
}

func (l *Locator) SchemaRepository() *jsonform.Repository {
	return l.SchemaRepo
}

func (l *Locator) ServiceSettings() Settings {
	return l.Config.Settings
}

func (l *Locator) AccessLog() ctxd.Logger {
	return l.AccessLogger
}

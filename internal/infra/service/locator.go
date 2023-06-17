package service

import (
	"github.com/bool64/brick"
	"github.com/vearutop/photo-blog/pkg/jsonform"
)

// Locator defines application resources.
type Locator struct {
	*brick.BaseLocator

	SchemaRepo *jsonform.Repository

	Config Config

	PhotoAlbumEnsurerProvider
	PhotoAlbumUpdaterProvider
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

	TextLabelEnsurerProvider
	TextLabelFinderProvider
	TextLabelDeleterProvider

	AuthVisitorEnsurerProvider
	AuthVisitorFinderProvider
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

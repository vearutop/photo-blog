package service

import (
	"github.com/bool64/brick"
	"github.com/vearutop/photo-blog/internal/infra/schema"
)

// Locator defines application resources.
type Locator struct {
	*brick.BaseLocator

	SchemaRepo *schema.Repository

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

func (l *Locator) SchemaRepository() *schema.Repository {
	return l.SchemaRepo
}

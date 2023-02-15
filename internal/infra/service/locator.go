package service

import (
	"github.com/bool64/brick"
)

// Locator defines application resources.
type Locator struct {
	*brick.BaseLocator

	PhotoAlbumEnsurerProvider
	PhotoAlbumImageAdderProvider
	PhotoAlbumImageDeleterProvider

	PhotoAlbumAdderProvider
	PhotoAlbumUpdaterProvider
	PhotoAlbumDeleterProvider
	PhotoAlbumFinderOldProvider

	PhotoImageEnsurerProvider
	PhotoImageUpdaterProvider
	PhotoImageFinderProvider
	PhotoImageIndexerProvider

	PhotoThumbnailerProvider

	PhotoExifEnsurerProvider
	PhotoExifFinderProvider

	PhotoGpsEnsurerProvider
	PhotoGpsFinderProvider
}

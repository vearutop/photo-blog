package service

import (
	"github.com/bool64/brick"
)

// Locator defines application resources.
type Locator struct {
	*brick.BaseLocator

	PhotoAlbumAdderProvider
	PhotoAlbumUpdaterProvider
	PhotoAlbumDeleterProvider
	PhotoAlbumFinderProvider

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

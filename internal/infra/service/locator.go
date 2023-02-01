package service

import (
	"github.com/bool64/brick"
)

// Locator defines application resources.
type Locator struct {
	*brick.BaseLocator

	PhotoAlbumAdderProvider
	PhotoAlbumFinderProvider

	PhotoImageEnsurerProvider
	PhotoImageUpdaterProvider
	PhotoImageFinderProvider

	PhotoThumbnailerProvider

	PhotoExifEnsurerProvider
	PhotoExifFinderProvider

	PhotoGpsEnsurerProvider
	PhotoGpsFinderProvider
}

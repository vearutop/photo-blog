package service

import (
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type PhotoAlbumAdderProvider interface {
	PhotoAlbumAdder() photo.AlbumAdder
}

type PhotoAlbumFinderProvider interface {
	PhotoAlbumFinder() photo.AlbumFinder
}

type PhotoImageEnsurerProvider interface {
	PhotoImageEnsurer() photo.ImageEnsurer
}

type PhotoImageUpdaterProvider interface {
	PhotoImageUpdater() photo.ImageUpdater
}

type PhotoImageFinderProvider interface {
	PhotoImageFinder() photo.ImageFinder
}

type PhotoThumbnailerProvider interface {
	PhotoThumbnailer() photo.Thumbnailer
}

type PhotoExifEnsurerProvider interface {
	PhotoExifEnsurer() photo.ExifEnsurer
}

type PhotoExifFinderProvider interface {
	PhotoExifFinder() photo.ExifFinder
}

type PhotoGpsEnsurerProvider interface {
	PhotoGpsEnsurer() photo.GpsEnsurer
}

type PhotoGpsFinderProvider interface {
	PhotoGpsFinder() photo.GpsFinder
}

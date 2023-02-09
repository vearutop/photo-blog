package service

import (
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type PhotoAlbumAdderProvider interface {
	PhotoAlbumAdder() photo.AlbumAdder
}

type PhotoAlbumUpdaterProvider interface {
	PhotoAlbumUpdater() photo.AlbumUpdater
}

type PhotoAlbumFinderProvider interface {
	PhotoAlbumFinder() photo.AlbumFinder
}

type PhotoAlbumDeleterProvider interface {
	PhotoAlbumDeleter() photo.AlbumDeleter
}

type PhotoImageEnsurerProvider interface {
	PhotoImageEnsurer() uniq.Ensurer[photo.Image]
}

type PhotoImageUpdaterProvider interface {
	PhotoImageUpdater() uniq.Updater[photo.Image]
}

type PhotoImageFinderProvider interface {
	PhotoImageFinder() uniq.Finder[photo.Image]
}

type PhotoImageIndexerProvider interface {
	PhotoImageIndexer() photo.ImageIndexer
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

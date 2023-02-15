package service

import (
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type PhotoAlbumImageAdderProvider interface {
	PhotoAlbumImageAdder() photo.AlbumImageAdder
}

type PhotoAlbumImageDeleterProvider interface {
	PhotoAlbumImageDeleter() photo.AlbumImageDeleter
}

type PhotoAlbumEnsurerProvider interface {
	PhotoAlbumEnsurer() uniq.Ensurer[photo.Album]
}

type PhotoAlbumAdderProvider interface {
	PhotoAlbumAdder() photo.AlbumAdder
}

type PhotoAlbumUpdaterProvider interface {
	PhotoAlbumUpdater() photo.AlbumUpdater
}

type PhotoAlbumFinderOldProvider interface {
	PhotoAlbumFinderOld() photo.AlbumFinder
}

type PhotoAlbumDeleterProvider interface {
	PhotoAlbumDeleter() photo.AlbumDeleter
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

type PhotoImageIndexerProvider interface {
	PhotoImageIndexer() photo.ImageIndexer
}

type PhotoThumbnailerProvider interface {
	PhotoThumbnailer() photo.Thumbnailer
}

type PhotoExifEnsurerProvider interface {
	PhotoExifEnsurer() uniq.Ensurer[photo.Exif]
}

type PhotoExifFinderProvider interface {
	PhotoExifFinder() uniq.Finder[photo.Exif]
}

type PhotoGpsEnsurerProvider interface {
	PhotoGpsEnsurer() uniq.Ensurer[photo.Gps]
}

type PhotoGpsFinderProvider interface {
	PhotoGpsFinder() uniq.Finder[photo.Gps]
}

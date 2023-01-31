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

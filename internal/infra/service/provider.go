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

type PhotoImageAdderProvider interface {
	PhotoImageAdder() photo.ImageAdder
}

type PhotoThumbAdderProvider interface {
	PhotoThumbAdder() photo.ThumbAdder
}

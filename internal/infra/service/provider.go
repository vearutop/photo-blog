package service

import (
	"github.com/bool64/ctxd"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/text"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
)

type PhotoAlbumImageAdderProvider interface {
	PhotoAlbumImageAdder() photo.AlbumImageAdder
}

type PhotoAlbumImageFinderProvider interface {
	PhotoAlbumImageFinder() photo.AlbumImageFinder
}

type PhotoAlbumImageDeleterProvider interface {
	PhotoAlbumImageDeleter() photo.AlbumImageDeleter
}

type PhotoAlbumEnsurerProvider interface {
	PhotoAlbumEnsurer() uniq.Ensurer[photo.Album]
}

type PhotoAlbumUpdaterProvider interface {
	PhotoAlbumUpdater() uniq.Updater[photo.Album]
}

type PhotoAlbumFinderProvider interface {
	PhotoAlbumFinder() uniq.Finder[photo.Album]
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

type TextLabelFinderProvider interface {
	TextLabelFinder() text.LabelFinder
}

type TextLabelEnsurerProvider interface {
	TextLabelEnsurer() text.LabelEnsurer
}

type TextLabelDeleterProvider interface {
	TextLabelDeleter() text.LabelDeleter
}

type AuthVisitorEnsurerProvider interface {
	AuthVisitorEnsurer() uniq.Ensurer[auth.Visitor]
}

type AuthVisitorFinderProvider interface {
	AuthVisitorFinder() uniq.Finder[auth.Visitor]
}

type AccessLogProvider interface {
	AccessLog() ctxd.Logger
}

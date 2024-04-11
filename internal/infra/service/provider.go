package service

import (
	"github.com/vearutop/photo-blog/internal/domain/comment"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/site"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/txt"
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

type PhotoAlbumDeleterProvider interface {
	PhotoAlbumDeleter() uniq.Deleter[photo.Album]
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

type PhotoMetaEnsurerProvider interface {
	PhotoMetaEnsurer() uniq.Ensurer[photo.Meta]
}

type PhotoMetaFinderProvider interface {
	PhotoMetaFinder() uniq.Finder[photo.Meta]
}

type PhotoGpxEnsurerProvider interface {
	PhotoGpxEnsurer() uniq.Ensurer[photo.Gpx]
}

type PhotoGpxFinderProvider interface {
	PhotoGpxFinder() uniq.Finder[photo.Gpx]
}

type TxtRendererProvider interface {
	TxtRenderer() *txt.Renderer
}

type SiteVisitorEnsurerProvider interface {
	SiteVisitorEnsurer() uniq.Ensurer[site.Visitor]
}

type SiteVisitorFinderProvider interface {
	SiteVisitorFinder() uniq.Finder[site.Visitor]
}

type CommentMessageEnsurerProvider interface {
	CommentMessageEnsurer() uniq.Ensurer[comment.Message]
}

type CommentMessageFinderProvider interface {
	CommentMessageFinder() uniq.Finder[comment.Message]
}

type CommentThreadEnsurerProvider interface {
	CommentThreadEnsurer() uniq.Ensurer[comment.Thread]
}

type CommentThreadFinderProvider interface {
	CommentThreadFinder() uniq.Finder[comment.Thread]
}

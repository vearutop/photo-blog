package control

import (
	"context"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/dep"
)

type addToAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoAlbumImageAdder() photo.AlbumImageAdder

	DepCache() *dep.Cache
}
type addToAlbumInput struct {
	DstAlbumName string    `path:"name" description:"Name of destination album to add photo."`
	SrcImageHash uniq.Hash `json:"image_hash,omitempty" title:"Image Hash" description:"Hash of an image to add to album."`
	SrcAlbumName string    `json:"album_name,omitempty" title:"Source Album Name" description:"Name of a source album to add photos from."`
}

// AddToAlbum creates use case interactor to add a single photo or photos from an album to another album.
func AddToAlbum(deps addToAlbumDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in addToAlbumInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "add_to_album", 1)
		deps.CtxdLogger().Info(ctx, "adding to album", "name", in.DstAlbumName, "hash", in.SrcImageHash)

		dstAlbum, err := deps.PhotoAlbumFinder().FindByHash(ctx, photo.AlbumHash(in.DstAlbumName))
		if err != nil {
			return err
		}

		if in.SrcAlbumName != "" {
			if images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, photo.AlbumHash(in.SrcAlbumName)); err == nil && len(images) > 0 {
				imgHashes := make([]uniq.Hash, 0, len(images))
				for _, img := range images {
					imgHashes = append(imgHashes, img.Hash)
				}

				return deps.PhotoAlbumImageAdder().AddImages(ctx, dstAlbum.Hash, imgHashes...)
			}
		}

		if in.SrcImageHash != 0 {
			img, err := deps.PhotoImageFinder().FindByHash(ctx, in.SrcImageHash)
			if err != nil {
				return err
			}

			err = deps.PhotoAlbumImageAdder().AddImages(ctx, dstAlbum.Hash, img.Hash)
		}

		if err == nil {
			err = deps.DepCache().AlbumChanged(ctx, dstAlbum.Name)
		}

		return err
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown)

	return u
}

package control

import (
	"context"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type addToAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoAlbumImageAdder() photo.AlbumImageAdder
}
type albumImageInput struct {
	Name string    `path:"name" description:"Name of destination album to add photo."`
	Hash uniq.Hash `path:"hash" description:"Hash of an image or album."`
}

// AddToAlbum creates use case interactor to add a photo to album.
func AddToAlbum(deps addToAlbumDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in albumImageInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "remove_from_album", 1)
		deps.CtxdLogger().Info(ctx, "removing from album", "name", in.Name, "hash", in.Hash)

		album, err := deps.PhotoAlbumFinder().FindByHash(ctx, photo.AlbumHash(in.Name))
		if err != nil {
			return err
		}

		if images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, in.Hash); err == nil && len(images) > 0 {
			imgHashes := make([]uniq.Hash, 0, len(images))
			for _, img := range images {
				imgHashes = append(imgHashes, img.Hash)
			}

			return deps.PhotoAlbumImageAdder().AddImages(ctx, album.Hash, imgHashes...)
		}

		img, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		return deps.PhotoAlbumImageAdder().AddImages(ctx, album.Hash, img.Hash)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown)

	return u
}

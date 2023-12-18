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

type removeFromAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoAlbumImageDeleter() photo.AlbumImageDeleter

	DepCache() *dep.Cache
}

type removeFromAlbumInput struct {
	AlbumName string    `path:"name" description:"Name of album to remove image from."`
	ImageHash uniq.Hash `path:"hash" description:"Hash of an image to remove from album."`
}

// RemoveFromAlbum creates use case interactor to delete a photo from album.
func RemoveFromAlbum(deps removeFromAlbumDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in removeFromAlbumInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "remove_from_album", 1)
		deps.CtxdLogger().Info(ctx, "removing from album", "name", in.AlbumName, "hash", in.ImageHash)

		albumHash := photo.AlbumHash(in.AlbumName)

		_, err := deps.PhotoAlbumFinder().FindByHash(ctx, albumHash)
		if err != nil {
			return err
		}

		images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, albumHash)
		if err != nil {
			return err
		}

		for _, img := range images {
			if img.Hash == in.ImageHash {
				return deps.PhotoAlbumImageDeleter().DeleteImages(ctx, albumHash, img.Hash)
			}
		}

		return deps.DepCache().AlbumChanged(ctx, in.AlbumName)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown)

	return u
}

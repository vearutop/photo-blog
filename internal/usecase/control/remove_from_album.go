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

type removeFromAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoAlbumImageDeleter() photo.AlbumImageDeleter
}

// RemoveFromAlbum creates use case interactor to delete a photo from album.
func RemoveFromAlbum(deps removeFromAlbumDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in albumImageInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "remove_from_album", 1)
		deps.CtxdLogger().Info(ctx, "removing from album", "name", in.Name, "hash", in.Hash)

		albumHash := photo.AlbumHash(in.Name)

		_, err := deps.PhotoAlbumFinder().FindByHash(ctx, albumHash)
		if err != nil {
			return err
		}

		images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, albumHash)
		if err != nil {
			return err
		}

		for _, img := range images {
			if img.Hash == in.Hash {
				return deps.PhotoAlbumImageDeleter().DeleteImages(ctx, albumHash, img.Hash)
			}
		}

		return nil
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown)

	return u
}

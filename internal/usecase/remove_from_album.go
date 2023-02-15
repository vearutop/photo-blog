package usecase

import (
	"context"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type removeFromAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinderOld() photo.AlbumFinder
	PhotoAlbumDeleter() photo.AlbumDeleter
}

// RemoveFromAlbum creates use case interactor to delete a photo from album.
func RemoveFromAlbum(deps removeFromAlbumDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in albumImageInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "remove_from_album", 1)
		deps.CtxdLogger().Info(ctx, "removing from album", "name", in.Name, "hash", in.Hash)

		album, err := deps.PhotoAlbumFinderOld().FindByName(ctx, in.Name)
		if err != nil {
			return err
		}

		images, err := deps.PhotoAlbumFinderOld().FindImages(ctx, album.ID)
		if err != nil {
			return err
		}

		for _, img := range images {
			if img.Hash == in.Hash {
				return deps.PhotoAlbumDeleter().DeleteImages(ctx, album.ID, img.ID)
			}
		}

		return nil
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown)

	return u
}

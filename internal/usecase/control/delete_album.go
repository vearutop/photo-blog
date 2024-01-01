package control

import (
	"context"
	"errors"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/dep"
)

type deleteAlbumDeps interface {
	removeFromAlbumDeps

	PhotoAlbumDeleter() uniq.Deleter[photo.Album]
	DepCache() *dep.Cache
}

type deleteAlbumInput struct {
	Name string `path:"name" description:"Name of album to delete."`
}

// DeleteAlbum creates use case interactor to delete album.
func DeleteAlbum(deps deleteAlbumDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in deleteAlbumInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "delete_album", 1)
		deps.CtxdLogger().Info(ctx, "deleting album", "name", in.Name)

		albumHash := photo.AlbumHash(in.Name)

		_, err := deps.PhotoAlbumFinder().FindByHash(ctx, albumHash)
		if err != nil {
			return err
		}

		images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, albumHash)
		if err != nil {
			return err
		}

		var errs []error

		for _, img := range images {
			errs = append(errs, deps.PhotoAlbumImageDeleter().DeleteImages(ctx, albumHash, img.Hash))
		}

		errs = append(errs, deps.PhotoAlbumDeleter().Delete(ctx, albumHash))

		err = errors.Join(errs...)

		if err == nil {
			err = errors.Join(
				deps.DepCache().AlbumListChanged(ctx),
				deps.DepCache().AlbumChanged(ctx, in.Name),
			)
		}

		return err
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown)

	return u
}

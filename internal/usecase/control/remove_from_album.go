package control

import (
	"context"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
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

type removeMultipleFromAlbumInput struct {
	AlbumName   string      `path:"name" description:"Name of album to remove images from."`
	ImageHashes []uniq.Hash `json:"image_hashes" title:"Image Hashes" description:"Hashes of images to remove from album."`
	CollabKey   string      `query:"collabKey" description:"Collaborator key to allow admin access."`
}

// RemoveMultipleFromAlbum creates use case interactor to delete one or more photos from album.
func RemoveMultipleFromAlbum(deps removeFromAlbumDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in removeMultipleFromAlbumInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "remove_from_album", float64(len(in.ImageHashes)))
		deps.CtxdLogger().Info(ctx, "removing from album", "name", in.AlbumName, "hashes", in.ImageHashes)

		if len(in.ImageHashes) == 0 {
			return nil
		}

		albumHash := photo.AlbumHash(in.AlbumName)

		album, err := deps.PhotoAlbumFinder().FindByHash(ctx, albumHash)
		if err != nil {
			return err
		}

		if !auth.IsAdmin(ctx) {
			if in.CollabKey == "" {
				return status.PermissionDenied
			}

			if album.Settings.CollabKey != in.CollabKey {
				return status.PermissionDenied
			}
		}

		if err := deps.PhotoAlbumImageDeleter().DeleteImages(ctx, albumHash, in.ImageHashes...); err != nil {
			return err
		}

		return deps.DepCache().AlbumChanged(ctx, in.AlbumName)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown)

	return u
}

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
	PhotoAlbumImageAdder() photo.AlbumImageAdder
}
type albumImageInput struct {
	Name string    `path:"name"`
	Hash uniq.Hash `path:"hash"`
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

		image, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		return deps.PhotoAlbumImageAdder().AddImages(ctx, album.Hash, image.Hash)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown)

	return u
}
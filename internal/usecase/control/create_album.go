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

type createAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PhotoAlbumEnsurer() uniq.Ensurer[photo.Album] // See storage.AlbumRepository.
}

// CreateAlbum creates use case interactor to add directory of photos.
func CreateAlbum(deps createAlbumDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in photo.Album, out *photo.Album) (err error) {
		deps.StatsTracker().Add(ctx, "create_album", 1)
		deps.CtxdLogger().Important(ctx, "creating album", "name", in.Name)

		in.Hash = uniq.StringHash(in.Name)

		*out, err = deps.PhotoAlbumEnsurer().Ensure(ctx, in)

		return err
	})

	u.SetDescription("Create a named album.")
	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

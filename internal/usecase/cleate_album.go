package usecase

import (
	"context"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type createAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PhotoAlbumAdder() photo.AlbumAdder
}

// CreateAlbum creates use case interactor to add directory of photos.
func CreateAlbum(deps createAlbumDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in photo.AlbumData, out *photo.Album) error {
		deps.StatsTracker().Add(ctx, "create_album", 1)
		deps.CtxdLogger().Important(ctx, "creating album", "name", in.Name)

		a, err := deps.PhotoAlbumAdder().Add(ctx, in)
		if err != nil {
			return err
		}

		*out = a

		return nil
	})

	u.SetDescription("Create a named album.")
	u.SetTags("Photos")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

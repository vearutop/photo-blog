package usecase

import (
	"context"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type updateAlbumDescDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PhotoAlbumFinderOld() photo.AlbumFinder
	PhotoAlbumUpdater() photo.AlbumUpdater
}

// UpdateAlbumDesc creates use case interactor to update album description.
func UpdateAlbumDesc(deps updateAlbumDescDeps) usecase.Interactor {
	type updateAlbumInput struct {
		ID int `path:"id"`
		TextBody
	}

	u := usecase.NewInteractor(func(ctx context.Context, in *updateAlbumInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "update_album", 1)
		deps.CtxdLogger().Important(ctx, "creating album", "id", in.ID)

		desc, err := in.Text()

		println(desc)

		return err

		// return deps.PhotoAlbumUpdater().Update(ctx, in.ID, in.AlbumData)
	})

	u.SetDescription("Update an album description.")
	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.NotFound)

	return u
}

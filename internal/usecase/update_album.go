package usecase

import (
	"context"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type updateAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PhotoAlbumUpdater() photo.AlbumUpdater
}

// UpdateAlbum updates use case interactor to add directory of photos.
func UpdateAlbum(deps updateAlbumDeps) usecase.Interactor {
	type updateAlbumInput struct {
		ID int `path:"id"`
		photo.AlbumData
	}

	u := usecase.NewInteractor(func(ctx context.Context, in updateAlbumInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "update_album", 1)
		deps.CtxdLogger().Important(ctx, "updating album", "id", in.ID)

		return deps.PhotoAlbumUpdater().Update(ctx, in.ID, in.AlbumData)
	})

	u.SetDescription("Update an album.")
	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.NotFound)

	return u
}

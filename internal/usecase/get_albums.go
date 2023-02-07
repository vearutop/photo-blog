package usecase

import (
	"context"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type getAlbumsDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PhotoAlbumFinder() photo.AlbumFinder
}

// GetAlbums creates use case interactor to get album data.
func GetAlbums(deps getAlbumsDeps) usecase.Interactor {
	type getAlbumsOutput struct {
		Albums []photo.Album `json:"albums"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *getAlbumsOutput) error {
		deps.StatsTracker().Add(ctx, "get_albums", 1)
		deps.CtxdLogger().Info(ctx, "getting albums")

		albums, err := deps.PhotoAlbumFinder().FindAll(ctx)
		if err != nil {
			return err
		}

		out.Albums = albums

		return nil
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown)

	return u
}

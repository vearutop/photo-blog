package usecase

import (
	"context"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// ShowAlbum creates use case interactor to show album.
func ShowAlbum(deps helloDeps) usecase.Interactor {
	type showAlbumInput struct {
		ID string `path:"id"`
	}

	type helloOutput struct {
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showAlbumInput, out *helloOutput) error {
		deps.StatsTracker().Add(ctx, "show_album", 1)
		deps.CtxdLogger().Info(ctx, "showing album", "path", in.ID)

		return nil
	})

	u.SetDescription("Add a directory of photos as an album (non-recursive).")
	u.SetTags("Photos")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

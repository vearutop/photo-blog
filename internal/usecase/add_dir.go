package usecase

import (
	"context"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// AddDirectory creates use case interactor to add directory of photos.
func AddDirectory(deps helloDeps) usecase.Interactor {
	type addDirInput struct {
		Path string `formData:"path"`
	}

	type helloOutput struct {
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in addDirInput, out *helloOutput) error {
		deps.StatsTracker().Add(ctx, "add_dir", 1)
		deps.CtxdLogger().Important(ctx, "adding directory", "path", in.Path)

		return nil
	})

	u.SetDescription("Add a directory of photos as an album (non-recursive).")
	u.SetTags("Photos")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

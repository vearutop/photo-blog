package usecase

import (
	"context"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"os"
	"strings"
)

type addDirectoryDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() photo.AlbumFinder
	PhotoImageAdder() photo.ImageAdder
	PhotoAlbumAdder() photo.AlbumAdder
}

// AddDirectory creates use case interactor to add directory of photos.
func AddDirectory(deps addDirectoryDeps) usecase.Interactor {
	type addDirInput struct {
		Path      string `formData:"path"`
		AlbumName string `formData:"album_name"`
	}

	type helloOutput struct {
		Names []string `json:"names"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in addDirInput, out *helloOutput) error {
		deps.StatsTracker().Add(ctx, "add_dir", 1)
		deps.CtxdLogger().Important(ctx, "adding directory", "path", in.Path)

		dir, err := os.Open(in.Path)
		if err != nil {
			return nil
		}

		names, err := dir.Readdirnames(0)
		if err != nil {
			return ctxd.WrapError(ctx, err, "read dir names", "path", in.Path)
		}

		deps.CtxdLogger().Important(ctx, "directory contents", "names", names)

		out.Names = names

		a, err := deps.PhotoAlbumFinder().FindByName(ctx, in.AlbumName)
		if err != nil {
			return ctxd.WrapError(ctx, err, "find album", "name", in.AlbumName)
		}

		var (
			imgIDs []int
			errs   []error
		)

		for _, name := range names {
			if strings.HasSuffix(strings.ToLower(name), ".jpg") {
				img, err := deps.PhotoImageAdder().Add(ctx, photo.ImageData{})

			}
		}

		deps.PhotoAlbumAdder().AddImages(ctx, a.ID, 123)

		return nil
	})

	u.SetDescription("Add a directory of photos as an album (non-recursive).")
	u.SetTags("Photos")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

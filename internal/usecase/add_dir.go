package usecase

import (
	"context"
	"os"
	"path"
	"strings"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type addDirectoryDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() photo.AlbumFinder
	PhotoImageEnsurer() photo.ImageEnsurer
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
			errs   []string
		)

		for _, name := range names {
			if strings.HasSuffix(strings.ToLower(name), ".jpg") {
				d := photo.ImageData{Path: path.Join(in.Path, name)}
				if img, err := deps.PhotoImageEnsurer().Ensure(ctx, d); err != nil {
					errs = append(errs, name+": "+err.Error())
				} else {
					imgIDs = append(imgIDs, img.ID)
				}
			}
		}

		if len(imgIDs) > 0 {
			if err := deps.PhotoAlbumAdder().AddImages(ctx, a.ID, imgIDs...); err != nil {
				if len(errs) > 0 {
					errs = append(errs, err.Error())
				} else {
					return err
				}
			}
		}

		if len(errs) > 0 {
			return ctxd.NewError(ctx, "there were errors", "errors", errs)
		}

		return nil
	})

	u.SetDescription("Add a directory of photos to an album (non-recursive).")
	u.SetTags("Photos")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}
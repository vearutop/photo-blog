package control

import (
	"context"
	"os"
	"path"
	"strings"
	"sync/atomic"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type addDirectoryDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageAdder() photo.AlbumImageAdder

	PhotoImageEnsurer() uniq.Ensurer[photo.Image]
	PhotoImageIndexer() photo.ImageIndexer
}

// AddDirectory creates use case interactor to add directory of photos to an album.
func AddDirectory(deps addDirectoryDeps, indexer usecase.IOInteractorOf[indexAlbumInput, struct{}]) usecase.Interactor {
	type addDirInput struct {
		Path string `formData:"path"`
		Name string `path:"name" description:"Album name."`
	}

	type addDirOutput struct {
		Names []string `json:"names"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in addDirInput, out *addDirOutput) error {
		deps.StatsTracker().Add(ctx, "add_dir", 1)
		deps.CtxdLogger().Important(ctx, "adding directory", "path", in.Path)

		a, err := deps.PhotoAlbumFinder().FindByHash(ctx, uniq.StringHash(in.Name))
		if err != nil {
			return ctxd.WrapError(ctx, err, "find album", "name", in.Name)
		}

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

		var (
			imgHashes []uniq.Hash
			errs      []string
		)

		for _, name := range names {
			if strings.HasSuffix(strings.ToLower(name), ".jpg") {
				d := photo.Image{Path: path.Join(in.Path, name)}
				if img, err := deps.PhotoImageEnsurer().Ensure(ctx, d); err != nil {
					errs = append(errs, name+": "+err.Error())
				} else {
					go func() {
						deps.StatsTracker().Set(ctx, "indexing_images_pending",
							float64(atomic.AddInt64(&indexInProgress, 1)))
						ctx := detachedContext{parent: ctx}
						if err := deps.PhotoImageIndexer().Index(ctx, img, photo.IndexingFlags{}); err != nil {
							deps.CtxdLogger().Error(ctx, "failed to index image", "error", err)
						}
						deps.StatsTracker().Set(ctx, "indexing_images_pending",
							float64(atomic.AddInt64(&indexInProgress, -1)))
					}()

					imgHashes = append(imgHashes, img.Hash)
				}
			}
		}

		if len(imgHashes) > 0 {
			if err := deps.PhotoAlbumImageAdder().AddImages(ctx, a.Hash, imgHashes...); err != nil {
				if len(errs) > 0 {
					errs = append(errs, err.Error())
				} else {
					return err
				}
			}
		}

		if err := indexer.Invoke(ctx, indexAlbumInput{Name: in.Name}, nil); err != nil {
			errs = append(errs, err.Error())
		}

		if len(errs) > 0 {
			return ctxd.NewError(ctx, "there were errors", "errors", errs)
		}

		return nil
	})

	u.SetDescription("Add a host-local directory of photos to an album (non-recursive).")
	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

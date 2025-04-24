package control

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
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/dep"
)

type addDirectoryDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumEnsurer() uniq.Ensurer[photo.Album]
	PhotoAlbumImageAdder() photo.AlbumImageAdder

	PhotoImageEnsurer() uniq.Ensurer[photo.Image]

	PhotoGpxEnsurer() uniq.Ensurer[photo.Gpx]

	DepCache() *dep.Cache
}

type addDirInput struct {
	Path string `formData:"path"`
	Name string `path:"name" description:"Album name."`
}

type addDirOutput struct {
	Names []string `json:"names"`
}

// AddDirectory creates use case interactor to add directory of photos to an album.
func AddDirectory(deps addDirectoryDeps, indexer usecase.IOInteractorOf[indexAlbumInput, struct{}]) usecase.IOInteractorOf[addDirInput, addDirOutput] {
	u := usecase.NewInteractor(func(ctx context.Context, in addDirInput, out *addDirOutput) error {
		deps.StatsTracker().Add(ctx, "add_dir", 1)
		deps.CtxdLogger().Important(ctx, "adding directory", "path", in.Path)

		a, err := deps.PhotoAlbumFinder().FindByHash(ctx, uniq.StringHash(in.Name))
		if err != nil {
			return ctxd.WrapError(ctx, err, "find album", "name", in.Name)
		}

		dir, err := os.Open(in.Path)
		if err != nil {
			return err
		}

		names, err := dir.Readdirnames(0)
		if err != nil {
			return ctxd.WrapError(ctx, err, "read dir names", "path", in.Path)
		}

		deps.CtxdLogger().Important(ctx, "directory contents", "names", names)

		out.Names = names

		var (
			imgHashes []uniq.Hash
			gpxHashes []uniq.Hash
			errs      []string
		)

		for _, name := range names {
			lName := strings.ToLower(name)
			if strings.HasSuffix(lName, ".jpg") || strings.HasSuffix(lName, ".jpeg") {
				d := photo.Image{}
				if err := d.SetPath(ctx, path.Join(in.Path, name)); err != nil {
					errs = append(errs, in.Path+": "+err.Error())

					continue
				}

				if img, err := deps.PhotoImageEnsurer().Ensure(ctx, d); err != nil {
					errs = append(errs, name+": "+err.Error())
				} else {
					imgHashes = append(imgHashes, img.Hash)
				}
			}

			if strings.HasSuffix(lName, ".gpx") {
				d := photo.Gpx{}
				if err := d.SetPath(ctx, path.Join(in.Path, name)); err != nil {
					errs = append(errs, name+": "+err.Error())

					continue
				}

				if err := d.Index(); err != nil {
					errs = append(errs, name+": "+err.Error())

					continue
				}

				deps.CtxdLogger().Info(ctx, "gpx", "settings", d.Settings.Val)

				if d, err := deps.PhotoGpxEnsurer().Ensure(ctx, d); err != nil {
					errs = append(errs, name+": "+err.Error())
				} else {
					gpxHashes = append(gpxHashes, d.Hash)
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

		if len(gpxHashes) > 0 {
			for _, h := range gpxHashes {
				found := false
				for _, hh := range a.Settings.GpxTracksHashes {
					if h == hh {
						found = true
						break
					}
				}
				if !found {
					a.Settings.GpxTracksHashes = append(a.Settings.GpxTracksHashes, h)
				}
			}

			if _, err := deps.PhotoAlbumEnsurer().Ensure(ctx, a); err != nil {
				return err
			}
		}

		if err := indexer.Invoke(ctx, indexAlbumInput{Name: in.Name}, nil); err != nil {
			errs = append(errs, err.Error())
		}

		if len(errs) > 0 {
			return ctxd.NewError(ctx, "there were errors", "errors", errs)
		}

		if err == nil {
			err = deps.DepCache().AlbumChanged(ctx, a.Name)
		}

		return err
	})

	u.SetDescription("Add a host-local directory of photos to an album (non-recursive).")
	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

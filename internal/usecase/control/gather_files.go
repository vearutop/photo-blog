package control

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/upload"
)

type gatherFilesDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoImageEnsurer() uniq.Ensurer[photo.Image]

	PhotoGpxFinder() uniq.Finder[photo.Gpx]
	PhotoGpxEnsurer() uniq.Ensurer[photo.Gpx]
}

func moveFileNoRewrite(from, to string) error {
	sfrom, err := os.Lstat(from)
	if err != nil {
		return err
	}

	sto, err := os.Lstat(to)
	if err == nil {
		if sto.Size() != sfrom.Size() {
			return fmt.Errorf("already exists: %s", to)
		}

		// Fake a rename.
		return nil
	}

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return os.Rename(from, to)
}

// GatherFiles moves album files into their canonical location.
func GatherFiles(deps gatherFilesDeps) usecase.Interactor {
	type gatherFilesInput struct {
		Name         string `path:"name" description:"Album name."`
		CheckMissing bool   `query:"check_missing"`
	}

	type move struct {
		OriginalName string `json:"orig,omitempty"`
		NewName      string `json:"new,omitempty"`
		Error        string `json:"error,omitempty"`
	}

	type albumReport struct {
		AlbumName string `json:"albumName"`
		Moves     []move `json:"moves,omitempty"`
	}

	type gatherFilesOutput struct {
		Reports []albumReport `json:"reports"`
	}

	// Forces canonical location to current album, disabled for "-" all albums to avoid unnecessary moves.
	forceAlbumPath := true

	u := usecase.NewInteractor(func(ctx context.Context, in gatherFilesInput, out *gatherFilesOutput) (err error) {
		deps.StatsTracker().Add(ctx, "gather_files", 1)
		deps.CtxdLogger().Info(ctx, "gathering files", "album_name", in.Name)

		processed := map[uniq.Hash]bool{}
		canonicalAlbumPath := upload.AlbumPath("")

		gatherImages := func(albumName string, rep *albumReport) error {
			var images []photo.Image

			rep.AlbumName = albumName

			album, err := deps.PhotoAlbumFinder().FindByHash(ctx, photo.AlbumHash(albumName))
			if err != nil {
				return err
			}

			images, err = deps.PhotoAlbumImageFinder().FindImages(ctx, album.Hash)
			if err != nil {
				return err
			}

			albumPath := upload.AlbumPath(album.Name)
			if err := os.MkdirAll(albumPath, 0o700); err != nil {
				deps.CtxdLogger().Error(ctx, "failed to create album directory", "error", err)

				return err
			}

			for _, img := range images {
				if in.CheckMissing {
					if _, err := os.Lstat(img.Path); err != nil {
						rep.Moves = append(rep.Moves, move{
							OriginalName: img.Path,
							Error:        err.Error(),
						})
					}

					continue
				}

				if strings.HasPrefix(img.Path, canonicalAlbumPath) && !forceAlbumPath {
					continue
				}

				if processed[img.Hash] {
					continue
				}

				processed[img.Hash] = true

				canonicalPath := upload.AlbumFilePath(albumPath, path.Base(img.Path))
				if img.Path != canonicalPath {
					m := move{
						OriginalName: img.Path,
						NewName:      canonicalPath,
					}

					ctx := ctxd.AddFields(ctx, "path", img.Path,
						"canonical", canonicalPath)

					if err := moveFileNoRewrite(img.Path, canonicalPath); err != nil {
						deps.CtxdLogger().Error(ctx, "failed to rename album file",
							"error", err.Error())

						m.Error = err.Error()
						rep.Moves = append(rep.Moves, m)

						continue
					}

					img.Path = canonicalPath

					if _, err := deps.PhotoImageEnsurer().Ensure(ctx, img); err != nil {
						deps.CtxdLogger().Error(ctx, "failed to update album image file",
							"error", err.Error())

						m.Error = err.Error()
						rep.Moves = append(rep.Moves, m)

						continue
					}

					rep.Moves = append(rep.Moves, m)
					deps.CtxdLogger().Info(ctx, "image file moved")
				}

			}

			for _, h := range album.Settings.GpxTracksHashes {
				g, err := deps.PhotoGpxFinder().FindByHash(ctx, h)
				if err != nil {
					return err
				}

				if in.CheckMissing {
					if _, err := os.Lstat(g.Path); err != nil {
						rep.Moves = append(rep.Moves, move{
							OriginalName: g.Path,
							Error:        err.Error(),
						})
					}

					continue
				}

				if strings.HasPrefix(g.Path, canonicalAlbumPath) {
					continue
				}

				canonicalPath := upload.AlbumFilePath(albumPath, path.Base(g.Path))
				m := move{
					OriginalName: g.Path,
					NewName:      canonicalPath,
				}

				ctx := ctxd.AddFields(ctx, "path", g.Path,
					"canonical", canonicalPath)

				if err := moveFileNoRewrite(g.Path, canonicalPath); err != nil {
					deps.CtxdLogger().Error(ctx, "failed to rename album file",
						"error", err.Error())

					m.Error = err.Error()
					rep.Moves = append(rep.Moves, m)

					continue
				}

				g.Path = canonicalPath

				if _, err := deps.PhotoGpxEnsurer().Ensure(ctx, g); err != nil {
					deps.CtxdLogger().Error(ctx, "failed to update album gpx file",
						"error", err.Error())

					m.Error = err.Error()
					rep.Moves = append(rep.Moves, m)

					continue
				}
			}

			return nil
		}

		if in.Name == "-" {
			forceAlbumPath = false

			albums, err := deps.PhotoAlbumFinder().FindAll(ctx)
			if err != nil {
				return err
			}

			for _, album := range albums {
				rep := albumReport{}

				err := gatherImages(album.Name, &rep)
				out.Reports = append(out.Reports, rep)
				if err != nil {
					return err
				}
			}
		} else {
			rep := albumReport{}

			err := gatherImages(in.Name, &rep)
			out.Reports = append(out.Reports, rep)
			if err != nil {
				return err
			}
		}

		return nil
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

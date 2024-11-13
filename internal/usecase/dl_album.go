package usecase

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync/atomic"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/storage"
)

type dlAlbumDeps interface {
	CtxdLogger() ctxd.Logger
	StatsTracker() stats.Tracker
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	Settings() settings.Values
	FavoriteRepository() *storage.FavoriteRepository
}

type dlAlbumInput struct {
	Name     string `path:"name"`
	Favorite bool   `query:"favorite"`
}

func DownloadAlbum(deps dlAlbumDeps) usecase.Interactor {
	var inProgress int64

	u := usecase.NewInteractor(func(ctx context.Context, in dlAlbumInput, out *response.EmbeddedSetter) (err error) {
		privacy := deps.Settings().Privacy()
		if (privacy.HideOriginal || privacy.HideBatchDownload) && !auth.IsAdmin(ctx) {
			return status.PermissionDenied
		}

		rw := out.ResponseWriter()

		album, err := deps.PhotoAlbumFinder().FindByHash(ctx, photo.AlbumHash(in.Name))
		if err != nil {
			return err
		}

		var images []photo.Image
		if in.Favorite {
			visitorHash := auth.VisitorFromContext(ctx)
			if visitorHash == 0 {
				return status.PermissionDenied
			}

			images, err = deps.FavoriteRepository().FindAlbumImages(ctx, visitorHash, album.Hash)
		} else {
			images, err = deps.PhotoAlbumImageFinder().FindImages(ctx, album.Hash)
		}

		if err != nil {
			return err
		}

		deps.StatsTracker().Set(ctx, "dl_in_progress", float64(atomic.AddInt64(&inProgress, 1)))

		defer func() {
			deps.StatsTracker().Set(ctx, "dl_in_progress", float64(atomic.AddInt64(&inProgress, -1)))
		}()

		rw.Header().Set("Content-Type", "application/zip")
		rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", album.Name))

		// Create a new zip archive.
		w := zip.NewWriter(rw)
		defer func() {
			// Make sure to check the error on Close.
			clErr := w.Close()
			if clErr != nil {
				if err == nil {
					err = clErr
				} else {
					deps.CtxdLogger().Error(ctx, "failed to close zip writer", "error", clErr)
				}
			}
		}()

		copyImg := func(ctx context.Context, f io.Writer, img photo.Image) error {
			if len(img.Settings.HTTPSources) > 0 {
				resp, err := http.Get(img.Settings.HTTPSources[0])
				if err != nil {
					deps.CtxdLogger().Error(ctx, "failed to open remote image",
						"error", err, "img", img)
					return err
				}
				defer func() {
					if err := resp.Body.Close(); err != nil {
						deps.CtxdLogger().Error(ctx, "failed to close remote image")
					}
				}()
				if _, err = io.Copy(f, resp.Body); err != nil {
					return err
				}
			} else {
				src, err := os.Open(img.Path)
				if err != nil {
					deps.CtxdLogger().Error(ctx, "failed to open image",
						"error", err, "img", img)
					return err
				}
				defer func() {
					if err := src.Close(); err != nil {
						deps.CtxdLogger().Error(ctx, "failed to close image")
					}
				}()

				if _, err = io.Copy(f, src); err != nil {
					return err
				}
			}

			return nil
		}

		for _, img := range images {
			if img.TakenAt == nil {
				img.TakenAt = &img.CreatedAt
			}

			f, err := w.CreateHeader(&zip.FileHeader{
				Name:     path.Base(strings.TrimSuffix(img.Path, "."+img.Hash.String()+".jpg")),
				Method:   zip.Store,
				Modified: *img.TakenAt,
			})
			if err != nil {
				return err
			}

			if err := copyImg(ctx, f, img); err != nil {
				return err
			}
		}

		return nil
	})
	u.SetTags("Album")

	return u
}

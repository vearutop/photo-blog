package usecase

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/bool64/ctxd"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/settings"
)

type dlAlbumDeps interface {
	CtxdLogger() ctxd.Logger
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	Settings() settings.Values
}

type dlAlbumInput struct {
	Name string `path:"name"`
}

func DownloadAlbum(deps dlAlbumDeps) usecase.Interactor {
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

		images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, album.Hash)
		if err != nil {
			return err
		}

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

		for _, img := range images {
			f, err := w.CreateHeader(&zip.FileHeader{
				Name:   path.Base(img.Path),
				Method: zip.Store,
			})
			if err != nil {
				return err
			}

			src, err := os.Open(img.Path)
			if err != nil {
				deps.CtxdLogger().Error(ctx, "failed to open image",
					"error", err, "img", img)
				continue
			}

			if _, err = io.Copy(f, src); err != nil {
				return err
			}
		}

		return nil
	})
	u.SetTags("Album")

	return u
}

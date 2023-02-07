package usecase

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/bool64/ctxd"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type dlAlbumDeps interface {
	CtxdLogger() ctxd.Logger
	PhotoAlbumFinder() photo.AlbumFinder
	PhotoImageFinder() photo.ImageFinder
}

type dlAlbumInput struct {
	Name string `path:"name"`
}

func DownloadAlbum(deps dlAlbumDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in dlAlbumInput, out *usecase.OutputWithEmbeddedWriter) (err error) {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		album, err := deps.PhotoAlbumFinder().FindByName(ctx, in.Name)
		if err != nil {
			return err
		}

		images, err := deps.PhotoAlbumFinder().FindImages(ctx, album.ID)
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

package control

import (
	"context"
	"io"
	"mime/multipart"
	"os"
	"path"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

type uploadPhotosDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageAdder() photo.AlbumImageAdder

	PhotoImageEnsurer() uniq.Ensurer[photo.Image]
	PhotoImageIndexer() photo.ImageIndexer

	ServiceSettings() service.Settings
}

func UploadImages(deps uploadPhotosDeps) usecase.Interactor {
	type photoFiles struct {
		Name   string                  `path:"name"`
		Photos []*multipart.FileHeader `formData:"photos"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in photoFiles, out *struct{}) error {
		albumDir := path.Join(deps.ServiceSettings().UploadStorage, in.Name)

		if err := os.MkdirAll(albumDir, 0o600); err != nil {
			return err
		}

		for _, f := range in.Photos {
			println(f.Filename)
			file, err := f.Open()
			if err != nil {
				return err
			}

			// targetName := path.Join(albumDir, f.Filename)

			target, err := os.Create(path.Join(albumDir, f.Filename))
			if err == nil {
				io.Copy(target, file)
			} else {
				deps.CtxdLogger().Error(ctx, "failed to ")
			}

			if err := file.Close(); err != nil {
				deps.CtxdLogger().Error(ctx, "failed to close uploaded file",
					"file", f.Filename, "error", err.Error())
			}

			_ = f
		}

		return nil
	})

	return u
}

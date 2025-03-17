package usecase

import (
	"context"
	"net/http"
	"strings"

	"github.com/bool64/ctxd"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type showThumbDeps interface {
	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoThumbnailer() photo.Thumbnailer
}

type showThumbInput struct {
	request.EmbeddedSetter
	Size photo.ThumbSize `path:"size"`
	Hash uniq.Hash       `path:"hash"`
}

func ShowThumb(deps showThumbDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in showThumbInput, out *response.EmbeddedSetter) error {
		rw := out.ResponseWriter()

		image, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		dctx := detachedContext{parent: ctx}
		cont, err := deps.PhotoThumbnailer().Thumbnail(dctx, image, in.Size)
		if err != nil {
			return ctxd.WrapError(ctx, err, "getting thumbnail")
		}

		rw.Header().Set("Cache-Control", "max-age=31536000")

		if cont.FilePath != "" {
			if strings.HasPrefix(cont.FilePath, "https://") || strings.HasPrefix(cont.FilePath, "http://") {
				http.Redirect(rw, in.Request(), cont.FilePath, http.StatusMovedPermanently)
				return nil
			}

			http.ServeFile(rw, in.Request(), cont.FilePath)
		} else {
			http.ServeContent(rw, in.Request(), "thumb.jpg", image.CreatedAt, cont.ReadSeeker())
		}

		return nil
	})
	u.SetTags("Image")

	return u
}

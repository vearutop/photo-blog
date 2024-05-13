package usecase

import (
	"context"
	"net/http"
	"strings"

	"github.com/bool64/ctxd"
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
	Size photo.ThumbSize `path:"size"`
	Hash uniq.Hash       `path:"hash"`
	req  *http.Request
}

func (s *showThumbInput) SetRequest(r *http.Request) {
	s.req = r
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
				http.Redirect(rw, in.req, cont.FilePath, http.StatusMovedPermanently)
				return nil
			}

			http.ServeFile(rw, in.req, cont.FilePath)
		} else {
			http.ServeContent(rw, in.req, "thumb.jpg", image.CreatedAt, cont.ReadSeeker())
		}

		return nil
	})
	u.SetTags("Image")

	return u
}

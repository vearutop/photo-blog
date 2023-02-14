package usecase

import (
	"context"
	"errors"
	"net/http"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type showThumbDeps interface {
	PhotoImageFinder() photo.ImageFinder
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
	u := usecase.NewInteractor(func(ctx context.Context, in showThumbInput, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		image, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		dctx := detachedContext{parent: ctx}
		cont, err := deps.PhotoThumbnailer().Thumbnail(dctx, image, in.Size)
		if err != nil {
			return err
		}

		http.ServeContent(rw, in.req, "thumb.jpg", image.CreatedAt, cont.ReadSeeker())

		return nil
	})
	u.SetTags("Image")

	return u
}

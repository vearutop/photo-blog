package usecase

import (
	"context"
	"errors"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"net/http"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
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

		cont, err := deps.PhotoThumbnailer().Thumbnail(ctx, image, in.Size)
		if err != nil {
			return err
		}

		http.ServeContent(rw, in.req, "thumb.jpg", image.CreatedAt, cont)

		return nil
	})
	u.SetTags("Image")

	return u
}

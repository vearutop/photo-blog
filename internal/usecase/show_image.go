package usecase

import (
	"context"
	"errors"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"net/http"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type showImageDeps interface {
	PhotoImageFinder() photo.ImageFinder
}

type showImageInput struct {
	Hash uniq.Hash `path:"hash"`
	req  *http.Request
}

func (s *showImageInput) SetRequest(r *http.Request) {
	s.req = r
}

func ShowImage(deps showImageDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in showImageInput, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		image, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		http.ServeFile(rw, in.req, image.Path)

		return nil
	})
	u.SetTags("Image")

	return u
}

package usecase

import (
	"context"
	"errors"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"net/http"
	"strconv"
)

type showImageDeps interface {
	PhotoImageFinder() photo.ImageFinder
}

type showImageInput struct {
	Hash string `path:"hash"`
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

		h, err := strconv.ParseUint(in.Hash, 36, 64)
		if err != nil {
			return err
		}

		image, err := deps.PhotoImageFinder().FindByHash(ctx, int64(h))
		if err != nil {
			return err
		}

		http.ServeFile(rw, in.req, image.Path)

		return nil
	})

	return u
}

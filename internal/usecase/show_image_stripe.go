package usecase

import (
	"context"
	"github.com/swaggest/usecase"
	"net/http"
)

type showImageStripeDeps interface {
}

type showImageStripeInput struct {
	Name  string `path:"name" title:"Album name"`
	Count int    `query:"count" title:"Number of images"`
	req   *http.Request
}

func (s *showImageStripeInput) SetRequest(r *http.Request) {
	s.req = r
}

func ShowImageStripe(deps getAlbumImagesDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in showImageStripeInput, out *usecase.OutputWithEmbeddedWriter) error {
		//rw, ok := out.Writer.(http.ResponseWriter)
		//if !ok {
		//	return errors.New("missing http.ResponseWriter")
		//}
		//
		//albumHash := photo.AlbumHash(in.Name)
		//
		//album, err := deps.PhotoAlbumFinder().FindByHash(ctx, albumHash)
		//if err != nil {
		//	return err
		//}
		//
		//image, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		//if err != nil {
		//	return err
		//}
		//
		//http.ServeFile(rw, in.req, image.Path)

		return nil
	})
	u.SetTags("Image")

	return u
}

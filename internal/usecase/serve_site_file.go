package usecase

import (
	"context"
	"net/http"
	"path"

	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
)

func ServeSiteFile(deps interface{}) usecase.Interactor {
	type fileReq struct {
		File string `path:"file"`
		request.EmbeddedSetter
	}

	u := usecase.NewInteractor(func(ctx context.Context, in fileReq, out *response.EmbeddedSetter) error {
		rw := out.ResponseWriter()

		rw.Header().Set("Cache-Control", "max-age=31536000")

		filePath := path.Join("site", in.File)

		http.ServeFile(rw, in.Request(), filePath)

		return nil
	})
	u.SetTags("Site")

	return u
}

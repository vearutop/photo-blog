package help

import (
	"context"
	"errors"
	"net/http"

	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/resources/help"
	"github.com/vearutop/statigz"
)

func ServeFile(deps any) usecase.Interactor {
	type fileReq struct {
		File string `path:"file"`
		request.EmbeddedSetter
	}

	fs := statigz.FileServer(help.Assets)
	u := usecase.NewInteractor(func(ctx context.Context, in fileReq, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		rw.Header().Set("Cache-Control", "max-age=31536000")
		r := in.Request()
		r.URL.Path = in.File

		fs.ServeHTTP(rw, in.Request())

		return nil
	})
	u.SetTags("Help")

	return u
}

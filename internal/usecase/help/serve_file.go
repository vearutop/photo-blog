package help

import (
	"context"

	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/resources/help"
	"github.com/vearutop/statigz"
)

func ServeFile(deps indexDeps) usecase.Interactor {
	type fileReq struct {
		File string `path:"file"`
		request.EmbeddedSetter
	}

	fs := statigz.FileServer(help.Assets)
	u := usecase.NewInteractor(func(ctx context.Context, in fileReq, out *response.EmbeddedSetter) error {
		if !deps.Settings().Privacy().PublicHelp && !auth.IsAdmin(ctx) {
			return status.PermissionDenied
		}

		rw := out.ResponseWriter()

		rw.Header().Set("Cache-Control", "max-age=31536000")
		r := in.Request()
		r.URL.Path = in.File

		fs.ServeHTTP(rw, in.Request())

		return nil
	})
	u.SetTags("Help")

	return u
}

package usecase

import (
	"context"
	"errors"
	"net/http"
	"path"

	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

type serveSiteFileDeps interface {
	ServiceConfig() service.Config
}

func ServeSiteFile(deps serveSiteFileDeps) usecase.Interactor {
	type fileReq struct {
		File string `path:"file"`
		request.EmbeddedSetter
	}

	u := usecase.NewInteractor(func(ctx context.Context, in fileReq, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		rw.Header().Set("Cache-Control", "max-age=31536000")

		filePath := path.Join(deps.ServiceConfig().StoragePath, "site", in.File)

		http.ServeFile(rw, in.Request(), filePath)

		return nil
	})
	u.SetTags("Site")

	return u
}

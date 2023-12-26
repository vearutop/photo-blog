package usecase

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strings"

	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/infra/nethttp/ui"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/settings"
)

type serveFaviconDeps interface {
	Settings() settings.Values
	ServiceConfig() service.Config
}

func ServeFavicon(deps serveFaviconDeps) usecase.Interactor {
	type fileReq struct {
		request.EmbeddedSetter
	}

	u := usecase.NewInteractor(func(ctx context.Context, in fileReq, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		rw.Header().Set("Cache-Control", "max-age=31536000")

		favicon := deps.Settings().Appearance().SiteFavicon

		r := in.Request()
		if strings.HasPrefix(favicon, "/site/") {
			http.ServeFile(rw, r, path.Join(deps.ServiceConfig().StoragePath, favicon))

			return nil
		}

		r.URL.Path = "/static/favicon.png"
		ui.Static.ServeHTTP(rw, r)

		return nil
	})
	u.SetTags("Site")

	return u
}

package usecase

import (
	"context"
	"net/http"
	"strings"

	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/infra/nethttp/ui"
	"github.com/vearutop/photo-blog/internal/infra/settings"
)

type serveFaviconDeps interface {
	Settings() settings.Values
}

func ServeFavicon(deps serveFaviconDeps) usecase.Interactor {
	type fileReq struct {
		request.EmbeddedSetter
	}

	u := usecase.NewInteractor(func(ctx context.Context, in fileReq, out *response.EmbeddedSetter) error {
		rw := out.ResponseWriter()

		rw.Header().Set("Cache-Control", "max-age=31536000")

		favicon := deps.Settings().Appearance().SiteFavicon

		r := in.Request()
		if strings.HasPrefix(favicon, "/site/") {
			http.ServeFile(rw, r, strings.TrimPrefix(favicon, "/"))

			return nil
		}

		r.URL.Path = "/favicon.png"
		ui.Static.ServeHTTP(rw, r)

		return nil
	})
	u.SetTags("Site")

	return u
}

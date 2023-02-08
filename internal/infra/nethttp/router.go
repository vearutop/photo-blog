// Package nethttp manages application http interface.
package nethttp

import (
	"github.com/swaggest/openapi-go/openapi3"
	"net/http"

	"github.com/bool64/brick"
	"github.com/swaggest/rest/nethttp"
	"github.com/vearutop/photo-blog/internal/infra/nethttp/ui"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/usecase"
)

// NewRouter creates an instance of router filled with handlers and docs.
func NewRouter(deps *service.Locator) http.Handler {
	r := brick.NewBaseWebService(deps.BaseLocator)

	r.Group()

	adminAuth := basicAuth("Admin Access", map[string]string{"admin": "admin"})
	r.Use(nethttp.AnnotateOpenAPI(r.OpenAPICollector, func(op *openapi3.Operation) error {
		op.Tags = []string{"Admin Mode"}

		return nil
	}))
	r.Use(adminAuth, nethttp.HTTPBasicSecurityMiddleware(r.OpenAPICollector, "Admin", "Admin access"))

	r.Post("/album", usecase.CreateAlbum(deps))

	r.Post("/directory", usecase.AddDirectory(deps))
	r.Get("/album/{name}.json", usecase.GetAlbum(deps))
	r.Get("/albums.json", usecase.GetAlbums(deps))
	r.Get("/image/{hash}.json", usecase.GetImage(deps))
	r.Post("/index/{name}", usecase.IndexAlbum(deps), nethttp.SuccessStatus(http.StatusAccepted))
	r.Delete("/album/{name}/{hash}", usecase.RemoveFromAlbum(deps))
	r.Post("/album/{name}/{hash}", usecase.AddToAlbum(deps))
	r.Get("/album/{name}.zip", usecase.DownloadAlbum(deps))

	r.Get("/image/{hash}.jpg", usecase.ShowImage(deps))
	r.Get("/thumb/{size}/{hash}.jpg", usecase.ShowThumb(deps))

	r.Method(http.MethodGet, "/", ui.Static)

	r.Get("/{name}/", usecase.ShowAlbum(deps))
	r.Get("/{name}/pano-{hash}.html", usecase.ShowPano(deps))

	r.Post("/make-pass-hash", usecase.MakePassHash())

	r.Mount("/static/", http.StripPrefix("/static", ui.Static))

	return r
}

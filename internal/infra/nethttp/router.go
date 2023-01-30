// Package nethttp manages application http interface.
package nethttp

import (
	"github.com/vearutop/photo-blog/internal/usecase"
	"net/http"

	"github.com/bool64/brick"
	"github.com/vearutop/photo-blog/internal/infra/nethttp/ui"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

// NewRouter creates an instance of router filled with handlers and docs.
func NewRouter(deps *service.Locator) http.Handler {
	r := brick.NewBaseWebService(deps.BaseLocator)

	r.Post("/album", usecase.CreateAlbum(deps))

	r.Post("/directory", usecase.AddDirectory(deps))
	r.Get("/album/{name}.json", usecase.GetAlbum(deps))
	r.Get("/album/{name}.zip", usecase.DownloadAlbum(deps))

	r.Get("/image/{hash}.jpg", usecase.ShowImage(deps))
	r.Get("/thumb/{size}/{hash}.jpg", usecase.ShowThumb(deps))

	r.Method(http.MethodGet, "/", ui.Index())
	r.Get("/{name}/", usecase.ShowAlbum(deps))

	r.Mount("/static/", http.StripPrefix("/static", ui.Static))

	return r
}

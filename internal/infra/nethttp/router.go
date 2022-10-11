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

	//r.Get("/hello", usecase.HelloWorld(deps))
	//r.Delete("/hello", usecase.Clear(deps))

	r.Post("/directory", usecase.AddDirectory(deps))
	r.Get("/album/{id}", usecase.ShowAlbum(deps))

	r.Method(http.MethodGet, "/", ui.Index())

	r.Mount("/static/", http.StripPrefix("/static", ui.Static))

	return r
}

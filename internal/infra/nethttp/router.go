// Package nethttp manages application http interface.
package nethttp

import (
	"net/http"

	"github.com/bool64/brick"
	"github.com/go-chi/chi/v5"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
	"github.com/vearutop/photo-blog/internal/infra/nethttp/ui"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/usecase"
)

// NewRouter creates an instance of router filled with handlers and docs.
func NewRouter(deps *service.Locator, cfg service.Config) http.Handler {
	s := brick.NewBaseWebService(deps.BaseLocator)

	s.Group(func(r chi.Router) {
		s := fork(s, r)

		if cfg.AdminPassHash != "" {
			adminAuth := basicAuth("Admin Access", cfg.AdminPassHash, cfg.AdminPassSalt)
			s.Use(nethttp.AnnotateOpenAPI(s.OpenAPICollector, func(op *openapi3.Operation) error {
				op.Tags = append(op.Tags, "Admin Mode")

				return nil
			}))
			s.Use(adminAuth, nethttp.HTTPBasicSecurityMiddleware(s.OpenAPICollector, "Admin", "Admin access"))
		}

		s.Post("/album", usecase.CreateAlbum(deps))
		s.Put("/album/{id}", usecase.UpdateAlbum(deps))
		s.Post("/directory", usecase.AddDirectory(deps))
		s.Get("/albums.json", usecase.GetAlbums(deps))
		s.Post("/index/{name}", usecase.IndexAlbum(deps), nethttp.SuccessStatus(http.StatusAccepted))

		s.Delete("/album/{name}/{hash}", usecase.RemoveFromAlbum(deps))
		s.Post("/album/{name}/{hash}", usecase.AddToAlbum(deps))
	})

	s.Get("/album/{name}.json", usecase.GetAlbum(deps))
	s.Get("/image/{hash}.json", usecase.GetImage(deps))
	s.Get("/album/{name}.zip", usecase.DownloadAlbum(deps))

	s.Get("/image/{hash}.jpg", usecase.ShowImage(deps))
	s.Get("/thumb/{size}/{hash}.jpg", usecase.ShowThumb(deps))

	s.Method(http.MethodGet, "/", ui.Static)

	s.Get("/{name}/", usecase.ShowAlbum(deps))
	s.Get("/{name}/pano-{hash}.html", usecase.ShowPano(deps))

	s.Post("/make-pass-hash", usecase.MakePassHash())

	s.Mount("/static/", http.StripPrefix("/static", ui.Static))

	return s
}

func fork(s *web.Service, r chi.Router) *web.Service {
	f := *s

	if w, ok := r.(*chirouter.Wrapper); ok {
		f.Wrapper = w

		return &f
	}

	w := *f.Wrapper
	w.Router = r
	f.Wrapper = &w

	return &f
}

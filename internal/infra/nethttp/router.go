// Package nethttp manages application http interface.
package nethttp

import (
	"context"
	"net/http"

	"github.com/bool64/brick"
	"github.com/go-chi/chi/v5"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/nethttp/ui"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/usecase"
	"github.com/vearutop/photo-blog/internal/usecase/control"
)

// NewRouter creates an instance of router filled with handlers and docs.
func NewRouter(deps *service.Locator, cfg service.Config) http.Handler {
	s := brick.NewBaseWebService(deps.BaseLocator)
	deps.CtxdLogger().Important(context.Background(), "initializing router")

	s.Group(func(r chi.Router) {
		s := fork(s, r)

		if cfg.AdminPassHash != "" {
			adminAuth := basicAuth("Admin Access", cfg.AdminPassHash, cfg.AdminPassSalt)
			s.Use(nethttp.AnnotateOpenAPI(s.OpenAPICollector, func(op *openapi3.Operation) error {
				op.Tags = append(op.Tags, "Control Panel")

				return nil
			}))
			s.Use(adminAuth, nethttp.HTTPBasicSecurityMiddleware(s.OpenAPICollector, "Admin", "Admin access"))
		}

		s.Post("/album", control.CreateAlbum(deps))
		s.Post("/album/{name}/directory", control.AddDirectory(deps, control.IndexAlbum(deps)))
		s.Post("/album/{name}/images", control.UploadImages(deps))

		s.Get("/albums.json", usecase.GetAlbums(deps))
		s.Post("/index/{name}", control.IndexAlbum(deps), nethttp.SuccessStatus(http.StatusAccepted))

		s.Delete("/album/{name}/{hash}", control.RemoveFromAlbum(deps))
		s.Post("/album/{name}/{hash}", control.AddToAlbum(deps))

		s.Get("/edit/image/{hash}.html", control.EditImage(deps))
		s.Get("/edit/album/{hash}.html", control.EditAlbum(deps))
		s.Get("/edit/settings.html", control.EditSettings(deps))

		s.Get("/album/{hash}.json", control.Get(deps, func() uniq.Finder[photo.Album] { return deps.PhotoAlbumFinder() }))
		s.Get("/image/{hash}.json", control.Get(deps, func() uniq.Finder[photo.Image] { return deps.PhotoImageFinder() }))
		s.Get("/exif/{hash}.json", control.Get(deps, func() uniq.Finder[photo.Exif] { return deps.PhotoExifFinder() }))
		s.Get("/gps/{hash}.json", control.Get(deps, func() uniq.Finder[photo.Gps] { return deps.PhotoGpsFinder() }))
		s.Get("/settings.json", control.GetSettings(deps))
		s.Put("/settings.json", control.UpdateSettings(deps))

		s.Put("/album", control.Update(deps, func() uniq.Ensurer[photo.Album] { return deps.PhotoAlbumEnsurer() }))
		s.Put("/image", control.UpdateImage(deps))
		s.Put("/exif", control.Update(deps, func() uniq.Ensurer[photo.Exif] { return deps.PhotoExifEnsurer() }))
		s.Put("/gps", control.Update(deps, func() uniq.Ensurer[photo.Gps] { return deps.PhotoGpsEnsurer() }))

		s.Delete("/album/{name}", control.DeleteAlbum(deps))

		s.Get("/image-info/{hash}.json", usecase.GetImageInfo(deps))

		s.Get("/login", control.Login())
	})

	acu := usecase.GetAlbumContents(deps)
	s.Get("/album-contents/{name}.json", acu)

	// Visitors access log.
	s.Group(func(r chi.Router) {
		s := fork(s, r)

		if deps.Config.Settings.TagVisitors {
			s.Use(auth.VisitorMiddleware(deps.AccessLog()))
		}

		s.Get("/{name}/", usecase.ShowAlbum(deps, acu))
		s.Get("/album/{name}.zip", usecase.DownloadAlbum(deps))
		s.Get("/{name}/photo-{hash}.html", usecase.ShowAlbumAtImage(usecase.ShowAlbum(deps, acu)))
		s.Get("/{name}/pano-{hash}.html", usecase.ShowPano(deps))

		s.Get("/image/{hash}.jpg", usecase.ShowImage(deps, false))
		s.Get("/image/{hash}.avif", usecase.ShowImage(deps, true))
		s.Get("/thumb/{size}/{hash}.jpg", usecase.ShowThumb(deps))
		s.Get("/track/{hash}.gpx", usecase.DownloadGpx(deps))

		s.Group(func(r chi.Router) {
			s := fork(s, r)

			if cfg.AdminPassHash != "" {
				adminAuth := maybeAuth(cfg.AdminPassHash, cfg.AdminPassSalt)

				s.Use(adminAuth)
			}

			s.Get("/", usecase.ShowMain(deps, acu))
		})
	})

	// Redirecting `/my-album` to `/my-album/`.
	s.Method(http.MethodGet, "/{name}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
	}))

	s.Post("/make-pass-hash", usecase.MakePassHash())

	s.Mount("/static/", http.StripPrefix("/static", ui.Static))

	deps.SchemaRepo.Mount(s, "/json-form/")

	deps.CtxdLogger().Important(context.Background(), "router initialized successfully")

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

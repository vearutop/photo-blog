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
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/nethttp/ui"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/usecase"
	"github.com/vearutop/photo-blog/internal/usecase/control"
)

// NewRouter creates an instance of router filled with handlers and docs.
func NewRouter(deps *service.Locator, cfg service.Config) http.Handler {
	s := brick.NewBaseWebService(deps.BaseLocator)

	//s.Wrap(func(handler http.Handler) http.Handler {
	//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//		var h uniq.Hash
	//
	//		c, err := r.Cookie("h")
	//		if err == nil {
	//			if err := h.UnmarshalText([]byte(c.Value)); err == nil {
	//			}
	//		} else {
	//
	//		}
	//
	//		r = r.WithContext(ctxd.AddFields(r.Context(), "visitor", h))
	//	})
	//})

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
		s.Post("/album/{name}/directory", control.AddDirectory(deps))
		s.Post("/album/{name}/images", control.UploadImages(deps))

		s.Get("/albums.json", usecase.GetAlbums(deps))
		s.Post("/index/{name}", control.IndexAlbum(deps), nethttp.SuccessStatus(http.StatusAccepted))

		s.Delete("/album/{name}/{hash}", control.RemoveFromAlbum(deps))
		s.Post("/album/{name}/{hash}", control.AddToAlbum(deps))

		s.Get("/control/{name}/{id}", control.ShowForm(deps))
		s.Get("/edit/image/{hash}.html", control.EditImage(deps))

		s.Get("/album/{hash}.json", control.Get(deps, func() uniq.Finder[photo.Album] { return deps.PhotoAlbumFinder() }))
		s.Get("/image/{hash}.json", control.Get(deps, func() uniq.Finder[photo.Image] { return deps.PhotoImageFinder() }))
		s.Get("/exif/{hash}.json", control.Get(deps, func() uniq.Finder[photo.Exif] { return deps.PhotoExifFinder() }))
		s.Get("/gps/{hash}.json", control.Get(deps, func() uniq.Finder[photo.Gps] { return deps.PhotoGpsFinder() }))
		s.Get("/schema/{name}.json", control.GetSchema(deps))

		s.Put("/album", control.Update(deps, func() uniq.Ensurer[photo.Album] { return deps.PhotoAlbumEnsurer() }))
		s.Put("/image", control.UpdateImage(deps))
		s.Put("/exif", control.Update(deps, func() uniq.Ensurer[photo.Exif] { return deps.PhotoExifEnsurer() }))
		s.Put("/gps", control.Update(deps, func() uniq.Ensurer[photo.Gps] { return deps.PhotoGpsEnsurer() }))

		s.Get("/image-info/{hash}.json", usecase.GetImageInfo(deps))
	})

	s.Get("/album-images/{name}.json", usecase.GetAlbumImages(deps))
	s.Get("/album/{name}.zip", usecase.DownloadAlbum(deps))

	s.Get("/image/{hash}.jpg", usecase.ShowImage(deps))
	s.Get("/thumb/{size}/{hash}.jpg", usecase.ShowThumb(deps))

	s.Method(http.MethodGet, "/", ui.Static)

	s.Get("/{name}/", usecase.ShowAlbum(deps))
	s.Get("/{name}/photo-{hash}.html", usecase.ShowAlbumAtImage(usecase.ShowAlbum(deps)))
	s.Get("/{name}/pano-{hash}.html", usecase.ShowPano(deps))

	s.Post("/make-pass-hash", usecase.MakePassHash())

	s.Mount("/static/", http.StripPrefix("/static", ui.Static))
	s.Handle("/json-form.html", ui.Static)

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

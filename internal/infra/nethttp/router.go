// Package nethttp manages application http interface.
package nethttp

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/bool64/brick"
	"github.com/go-chi/chi/v5"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/nethttp/ui"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/upload"
	"github.com/vearutop/photo-blog/internal/infra/webdav"
	"github.com/vearutop/photo-blog/internal/usecase"
	"github.com/vearutop/photo-blog/internal/usecase/control"
	"github.com/vearutop/photo-blog/internal/usecase/control/debug"
	"github.com/vearutop/photo-blog/internal/usecase/control/settings"
	"github.com/vearutop/photo-blog/internal/usecase/help"
	"github.com/vearutop/photo-blog/pkg/txt"
	"golang.org/x/text/language"
)

// NewRouter creates an instance of router filled with handlers and docs.
func NewRouter(deps *service.Locator) *web.Service {
	deps.BaseLocator.HTTPServerMiddlewares = append(deps.BaseLocator.HTTPServerMiddlewares, func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := time.Now()
			h.ServeHTTP(w, r)

			ctx := r.Context()
			if err := ctx.Err(); err != nil {
				deps.CtxdLogger().Error(ctx, "http ctx err",
					"error", err,
					"elapsed", time.Since(s).String(),
					"req", r.URL.String(),
				)
			}
		})
	})

	s := brick.NewBaseWebService(deps.BaseLocator)
	deps.CtxdLogger().Important(context.Background(), "initializing router")

	s.Group(func(r chi.Router) {
		s := fork(s, r)

		adminAuth := auth.BasicAuth("Admin Access", deps.Settings)
		s.Use(nethttp.OpenAPIAnnotationsMiddleware(s.OpenAPICollector, func(oc openapi.OperationContext) error {
			oc.SetTags(append(oc.Tags(), "Control Panel")...)

			return nil
		}))

		s.Use(adminAuth, nethttp.HTTPBasicSecurityMiddleware(s.OpenAPICollector, "Admin", "Admin access"))

		// WebDAV server configuration.
		for _, m := range strings.Split("OPTIONS, MKCOL, LOCK, GET, HEAD, POST, DELETE, PROPPATCH, COPY, MOVE, UNLOCK, PROPFIND, PUT", ", ") {
			chi.RegisterMethod(m)
		}
		wh := webdav.NewHandler(deps.CtxdLogger(), deps.Settings())
		s.Handle("/webdav", wh)
		s.Mount("/webdav/", wh)
		// End of WebDAV.

		s.Post("/album", control.CreateAlbum(deps))
		s.Post("/album/{name}/directory", control.AddDirectory(deps, control.IndexAlbum(deps)))

		s.Get("/albums.json", usecase.GetAlbums(deps))
		s.Post("/index/{name}", control.IndexAlbum(deps), nethttp.SuccessStatus(http.StatusAccepted))
		s.Post("/gather/{name}", control.GatherFiles(deps))

		s.Delete("/album/{name}/{hash}", control.RemoveFromAlbum(deps))
		s.Post("/album/{name}", control.AddToAlbum(deps))

		s.Get("/add-album.html", control.AddAlbum(deps))

		s.Get("/edit/image/{hash}.html", control.EditImage(deps))
		s.Get("/edit/album/{hash}.html", control.EditAlbum(deps))

		s.Get("/edit/password.html", settings.EditAdminPassword(deps))
		s.Post("/settings/password.json", settings.SetPassword(deps))
		s.Get("/edit/settings.html", settings.Edit(deps))
		s.Post("/settings/appearance.json", settings.SetAppearance(deps))
		s.Post("/settings/maps.json", settings.SetMaps(deps))
		s.Post("/settings/visitors.json", settings.SetVisitors(deps))
		s.Post("/settings/storage.json", settings.SetStorage(deps))
		s.Post("/settings/privacy.json", settings.SetPrivacy(deps))

		s.Get("/album/{hash}.json", control.Get(deps, func() uniq.Finder[photo.Album] { return deps.PhotoAlbumFinder() }))
		s.Get("/image/{hash}.json", control.Get(deps, func() uniq.Finder[photo.Image] { return deps.PhotoImageFinder() }))
		s.Get("/exif/{hash}.json", control.Get(deps, func() uniq.Finder[photo.Exif] { return deps.PhotoExifFinder() }))
		s.Get("/gps/{hash}.json", control.Get(deps, func() uniq.Finder[photo.Gps] { return deps.PhotoGpsFinder() }))

		s.Put("/album", control.Update(deps, func() uniq.Ensurer[photo.Album] { return deps.PhotoAlbumEnsurer() }))
		s.Put("/image", control.Update(deps, func() uniq.Ensurer[photo.Image] { return deps.PhotoImageEnsurer() }))
		s.Put("/exif", control.Update(deps, func() uniq.Ensurer[photo.Exif] { return deps.PhotoExifEnsurer() }))
		s.Put("/gps", control.Update(deps, func() uniq.Ensurer[photo.Gps] { return deps.PhotoGpsEnsurer() }))

		s.Put("/album-image-time", control.SetAlbumImageTime(deps))

		s.Delete("/album/{name}", control.DeleteAlbum(deps))

		s.Post("/message/approve", control.ApproveMessage(deps))

		if err := upload.MountTus(s, deps); err != nil {
			panic(err)
		}

		s.Get("/image-info/{hash}.json", usecase.GetImageInfo(deps))

		s.Get("/login", control.Login())

		s.Get("/db.html", debug.DBConsole(deps))
		s.Post("/query-db", debug.DBQuery(deps))
		s.Get("/query-db.csv", debug.DBQueryCSV(deps))
	})

	s.Get("/album-contents/{name}.json", usecase.GetAlbumContents(deps))

	// Visitors access log.
	s.Group(func(r chi.Router) {
		s := fork(s, r)

		adminAuth := auth.MaybeAuth(deps.Settings())
		s.Use(adminAuth)

		s.Use(auth.VisitorMiddleware(deps.AccessLog(), deps.Settings()))

		s.Use(func(handler http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Accept-Ch", "Downlink, Sec-CH-UA-Model, Sec-CH-UA-Platform, Sec-CH-UA-Platform-Version")

				handler.ServeHTTP(w, r)
			})
		})

		// Supported content language matching.
		s.Use(func(handler http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				matcher, languages := deps.Settings().Appearance().LanguageMatcher()

				if matcher != nil {
					queryLang := r.URL.Query().Get("lang")
					cookieLang := ""
					if c, err := r.Cookie("lang"); err == nil {
						cookieLang = c.Value
					}

					accept := r.Header.Get("Accept-Language")
					tag, i := language.MatchStrings(matcher, queryLang, cookieLang, accept)
					lang := languages[i]

					deps.CtxdLogger().Debug(context.Background(), "matching language",
						"lang", tag.String(),
						"idx", i,
						"cookie", cookieLang,
						"query", queryLang,
						"accept", accept,
						"lang", lang,
					)

					if queryLang != "" {
						c := http.Cookie{
							Name: "lang", Value: queryLang,
							SameSite: http.SameSiteStrictMode, MaxAge: 3 * 365 * 86400,
						} // Around 3 years.

						http.SetCookie(w, &c)
					}

					r = r.WithContext(txt.WithLanguage(r.Context(), lang))
				}

				handler.ServeHTTP(w, r)
			})
		})

		s.Get("/help/", help.Index(deps))
		s.Get("/help/{file}.md", help.Markdown(deps))
		s.Get("/help/{file}", help.ServeFile(deps))

		s.Get("/", usecase.ShowMain(deps))
		s.Get("/{name}/", usecase.ShowAlbum(deps))
		s.Get("/{name}/photo-{hash}.html", usecase.ShowAlbumAtImage(usecase.ShowAlbum(deps)))

		s.Get("/poi/photos-{name}.gpx", usecase.DownloadImagesPoiGpx(deps))
		s.Get("/album/{name}.zip", usecase.DownloadAlbum(deps))
		s.Get("/{name}/pano-{hash}.html", usecase.ShowPano(deps))

		s.Get("/image/{hash}.jpg", usecase.ShowImage(deps, false))
		s.Get("/image/{hash}.avif", usecase.ShowImage(deps, true))
		s.Get("/thumb/{size}/{hash}.jpg", usecase.ShowThumb(deps))
		s.Get("/track/{hash}.gpx", usecase.DownloadGpx(deps))

		s.Post("/message", usecase.AddMessage(deps))

		s.Get("/site/{file}", usecase.ServeSiteFile(deps))
		s.Get("/stats", usecase.CollectStats(deps))
	})

	s.Get("/favicon.ico", usecase.ServeFavicon(deps))
	s.Method(http.MethodGet, "/robots.txt", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nAllow: /"))
	}))

	// Redirecting `/my-album` to `/my-album/`.
	s.Method(http.MethodGet, "/{name}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
	}))

	s.Get("/map-tile/{r}/{z}/{x}/{y}.png", usecase.MapTile(deps))

	s.Mount("/static/", http.StripPrefix("/static", ui.Static))

	deps.SchemaRepo.Mount(s, "/json-form/")

	s.Get("/og.html", usecase.OG(deps))

	s.OnNotFound(usecase.NotFound(deps))
	s.OnMethodNotAllowed(usecase.NotFound(deps))

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

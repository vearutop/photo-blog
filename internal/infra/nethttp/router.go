// Package nethttp manages application http interface.
package nethttp

import (
	"context"
	"encoding/base64"
	"net/http"
	"runtime/coverage"
	"strings"
	"time"

	"github.com/bool64/brick"
	"github.com/go-chi/chi/v5"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/rest/chirouter"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
	"github.com/vearutop/dbcon/dbcon"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/nethttp/ui"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/upload"
	"github.com/vearutop/photo-blog/internal/infra/webdav"
	"github.com/vearutop/photo-blog/internal/usecase"
	"github.com/vearutop/photo-blog/internal/usecase/control"
	"github.com/vearutop/photo-blog/internal/usecase/control/settings"
	"github.com/vearutop/photo-blog/internal/usecase/help"
	"github.com/vearutop/photo-blog/internal/usecase/stats"
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
				deps.CtxdLogger().Warn(ctx, "http ctx err",
					"error", err,
					"elapsed", time.Since(s).String(),
					"req", r.URL.String(),
				)
			}
		})
	})

	s := brick.NewBaseWebService(deps.BaseLocator)
	deps.CtxdLogger().Important(context.Background(), "initializing router")

	s.AddHeadToGet = true

	if deps.DebugRouter != nil {
		deps.DebugRouter.Get("/coverage-meta", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "binary/octet-stream")
			if err := coverage.WriteMeta(w); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
		deps.DebugRouter.AddLink("coverage-meta", "Coverage Meta")

		deps.DebugRouter.Get("/coverage-counters", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "binary/octet-stream")
			if err := coverage.WriteCounters(w); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
		deps.DebugRouter.AddLink("coverage-counters", "Coverage Counters")

		deps.DebugRouter.Mount("/db", dbcon.Handler("/debug/db/", deps, func(options *dbcon.Options) {
			options.AddValueProcessor("hash", func(v any) any {
				if i, ok := v.(int64); ok {
					return uniq.Hash(i).String()
				}

				return v
			})

			options.AddValueProcessor("thumb", func(v any) any {
				if i, ok := v.(int64); ok {
					s := uniq.Hash(i).String()
					return `<a href="/list-` + s + `/"><img src="/thumb/300w/` + s + `.jpg" /></a>`
				}

				return v
			})

			options.AddValueProcessor("visitor", func(v any) any {
				if i, ok := v.(int64); ok {
					s := uniq.Hash(i).String()
					return `<a href="/stats/visitor/` + s + `.html">` + s + `</a>`
				}

				return v
			})

			options.AddValueProcessor("img", func(v any) any {
				if b, ok := v.([]byte); ok {
					ct := http.DetectContentType(b)
					return `<img src="data:` + ct + `;base64,` + base64.StdEncoding.EncodeToString(b) + `" />`
				}

				if s, ok := v.(string); ok {
					b, err := base64.StdEncoding.DecodeString(s)
					if err != nil {
						return err.Error()
					}
					ct := http.DetectContentType(b)

					return `<img src="data:` + ct + `;base64,` + s + `" />`
				}

				return v
			})

			options.AddValueProcessor("msDuration", func(v any) any {
				if i, ok := v.(int64); ok {
					return (time.Duration(i) * time.Millisecond).String()
				}

				return v
			})
		}))
		deps.DebugRouter.AddLink("db", "DB Console")
	}

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

		addDir := control.AddDirectory(deps, control.IndexAlbum(deps))
		s.Post("/album/{name}/directory", addDir)
		s.Post("/album/add-recursive", control.AddDirectoryRecursive(deps, addDir))
		s.Post("/album/{name}/url", control.AddRemote(deps))

		s.Get("/albums.json", usecase.GetAlbums(deps))
		s.Post("/index/{name}", control.IndexAlbum(deps), nethttp.SuccessStatus(http.StatusAccepted))
		s.Post("/index-remote", control.IndexRemote(deps), nethttp.SuccessStatus(http.StatusAccepted))
		s.Post("/cleanup-remote", control.CleanupRemote(deps), nethttp.SuccessStatus(http.StatusAccepted))
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
		s.Post("/settings/external_api.json", settings.SetExternalAPI(deps))
		s.Post("/settings/image_prompt.json", settings.SetImagePrompt(deps))
		s.Post("/settings/indexing.json", settings.SetIndexing(deps))

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

		s.Get("/image-info/{hash}.json", usecase.GetImageInfo(deps))

		s.Get("/login", control.Login())
		s.Get("/settings/version.html", control.Version())
		s.Get("/settings/self-update", control.SelfUpdate())

		// Stats.
		s.Get("/stats/daily.html", stats.ShowDailyTotal(deps))
		s.Get("/stats/top-pages.html", stats.TopPages(deps))
		s.Get("/stats/top-images.html", stats.TopImages(deps))
		s.Get("/stats/refers.html", stats.ShowRefers(deps))
		s.Get("/stats/visitor/{hash}.html", stats.ShowVisitor(deps))
	})

	maybeAuth := auth.MaybeAuth(deps.Settings())

	// CollabKey or Admin
	s.Group(func(r chi.Router) {
		s := fork(s, r)

		s.Use(maybeAuth)

		if err := upload.MountTus(s, deps); err != nil {
			panic(err)
		}
	})

	s.Get("/album-contents/{name}.json", usecase.GetAlbumContents(deps))

	// Visitors access log.
	s.Group(func(r chi.Router) {
		s := fork(s, r)

		s.Use(maybeAuth)

		s.Use(auth.VisitorMiddleware(deps.AccessLog(), deps.Settings(), deps.VisitorStats()))

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
		showAlbum := usecase.ShowAlbum(deps)
		s.Get("/{name}/", showAlbum)
		s.Get("/{name}/photo-{hash}.html", usecase.ShowAlbumAtImage(showAlbum))

		s.Get("/search/", usecase.SearchImages(deps))

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

		s.Post("/favorite", usecase.AddFavorite(deps))
		s.Delete("/favorite", usecase.DeleteFavorite(deps))
		s.Get("/favorite", usecase.GetFavorite(deps))
	})

	s.Get("/favicon.ico", usecase.ServeFavicon(deps))
	s.Method(http.MethodGet, "/robots.txt", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("User-agent: *\nAllow: /"))
	}))

	// Redirecting `/my-album` to `/my-album/`.
	s.Method(http.MethodGet, "/{name}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
	}))

	// s.Get("/map-tile/{s}/{r}/{z}/{x}/{y}.png", usecase.MapTile(deps))
	s.Get("/map-tile/{s}/{r}/{z}/{x}/{y}.png", usecase.MapTile(deps))

	s.Mount("/static/", http.StripPrefix("/static", ui.Static))

	deps.SchemaRepo.Mount(s, "/json-form/")

	s.Handle("/og.html", nethttp.NewHandler(usecase.OG(deps)))

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

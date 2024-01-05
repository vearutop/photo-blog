package webdav

import (
	"net/http"

	"github.com/bool64/ctxd"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"golang.org/x/net/webdav"
)

func NewHandler(logger ctxd.Logger, cfg settings.Values) http.Handler {
	handler := &webdav.Handler{
		Prefix:     "/webdav",
		FileSystem: webdav.Dir("."),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				logger.Error(r.Context(), "WebDAV failed",
					"method", r.Method,
					"url", r.URL.String(),
					"error", err)
			} else {
				logger.Info(r.Context(), "WebDAV served",
					"method", r.Method,
					"url", r.URL.String())
			}
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cfg.Storage().WebDAV {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("WebDAV access is disable by configuration\n"))

			return
		}

		handler.ServeHTTP(w, r)
	})
}

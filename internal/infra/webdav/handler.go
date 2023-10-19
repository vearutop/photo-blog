package webdav

import (
	"net/http"

	"github.com/bool64/ctxd"
	"golang.org/x/net/webdav"
)

func NewHandler(path string, logger ctxd.Logger) *webdav.Handler {
	handler := &webdav.Handler{
		Prefix:     "/webdav/",
		FileSystem: webdav.Dir(path),
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

	return handler
}

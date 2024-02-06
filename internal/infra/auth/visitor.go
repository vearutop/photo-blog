package auth

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"time"

	"github.com/bool64/ctxd"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/settings"
)

type Visitor struct {
	uniq.Head
	Latest      time.Time `db:"latest"`
	Hits        int       `db:"hits"`
	UserAgent   string    `db:"user_agent"`
	Referrer    string    `db:"referrer"`
	Destination string    `db:"destination"`
	RemoteAddr  string    `db:"remote_addr"`
}

type visitorCtxKey struct{}

// ContextWithVisitor adds visitor id to context.
func ContextWithVisitor(ctx context.Context, id uniq.Hash) context.Context {
	return context.WithValue(ctx, visitorCtxKey{}, id)
}

// VisitorFromContext returns visitor hash or 0.
func VisitorFromContext(ctx context.Context) uniq.Hash {
	if v, ok := ctx.Value(visitorCtxKey{}).(uniq.Hash); ok {
		return v
	}

	return 0
}

func VisitorMiddleware(logger ctxd.Logger, cfg settings.Values) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			visitors := cfg.Visitors()
			isNew := true

			if visitors.Tag {
				var h uniq.Hash

				c, err := r.Cookie("v")
				if err == nil {
					if err = h.UnmarshalText([]byte(c.Value)); err != nil || h == 0 {
						h = uniq.Hash(rand.Int())
					}

					c := http.Cookie{Name: "v", Value: h.String(),
						SameSite: http.SameSiteStrictMode, MaxAge: 3 * 365 * 86400} // Around 3 years.

					http.SetCookie(w, &c)

					isNew = false
				} else if errors.Is(err, http.ErrNoCookie) {
					if v := r.URL.Query().Get("v"); v != "" {
						_ = h.UnmarshalText([]byte(v))
						isNew = false
					} else {
						h = uniq.Hash(rand.Int())
					}

					c := http.Cookie{Name: "v", Value: h.String(),
						SameSite: http.SameSiteStrictMode, MaxAge: 3 * 365 * 86400} // Around 3 years.

					http.SetCookie(w, &c)
				}

				if h != 0 {
					r = r.WithContext(ContextWithVisitor(ctxd.AddFields(r.Context(), "visitor", h), h))
				}
			}

			if logger != nil && visitors.AccessLog {
				logger.Important(r.Context(), "access",
					"new_visitor", isNew,
					"host", r.Host,
					"url", r.URL.String(),
					"user_agent", r.Header.Get("User-Agent"),
					"referer", r.Header.Get("Referer"),
					"forwarded_for", r.Header.Get("X-Forwarded-For"),
				)
			}

			handler.ServeHTTP(w, r)
		})
	}
}

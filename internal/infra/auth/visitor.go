package auth

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"time"

	"github.com/bool64/ctxd"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
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

func VisitorMiddleware(logger ctxd.Logger) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var h uniq.Hash

			isNew := false

			c, err := r.Cookie("v")
			if err == nil {
				_ = h.UnmarshalText([]byte(c.Value))
			} else if errors.Is(err, http.ErrNoCookie) {
				h = uniq.Hash(rand.Int())

				c := http.Cookie{Name: "v", Value: h.String(), HttpOnly: true, MaxAge: 3 * 30 * 86400}
				http.SetCookie(w, &c)

				isNew = true
			}

			if h != 0 {
				r = r.WithContext(ContextWithVisitor(ctxd.AddFields(r.Context(), "visitor", h), h))
			}

			if logger != nil {
				logger.Important(r.Context(), "access",
					"is_new", isNew,
					"url", r.URL.String(),
					"user_agent", r.Header.Get("User-Agent"),
					"referer", r.Header.Get("Referer"),
					"remote_addr", r.RemoteAddr,
					"forwarded_for", r.Header.Get("X-Forwarded-For"),
				)
			}

			handler.ServeHTTP(w, r)
		})
	}
}

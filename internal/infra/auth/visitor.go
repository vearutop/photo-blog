package auth

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strings"

	"github.com/bool64/ctxd"
	"github.com/cespare/xxhash/v2"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/storage/visitor"
	"github.com/vearutop/photo-blog/pkg/webstats"
)

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

func VisitorMiddleware(logger ctxd.Logger, cfg settings.Values, st *visitor.Stats) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			visitors := cfg.Visitors()
			isNew := true

			if visitors.Tag {
				var h uniq.Hash

				c, err := r.Cookie("v")
				if err == nil {
					if err = h.UnmarshalText([]byte(c.Value)); err != nil || h == 0 {
						if webstats.IsBot(r.UserAgent()) {
							h = uniq.Hash(xxhash.Sum64String(r.UserAgent())) // Fixed value of visitor for bots.
						} else {
							h = uniq.Hash(rand.Int())
						}

						c := http.Cookie{
							Name: "v", Value: h.String(),
							SameSite: http.SameSiteStrictMode, MaxAge: 3 * 365 * 86400,
						} // Around 3 years.

						http.SetCookie(w, &c)
					} else {
						isNew = false
					}
				} else if errors.Is(err, http.ErrNoCookie) {
					if v := r.URL.Query().Get("v"); v != "" {
						_ = h.UnmarshalText([]byte(v))
						isNew = false
					} else {
						if webstats.IsBot(r.UserAgent()) {
							h = uniq.Hash(xxhash.Sum64String(r.UserAgent())) // Fixed value of visitor for bots.
						} else {
							h = uniq.Hash(rand.Int())
						}
					}

					c := http.Cookie{
						Name: "v", Value: h.String(),
						SameSite: http.SameSiteStrictMode, MaxAge: 3 * 365 * 86400,
					} // Around 3 years.

					http.SetCookie(w, &c)
				}

				if h != 0 {
					r = r.WithContext(ContextWithVisitor(ctxd.AddFields(r.Context(), "visitor", h), h))
				}

				st.CollectVisitor(h, r)
			}

			if logger != nil && visitors.AccessLog {
				h := r.Header
				logger.Important(r.Context(), "access",
					"new_visitor", isNew,
					"host", r.Host,
					"url", r.URL.String(),
					"user_agent", h.Get("User-Agent"),
					"device", strings.TrimSpace(
						strings.Trim(h.Get("Sec-Ch-Ua-Model"), `"`)+" "+
							strings.Trim(h.Get("Sec-Ch-Ua-Platform"), `"`)+" "+
							strings.Trim(h.Get("Sec-Ch-Ua-Platform-Version"), `"`),
					),
					"referer", h.Get("Referer"),
					"forwarded_for", h.Get("X-Forwarded-For"),
					"admin", IsAdmin(r.Context()),
					"lang", r.Header.Get("Accept-Language"),
				)
			}

			handler.ServeHTTP(w, r)
		})
	}
}

package auth

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/bool64/cache"
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

func VisitorMiddleware(logger ctxd.Logger, cfg settings.Values, st *visitor.StatsRepository) func(handler http.Handler) http.Handler {
	recentVisitors := cache.NewFailoverOf[uniq.Hash](func(cfg *cache.FailoverConfigOf[uniq.Hash]) {
		cfg.BackendConfig.TimeToLive = 15 * time.Minute
	})

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			visitors := cfg.Visitors()
			isNew := true

			device := strings.TrimSpace(
				strings.Trim(r.Header.Get("Sec-Ch-Ua-Model"), `"`) + " " +
					strings.Trim(r.Header.Get("Sec-Ch-Ua-Platform"), `"`) + " " +
					strings.Trim(r.Header.Get("Sec-Ch-Ua-Platform-Version"), `"`),
			)

			ctx := r.Context()

			isAdmin := IsAdmin(ctx)
			isBot := webstats.IsBot(r.UserAgent())

			setNewVisitorCookie := func(ctx context.Context) (h uniq.Hash) {
				if isBot {
					h = uniq.Hash(xxhash.Sum64String(r.UserAgent())) // Fixed value of visitor for bots.
				} else {
					h, _ = recentVisitors.Get(ctx, []byte(r.UserAgent()+device+r.Header.Get("Accept-Language")+r.Header.Get("X-Forwarded-For")),
						func(ctx context.Context) (uniq.Hash, error) {
							return uniq.Hash(rand.Int()), nil
						})
				}

				c := http.Cookie{
					Name: "v", Value: h.String(),
					SameSite: http.SameSiteStrictMode, MaxAge: 3 * 365 * 86400,
				} // Around 3 years.

				http.SetCookie(w, &c)

				return h
			}

			if visitors.Tag {
				var h uniq.Hash

				c, err := r.Cookie("v")
				if err == nil {
					if err = h.UnmarshalText([]byte(c.Value)); err != nil || h == 0 {
						h = setNewVisitorCookie(ctx)
					} else {
						isNew = false
					}
				} else if errors.Is(err, http.ErrNoCookie) {
					if v := r.URL.Query().Get("v"); v != "" {
						_ = h.UnmarshalText([]byte(v))
						isNew = false

						c := http.Cookie{
							Name: "v", Value: h.String(),
							SameSite: http.SameSiteStrictMode, MaxAge: 3 * 365 * 86400,
						} // Around 3 years.

						http.SetCookie(w, &c)
					} else {
						h = setNewVisitorCookie(ctx)
					}
				}

				if h != 0 {
					r = r.WithContext(ContextWithVisitor(ctxd.AddFields(r.Context(), "visitor", h), h))
				}

				st.CollectVisitor(h, isBot, isAdmin, r)
			}

			if logger != nil && visitors.AccessLog {
				h := r.Header
				logger.Important(r.Context(), "access",
					"new_visitor", isNew,
					"host", r.Host,
					"url", r.URL.String(),
					"user_agent", r.UserAgent(),
					"device", device,
					"referer", h.Get("Referer"),
					"forwarded_for", h.Get("X-Forwarded-For"),
					"admin", isAdmin,
					"bot", isBot,
					"lang", r.Header.Get("Accept-Language"),
				)
			}

			handler.ServeHTTP(w, r)
		})
	}
}

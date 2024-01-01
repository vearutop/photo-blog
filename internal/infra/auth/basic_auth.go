package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bool64/cache"
	"github.com/vearutop/photo-blog/internal/infra/settings"
)

var authCache = cache.NewFailoverOf[string](func(cfg *cache.FailoverConfigOf[string]) {
	cfg.BackendConfig.CountSoftLimit = 100
})

func MaybeAuth(deps settings.Values) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sec := deps.Security()

			if sec.Disabled() {
				r = r.WithContext(SetAdmin(r.Context()))
				next.ServeHTTP(w, r)

				return
			}

			_, pass, ok := r.BasicAuth()
			if ok {
				h, _ := authCache.Get(r.Context(), []byte(pass+sec.PassHash), func(ctx context.Context) (string, error) {
					return Hash(HashInput{
						Pass: pass,
						Salt: Salt(sec.PassSalt),
					}), nil
				})

				if h == sec.PassHash {
					r = r.WithContext(SetAdmin(r.Context()))
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// BasicAuth implements a simple middleware handler for adding basic http auth to a route.
func BasicAuth(realm string, deps func() settings.Values) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sec := deps().Security()

			if sec.Disabled() {
				r = r.WithContext(SetAdmin(r.Context()))
				next.ServeHTTP(w, r)

				return
			}

			_, pass, ok := r.BasicAuth()
			if !ok {
				basicAuthFailed(w, realm)
				return
			}

			h, _ := authCache.Get(r.Context(), []byte(pass+sec.PassHash), func(ctx context.Context) (string, error) {
				return Hash(HashInput{
					Pass: pass,
					Salt: Salt(sec.PassSalt),
				}), nil
			})

			if h != sec.PassHash {
				basicAuthFailed(w, realm)
				return
			}

			r = r.WithContext(SetAdmin(r.Context()))
			next.ServeHTTP(w, r)
		})
	}
}

func basicAuthFailed(w http.ResponseWriter, realm string) {
	w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
	w.WriteHeader(http.StatusUnauthorized)
}

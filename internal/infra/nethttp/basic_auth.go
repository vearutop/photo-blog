package nethttp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bool64/cache"
	"github.com/vearutop/photo-blog/internal/infra/auth"
)

var authCache = cache.NewFailoverOf[string](func(cfg *cache.FailoverConfigOf[string]) {
	cfg.BackendConfig.CountSoftLimit = 1000
})

func maybeAuth(hash, salt string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, pass, ok := r.BasicAuth()
			if ok {
				h, _ := authCache.Get(r.Context(), []byte(pass), func(ctx context.Context) (string, error) {
					return auth.Hash(auth.HashInput{
						Pass: pass,
						Salt: auth.Salt(salt),
					}), nil
				})

				if h == hash {
					r = r.WithContext(auth.SetAdmin(r.Context()))
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// basicAuth implements a simple middleware handler for adding basic http auth to a route.
func basicAuth(realm string, hash, salt string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, pass, ok := r.BasicAuth()
			if !ok {
				basicAuthFailed(w, realm)
				return
			}

			h, _ := authCache.Get(r.Context(), []byte(pass), func(ctx context.Context) (string, error) {
				return auth.Hash(auth.HashInput{
					Pass: pass,
					Salt: auth.Salt(salt),
				}), nil
			})

			if h != hash {
				basicAuthFailed(w, realm)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func basicAuthFailed(w http.ResponseWriter, realm string) {
	w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
	w.WriteHeader(http.StatusUnauthorized)
}

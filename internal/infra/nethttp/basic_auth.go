package nethttp

import (
	"fmt"
	"net/http"

	"github.com/vearutop/photo-blog/internal/infra/auth"
)

// basicAuth implements a simple middleware handler for adding basic http auth to a route.
func basicAuth(realm string, hash, salt string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, pass, ok := r.BasicAuth()
			if !ok {
				basicAuthFailed(w, realm)
				return
			}

			h := auth.Hash(auth.HashInput{
				Pass: pass,
				Salt: auth.Salt(salt),
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

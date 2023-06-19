package jsonform

import (
	"github.com/swaggest/rest/web"
	"net/http"
)

func (r *Repository) NewHandler(prefix string) http.Handler {
	return http.StripPrefix(prefix, staticServer)
}

type handler struct {
	r *Repository
}

func (r *Repository) Mount(s *web.Service, prefix string) {
	s.Get(prefix + "/{name}-schema.json")
	s.Mount(prefix, http.StripPrefix(prefix, staticServer))
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}

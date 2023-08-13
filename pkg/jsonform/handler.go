package jsonform

import (
	"context"
	"net/http"

	"github.com/swaggest/rest/web"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

func (r *Repository) Handler() http.Handler {
	return nil
}

func (r *Repository) Mount(s *web.Service, prefix string) {
	s.Get(prefix+"{name}-schema.json", r.GetSchema())
	s.Mount(prefix, http.StripPrefix(prefix, staticServer))
}

func (r *Repository) GetSchema() usecase.Interactor {
	type schemaReq struct {
		Name string `path:"name"`
	}

	u := usecase.NewInteractor[schemaReq, FormSchema](func(ctx context.Context, input schemaReq, output *FormSchema) error {
		if fs, found := r.schemas[input.Name]; found {
			*output = fs

			return nil
		}

		return status.NotFound
	})

	return u
}

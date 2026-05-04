package usecase

import (
	"context"

	"github.com/swaggest/usecase"
)

type shortenDeps interface{}

func Shorten(deps shortenDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input i, output *o) error {
	})

	return u
}

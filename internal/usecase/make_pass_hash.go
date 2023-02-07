package usecase

import (
	"context"
	"time"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/infra/auth"
)

func MakePassHash() usecase.Interactor {
	type makePassHashOutput struct {
		usecase.OutputWithEmbeddedWriter
		Elapsed string `header:"X-Elapsed"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in auth.HashInput, out *makePassHashOutput) error {
		start := time.Now()
		out.Elapsed = time.Since(start).String()

		h := auth.Hash(in)
		_, _ = out.Write([]byte("ADMIN_PASS_HASH=" + h + "\n"))
		_, _ = out.Write([]byte("ADMIN_PASS_SALT=" + in.Salt + "\n"))

		return nil
	})

	u.SetTags("Util")

	return u
}

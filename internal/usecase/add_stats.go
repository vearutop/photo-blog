package usecase

import (
	"context"
	"time"

	"github.com/bool64/ctxd"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/storage/visitor"
)

type collectStatsDeps interface {
	CtxdLogger() ctxd.Logger
	VisitorStats() *visitor.StatsRepository
}

func CollectStats(deps collectStatsDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input visitor.CollectStats, output *struct{}) error {
		deps.CtxdLogger().Info(ctx, "stats", "input", input,
			"admin", auth.IsAdmin(ctx), "bot", auth.IsBot(ctx))

		if auth.IsBot(ctx) || auth.IsAdmin(ctx) {
			return nil
		}

		deps.VisitorStats().CollectRequest(ctx, input, time.Now())

		return nil
	})

	return u
}

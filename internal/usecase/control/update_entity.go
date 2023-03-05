package control

import (
	"context"
	"fmt"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type updateEntityDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
}

// Update creates use case interactor to update entity data.
func Update[V any](deps updateEntityDeps, ensurer func() uniq.Ensurer[V]) usecase.Interactor {
	var v V
	t := fmt.Sprintf("%T", v)

	u := usecase.NewInteractor(func(ctx context.Context, in V, out *struct{}) (err error) {
		deps.StatsTracker().Add(ctx, "update_"+t, 1)
		deps.CtxdLogger().Info(ctx, "updating "+t, "value", in)

		return stripVal(ensurer().Ensure(ctx, in))
	})

	u.SetTitle("Update " + t)
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

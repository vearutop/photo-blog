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

type getEntityDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
}

type hashPath struct {
	Hash uniq.Hash `path:"hash"`
}

// Get creates use case interactor to get entity data.
func Get[V any](deps getEntityDeps, finder func() uniq.Finder[V]) usecase.Interactor {
	var v V

	t := fmt.Sprintf("%T", v)

	u := usecase.NewInteractor(func(ctx context.Context, in hashPath, out *V) (err error) {
		deps.StatsTracker().Add(ctx, "get_"+t, 1)
		deps.CtxdLogger().Info(ctx, "getting "+t, "hash", in.Hash)

		*out, err = finder().FindByHash(ctx, in.Hash)

		return err
	})

	u.SetTitle("Get " + t)
	u.SetName("control.Get[" + t + "]")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

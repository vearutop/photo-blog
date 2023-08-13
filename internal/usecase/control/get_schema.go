package control

import (
	"context"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/pkg/jsonform"
)

type getSchemaDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	SchemaRepository() *jsonform.Repository
}

// GetSchema creates use case interactor to get entity schema.
func GetSchema(deps getSchemaDeps) usecase.Interactor {
	type getSchemaInput struct {
		Name string `path:"name"`
	}
	u := usecase.NewInteractor(func(ctx context.Context, in getSchemaInput, out *jsonform.FormSchema) error {
		deps.StatsTracker().Add(ctx, "get_schema", 1)
		deps.CtxdLogger().Info(ctx, "getting schema", "name", in.Name)

		*out = deps.SchemaRepository().Schema(in.Name)

		return nil
	})

	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

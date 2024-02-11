package usecase

import (
	"context"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type collectStatsDeps interface{}

func CollectStats(deps collectStatsDeps) usecase.Interactor {
	type collectStatsRequest struct {
		Visitor uniq.Hash `query:"v"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input collectStatsRequest, output *struct{}) error {
		return nil
	})

	return u
}

package queue

import (
	"context"

	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/usecase/control"
	"github.com/vearutop/photo-blog/pkg/qlite"
)

func SetupBroker(deps *service.Locator) error {
	doIndexRemote := control.DoIndexRemote(deps)
	if err := qlite.AddConsumer[string](deps.QueueBroker(), doIndexRemote.Name(), func(ctx context.Context, v string) error {
		return doIndexRemote.Invoke(ctx, v, nil)
	}, func(o *qlite.ConsumerOptions) {
		o.Concurrency = 20
	}); err != nil {
		return err
	}

	return nil
}

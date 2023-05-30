package control

import (
	"context"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

type getSettingsDeps interface {
	ServiceSettings() service.Settings
}

// GetSettings creates use case interactor to get settings.
func GetSettings(deps getSettingsDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, _ struct{}, out *service.Settings) (err error) {
		*out = deps.ServiceSettings()

		return err
	})

	u.SetTitle("Get Settings")

	return u
}

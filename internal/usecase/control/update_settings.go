package control

import (
	"context"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

// UpdateSettings creates use case interactor to update settings.
func UpdateSettings(deps *service.Locator) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in service.Settings, out *struct{}) (err error) {
		_, err = deps.Storage.DB().Exec("UPDATE app SET settings = ?", in)
		if err != nil {
			return err
		}

		deps.Config.Settings = in

		return nil
	})

	u.SetTitle("Update Settings")
	u.SetTags("Control")
	u.SetExpectedErrors(status.Unknown)

	return u
}

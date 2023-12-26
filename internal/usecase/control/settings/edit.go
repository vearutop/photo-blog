package settings

import (
	"context"
	"net/http"

	"github.com/bool64/ctxd"
	"github.com/swaggest/jsonform-go"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/settings"
)

type editSettingsDeps interface {
	CtxdLogger() ctxd.Logger
	Settings() settings.Values
	SchemaRepository() *jsonform.Repository
}

// Edit creates use case interactor to show form.
func Edit(deps editSettingsDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *usecase.OutputWithEmbeddedWriter) error {
		form := func(title, url string, value any) jsonform.Form {
			return jsonform.Form{
				Title:         title,
				SubmitURL:     url,
				SubmitMethod:  http.MethodPost,
				SuccessStatus: http.StatusNoContent,
				Value:         value,
				SubmitText:    "Save",
				OnSuccess:     `formSaved`,
			}
		}

		return deps.SchemaRepository().Render(out.Writer,
			jsonform.Page{
				Title:       "Settings",
				PrependHTML: `<div style="margin: 2em"><a href="/">Back to main page</a></div> <script>function formSaved(x, ctx) { $(ctx.result).html('Saved.') } </script>`,
			},
			form("Appearance", "/settings/appearance.json", deps.Settings().Appearance()),
			form("Maps", "/settings/maps.json", deps.Settings().Maps()),
			form("Visitors", "/settings/visitors.json", deps.Settings().Visitors()),
			form("Storage", "/settings/storage.json", deps.Settings().Storage()),
		)
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown)

	return u
}

func SetAppearance(deps setSettingsDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input settings.Appearance, output *struct{}) error {
		return deps.SettingsManager().SetAppearance(ctx, input)
	})

	return u
}

func SetMaps(deps setSettingsDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input settings.Maps, output *struct{}) error {
		return deps.SettingsManager().SetMaps(ctx, input)
	})

	return u
}

func SetVisitors(deps setSettingsDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input settings.Visitors, output *struct{}) error {
		return deps.SettingsManager().SetVisitors(ctx, input)
	})

	return u
}

func SetStorage(deps setSettingsDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input settings.Storage, output *struct{}) error {
		return deps.SettingsManager().SetStorage(ctx, input)
	})

	return u
}

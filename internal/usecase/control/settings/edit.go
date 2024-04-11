package settings

import (
	"context"
	"html/template"
	"net/http"

	"github.com/bool64/ctxd"
	"github.com/bool64/dev/version"
	"github.com/swaggest/jsonform-go"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/upload"
)

type editSettingsDeps interface {
	CtxdLogger() ctxd.Logger
	Settings() settings.Values
	SchemaRepository() *jsonform.Repository
}

// Edit creates use case interactor to show form.
func Edit(deps editSettingsDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *usecase.OutputWithEmbeddedWriter) error {
		form := func(title, url string, value any, options ...func(f *jsonform.Form)) jsonform.Form {
			f := jsonform.Form{
				Title:         title,
				SubmitURL:     url,
				SubmitMethod:  http.MethodPost,
				SuccessStatus: http.StatusNoContent,
				Value:         value,
				SubmitText:    "Save",
				OnSuccess:     `formSaved`,
			}

			for _, o := range options {
				o(&f)
			}

			return f
		}

		return deps.SchemaRepository().Render(out.Writer,
			jsonform.Page{
				AppendHTMLHead: `
    <link rel="stylesheet" href="/static/style.css">
    <link rel="stylesheet" href="/static/tus/uppy.min.css">
    <script src="/static/tus/uppy.legacy.min.js"></script>

`,
				Title: "Settings",
				PrependHTML: `<a style="margin-left: 2em" href ="/">Back to main page</a> ` +
					upload.TusUploadsButton() +
					`<script>function formSaved(x, ctx) { $(ctx.result).html('Saved.') } </script>`,
				AppendHTML: `<div style="margin:2em">` + template.HTML(version.Info().String()) + `</div>`,
			},
			form("Appearance", "/settings/appearance.json", deps.Settings().Appearance()),
			form("Maps", "/settings/maps.json", deps.Settings().Maps()),
			form("Visitors", "/settings/visitors.json", deps.Settings().Visitors()),
			form("Storage", "/settings/storage.json", deps.Settings().Storage()),
			form("Privacy", "/settings/privacy.json", deps.Settings().Privacy(), func(f *jsonform.Form) {
				f.Description = "These settings do not affect how pages look for admin user, only for guests."
			}),
			jsonform.Form{
				Title: "Set Admin Password",
				Description: "<p>Add password protection to editing of albums and uploading of images.</p>" +
					"<p>For local or externally protected instance, password protection can be removed by setting an empty password.</p>",
				SubmitURL:     "/settings/password.json",
				SubmitMethod:  http.MethodPost,
				SuccessStatus: http.StatusNoContent,
				Value:         adminPass{},
				SubmitText:    "Save",
			},
			form("External API Integration", "/settings/external_api.json", deps.Settings().ExternalAPI()),
		)
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown)

	return u
}

func SetExternalAPI(deps setSettingsDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input settings.ExternalAPI, output *struct{}) error {
		return deps.SettingsManager().SetExternalAPI(ctx, input)
	})

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

func SetPrivacy(deps setSettingsDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input settings.Privacy, output *struct{}) error {
		return deps.SettingsManager().SetPrivacy(ctx, input)
	})

	return u
}

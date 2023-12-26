package settings

import (
	"context"
	"net/http"

	"github.com/swaggest/jsonform-go"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

type adminPass struct {
	Password       string `json:"password" formType:"password" title:"Password"`
	RepeatPassword string `json:"repeatPassword" formType:"password" title:"Repeat Password"`
}

// EditAdminPassword creates use case interactor to show form.
func EditAdminPassword(deps *service.Locator) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *usecase.OutputWithEmbeddedWriter) error {
		return deps.SchemaRepository().Render(out.Writer,
			jsonform.Page{
				Title: "Set Admin Password",
			},

			jsonform.Form{
				Title: "Set Admin Password",
				Description: "<p>Add password protection to editing of albums and uploading of images.</p>" +
					"<p>For local or externally protected instance, password protection can be removed by setting an empty password.</p>" +
					`<a href="/">Back to main page.</a>`,
				SubmitURL:     "/settings/password.json",
				SubmitMethod:  http.MethodPost,
				SuccessStatus: http.StatusNoContent,
				Value:         adminPass{},
				SubmitText:    "Save",
			},
		)
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown)

	return u
}

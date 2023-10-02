package control

import (
	"context"
	"net/http"

	"github.com/swaggest/jsonform-go"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

// EditSettings creates use case interactor to show form.
func EditSettings(deps *service.Locator) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *usecase.OutputWithEmbeddedWriter) error {
		return deps.SchemaRepository().Render(out.Writer, jsonform.Page{}, jsonform.Form{
			Title:         "Settings",
			SubmitURL:     "/settings.json",
			SubmitMethod:  http.MethodPut,
			SuccessStatus: http.StatusNoContent,
			Value:         deps.ServiceSettings(),
		})
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown)

	return u
}

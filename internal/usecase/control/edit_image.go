package control

import (
	"context"
	"github.com/swaggest/jsonform-go"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"html/template"
	"net/http"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type editImagePageDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	SchemaRepository() *jsonform.Repository
	PhotoImageFinder() uniq.Finder[photo.Image]
}

// EditImage creates use case interactor to show form.
func EditImage(deps editImagePageDeps) usecase.Interactor {
	type editImageInput struct {
		Hash uniq.Hash `path:"hash"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in editImageInput, out *usecase.OutputWithEmbeddedWriter) error {
		img, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		return deps.SchemaRepository().Render(out.Writer, jsonform.Page{
			PrependHTML: template.HTML(`
<div style="margin:2em" class="pure-u-2-5">
    <h1>Manage photo</h1>
    <img alt="" src="/thumb/400h/` + img.Hash.String() + `.jpg" />
    <form id="schema-form" class="pure-form"></form>
    <div id="res" class="alert"></div>
</div>`),
		}, jsonform.Form{
			Title:         "Manage Photo",
			SubmitURL:     "/album",
			SubmitMethod:  http.MethodPut,
			SuccessStatus: http.StatusNoContent,
			Value:         img,
		})
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

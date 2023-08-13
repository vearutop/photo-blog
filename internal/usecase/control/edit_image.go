package control

import (
	"context"
	"encoding/json"
	"html/template"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/jsonform"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type editImagePageDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	SchemaRepository() *jsonform.Repository
}

// EditImage creates use case interactor to show form.
func EditImage(deps editImagePageDeps) usecase.Interactor {
	type editImageInput struct {
		Hash uniq.Hash `path:"hash"`
	}

	tmpl := must(static.Template("edit-image.html"))

	type pageData struct {
		ImageHash string
		Schema    template.JS
		Value     template.JS
	}

	u := usecase.NewInteractor(func(ctx context.Context, in editImageInput, out *web.Page) error {
		s := deps.SchemaRepository().Schema("update-image-input")
		j, err := json.Marshal(s)
		if err != nil {
			return err
		}

		d := pageData{}
		d.ImageHash = in.Hash.String()
		d.Value = `{}`
		d.Schema = template.JS(j)

		return out.Render(tmpl, d)
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

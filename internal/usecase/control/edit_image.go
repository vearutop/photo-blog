package control

import (
	"context"
	"encoding/json"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/schema"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
	"html/template"
)

type editImagePageDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	SchemaRepository() *schema.Repository
}

// EditImage creates use case interactor to show form.
func EditImage(deps editImagePageDeps) usecase.Interactor {
	type editImageInput struct {
		Hash uniq.Hash `path:"hash"`
	}

	tpl, err := static.Assets.ReadFile("edit-image.html")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		panic(err)
	}

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

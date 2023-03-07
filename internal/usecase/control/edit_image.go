package control

import (
	"context"
	"encoding/json"
	"html/template"
	"io"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/schema"
	"github.com/vearutop/photo-blog/resources/static"
)

type editImagePage struct {
	ImageHash string
	Schema    template.JS
	Value     template.JS

	writer io.Writer
}

func (o *editImagePage) SetWriter(w io.Writer) {
	o.writer = w
}

func (o *editImagePage) Render(tmpl *template.Template) error {
	return tmpl.Execute(o.writer, o)
}

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

	u := usecase.NewInteractor(func(ctx context.Context, in editImageInput, out *editImagePage) error {
		s := deps.SchemaRepository().Schema("update-image-input")
		j, err := json.Marshal(s)
		if err != nil {
			return err
		}

		out.ImageHash = in.Hash.String()
		out.Value = `{}`
		out.Schema = template.JS(j)

		return out.Render(tmpl)
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

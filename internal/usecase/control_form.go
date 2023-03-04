package usecase

import (
	"context"
	"encoding/json"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/schema"
	"html/template"
	"io"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/resources/static"
)

type formPage struct {
	EntityName string
	Title      string
	Schema     template.JS
	Value      template.JS

	writer io.Writer
}

func (o *formPage) SetWriter(w io.Writer) {
	o.writer = w
}

func (o *formPage) Render(tmpl *template.Template) error {
	return tmpl.Execute(o.writer, o)
}

type getFormDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	SchemaRepository() *schema.Repository
}

// ShowForm creates use case interactor to show form.
func ShowForm(deps getFormDeps) usecase.Interactor {
	type getFormInput struct {
		Name string    `path:"name"`
		ID   uniq.Hash `path:"id"`
	}

	tpl, err := static.Assets.ReadFile("form.html")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in getFormInput, out *formPage) error {
		deps.StatsTracker().Add(ctx, "show_form", 1)
		deps.CtxdLogger().Info(ctx, "showing form", "name", in.Name)

		s := deps.SchemaRepository().Schema(in.Name)
		j, err := json.Marshal(s)
		if err != nil {
			return err
		}

		out.EntityName = in.Name
		out.Value = `{}`
		out.Schema = template.JS(j)

		return out.Render(tmpl)
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

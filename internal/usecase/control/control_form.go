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

type getFormDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	SchemaRepository() *jsonform.Repository
}

// ShowForm creates use case interactor to show form.
func ShowForm(deps getFormDeps) usecase.Interactor {
	type getFormInput struct {
		Schema string    `query:"schema"`
		ID     uniq.Hash `query:"id"`
	}

	tpl, err := static.Assets.ReadFile("form.html")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		panic(err)
	}

	type pageData struct {
		EntityName string
		Title      string
		Schema     template.JS
		Value      template.JS
	}

	u := usecase.NewInteractor(func(ctx context.Context, in getFormInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_form", 1)
		deps.CtxdLogger().Info(ctx, "showing form", "schema", in.Schema)

		s := deps.SchemaRepository().Schema(in.Schema)
		j, err := json.Marshal(s)
		if err != nil {
			return err
		}

		d := pageData{}
		d.EntityName = in.Schema
		d.Value = `{}`
		d.Schema = template.JS(j)

		return out.Render(tmpl, d)
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

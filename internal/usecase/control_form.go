package usecase

import (
	"context"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"html/template"
	"io"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/resources/static"
)

type formPage struct {
	Title      string
	Name       string
	CoverImage string

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
}

// ShowForm creates use case interactor to show form.
func ShowForm(deps getFormDeps) usecase.Interactor {
	type getFormInput struct {
		Name string `query:"name"`
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

		return out.Render(tmpl)
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

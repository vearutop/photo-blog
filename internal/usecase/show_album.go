package usecase

import (
	"context"
	"errors"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/nethttp/ui"
	"html/template"
	"io"
	"net/http"
)

type htmlResponseOutput struct {
	ID         int
	Filter     string
	Title      string
	Items      []string
	AntiHeader bool `header:"X-Anti-Header"`

	writer io.Writer
}

func (o *htmlResponseOutput) SetWriter(w io.Writer) {
	o.writer = w
}

func (o *htmlResponseOutput) Render(tmpl *template.Template) error {
	return tmpl.Execute(o.writer, o)
}

func htmlResponse() usecase.Interactor {
	type htmlResponseInput struct {
		ID     int    `path:"id"`
		Filter string `query:"filter"`
		Header bool   `header:"X-Header"`
	}

	const tpl = `<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
	</head>
	<body>
		<a href="/html-response/{{.ID}}?filter={{.Filter}}">Next {{.Title}}</a><br />
		{{range .Items}}<div>{{ . }}</div>{{else}}<div><strong>no rows</strong></div>{{end}}
	</body>
</html>`

	tmpl, err := template.New("htmlResponse").Parse(tpl)
	if err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in htmlResponseInput, out *htmlResponseOutput) (err error) {
		out.AntiHeader = !in.Header
		out.Filter = in.Filter
		out.ID = in.ID + 1
		out.Title = "Foo"
		out.Items = []string{"foo", "bar", "baz"}

		return out.Render(tmpl)
	})

	u.SetTitle("Request With HTML Response")
	u.SetDescription("Request with templated HTML response.")

	return u
}

// ShowAlbum creates use case interactor to show album.
func ShowAlbum(deps getAlbumDeps) usecase.Interactor {
	type getAlbumInput struct {
		Name string `path:"name"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in getAlbumInput, out *usecase.OutputWithEmbeddedWriter) error {
		deps.StatsTracker().Add(ctx, "show_album", 1)
		deps.CtxdLogger().Info(ctx, "showing album", "name", in.Name)

		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		r, _ := http.NewRequest(http.MethodGet, "/album.html", nil)
		ui.Static.ServeHTTP(rw, r)

		return nil
	})

	u.SetTags("Photos")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

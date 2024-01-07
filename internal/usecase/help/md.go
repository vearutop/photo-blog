package help

import (
	"context"
	uc "github.com/vearutop/photo-blog/internal/usecase"
	"html/template"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/pkg/txt"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/help"
	"github.com/vearutop/photo-blog/resources/static"
)

func Markdown(deps indexDeps) usecase.Interactor {
	tmpl, err := static.Template("help.html")
	if err != nil {
		panic(err)
	}

	type pageData struct {
		uc.PageCommon
		Content template.HTML
	}

	type req struct {
		File string `path:"file"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input req, output *web.Page) error {
		if !deps.Settings().Privacy().PublicHelp && !auth.IsAdmin(ctx) {
			return status.PermissionDenied
		}

		md, err := help.Assets.ReadFile(input.File + ".md")
		if err != nil {
			return err
		}

		d := pageData{}
		c, err := deps.TxtRenderer().RenderLang(ctx, string(md))
		if err != nil {
			return err
		}

		d.Title = deps.TxtRenderer().MustRenderLang(ctx, `
:::{lang=en}
Help
:::

:::{lang=ru}
Справка
:::
`, func(o *txt.RenderOptions) {
			o.StripTags = true
		})

		d.Content = template.HTML(c)

		return output.Render(tmpl, d)
	})
	u.SetTags("Help")

	return u
}

package help

import (
	"context"
	"html/template"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/pkg/txt"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/help"
	"github.com/vearutop/photo-blog/resources/static"
)

type indexDeps interface {
	TxtRenderer() *txt.Renderer
}

type pageCommon struct {
	Title   string
	Lang    string
	Favicon string
	Head    template.HTML
	Header  template.HTML
	Footer  template.HTML
}

func (p *pageCommon) fill(ctx context.Context, r *txt.Renderer, a settings.Appearance) {
	if p.Title == "" {
		p.Title = r.MustRenderLang(ctx, a.SiteTitle, func(o *txt.RenderOptions) {
			o.StripTags = true
		})
	}

	p.Lang = txt.Language(ctx)
	p.Head = template.HTML(r.MustRenderLang(ctx, a.SiteHead))
	p.Header = template.HTML(r.MustRenderLang(ctx, a.SiteHeader))
	p.Footer = template.HTML(r.MustRenderLang(ctx, a.SiteFooter))
	p.Favicon = a.SiteFavicon

	if p.Favicon == "" {
		p.Favicon = "/static/favicon.png"
	}
}

func Index(deps indexDeps) usecase.Interactor {
	tmpl, err := static.Template("help.html")
	if err != nil {
		panic(err)
	}

	md, err := help.Assets.ReadFile("index.md")
	if err != nil {
		panic(err)
	}

	type pageData struct {
		pageCommon
		Content template.HTML
	}

	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *web.Page) error {
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

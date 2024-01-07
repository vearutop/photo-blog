package usecase

import (
	"context"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/pkg/txt"
	"html/template"
	"time"

	"github.com/swaggest/rest/request"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type hashInPath struct {
	Hash uniq.Hash `path:"hash"`
	request.EmbeddedSetter
}

// detachedContext exposes parent values, but suppresses parent cancellation.
type detachedContext struct {
	parent context.Context //nolint:containedctx // This wrapping is here on purpose.
}

func (d detachedContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (d detachedContext) Done() <-chan struct{} {
	return nil
}

func (d detachedContext) Err() error {
	return nil
}

func (d detachedContext) Value(key interface{}) interface{} {
	return d.parent.Value(key)
}

type PageCommon struct {
	Title   string
	Lang    string
	Favicon string
	Head    template.HTML
	Header  template.HTML
	Footer  template.HTML
}

func (p *PageCommon) Fill(ctx context.Context, r *txt.Renderer, a settings.Appearance) {
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

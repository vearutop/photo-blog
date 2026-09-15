package usecase

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"strings"

	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/pkg/txt"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type showMainInput struct {
	request.EmbeddedSetter

	hasAuth bool
}

type showMainDeps interface {
	getAlbumImagesDeps

	DepCache() *dep.Cache
	Settings() settings.Values
}

type pageCommon struct {
	Title    string
	Lang     string
	Favicon  string
	Head     template.HTML
	Header   template.HTML
	Footer   template.HTML
	MainMenu []settings.MenuItem

	Secure          bool
	IsAdmin         bool
	IsBot           bool
	ShowLoginButton bool

	ThumbBaseURL     string
	ImageBaseURL     string
	ThumbBaseHref    string
	ImageBaseHref    string
	CanonicalBaseURL string

	SubAlbums []getAlbumOutput
}

func (p *pageCommon) fill(ctx context.Context, r *txt.Renderer, a settings.Values) {
	ap := a.Appearance()

	if p.Title == "" {
		p.Title = r.MustRenderLang(ctx, ap.SiteTitle, func(o *txt.RenderOptions) {
			o.StripTags = true
		})
	}

	p.Lang = txt.Language(ctx)

	p.Head = template.HTML(r.MustRenderLang(ctx, ap.SiteHead))
	p.Header = template.HTML(r.MustRenderLang(ctx, ap.SiteHeader))
	p.Footer = template.HTML(r.MustRenderLang(ctx, ap.SiteFooter))
	p.Favicon = ap.SiteFavicon

	p.ThumbBaseURL = ap.ThumbBaseURL
	p.ImageBaseURL = ap.ImageBaseURL
	p.ThumbBaseHref = p.ThumbBaseURL
	p.ImageBaseHref = p.ImageBaseURL
	p.CanonicalBaseURL = strings.TrimSuffix(ap.CanonicalBaseURL, "/")

	if p.Favicon == "" {
		p.Favicon = "/static/favicon.png"
	}
	if p.ThumbBaseHref == "" {
		p.ThumbBaseHref = "/thumb"
	}
	if p.ImageBaseHref == "" {
		p.ImageBaseHref = "/image"
	}

	p.IsAdmin = auth.IsAdmin(ctx)
	p.IsBot = auth.IsBot(ctx)
	p.Secure = !a.Security().Disabled()
	p.ShowLoginButton = !a.Privacy().HideLoginButton

	for _, i := range ap.MainMenu {
		if i.AdminOnly && !p.IsAdmin {
			continue
		}

		p.MainMenu = append(p.MainMenu, settings.MenuItem{
			URL: i.URL,
			Text: strings.TrimSpace(r.MustRenderLang(ctx, i.Text, func(o *txt.RenderOptions) {
				o.StripTags = true
			})),
		})
	}

	if len(p.MainMenu) == 0 {
		p.MainMenu = append(p.MainMenu, settings.MenuItem{
			URL:  "/",
			Text: "Home",
		})
	}
}

// ShowMain2 creates use case interactor to show album.
func ShowMain2(deps showMainDeps) usecase.IOInteractorOf[showMainInput, web.Page] {
	var (
		tmpl     *template.Template
		err      error
		notFound usecase.IOInteractorOf[struct{}, web.Page]
		b        *AlbumPageBuilder
	)

	if deps.DepCache() != nil {
		tmpl, err = static.Template("album.gohtml")
		if err != nil {
			panic(err)
		}

		notFound = NotFound(deps)

		b = NewAlbumPageBuilder(deps)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showMainInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_main", 1)
		deps.CtxdLogger().Debug(ctx, "showing main page")

		featured := deps.Settings().Appearance().FeaturedAlbumName

		cont, err := b.getCachedAlbum(ctx, featured, false, "", false)
		if err != nil {
			if errors.Is(err, status.NotFound) {
				return notFound.Invoke(ctx, struct{}{}, out)
			}

			return fmt.Errorf("get album contents: %w", err)
		}

		list, err := deps.PhotoAlbumFinder().FindAll(ctx)
		if err != nil {
			return fmt.Errorf("find all albums: %w", err)
		}

		if len(list) > 4 {
			list = list[:4]
		}

		for _, a := range list {
			cont.Album.Settings.SubAlbumNames = append(cont.Album.Settings.SubAlbumNames, a.Name)
		}

		cont.Album.Settings.HideMap = true

		d, err := b.cachedBuild(ctx, cont)
		if err != nil {
			return err
		}
		d.OGPageURL = "https://" + in.Request().Host + in.Request().URL.Path

		d.Title = deps.Settings().Appearance().SiteTitle
		d.Title, err = deps.TxtRenderer().RenderLang(ctx, d.Title, txt.StripTags)
		if err != nil {
			return err
		}
		d.TotalSize = ""

		if d.IsAdmin {
			ps, err := deps.VisitorStats().AlbumViews(ctx, d.AlbumData.Album.Hash)
			if err != nil {
				deps.CtxdLogger().Error(ctx, "failed to get album views", "error", err)
			} else {
				d.Visits = fmt.Sprintf("%d/%d/%d", ps.Uniq, ps.Views, ps.Refers)
			}
		}

		return out.Render(tmpl, d)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument, status.PermissionDenied)

	return u
}

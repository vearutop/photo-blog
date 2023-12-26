package usecase

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"sort"
	"strconv"
	"time"

	"github.com/bool64/brick"
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
	hasAuth bool
}

type showMainDeps interface {
	getAlbumImagesDeps

	DepCache() *dep.Cache
	Settings() settings.Values
}

type pageCommon struct {
	Title   string
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

	p.Head = template.HTML(r.MustRenderLang(ctx, a.SiteHead))
	p.Header = template.HTML(r.MustRenderLang(ctx, a.SiteHeader))
	p.Footer = template.HTML(r.MustRenderLang(ctx, a.SiteFooter))
	p.Favicon = a.SiteFavicon

	if p.Favicon == "" {
		p.Favicon = "/static/favicon.png"
	}
}

// ShowMain creates use case interactor to show album.
func ShowMain(deps showMainDeps) usecase.IOInteractorOf[showMainInput, web.Page] {
	tpl, err := static.Assets.ReadFile("index.html")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		panic(err)
	}

	type pageData struct {
		pageCommon

		Lang              string
		Name              string
		CoverImage        string
		Secure            bool
		NonAdmin          bool
		Public            bool
		Hash              string
		Featured          string
		FeaturedAlbumData getAlbumOutput
		Albums            []getAlbumOutput
	}

	cacheName := "main-page"
	c := brick.MakeCacheOf[pageData](deps, cacheName, time.Hour)

	u := usecase.NewInteractor(func(ctx context.Context, in showMainInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_main", 1)
		deps.CtxdLogger().Info(ctx, "showing main")

		cacheKey := []byte("main" + strconv.FormatBool(auth.IsAdmin(ctx)) + txt.Language(ctx))
		d, err := c.Get(ctx, cacheKey, func(ctx context.Context) (pageData, error) {
			d := pageData{}

			deps.DepCache().ServiceSettingsDependency(cacheName, cacheKey)
			deps.DepCache().AlbumListDependency(cacheName, cacheKey)

			d.fill(ctx, deps.TxtRenderer(), deps.Settings().Appearance())

			d.Lang = txt.Language(ctx)
			d.NonAdmin = !auth.IsAdmin(ctx)
			d.Secure = !deps.Settings().Security().Disabled()
			d.Featured = deps.Settings().Appearance().FeaturedAlbumName

			if d.Featured != "" {
				cont, err := getAlbumContents(ctx, deps, d.Featured, false)
				if err != nil && !errors.Is(err, status.NotFound) {
					return d, fmt.Errorf("featured: %w", err)
				}

				if cont.Album.CoverImage != 0 {
					d.CoverImage = "/thumb/1200w/" + cont.Album.CoverImage.String() + ".jpg"
				}

				d.FeaturedAlbumData = cont

				deps.DepCache().AlbumDependency(cacheName, cacheKey, d.Featured)
			}

			list, err := deps.PhotoAlbumFinder().FindAll(ctx)
			if err != nil {
				return d, err
			}

			sort.Slice(list, func(i, j int) bool {
				return list[i].CreatedAt.After(list[j].CreatedAt)
			})

			for _, a := range list {
				if a.Hidden {
					continue
				}

				if !a.Public || a.Name == "" {
					if d.NonAdmin {
						continue
					}
				}

				cont, err := getAlbumContents(ctx, deps, a.Name, true)
				if err != nil {
					return d, err
				}

				if len(cont.Images) == 0 && d.NonAdmin {
					continue
				}

				d.Albums = append(d.Albums, cont)
				deps.DepCache().AlbumDependency(cacheName, cacheKey, cont.Album.Name)
			}

			return d, nil
		})
		if err != nil {
			return err
		}

		return out.Render(tmpl, d)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

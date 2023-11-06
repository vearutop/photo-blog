package usecase

import (
	"context"
	"github.com/bool64/cache"
	"html/template"
	"sort"
	"strconv"
	"time"

	"github.com/bool64/brick"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/pkg/txt"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type showMainInput struct {
	hasAuth bool
}

type showMainDeps interface {
	getAlbumImagesDeps

	CacheInvalidationIndex() *cache.InvalidationIndex
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
		Title             string
		Lang              string
		Name              string
		CoverImage        string
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

			invalidationLabels := []string{"service-settings"}

			d.Title = deps.TxtRenderer().MustRenderLang(ctx, deps.ServiceSettings().SiteTitle, func(o *txt.RenderOptions) {
				o.StripTags = true
			})
			d.Lang = txt.Language(ctx)
			d.NonAdmin = !auth.IsAdmin(ctx)
			d.Featured = deps.ServiceSettings().FeaturedAlbumName

			if d.Featured != "" {
				fa, err := deps.PhotoAlbumFinder().FindByHash(ctx, photo.AlbumHash(d.Featured))
				if err != nil {
					return d, err
				}

				if fa.CoverImage != 0 {
					d.CoverImage = "/thumb/1200w/" + fa.CoverImage.String() + ".jpg"
				}

				cont, err := getAlbumContents(ctx, deps, d.Featured, false)
				if err != nil {
					return d, err
				}

				d.FeaturedAlbumData = cont

				invalidationLabels = append(invalidationLabels, "album/"+d.Featured)
			}

			list, err := deps.PhotoAlbumFinder().FindAll(ctx)
			if err != nil {
				return d, err
			}

			sort.Slice(list, func(i, j int) bool {
				return list[i].CreatedAt.After(list[j].CreatedAt)
			})

			for _, a := range list {
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
				invalidationLabels = append(invalidationLabels, "album/"+cont.Album.Name)
			}

			deps.CacheInvalidationIndex().AddLabels(cacheName, cacheKey, invalidationLabels...)

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

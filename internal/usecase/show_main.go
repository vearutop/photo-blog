package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/bool64/cache"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	infraService "github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/pkg/txt"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

// ShowMain creates use case interactor to show album.
func ShowMain(deps showMainDeps) usecase.IOInteractorOf[showMainInput, web.Page] {
	tmpl, err := static.Template("index.html")
	if err != nil {
		panic(err)
	}

	type pageData struct {
		pageCommon

		CoverImage        string
		Featured          string
		FeaturedAlbumData getAlbumOutput
	}

	cacheName := "main-page"
	var c *cache.FailoverOf[pageData]
	if deps.DepCache() != nil {
		c = infraService.MakePersistentCacheOf[pageData](deps, cacheName, time.Hour)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showMainInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_main", 1)
		deps.CtxdLogger().Info(ctx, "showing main")

		cacheKey := []byte("main" + strconv.FormatBool(auth.IsAdmin(ctx)) + txt.Language(ctx))
		cacheMiss := false
		d, err := c.Get(ctx, cacheKey, func(ctx context.Context) (pageData, error) {
			cacheMiss = true
			d := pageData{}

			d.fill(ctx, deps.TxtRenderer(), deps.Settings())

			d.Featured = deps.Settings().Appearance().FeaturedAlbumName

			if d.Featured != "" {
				cont, err := getAlbumContents(ctx, deps, imagesFilter{albumName: d.Featured}, false)
				if err != nil && !errors.Is(err, status.NotFound) {
					return d, fmt.Errorf("featured: %w", err)
				}

				if cont.Album.CoverImage != 0 {
					d.CoverImage = "/thumb/1200w/" + cont.Album.CoverImage.String() + ".jpg"
				}

				d.FeaturedAlbumData = cont
			}

			list, err := deps.PhotoAlbumFinder().FindAll(ctx)
			if err != nil {
				return d, fmt.Errorf("find all albums: %w", err)
			}

			sort.Slice(list, func(i, j int) bool {
				return list[i].CreatedAt.After(list[j].CreatedAt)
			})

			for _, a := range list {
				if a.Hidden {
					continue
				}

				if !a.Public || a.Name == "" {
					if !d.IsAdmin {
						continue
					}
				}

				cont, err := getAlbumContents(ctx, deps, imagesFilter{albumName: a.Name}, true)
				if err != nil {
					return d, fmt.Errorf("find album %s: %w", a.Name, err)
				}

				if len(cont.Images) == 0 && !d.IsAdmin {
					continue
				}

				d.SubAlbums = append(d.SubAlbums, cont)
			}

			return d, nil
		})
		if err != nil {
			return err
		}

		if cacheMiss {
			// Labels are reset right before (re)registering them, as close to the cache write as
			// possible, so a concurrent invalidation during the (slow) build above can't be lost
			// by finding no labels to match against.
			if err := deps.DepCache().ResetKey(ctx, cacheName, cacheKey); err != nil {
				return fmt.Errorf("reset cache deps: %w", err)
			}

			deps.DepCache().ServiceSettingsDependency(cacheName, cacheKey)
			deps.DepCache().AlbumListDependency(cacheName, cacheKey)

			if d.Featured != "" {
				deps.DepCache().AlbumDependency(cacheName, cacheKey, d.Featured)
			}

			for _, cont := range d.SubAlbums {
				deps.DepCache().AlbumDependency(cacheName, cacheKey, cont.Album.Name)
			}
		}

		return out.Render(tmpl, d)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

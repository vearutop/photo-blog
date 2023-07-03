package usecase

import (
	"context"
	"html/template"
	"sort"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type showMainInput struct {
	hasAuth bool
}

// ShowMain creates use case interactor to show album.
func ShowMain(deps getAlbumImagesDeps) usecase.IOInteractorOf[showMainInput, web.Page] {
	tpl, err := static.Assets.ReadFile("index.html")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		panic(err)
	}

	type album struct {
		photo.Album
		Images []photo.Image
	}

	type pageData struct {
		Title      string
		Name       string
		CoverImage string
		NonAdmin   bool
		Public     bool
		Hash       string
		Featured   string
		Albums     []album
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showMainInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_main", 1)
		deps.CtxdLogger().Info(ctx, "showing main")

		d := pageData{}

		d.Title = deps.ServiceSettings().SiteTitle
		d.NonAdmin = !auth.IsAdmin(ctx)
		d.Featured = deps.ServiceSettings().FeaturedAlbumName

		if d.Featured != "" {
			fa, err := deps.PhotoAlbumFinder().FindByHash(ctx, photo.AlbumHash(d.Featured))
			if err != nil {
				return err
			}

			if fa.CoverImage != 0 {
				d.CoverImage = "/thumb/1200w/" + fa.CoverImage.String() + ".jpg"
			}
		}

		list, err := deps.PhotoAlbumFinder().FindAll(ctx)
		if err != nil {
			return err
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

			images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, a.Hash)
			if err != nil {
				return err
			}

			if len(images) == 0 && d.NonAdmin {
				continue
			}

			if a.CoverImage != 0 {
				for i, img := range images {
					if img.Hash == a.CoverImage {
						img0 := images[0]

						images[0] = img
						images[i] = img0
						break
					}
				}
			}

			if len(images) > 4 {
				images = images[:4]
			}

			aa := album{}
			aa.Album = a
			aa.Images = images

			d.Albums = append(d.Albums, aa)
		}

		//album, err := deps.PhotoAlbumFinder().FindByHash(ctx, albumHash)
		//if err != nil {
		//	return err
		//}
		//
		//images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, albumHash)
		//if err != nil {
		//	return err
		//}
		//
		//if len(images) == 0 {
		//	return errors.New("no images")
		//}
		//
		//d := pageData{}
		//d.Title = album.Title
		//d.Name = album.Name
		//d.NonAdmin = !in.hasAuth
		//d.Public = album.Public
		//d.Hash = album.Hash.String()
		//
		//switch {
		//case album.CoverImage != 0:
		//	d.CoverImage = "/thumb/1200w/" + album.CoverImage.String() + ".jpg"
		//default:
		//	d.CoverImage = "/thumb/1200w/" + images[0].Hash.String() + ".jpg"
		//}

		return out.Render(tmpl, d)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

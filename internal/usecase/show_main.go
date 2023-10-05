package usecase

import (
	"context"
	"html/template"
	"math"
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
func ShowMain(deps getAlbumImagesDeps, contents usecase.IOInteractorOf[getAlbumInput, getAlbumOutput]) usecase.IOInteractorOf[showMainInput, web.Page] {
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
		Title             string
		Name              string
		CoverImage        string
		NonAdmin          bool
		Public            bool
		Hash              string
		Featured          string
		FeaturedAlbumData getAlbumOutput
		Albums            []album
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

			cont := getAlbumOutput{}

			if err := contents.Invoke(ctx, getAlbumInput{
				Name: d.Featured,
			}, &cont); err != nil {
				return err
			}

			d.FeaturedAlbumData = cont
		}

		list, err := deps.PhotoAlbumFinder().FindAll(ctx)
		if err != nil {
			return err
		}

		sort.Slice(list, func(i, j int) bool {
			return list[i].CreatedAt.After(list[j].CreatedAt)
		})

		aspectRatio := 3.0 / 2.0

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

			res := make([]photo.Image, 0, 4)

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

			for _, img := range images {
				ar := float64(img.Width) / float64(img.Height)

				if math.Abs(ar-aspectRatio) > 1e-2 {
					continue
				}

				res = append(res, img)
				if len(res) >= 4 {
					break
				}
			}

			if len(res) == 0 && d.NonAdmin {
				continue
			}

			aa := album{}
			aa.Album = a
			aa.Images = res

			d.Albums = append(d.Albums, aa)
		}

		return out.Render(tmpl, d)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

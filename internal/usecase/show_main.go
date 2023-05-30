package usecase

import (
	"context"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"html/template"
	"net/http"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type showMainInput struct {
	hasAuth bool
}

func (i *showMainInput) SetRequest(r *http.Request) {
	if r.Header.Get("Authorization") != "" {
		i.hasAuth = true
	}
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

	type pageData struct {
		Title      string
		Name       string
		CoverImage string
		NonAdmin   bool
		Public     bool
		Hash       string
		Albums     []photo.Album
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showMainInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_main", 1)
		deps.CtxdLogger().Info(ctx, "showing main")

		d := pageData{}

		list, err := deps.PhotoAlbumFinder().FindAll(ctx)
		if err != nil {
			return err
		}

		for _, a := range list {
			if !a.Public {
				continue
			}

			d.Albums = append(d.Albums, a)
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

package usecase

import (
	"context"
	"errors"
	"github.com/vearutop/photo-blog/pkg/web"
	"html/template"
	"net/http"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/resources/static"
)

type showAlbumAtImageInput struct {
	showAlbumInput
	Hash uniq.Hash `path:"hash"`
}

type showAlbumInput struct {
	Name    string `path:"name"`
	hasAuth bool
	imgHash uniq.Hash
}

func (i *showAlbumInput) SetRequest(r *http.Request) {
	if r.Header.Get("Authorization") != "" {
		i.hasAuth = true
	}
}

func ShowAlbumAtImage(up usecase.IOInteractorOf[showAlbumInput, web.Page]) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in showAlbumAtImageInput, out *web.Page) error {
		in.imgHash = in.Hash

		return up.Invoke(ctx, in.showAlbumInput, out)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

// ShowAlbum creates use case interactor to show album.
func ShowAlbum(deps getAlbumImagesDeps) usecase.IOInteractorOf[showAlbumInput, web.Page] {
	tpl, err := static.Assets.ReadFile("album.html")
	if err != nil {
		panic(err)
	}

	type albumPage struct {
		Title      string
		Name       string
		CoverImage string
		NonAdmin   bool
		Public     bool
		Hash       string
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showAlbumInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_album", 1)
		deps.CtxdLogger().Info(ctx, "showing album", "name", in.Name)

		albumHash := photo.AlbumHash(in.Name)

		album, err := deps.PhotoAlbumFinder().FindByHash(ctx, albumHash)
		if err != nil {
			return err
		}

		images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, albumHash)
		if err != nil {
			return err
		}

		if len(images) == 0 {
			return errors.New("no images")
		}

		data := albumPage{}
		data.Title = album.Title
		data.Name = album.Name
		data.NonAdmin = !in.hasAuth
		data.Public = album.Public
		data.Hash = album.Hash.String()

		switch {
		case in.imgHash != 0:
			data.CoverImage = "/thumb/1200w/" + in.imgHash.String() + ".jpg"
		case album.CoverImage != 0:
			data.CoverImage = "/thumb/1200w/" + album.CoverImage.String() + ".jpg"
		default:
			data.CoverImage = "/thumb/1200w/" + images[0].Hash.String() + ".jpg"
		}

		return out.Render(tmpl, data)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

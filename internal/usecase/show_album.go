package usecase

import (
	"context"
	"errors"
	"html/template"
	"io"
	"net/http"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/resources/static"
)

type albumPage struct {
	Title      string
	Name       string
	CoverImage string
	NonAdmin   bool

	writer io.Writer
}

func (o *albumPage) SetWriter(w io.Writer) {
	o.writer = w
}

func (o *albumPage) Render(tmpl *template.Template) error {
	return tmpl.Execute(o.writer, o)
}

type showAlbumInput struct {
	Name    string `path:"name"`
	hasAuth bool
}

func (i *showAlbumInput) SetRequest(r *http.Request) {
	if r.Header.Get("Authorization") != "" {
		i.hasAuth = true
	}
}

// ShowAlbum creates use case interactor to show album.
func ShowAlbum(deps getAlbumImagesDeps) usecase.Interactor {
	tpl, err := static.Assets.ReadFile("album.html")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showAlbumInput, out *albumPage) error {
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

		out.Title = album.Title
		out.Name = album.Name
		out.NonAdmin = !in.hasAuth
		if album.CoverImage != 0 {
			out.CoverImage = "/thumb/1200w/" + album.CoverImage.String() + ".jpg"
		} else {
			out.CoverImage = "/thumb/1200w/" + images[0].Hash.String() + ".jpg"
		}

		return out.Render(tmpl)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

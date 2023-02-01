package usecase

import (
	"context"
	"errors"
	"html/template"
	"io"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/resources/static"
)

type albumPage struct {
	Title      string
	Name       string
	CoverImage string

	writer io.Writer
}

func (o *albumPage) SetWriter(w io.Writer) {
	o.writer = w
}

func (o *albumPage) Render(tmpl *template.Template) error {
	return tmpl.Execute(o.writer, o)
}

// ShowAlbum creates use case interactor to show album.
func ShowAlbum(deps getAlbumDeps) usecase.Interactor {
	type getAlbumInput struct {
		Name string `path:"name"`
	}

	tpl, err := static.Assets.ReadFile("album.html")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in getAlbumInput, out *albumPage) error {
		deps.StatsTracker().Add(ctx, "show_album", 1)
		deps.CtxdLogger().Info(ctx, "showing album", "name", in.Name)

		album, err := deps.PhotoAlbumFinder().FindByName(ctx, in.Name)
		if err != nil {
			return err
		}

		images, err := deps.PhotoAlbumFinder().FindImages(ctx, album.ID)
		if err != nil {
			return err
		}

		if len(images) == 0 {
			return errors.New("no images")
		}

		out.Title = album.Title
		out.Name = album.Name
		out.CoverImage = "/thumb/1200w/" + images[0].StringHash() + ".jpg"

		return out.Render(tmpl)
	})

	u.SetTags("Photos")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

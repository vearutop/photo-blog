package usecase

import (
	"context"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"html/template"
	"io"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/resources/static"
)

type panoPage struct {
	Title      string
	Name       string
	CoverImage string
	Image      string

	writer io.Writer
}

func (o *panoPage) SetWriter(w io.Writer) {
	o.writer = w
}

func (o *panoPage) Render(tmpl *template.Template) error {
	return tmpl.Execute(o.writer, o)
}

type showPanoDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PhotoAlbumFinder() photo.AlbumFinder
}

// ShowPano creates use case interactor to show pano.
func ShowPano(deps showPanoDeps) usecase.Interactor {
	type getPanoInput struct {
		Name string     `path:"name"`
		Hash photo.Hash `path:"hash"`
	}

	tpl, err := static.Assets.ReadFile("pano.html")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in getPanoInput, out *panoPage) error {
		deps.StatsTracker().Add(ctx, "show_pano", 1)
		deps.CtxdLogger().Info(ctx, "showing pano", "name", in.Name)

		album, err := deps.PhotoAlbumFinder().FindByName(ctx, in.Name)
		if err != nil {
			return err
		}

		out.Title = album.Title
		out.Name = album.Name
		out.CoverImage = "/thumb/1200w/" + in.Hash.String() + ".jpg"
		out.Image = "/image/" + in.Hash.String() + ".jpg"

		return out.Render(tmpl)
	})

	u.SetTags("Pano")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

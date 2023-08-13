package usecase

import (
	"context"
	"html/template"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type showPanoDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PhotoAlbumFinder() uniq.Finder[photo.Album]
}

// ShowPano creates use case interactor to show pano.
func ShowPano(deps showPanoDeps) usecase.Interactor {
	type getPanoInput struct {
		Name string    `path:"name"`
		Hash uniq.Hash `path:"hash"`
	}

	tpl, err := static.Assets.ReadFile("pano.html")
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
		Image      string
	}

	u := usecase.NewInteractor(func(ctx context.Context, in getPanoInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_pano", 1)
		deps.CtxdLogger().Info(ctx, "showing pano", "album", in.Name, "pano", in.Hash)

		album, err := deps.PhotoAlbumFinder().FindByHash(ctx, photo.AlbumHash(in.Name))
		if err != nil {
			return err
		}

		d := pageData{}
		d.Title = album.Title
		d.Name = album.Name
		d.CoverImage = "/thumb/1200w/" + in.Hash.String() + ".jpg"
		d.Image = "/image/" + in.Hash.String() + ".jpg"

		return out.Render(tmpl, d)
	})

	u.SetTags("Pano")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

package control

import (
	"context"
	"encoding/json"
	"github.com/vearutop/photo-blog/pkg/jsonform"
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

type editAlbumPageDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	SchemaRepository() *jsonform.Repository
	PhotoAlbumFinder() uniq.Finder[photo.Album]
}

// EditAlbum creates use case interactor to show form.
func EditAlbum(deps editAlbumPageDeps) usecase.Interactor {
	type editAlbumInput struct {
		Hash uniq.Hash `path:"hash"`
	}

	tmpl := must(static.Template("edit-album.html"))

	type pageData struct {
		AlbumHash string
		Schema    template.JS
		Value     template.JS
	}

	u := usecase.NewInteractor(func(ctx context.Context, in editAlbumInput, out *web.Page) error {
		s := deps.SchemaRepository().Schema("album")
		j, err := json.Marshal(s)
		if err != nil {
			return err
		}

		a, err := deps.PhotoAlbumFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		aj, err := json.Marshal(a)
		if err != nil {
			return err
		}

		d := pageData{}
		d.AlbumHash = in.Hash.String()
		d.Value = template.JS(aj)
		d.Schema = template.JS(j)

		return out.Render(tmpl, d)
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

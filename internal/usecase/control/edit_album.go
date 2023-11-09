package control

import (
	"context"
	"github.com/vearutop/photo-blog/internal/infra/upload"
	"html/template"
	"net/http"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/jsonform-go"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
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

	u := usecase.NewInteractor(func(ctx context.Context, in editAlbumInput, out *usecase.OutputWithEmbeddedWriter) error {
		a, err := deps.PhotoAlbumFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		p := jsonform.Page{
			AppendHTMLHead: `
    <link rel="stylesheet" href="/static/style.css">
    <script src="/static/client.js"></script>
    <script src="/static/album.js"></script>
    <link rel="stylesheet" href="/static/tus/uppy.min.css">
    <script src="/static/tus/uppy.legacy.min.js"></script>

`,
			PrependHTML: upload.TusAlbumHTMLButton(a.Name),
			AppendHTML: template.HTML(`
<hr /><button style="margin: 2em" class="btn btn-danger" onclick="deleteAlbum('` + a.Name + `')">Delete this album</button>
`),
		}

		return deps.SchemaRepository().Render(out.Writer, p,
			jsonform.Form{
				Title:         "Manage Album",
				SubmitURL:     "/album",
				SubmitMethod:  http.MethodPut,
				SuccessStatus: http.StatusNoContent,
				Value:         a,
			})
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

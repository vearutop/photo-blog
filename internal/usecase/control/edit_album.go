package control

import (
	"context"
	"html/template"
	"net/http"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/jsonform-go"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/upload"
)

type editAlbumPageDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	SchemaRepository() *jsonform.Repository
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	Settings() settings.Values
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
			Title: "✏️ " + a.Name,
			AppendHTMLHead: `
    <link rel="stylesheet" href="/static/style.css">
    <script src="/static/client.js"></script>
    <script src="/static/album.js"></script>
    <script src="/static/album_extra.js"></script>
    <link rel="stylesheet" href="/static/tus/uppy.min.css">
    <script src="/static/tus/uppy.legacy.min.js"></script>

`,
			PrependHTML: template.HTML(`<a style="margin-left: 2em" href ="/`+a.Name+`">Back to album</a> `) +
				upload.TusAlbumHTMLButton(a.Name) +
				`<script>
function formSaved(x, ctx) { $(ctx.result).html('Saved.').show() } 
function formDone(x, ctx) { $(ctx.result).html('Done.').show() } 
</script>`,
			AppendHTML: template.HTML(`
<hr />
<button style="margin: 2em" class="btn btn-danger" onclick="deleteAlbum('` + a.Name + `')">Delete this album</button>
<button style="margin: 2em" class="btn" onclick="reindexAlbum('` + a.Name + `')">Reindex this album</button>
`),
		}

		return deps.SchemaRepository().Render(out.Writer, p,
			jsonform.Form{
				Title:         "Edit Album Details",
				SubmitURL:     "/album",
				SubmitMethod:  http.MethodPut,
				SuccessStatus: http.StatusNoContent,
				Value:         a,
				SubmitText:    "Save",
				OnSuccess:     `formSaved`,
			},
			jsonform.Form{
				Title:         "Add Images From Another Album",
				SubmitURL:     "/album/" + a.Name,
				SubmitMethod:  http.MethodPost,
				SuccessStatus: http.StatusNoContent,
				Value:         addToAlbumInput{},
				SubmitText:    "Add",
				OnSuccess:     `formDone`,
			},
		)
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

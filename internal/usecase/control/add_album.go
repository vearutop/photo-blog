package control

import (
	"context"
	"html/template"
	"math/rand"
	"net/http"
	"time"

	"github.com/swaggest/jsonform-go"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

// AddAlbum creates use case interactor to show form.
func AddAlbum(deps editAlbumPageDeps) usecase.Interactor {
	type albumData struct {
		Title string `json:"title" formType:"textarea" title:"Title" description:"Title of an album."`
		Name  string `json:"name" title:"Name" required:"true" description:"A slug value that is used in album URL, secret prefix is added by default, remove it for public albums."`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *usecase.OutputWithEmbeddedWriter) error {
		return deps.SchemaRepository().Render(
			out.Writer,
			jsonform.Page{
				AppendHTMLHead: `
    <link rel="stylesheet" href="/static/style.css">
    <script src="/static/client.js"></script>
    <script src="/static/album.js"></script>
    <script src="/static/album_extra.js"></script>

`,
				AppendHTML: template.HTML(`
<script>
document.addEventListener("DOMContentLoaded", () => {
 document.getElementById("jsonform-1-elt-name").value += "` + time.Now().Format(time.DateOnly) + "-meaningful-name-" + uniq.Hash(rand.Int()).String() + `";
});
</script>`),
			},
			jsonform.Form{
				Title:         "Add Album",
				SubmitURL:     "/album",
				SubmitMethod:  http.MethodPost,
				SuccessStatus: http.StatusOK,
				Value:         albumData{},
				OnSuccess:     `onAlbumCreated`,
			})
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

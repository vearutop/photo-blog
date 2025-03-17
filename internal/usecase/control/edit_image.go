package control

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/jsonform-go"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type editImagePageDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	SchemaRepository() *jsonform.Repository
	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoExifFinder() uniq.Finder[photo.Exif]
	PhotoGpsFinder() uniq.Finder[photo.Gps]
}

// EditImage creates use case interactor to show form.
func EditImage(deps editImagePageDeps) usecase.Interactor {
	type editImageInput struct {
		Hash uniq.Hash `path:"hash"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in editImageInput, out *usecase.OutputWithEmbeddedWriter) error {
		img, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return fmt.Errorf("find image: %w", err)
		}

		exif, err := deps.PhotoExifFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			if !errors.Is(err, status.NotFound) {
				return fmt.Errorf("find exif: %w", err)
			}

			exif.Hash = in.Hash
		}

		gps, err := deps.PhotoGpsFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			if !errors.Is(err, status.NotFound) {
				return fmt.Errorf("find gps: %w", err)
			}

			gps.Hash = in.Hash
			gps.GpsTime = time.Now()
		}

		return deps.SchemaRepository().Render(out.Writer,
			jsonform.Page{
				Title: "Edit Photo Details",
				PrependHTML: template.HTML(`
<div style="margin:2em" class="pure-u-2-5">
    <h1>Manage photo</h1>
    <img alt="" style="width:100%" src="/thumb/600w/` + img.Hash.String() + `.jpg" />
</div>` +
					`<script>
function formSaved(x, ctx) { $(ctx.result).html('Saved.').show() } 
</script>`),
			},
			jsonform.Form{
				Title:         "Exif",
				SubmitURL:     "/exif",
				SubmitMethod:  http.MethodPut,
				SuccessStatus: http.StatusNoContent,
				Value:         exif,
				SubmitText:    "Save",
				OnSuccess:     `formSaved`,
			},
			jsonform.Form{
				Title:         "Image",
				SubmitURL:     "/image",
				SubmitMethod:  http.MethodPut,
				SuccessStatus: http.StatusNoContent,
				Value:         img,
				SubmitText:    "Save",
				OnSuccess:     `formSaved`,
			},
			jsonform.Form{
				Title:         "GPS",
				SubmitURL:     "/gps",
				SubmitMethod:  http.MethodPut,
				SuccessStatus: http.StatusNoContent,
				Value:         gps,
				SubmitText:    "Save",
				OnSuccess:     `formSaved`,
			},
		)
	})

	u.SetTags("Control Panel")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

package usecase

import (
	"context"
	"net/http"
	"strings"

	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/settings"
)

type showImageDeps interface {
	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoExifFinder() uniq.Finder[photo.Exif]
	Settings() settings.Values
}

func ShowImage(deps showImageDeps, useAvif bool) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in hashInPath, out *response.EmbeddedSetter) error {
		if deps.Settings().Privacy().HideOriginal && !auth.IsAdmin(ctx) {
			if exif, err := deps.PhotoExifFinder().FindByHash(ctx, in.Hash); err != nil {
				return err
			} else {
				if exif.ProjectionType == "" {
					return status.PermissionDenied
				}
			}
		}

		rw := out.ResponseWriter()

		image, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		if len(image.Settings.HTTPSources) > 0 {
			http.Redirect(rw, in.Request(), image.Settings.HTTPSources[0], http.StatusMovedPermanently)
			return nil
		}

		p := image.Path
		if useAvif {
			p = p[0:strings.LastIndex(p, ".")] + ".avif"
		}

		rw.Header().Set("Cache-Control", "max-age=31536000")

		http.ServeFile(rw, in.Request(), p)

		return nil
	})
	u.SetTags("Image")

	return u
}

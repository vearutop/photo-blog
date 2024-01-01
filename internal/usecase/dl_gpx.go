package usecase

import (
	"context"
	"net/http"

	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type dlGpxDeps interface {
	PhotoGpxFinder() uniq.Finder[photo.Gpx]
}

func DownloadGpx(deps dlGpxDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in hashInPath, out *response.EmbeddedSetter) error {
		rw := out.ResponseWriter()

		gpx, err := deps.PhotoGpxFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		rw.Header().Set("Cache-Control", "max-age=31536000")
		http.ServeFile(rw, in.Request(), gpx.Path)

		return nil
	})
	u.SetTags("Gpx")

	return u
}

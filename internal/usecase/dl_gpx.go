package usecase

import (
	"context"
	"errors"
	"net/http"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type dlGpxDeps interface {
	PhotoGpxFinder() uniq.Finder[photo.Gpx]
}

func DownloadGpx(deps dlGpxDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in hashInPath, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		gpx, err := deps.PhotoGpxFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		rw.Header().Set("Cache-Control", "max-age=31536000")
		http.ServeFile(rw, in.req, gpx.Path)

		return nil
	})
	u.SetTags("Gpx")

	return u
}

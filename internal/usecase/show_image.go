package usecase

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/files"
)

type showImageDeps interface {
	PhotoImageFinder() uniq.Finder[photo.Image]
}

func ShowImage(deps showImageDeps, useAvif bool) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in hashInPath, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		image, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		p := image.Path
		if useAvif {
			p = p[0:strings.LastIndex(p, ".")] + ".avif"
		}

		rw.Header().Set("Cache-Control", "max-age=31536000")

		http.ServeFile(rw, in.req, files.Path(p))

		return nil
	})
	u.SetTags("Image")

	return u
}

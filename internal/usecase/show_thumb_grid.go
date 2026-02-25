package usecase

import (
	"bytes"
	"context"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/bool64/cache"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/ultrahdr"
)

type showThumbGridInput struct {
	request.EmbeddedSetter
	Name string `path:"name"`
	Cols int    `query:"cols" default:"6"`
	Rows int    `query:"rows" default:"2"`
}

type showThumbGridDeps interface {
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoThumbnailer() photo.Thumbnailer
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	MapTilesCache() *cache.FailoverOf[[]byte]
}

func ShowThumbGrid(deps showThumbGridDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in showThumbGridInput, out *response.EmbeddedSetter) error {
		rw := out.ResponseWriter()

		deps.StatsTracker().Add(ctx, "show_thumb_grid", 1)
		deps.CtxdLogger().Info(ctx, "showing thumb grid", "req", in.Request().Header, "name", in.Name, "cols", in.Cols, "rows", in.Rows)

		body, err := deps.MapTilesCache().Get(ctx, []byte("grid/"+in.Name+"/"+strconv.Itoa(in.Cols)+"/"+strconv.Itoa(in.Rows)),
			func(ctx context.Context) ([]byte, error) {
				images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, photo.AlbumHash(in.Name))
				if err != nil {
					return nil, err
				}

				n := in.Cols * in.Rows

				sample := sampleN(images, n)
				var readers []io.Reader

				for _, img := range sample {
					th, err := deps.PhotoThumbnailer().Thumbnail(ctx, img, "600w")
					if err != nil {
						return nil, ctxd.WrapError(ctx, err, "getting thumbnail")
					}

					r, err := th.Reader()
					if err != nil {
						return nil, ctxd.WrapError(ctx, err, "getting thumbnail reader")
					}

					readers = append(readers, r)
				}

				res, err := ultrahdr.Grid(readers, in.Cols, 300, 200, &ultrahdr.GridOptions{
					Quality:       80,
					Interpolation: ultrahdr.InterpolationLanczos2,
				})
				if err != nil {
					return nil, ctxd.WrapError(ctx, err, "creating grid")
				}

				return res.Container, nil
			},
		)

		if err != nil {
			return err
		}

		http.ServeContent(rw, in.Request(), "grid.jpg", time.Now(), bytes.NewReader(body))

		return nil
	})

	u.SetTags("Image")

	return u
}

// sampleN returns up to n randomly selected elements from the input slice.
// It does NOT modify the input slice.
// If n >= len(items), it returns a copy of the whole slice (in original order).
func sampleN[T any](items []T, n int) []T {
	if n <= 0 {
		return nil
	}

	total := len(items)
	if n >= total {
		result := make([]T, total)
		copy(result, items)
		return result
	}

	result := make([]T, 0, n)
	seen := make(map[int]struct{}, n)

	for len(result) < n {
		idx := rand.IntN(total)

		if _, already := seen[idx]; already {
			continue
		}

		seen[idx] = struct{}{}
		result = append(result, items[idx])
	}

	return result
}

package usecase

import (
	"bytes"
	"context"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/stats"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/ultrahdr"
)

type showThumbGridInput struct {
	request.EmbeddedSetter
	Name   string `path:"name"`
	Cols   int    `query:"cols" default:"6"`
	Rows   int    `query:"rows" default:"2"`
	CellW  int    `query:"cell_w" default:"300"`
	CellH  int    `query:"cell_h" default:"200"`
	Offset int    `query:"offset" default:"-1"`
}

func (in showThumbGridInput) string() string {
	return in.Name + "/" + strconv.Itoa(in.Cols) + "/" + strconv.Itoa(in.Rows) + "/" + strconv.Itoa(in.Offset) + "/" +
		strconv.Itoa(in.CellW) + "/" + strconv.Itoa(in.CellH)
}

type showThumbGridDeps interface {
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoThumbnailer() photo.Thumbnailer
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	DepCache() *dep.Cache
	PersistentCacheStorage() *sqluct.Storage
}

func ShowThumbGrid(deps showThumbGridDeps) usecase.Interactor {
	const cacheName = "thumb-grid"

	c := service.MakePersistentCacheOf[[]byte](deps, cacheName, time.Hour)

	u := usecase.NewInteractor(func(ctx context.Context, in showThumbGridInput, out *response.EmbeddedSetter) error {
		rw := out.ResponseWriter()

		deps.StatsTracker().Add(ctx, "show_thumb_grid", 1)
		deps.CtxdLogger().Info(ctx, "showing thumb grid", "req", in.Request().Header, "name", in.Name, "cols", in.Cols, "rows", in.Rows)

		cacheKey := []byte("grid/" + in.string())
		body, err := c.Get(ctx,
			cacheKey,
			func(ctx context.Context) ([]byte, error) {
				if err := deps.DepCache().ResetKey(ctx, cacheName, cacheKey); err != nil {
					return nil, err
				}

				deps.DepCache().AlbumDependency(cacheName, cacheKey, in.Name)

				images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, photo.AlbumHash(in.Name))
				if err != nil {
					return nil, err
				}

				n := in.Cols * in.Rows

				var sample []photo.Image
				if in.Offset >= 0 {
					for i := in.Offset; i < in.Offset+n; i++ {
						sample = append(sample, images[i%len(images)])
					}
				} else {
					sample = sampleN(images, n)
				}

				var readers []io.Reader

				for _, img := range sample {
					th, err := deps.PhotoThumbnailer().Thumbnail(ctx, img, "1200w")
					if err != nil {
						return nil, ctxd.WrapError(ctx, err, "getting thumbnail")
					}

					r, err := th.Reader()
					if err != nil {
						return nil, ctxd.WrapError(ctx, err, "getting thumbnail reader")
					}

					readers = append(readers, r)
				}

				res, err := ultrahdr.Grid(readers, in.Cols, in.CellW, in.CellH, &ultrahdr.GridOptions{
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

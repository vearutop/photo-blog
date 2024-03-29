package usecase

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bool64/brick"
	"github.com/bool64/cache"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

func MapTile(deps *service.Locator) usecase.Interactor {
	type mapTileID struct {
		request.EmbeddedSetter

		Retina string `path:"r"`
		Zoom   string `path:"z"`
		X      string `path:"x"`
		Y      string `path:"y"`
	}

	mapCache := brick.MakeCacheOf[photo.MapTile](deps, "map-tiles", 7*24*time.Hour,
		func(cfg *cache.FailoverConfigOf[photo.MapTile]) {
			cfg.BackendConfig.CountSoftLimit = 1000
			cfg.BackendConfig.DeleteExpiredJobInterval = time.Hour
			cfg.BackendConfig.DeleteExpiredAfter = 2 * time.Hour
		},
	)

	return usecase.NewInteractor(func(ctx context.Context, input mapTileID, output *response.EmbeddedSetter) error {
		rw := output.ResponseWriter()

		t, err := mapCache.Get(ctx, []byte(input.Retina+"/"+input.Zoom+"/"+input.X+"/"+input.Y),
			func(ctx context.Context) (photo.MapTile, error) {
				u := deps.Settings().Maps().Tiles
				if u == "" {
					u = "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
				}

				u = strings.ReplaceAll(u, "{r}", input.Retina)
				u = strings.ReplaceAll(u, "{z}", input.Zoom)
				u = strings.ReplaceAll(u, "{x}", input.X)
				u = strings.ReplaceAll(u, "{y}", input.Y)

				resp, err := http.Get(u)
				if err != nil {
					return photo.MapTile{}, err
				}

				defer func() {
					_ = resp.Body.Close()
				}()

				d, err := io.ReadAll(resp.Body)
				if err != nil {
					return photo.MapTile{}, err
				}

				return photo.MapTile{
					Data:       d,
					ModifiedAt: time.Now(),
				}, nil
			},
		)
		if err != nil {
			return err
		}

		http.ServeContent(rw, input.Request(), "image.png", t.ModifiedAt, bytes.NewReader(t.Data))

		return nil
	})
}

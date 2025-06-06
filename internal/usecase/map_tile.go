package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

func MapTile(deps *service.Locator) usecase.Interactor {
	type mapTileID struct {
		request.EmbeddedSetter

		Retina string `path:"r"`
		Zoom   string `path:"z"`
		X      string `path:"x"`
		Y      string `path:"y"`
		S      string `path:"s"`
	}

	return usecase.NewInteractor(func(ctx context.Context, input mapTileID, output *response.EmbeddedSetter) error {
		rw := output.ResponseWriter()

		u := deps.Settings().Maps().Tiles
		if u == "" {
			u = "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
		}

		u = strings.ReplaceAll(u, "{r}", input.Retina)
		u = strings.ReplaceAll(u, "{z}", input.Zoom)
		u = strings.ReplaceAll(u, "{x}", input.X)
		u = strings.ReplaceAll(u, "{y}", input.Y)
		u = strings.ReplaceAll(u, "{s}", input.S)

		mu := sync.Mutex{}

		t, err := deps.MapTilesCache().Get(ctx, []byte(u+"-png"),
			func(ctx context.Context) ([]byte, error) {
				mu.Lock()
				defer mu.Unlock()

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
				if err != nil {
					return nil, err
				}

				req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:128.0) Gecko/20100101 Firefox/128.0")
				req.Header.Set("Referer", input.Request().Header.Get("Referer"))
				req.Header.Set("Accept", "mage/avif,image/jxl,image/webp,image/png,image/svg+xml,image/*;q=0.8,*/*;q=0.5")
				req.Header.Set("Accept-Language", "en-US,en;q=0.5")
				req.Header.Set("Accept-Encoding", "gzip, deflate, br")

				resp, err := http.DefaultTransport.RoundTrip(req)
				if err != nil {
					return nil, err
				}

				defer func() {
					_ = resp.Body.Close()
				}()

				d, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, err
				}

				if resp.StatusCode != http.StatusOK {
					return nil, fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode, string(d))
				}

				deps.CtxdLogger().Debug(ctx, "map tile cached", "size", len(d))

				return d, nil
			},
		)
		if err != nil {
			return err
		}

		rw.Header().Set("Etag", strconv.FormatUint(xxhash.Sum64(t), 36))
		rw.Header().Set("Content-Type", "image/png")
		rw.Header().Set("Cache-Control", "max-age=31536000")

		http.ServeContent(rw, input.Request(), "image.png", time.Time{}, bytes.NewReader(t))

		return nil
	})
}

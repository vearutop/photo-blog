package usecase

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/bool64/ctxd"
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
	CtxdLogger() ctxd.Logger
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

		r := in.Request()
		rw := out.ResponseWriter()

		image, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		if len(image.Settings.HTTPSources) > 0 {
			remoteURL := image.Settings.HTTPSources[0]

			if r.Header.Get("X-Mirror") != "" {
				deps.CtxdLogger().Info(ctx, "serving image from remote address", "img", image, "url", image.Settings.HTTPSources[0])

				return serveRemote(ctx, rw, r, remoteURL)
			}

			deps.CtxdLogger().Info(ctx, "redirecting image to remote address", "img", image, "url", image.Settings.HTTPSources[0])
			http.Redirect(rw, in.Request(), remoteURL, http.StatusMovedPermanently)
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

func serveRemote(ctx context.Context, w http.ResponseWriter, r *http.Request, remoteURL string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", remoteURL, nil)
	if err != nil {
		return err
	}

	req.Header = r.Header
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		w.Header()[k] = vv
	}
	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)

	return err
}

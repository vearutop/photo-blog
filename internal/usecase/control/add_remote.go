package control

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/image"
)

type addRemoteDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageAdder() photo.AlbumImageAdder

	PhotoImageEnsurer() uniq.Ensurer[photo.Image]
	PhotoImageFinder() uniq.Finder[photo.Image]

	PhotoExifEnsurer() uniq.Ensurer[photo.Exif]
	PhotoGpsEnsurer() uniq.Ensurer[photo.Gps]

	PhotoThumbnailer() photo.Thumbnailer

	DepCache() *dep.Cache
}

// AddRemote creates use case interactor to add remote directory of photos to an album.
func AddRemote(deps addRemoteDeps) usecase.Interactor {
	type addDirInput struct {
		URL  string `formData:"url" description:"URL to JSON list file."`
		Name string `path:"name" description:"Album name."`
	}

	type addDirOutput struct {
		Names []string `json:"names"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in addDirInput, out *addDirOutput) error {
		deps.StatsTracker().Add(ctx, "add_remote", 1)
		deps.CtxdLogger().Important(ctx, "adding remote directory", "url", in.URL)

		a, err := deps.PhotoAlbumFinder().FindByHash(ctx, uniq.StringHash(in.Name))
		if err != nil {
			return ctxd.WrapError(ctx, err, "find album", "name", in.Name)
		}

		listResp, err := http.Get(in.URL)
		if err != nil {
			return fmt.Errorf("getting JSON list: %w", err)
		}

		defer func() {
			if err := listResp.Body.Close(); err != nil {
				deps.CtxdLogger().Warn(ctx, "error closing response body", "error", err)
			}
		}()

		j, err := io.ReadAll(listResp.Body)
		if err != nil {
			return fmt.Errorf("reading JSON list: %w", err)
		}

		var list []image.Data
		if err := json.Unmarshal(j, &list); err != nil {
			return fmt.Errorf("unmarshalling JSON list: %w", err)
		}

		baseURL, _ := path.Split(in.URL)

		var (
			imgHashes    []uniq.Hash
			imgByHash    = map[uniq.Hash]photo.Image{}
			zippedThumbs = map[string][]photo.Thumb{}
		)

		for _, l := range list {
			newImg := l.Image
			img, err := deps.PhotoImageFinder().FindByHash(ctx, newImg.Hash)
			if errors.Is(err, status.NotFound) {
				img = newImg
			}

			img.Settings.HTTPSources = append(img.Settings.HTTPSources, baseURL+img.Path)
			if _, err := deps.PhotoImageEnsurer().Ensure(ctx, img); err != nil {
				return fmt.Errorf("ensure image: %w", err)
			}

			imgByHash[img.Hash] = img

			if l.Exif != nil {
				l.Exif.Hash = img.Hash
				if _, err := deps.PhotoExifEnsurer().Ensure(ctx, *l.Exif); err != nil {
					return fmt.Errorf("ensure exif: %w", err)
				}
			}

			if l.Gps != nil {
				l.Gps.Hash = img.Hash
				if _, err := deps.PhotoGpsEnsurer().Ensure(ctx, *l.Gps); err != nil {
					return fmt.Errorf("ensure gps: %w", err)
				}
			}

			for _, th := range l.Thumbs {
				if th.FilePath == "" {
					continue
				}

				if strings.Contains(th.FilePath, ".zip/") {
					zipURL := baseURL + path.Dir(th.FilePath)
					zippedThumbs[zipURL] = append(zippedThumbs[zipURL], th)
				} else {
					th.FilePath = baseURL + th.FilePath
					ctx := image.LargerThumbToContext(ctx, th)
					if _, err := deps.PhotoThumbnailer().Thumbnail(ctx, img, th.Format); err != nil {
						return fmt.Errorf("thumbnail: %w", err)
					}
				}
			}

			imgHashes = append(imgHashes, img.Hash)
		}

		if len(imgHashes) > 0 {
			if err := deps.PhotoAlbumImageAdder().AddImages(ctx, a.Hash, imgHashes...); err != nil {
				return fmt.Errorf("add album images: %w", err)
			}
		}

		for zipURL, thumbs := range zippedThumbs {
			resp, err := http.Get(zipURL)
			if err != nil {
				return fmt.Errorf("getting thumbs zip contents %s: %w", zipURL, err)
			}

			defer func() {
				if err := resp.Body.Close(); err != nil {
					deps.CtxdLogger().Warn(ctx, "error closing response body", "error", err)
				}
			}()

			zipFn := "temp/" + path.Base(zipURL)
			f, err := os.Create(zipFn)
			if err != nil {
				return fmt.Errorf("creating zip file: %w", err)
			}

			size, err := io.Copy(f, resp.Body)
			if err != nil {
				return err
			}

			zr, err := zip.NewReader(f, size)
			if err != nil {
				return err
			}

			for _, th := range thumbs {
				tf, err := zr.Open(path.Base(th.FilePath))
				if err != nil {
					return err
				}

				th.Data, err = io.ReadAll(tf)
				if err != nil {
					return err
				}

				if err := tf.Close(); err != nil {
					return err
				}

				th.FilePath = ""

				ctx := image.LargerThumbToContext(ctx, th)
				_, err = deps.PhotoThumbnailer().Thumbnail(ctx, imgByHash[th.Hash], th.Format)
				if err != nil {
					return err
				}
			}

			if err := os.Remove(zipFn); err != nil {
				return fmt.Errorf("removing zip file %s: %w", zipFn, err)
			}
		}

		if err == nil {
			err = deps.DepCache().AlbumChanged(ctx, a.Name)
		}

		return err
	})

	u.SetDescription("Add a http-remote directory of photos to an album.")
	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

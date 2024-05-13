package control

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/files"
	"github.com/vearutop/photo-blog/internal/infra/image"
	"github.com/vearutop/photo-blog/internal/infra/upload"
)

type addToAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoAlbumImageAdder() photo.AlbumImageAdder

	PhotoGpsEnsurer() uniq.Ensurer[photo.Gps]
	PhotoImageEnsurer() uniq.Ensurer[photo.Image]

	FilesProcessor() *files.Processor

	DepCache() *dep.Cache
}
type addToAlbumInput struct {
	DstAlbumName        string    `path:"name" description:"Name of destination album to add photo."`
	SrcImageHash        uniq.Hash `json:"image_hash,omitempty" title:"Image Hash" description:"Hash of an image to add to album."`
	SrcAlbumName        string    `json:"album_name,omitempty" title:"Source Album Name" description:"Name of a source album to add photos from."`
	SrcImageURL         string    `json:"image_url,omitempty" title:"Fetch image from a publicly available URL."`
	SrcGPS              string    `json:"image_lat_lon,omitempty" title:"Set image GPS location after adding from URL." description:"In latitude,longitude format."`
	SrcImageDescription string    `json:"image_description,omitempty" title:"Set image description after adding from URL." formType:"textarea" description:"Description of an image, can contain HTML."`
}

// AddToAlbum creates use case interactor to add a single photo or photos from an album to another album.
func AddToAlbum(deps addToAlbumDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in addToAlbumInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "add_to_album", 1)
		deps.CtxdLogger().Info(ctx, "adding to album", "name", in.DstAlbumName, "hash", in.SrcImageHash)

		dstAlbum, err := deps.PhotoAlbumFinder().FindByHash(ctx, photo.AlbumHash(in.DstAlbumName))
		if err != nil {
			return err
		}

		if in.SrcAlbumName != "" {
			if images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, photo.AlbumHash(in.SrcAlbumName)); err == nil && len(images) > 0 {
				imgHashes := make([]uniq.Hash, 0, len(images))
				for _, img := range images {
					imgHashes = append(imgHashes, img.Hash)
				}

				return deps.PhotoAlbumImageAdder().AddImages(ctx, dstAlbum.Hash, imgHashes...)
			}
		}

		if in.SrcImageHash != 0 {
			img, err := deps.PhotoImageFinder().FindByHash(ctx, in.SrcImageHash)
			if err != nil {
				return err
			}

			err = deps.PhotoAlbumImageAdder().AddImages(ctx, dstAlbum.Hash, img.Hash)
		}

		if in.SrcImageURL != "" {
			u, err := url.Parse(in.SrcImageURL)
			if err != nil {
				return nil
			}

			albumPath := upload.AlbumPath(in.DstAlbumName)
			if err := os.MkdirAll(albumPath, 0o700); err != nil {
				return err
			}

			dstFilePath := upload.AlbumFilePath(albumPath, path.Base(u.Path))
			// TODO: Add file exists check here.

			if !strings.HasSuffix(dstFilePath, ".jpg") {
				dstFilePath += ".jpg"
			}

			dst, err := os.Create(dstFilePath)
			if err != nil {
				return fmt.Errorf("create dst file %s: %w", dstFilePath, err)
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, in.SrcImageURL, nil)
			if err != nil {
				return err
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")

			resp, err := http.DefaultTransport.RoundTrip(req)
			if err != nil {
				return fmt.Errorf("fetching image url %s: %w", in.SrcImageURL, err)
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading image %s: %w", in.SrcImageURL, err)
			}

			data, err = image.ToJpeg(data)
			if err != nil {
				return fmt.Errorf("converting image %s: %w", in.SrcImageURL, err)
			}

			_, err = dst.Write(data)
			if err != nil {
				return fmt.Errorf("writing image %s: %w", dstFilePath, err)
			}

			if err := dst.Close(); err != nil {
				return err
			}

			err = deps.FilesProcessor().AddFile(ctx, in.DstAlbumName, dstFilePath, func(hash uniq.Hash) {
				ctx := detachedContext{
					parent: ctx,
				}

				if in.SrcGPS != "" {
					parts := strings.Split(in.SrcGPS, ",")

					gps := photo.Gps{}
					gps.Hash = hash

					if len(parts) != 2 {
						deps.CtxdLogger().Error(ctx, "invalid gps location", "location", in.SrcGPS)
						return
					}

					gps.Latitude, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 32)
					if err != nil {
						deps.CtxdLogger().Error(ctx, "invalid gps location", "location", in.SrcGPS)
						return
					}

					gps.Longitude, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 32)
					if err != nil {
						deps.CtxdLogger().Error(ctx, "invalid gps location", "location", in.SrcGPS)
						return
					}

					if _, err := deps.PhotoGpsEnsurer().Ensure(ctx, gps); err != nil {
						deps.CtxdLogger().Error(ctx, "ensure gps location", "error", err)
						return
					}
				}

				if in.SrcImageDescription != "" {
					time.Sleep(time.Second)

					img, err := deps.PhotoImageFinder().FindByHash(ctx, hash)
					if err != nil {
						deps.CtxdLogger().Error(ctx, "find image", "error", err)
						return
					}

					img.Settings.Description = in.SrcImageDescription

					if _, err := deps.PhotoImageEnsurer().Ensure(ctx, img); err != nil {
						deps.CtxdLogger().Error(ctx, "ensure image", "error", err)
						return
					}
				}
			})
			if err != nil {
				return err
			}

		}

		if err == nil {
			err = deps.DepCache().AlbumChanged(ctx, dstAlbum.Name)
		}

		return err
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown)

	return u
}

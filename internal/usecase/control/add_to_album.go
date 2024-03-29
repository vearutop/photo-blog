package control

import (
	"context"
	"fmt"
	"github.com/vearutop/photo-blog/internal/infra/files"
	"github.com/vearutop/photo-blog/internal/infra/upload"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/dep"
)

type addToAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoAlbumImageAdder() photo.AlbumImageAdder

	FilesProcessor() *files.Processor

	DepCache() *dep.Cache
}
type addToAlbumInput struct {
	DstAlbumName string    `path:"name" description:"Name of destination album to add photo."`
	SrcImageHash uniq.Hash `json:"image_hash,omitempty" title:"Image Hash" description:"Hash of an image to add to album."`
	SrcAlbumName string    `json:"album_name,omitempty" title:"Source Album Name" description:"Name of a source album to add photos from."`
	SrcImageURL  string    `json:"image_url,omitempty" title:"Fetch image from a publicly available URL."`
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

			dstFilePath := upload.AlbumFilePath(upload.AlbumPath(in.DstAlbumName), path.Base(u.Path))
			// TODO: Add file exists check here.

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

			if _, err := io.Copy(dst, resp.Body); err != nil {
				return fmt.Errorf("saving remote file: %w", err)
			}

			if err := dst.Close(); err != nil {
				return err
			}

			if err := deps.FilesProcessor().AddFile(ctx, in.DstAlbumName, dstFilePath); err != nil {
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

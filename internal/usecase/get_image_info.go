package usecase

import (
	"context"
	"errors"
	"os"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/image"
)

type getImageInfoDeps interface {
	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoGpsFinder() uniq.Finder[photo.Gps]
	PhotoExifFinder() uniq.Finder[photo.Exif]
}

func GetImageInfo(deps getImageInfoDeps) usecase.Interactor {
	type getImageInput struct {
		Hash     uniq.Hash `path:"hash"`
		ReadMeta bool      `query:"read_meta" description:"Read meta from original file."`
	}
	type imageInfo struct {
		Image photo.Image `json:"image"`
		Exif  photo.Exif  `json:"exif"`
		Gps   *photo.Gps  `json:"gps"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in getImageInput, out *imageInfo) error {
		img, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
		if err != nil {
			return err
		}

		out.Image = img

		if in.ReadMeta {
			f, err := os.Open(img.Path)
			if err != nil {
				return err
			}
			defer f.Close()

			meta, err := image.ReadMeta(f)
			if err != nil {
				return err
			}

			out.Exif = meta.Exif
			out.Gps = meta.GpsInfo
		} else {
			out.Exif, err = deps.PhotoExifFinder().FindByHash(ctx, in.Hash)
			if err != nil && !errors.Is(err, status.NotFound) {
				return err
			}

			gps, err := deps.PhotoGpsFinder().FindByHash(ctx, in.Hash)
			if err != nil && !errors.Is(err, status.NotFound) {
				return err
			}

			if err == nil {
				out.Gps = &gps
			}
		}

		return nil
	})
	u.SetTags("Image")

	return u
}

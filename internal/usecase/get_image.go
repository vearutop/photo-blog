package usecase

import (
	"context"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/infra/image"
	"os"
)

func GetImage(deps showImageDeps) usecase.Interactor {
	type getImageInput struct {
		Hash string `path:"hash"`
	}
	type imageInfo struct {
		Image photo.Image `json:"image"`
		Meta  image.Meta  `json:"meta"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in getImageInput, out *imageInfo) error {
		h, err := photo.StringHashToInt64(in.Hash)
		if err != nil {
			return err
		}

		img, err := deps.PhotoImageFinder().FindByHash(ctx, h)
		if err != nil {
			return err
		}

		out.Image = img

		f, err := os.Open(img.Path)
		if err != nil {
			return err
		}
		defer f.Close()

		out.Meta, err = image.ReadMeta(f)

		return err
	})

	return u
}

package usecase

import (
	"context"
	"os"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/infra/image"
)

func GetImage(deps showImageDeps) usecase.Interactor {
	type getImageInput struct {
		Hash photo.Hash `path:"hash"`
	}
	type imageInfo struct {
		Image photo.Image `json:"image"`
		Meta  image.Meta  `json:"meta"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in getImageInput, out *imageInfo) error {
		img, err := deps.PhotoImageFinder().FindByHash(ctx, in.Hash)
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

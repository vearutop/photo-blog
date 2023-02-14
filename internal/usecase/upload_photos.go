package usecase

import (
	"context"
	"mime/multipart"

	"github.com/swaggest/usecase"
)

func UploadPhotos() usecase.Interactor {
	type photoFiles struct {
		Photos []*multipart.FileHeader `formData:"photos"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in photoFiles, out *struct{}) error {
		for _, f := range in.Photos {
			_ = f
		}

		return nil
	})

	return u
}

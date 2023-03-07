package control

import (
	"context"
	"github.com/bool64/ctxd"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/text"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type updateImageDeps interface {
	CtxdLogger() ctxd.Logger
}

type UpdateImageInput struct {
	Hash         uniq.Hash    `json:"hash" formType:"hidden"`
	Exif         *photo.Exif  `json:"exif,omitempty"`
	Gps          *photo.Gps   `json:"gps,omitempty"`
	Descriptions []text.Label `json:"descriptions,omitempty"`
}

// UpdateImage creates use case interactor to update entity data.
func UpdateImage(deps updateImageDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in UpdateImageInput, out *struct{}) (err error) {
		deps.CtxdLogger().Important(ctx, "update image", "input", in)

		return nil
	})

	u.SetTitle("Update Image")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

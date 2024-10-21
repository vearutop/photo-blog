package usecase

import (
	"context"
	"errors"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
)

func DeleteFavorite(deps FavoriteDeps) usecase.Interactor {
	type deleteFavorite struct {
		ImageHash uniq.Hash `query:"image_hash"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input deleteFavorite, output *struct{}) error {
		visitorHash := auth.VisitorFromContext(ctx)

		if visitorHash == 0 {
			return errors.New("missing visitor hash")
		}

		return deps.FavoriteRepository().DeleteImages(ctx, visitorHash, input.ImageHash)
	})

	return u
}

package usecase

import (
	"context"
	"errors"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/storage"
)

type FavoriteDeps interface {
	FavoriteRepository() *storage.FavoriteRepository
}

func AddFavorite(deps FavoriteDeps) usecase.Interactor {
	type AddFavorite struct {
		ImageHash uniq.Hash `query:"image_hash"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input AddFavorite, output *struct{}) error {
		visitorHash := auth.VisitorFromContext(ctx)

		if visitorHash == 0 {
			return errors.New("missing visitor hash")
		}

		return deps.FavoriteRepository().AddImages(ctx, visitorHash, input.ImageHash)
	})

	return u
}

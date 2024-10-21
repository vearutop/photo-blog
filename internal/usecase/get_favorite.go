package usecase

import (
	"context"
	"errors"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
)

func GetFavorite(deps FavoriteDeps) usecase.Interactor {
	type getFavorite struct {
		AlbumHash uniq.Hash `query:"album_hash,omitempty"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getFavorite, output *[]uniq.Hash) error {
		visitorHash := auth.VisitorFromContext(ctx)

		if visitorHash == 0 {
			return errors.New("missing visitor hash")
		}

		res, err := deps.FavoriteRepository().FindImageHashes(ctx, visitorHash, input.AlbumHash)
		*output = res

		return err
	})

	return u
}

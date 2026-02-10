package control

import (
	"context"
	"fmt"

	"github.com/bool64/ctxd"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/dep"
)

type setAlbumImageTimeDeps interface {
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageAdder() photo.AlbumImageAdder
	CtxdLogger() ctxd.Logger
	DepCache() *dep.Cache
}

func SetAlbumImageTime(deps setAlbumImageTimeDeps) usecase.Interactor {
	type req struct {
		AlbumName string    `json:"album_name"`
		ImageHash uniq.Hash `json:"image_hash"`
		Timestamp int64     `json:"timestamp"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input req, output *struct{}) error {
		deps.CtxdLogger().Info(ctx, "set album image timestamp", "input", input)

		album, err := deps.PhotoAlbumFinder().FindByHash(ctx, photo.AlbumHash(input.AlbumName))
		if err != nil {
			return fmt.Errorf("find album %s: %w", input.AlbumName, err)
		}

		delta := int64(-1)
		if album.Settings.NewestFirst {
			delta = 1
		}

		if err := deps.PhotoAlbumImageAdder().SetAlbumImageTimestamp(ctx,
			photo.AlbumHash(input.AlbumName),
			input.ImageHash,
			input.Timestamp+delta); err != nil {
			return err
		}

		return deps.DepCache().AlbumChanged(ctx, input.AlbumName)
	})

	return u
}

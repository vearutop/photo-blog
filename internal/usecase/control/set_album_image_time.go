package control

import (
	"context"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"time"
)

type setAlbumImageTimeDeps interface {
	PhotoAlbumImageAdder() photo.AlbumImageAdder
}

func SetAlbumImageTime(deps setAlbumImageTimeDeps) usecase.Interactor {
	type req struct {
		AlbumName string    `json:"album_name"`
		ImageHash uniq.Hash `json:"image_hash"`
		Timestamp time.Time `json:"timestamp"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input req, output *struct{}) error {
		return deps.PhotoAlbumImageAdder().SetAlbumImageTimestamp(ctx, photo.AlbumHash(input.AlbumName), input.ImageHash, input.Timestamp)
	})

	return u
}

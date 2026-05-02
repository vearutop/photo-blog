package usecase

import (
	"context"
	"errors"
	"io"
	"strconv"

	"github.com/bool64/cache"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/image/sprite"
)

type showAlbumSpriteDeps interface {
	AlbumSprites() *sprite.Service
}

type showAlbumSpriteInput struct {
	request.EmbeddedSetter
	Key string `path:"key"`
}

func ShowAlbumSprite(deps showAlbumSpriteDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in showAlbumSpriteInput, out *response.EmbeddedSetter) error {
		entry, err := deps.AlbumSprites().Open(ctx, in.Key)
		if err != nil {
			if errors.Is(err, cache.ErrNotFound) {
				return status.NotFound
			}

			return err
		}

		rc, err := entry.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		rw := out.ResponseWriter()
		rw.Header().Set("Cache-Control", "max-age=31536000")
		rw.Header().Set("Content-Type", "image/jpeg")
		if size := entry.Meta().Size; size > 0 {
			rw.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		}

		_, err = io.Copy(rw, rc)
		if err != nil {
			return err
		}

		return nil
	})
	u.SetTags("Image")

	return u
}

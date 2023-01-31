package usecase

import (
	"context"
	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"image/jpeg"
	"os"
	"sync/atomic"
	"time"
)

type indexAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PhotoAlbumFinder() photo.AlbumFinder
	PhotoThumbnailer() photo.Thumbnailer
	PhotoImageUpdater() photo.ImageUpdater
}

// IndexAlbum creates use case interactor to index album.
func IndexAlbum(deps indexAlbumDeps) usecase.Interactor {
	type getAlbumInput struct {
		Name string `path:"name"`
	}

	var inProgress int64

	u := usecase.NewInteractor(func(ctx context.Context, in getAlbumInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "get_album", 1)
		deps.CtxdLogger().Info(ctx, "getting album", "name", in.Name)

		album, err := deps.PhotoAlbumFinder().FindByName(ctx, in.Name)
		if err != nil {
			return err
		}

		images, err := deps.PhotoAlbumFinder().FindImages(ctx, album.ID)
		if err != nil {
			return err
		}

		deps.StatsTracker().Set(ctx, "indexing_images_pending",
			float64(atomic.AddInt64(&inProgress, int64(len(images)))))

		detached := detachedContext{parent: ctx}
		go func() {
			for _, img := range images {
				if img.Width == 0 {
					f, err := os.Open(img.Path)
					if err != nil {
						deps.CtxdLogger().Error(detached, "failed to open image file",
							"error", err, "image", img)

						continue
					}
					c, err := jpeg.DecodeConfig(f)
					f.Close()

					if err != nil {
						deps.CtxdLogger().Error(detached, "failed to get image dimensions",
							"error", err, "image", img)

						continue
					}

					img.Width = int64(c.Width)
					img.Height = int64(c.Height)

					if err := deps.PhotoImageUpdater().Update(detached, img.ImageData); err != nil {
						deps.CtxdLogger().Error(detached, "failed to update image",
							"error", err, "image", img)

						continue
					}
				}

				for _, size := range photo.ThumbSizes {
					_, err := deps.PhotoThumbnailer().Thumbnail(detached, img, size)
					if err != nil {
						deps.CtxdLogger().Error(detached, "failed to get thumbnail",
							"error", err, "image", img, "size", size)
					}
					deps.StatsTracker().Set(ctx, "indexing_images_pending",
						float64(atomic.AddInt64(&inProgress, -1)))

				}
			}
		}()

		return nil
	})

	u.SetTags("Photos")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

// detachedContext exposes parent values, but suppresses parent cancellation.
type detachedContext struct {
	parent context.Context //nolint:containedctx // This wrapping is here on purpose.
}

func (d detachedContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (d detachedContext) Done() <-chan struct{} {
	return nil
}

func (d detachedContext) Err() error {
	return nil
}

func (d detachedContext) Value(key interface{}) interface{} {
	return d.parent.Value(key)
}

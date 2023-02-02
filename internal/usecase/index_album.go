package usecase

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type indexAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() photo.AlbumFinder
	PhotoImageIndexer() photo.ImageIndexer
}

// IndexAlbum creates use case interactor to index album.
func IndexAlbum(deps indexAlbumDeps) usecase.Interactor {
	type getAlbumInput struct {
		Name string `path:"name"`
	}

	var inProgress int64

	u := usecase.NewInteractor(func(ctx context.Context, in getAlbumInput, out *struct{}) error {
		deps.StatsTracker().Add(ctx, "index_album", 1)
		deps.CtxdLogger().Info(ctx, "indexing album", "name", in.Name)

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

		go func() {
			ctx := detachedContext{parent: ctx}
			for _, img := range images {
				if err := deps.PhotoImageIndexer().Index(ctx, img); err != nil {
					deps.CtxdLogger().Error(ctx, "failed to index image", "error", err)
				}
				deps.StatsTracker().Set(ctx, "indexing_images_pending",
					float64(atomic.AddInt64(&inProgress, -1)))
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

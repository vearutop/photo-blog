package control

import (
	"context"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/topic"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/image"
	"github.com/vearutop/photo-blog/pkg/qlite"
)

type indexAlbumDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger

	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumUpdater() uniq.Updater[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoImageFinder() uniq.Finder[photo.Image]

	QueueBroker() *qlite.Broker
}

type indexAlbumInput struct {
	Name string `path:"name" description:"Album name, use '-' for all images and albums."`
	photo.IndexingFlags
}

// IndexAlbum creates use case interactor to index album.
func IndexAlbum(deps indexAlbumDeps) usecase.IOInteractorOf[indexAlbumInput, struct{}] {
	u := usecase.NewInteractor(func(ctx context.Context, in indexAlbumInput, out *struct{}) (err error) {
		deps.StatsTracker().Add(ctx, "index_album", 1)
		deps.CtxdLogger().Info(ctx, "indexing album", "name", in.Name)

		var images []photo.Image

		if in.Name != "-" {
			album, err := deps.PhotoAlbumFinder().FindByHash(ctx, photo.AlbumHash(in.Name))
			if err != nil {
				return err
			}

			images, err = deps.PhotoAlbumImageFinder().FindImages(ctx, album.Hash)
			if err != nil {
				return err
			}
		} else {
			albums, err := deps.PhotoAlbumFinder().FindAll(ctx)
			if err != nil {
				return err
			}

			for _, album := range albums {
				if album.Hash == 0 {
					album.Hash = uniq.StringHash(album.Name)

					if err := deps.PhotoAlbumUpdater().Update(ctx, album); err != nil {
						return err
					}
				}
			}

			images, err = deps.PhotoImageFinder().FindAll(ctx)
			if err != nil {
				return err
			}
		}

		deps.CtxdLogger().Info(ctx, "indexing album", "num_images", len(images))

		for _, img := range images {
			if err := deps.QueueBroker().Publish(ctx, topic.IndexImage, image.IndexJob{Image: img}, func(msg *qlite.Message) {
				msg.PublishOnSuccess(topic.AlbumChanged, in.Name)
			}); err != nil {
				deps.CtxdLogger().Error(ctx, "error publishing album index", "error", err)
				return err
			}
		}

		return nil
	})

	u.SetTags("Album")
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

package usecase

import (
	"context"
	"errors"
	"path"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type getAlbumImagesDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoGpsFinder() uniq.Finder[photo.Gps]
	PhotoExifFinder() uniq.Finder[photo.Exif]
}

// GetAlbumImages creates use case interactor to get album data.
func GetAlbumImages(deps getAlbumImagesDeps) usecase.Interactor {
	type getAlbumInput struct {
		Name string `path:"name"`
	}

	type image struct {
		Name     string      `json:"name"`
		Hash     string      `json:"hash"`
		Width    int64       `json:"width"`
		Height   int64       `json:"height"`
		BlurHash string      `json:"blur_hash,omitempty"`
		Gps      *photo.Gps  `json:"gps,omitempty"`
		Exif     *photo.Exif `json:"exif,omitempty"`
	}

	type getAlbumOutput struct {
		Album  photo.Album `json:"album"`
		Images []image     `json:"images,omitempty"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in getAlbumInput, out *getAlbumOutput) error {
		deps.StatsTracker().Add(ctx, "get_album_images", 1)
		deps.CtxdLogger().Info(ctx, "getting album images", "name", in.Name)

		albumHash := photo.AlbumHash(in.Name)

		album, err := deps.PhotoAlbumFinder().FindByHash(ctx, albumHash)
		if err != nil {
			return err
		}

		images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, albumHash)
		if err != nil {
			return err
		}

		out.Album = album
		out.Images = make([]image, 0, len(images))
		for _, i := range images {
			img := image{
				Name:     path.Base(i.Path),
				Hash:     i.Hash.String(),
				Width:    i.Width,
				Height:   i.Height,
				BlurHash: i.BlurHash,
			}

			gps, err := deps.PhotoGpsFinder().FindByHash(ctx, i.Hash)
			if err == nil {
				img.Gps = &gps
			} else if !errors.Is(err, status.NotFound) {
				deps.CtxdLogger().Warn(ctx, "failed to find gps",
					"hash", i.Hash.String(), "error", err.Error())
			}

			exif, err := deps.PhotoExifFinder().FindByHash(ctx, i.Hash)
			if err == nil {
				img.Exif = &exif
			} else if !errors.Is(err, status.NotFound) {
				deps.CtxdLogger().Warn(ctx, "failed to find exif",
					"hash", i.Hash.String(), "error", err.Error())
			}

			out.Images = append(out.Images, img)
		}

		return nil
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}
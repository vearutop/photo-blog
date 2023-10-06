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
	"github.com/vearutop/photo-blog/internal/infra/service"
)

type getAlbumImagesDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoGpsFinder() uniq.Finder[photo.Gps]
	PhotoExifFinder() uniq.Finder[photo.Exif]
	ServiceSettings() service.Settings
	PhotoGpxFinder() uniq.Finder[photo.Gpx]
}

type getAlbumInput struct {
	Name   string `path:"name"`
	Locale string `cookie:"locale" default:"en-US"`
}

type Image struct {
	Name        string      `json:"name"`
	Hash        string      `json:"hash"`
	Width       int64       `json:"width"`
	Height      int64       `json:"height"`
	BlurHash    string      `json:"blur_hash,omitempty"`
	Gps         *photo.Gps  `json:"gps,omitempty"`
	Exif        *photo.Exif `json:"exif,omitempty"`
	Description string      `json:"description,omitempty"`
	size        int64
}

type track struct {
	Hash uniq.Hash `json:"hash"`
	photo.GpxSettings
}

type getAlbumOutput struct {
	Album       photo.Album `json:"album"`
	Description string      `json:"description,omitempty"`
	Images      []Image     `json:"images,omitempty"`
	Tracks      []track     `json:"tracks,omitempty"`
}

// GetAlbumContents creates use case interactor to get album data.
func GetAlbumContents(deps getAlbumImagesDeps) usecase.IOInteractorOf[getAlbumInput, getAlbumOutput] {
	u := usecase.NewInteractor(func(ctx context.Context, in getAlbumInput, out *getAlbumOutput) (err error) {
		deps.StatsTracker().Add(ctx, "get_album_images", 1)
		deps.CtxdLogger().Info(ctx, "getting album images", "name", in.Name)

		*out, err = getAlbumContents(ctx, deps, in.Name, false)

		return err
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func getAlbumContents(ctx context.Context, deps getAlbumImagesDeps, name string, preview bool) (out getAlbumOutput, err error) {
	albumHash := photo.AlbumHash(name)

	album, err := deps.PhotoAlbumFinder().FindByHash(ctx, albumHash)
	if err != nil {
		return out, err
	}

	var images []photo.Image

	if preview {
		images, err = deps.PhotoAlbumImageFinder().FindPreviewImages(ctx, albumHash, album.CoverImage, 4)
		if err != nil {
			return out, err
		}
	} else {
		images, err = deps.PhotoAlbumImageFinder().FindImages(ctx, albumHash)
		if err != nil {
			return out, err
		}
	}

	for _, h := range album.Settings.GpxTracksHashes {
		gpx, err := deps.PhotoGpxFinder().FindByHash(ctx, h)
		if err != nil {
			return out, err
		}

		s := gpx.Settings.Val

		if s.Name == "" {
			s.Name = path.Base(gpx.Path)
		}

		out.Tracks = append(out.Tracks, track{
			Hash:        h,
			GpxSettings: s,
		})
	}

	out.Album = album
	out.Images = make([]Image, 0, len(images))
	for _, i := range images {
		img := Image{
			Name:     path.Base(i.Path),
			Hash:     i.Hash.String(),
			Width:    i.Width,
			Height:   i.Height,
			BlurHash: i.BlurHash,
			size:     i.Size,
		}

		// Skip unprocessed images.
		if i.Width == 0 {
			continue
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

	if album.Settings.NewestFirst {
		reverse(out.Images)
	}

	return out, nil
}

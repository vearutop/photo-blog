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
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/pkg/txt"
)

type getAlbumImagesDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoGpsFinder() uniq.Finder[photo.Gps]
	PhotoExifFinder() uniq.Finder[photo.Exif]
	Settings() settings.Values
	PhotoGpxFinder() uniq.Finder[photo.Gpx]

	service.TxtRendererProvider
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
	Is360Pano   bool        `json:"is_360_pano,omitempty"`
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

	var (
		images  []photo.Image
		privacy settings.Privacy
	)

	// Privacy settings are only enabled for guests.
	if !auth.IsAdmin(ctx) {
		privacy = deps.Settings().Privacy()
	}

	if preview {
		images, err = deps.PhotoAlbumImageFinder().FindPreviewImages(ctx, albumHash, album.CoverImage, 4)
		if err != nil {
			return out, err
		}

		album.Settings = photo.AlbumSettings{}
	} else {
		images, err = deps.PhotoAlbumImageFinder().FindImages(ctx, albumHash)
		if err != nil {
			return out, err
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

		for i, t := range album.Settings.Texts {
			t.Text, err = deps.TxtRenderer().RenderLang(ctx, t.Text)
			if err != nil {
				return out, err
			}

			album.Settings.Texts[i] = t
		}

		album.Settings.Description, err = deps.TxtRenderer().RenderLang(ctx, album.Settings.Description)
		if err != nil {
			return out, err
		}
	}

	album.Title, err = deps.TxtRenderer().RenderLang(ctx, album.Title, func(o *txt.RenderOptions) {
		o.StripTags = true
	})
	if err != nil {
		return out, err
	}

	out.Album = album
	out.Images = make([]Image, 0, len(images))
	for _, i := range images {
		img := Image{
			Name:        path.Base(i.Path),
			Hash:        i.Hash.String(),
			Width:       i.Width,
			Height:      i.Height,
			BlurHash:    i.BlurHash,
			Description: deps.TxtRenderer().MustRenderLang(ctx, i.Settings.Description),
			size:        i.Size,
		}

		// Skip unprocessed images.
		if i.BlurHash == "" {
			continue
		}

		if !preview {
			if !privacy.HideGeoPosition {
				gps, err := deps.PhotoGpsFinder().FindByHash(ctx, i.Hash)
				if err == nil {
					img.Gps = &gps
				} else if !errors.Is(err, status.NotFound) {
					deps.CtxdLogger().Warn(ctx, "failed to find gps",
						"hash", i.Hash.String(), "error", err.Error())
				}
			}

			exif, err := deps.PhotoExifFinder().FindByHash(ctx, i.Hash)
			if err == nil {
				if !privacy.HideTechDetails {
					img.Exif = &exif
				}

				img.Is360Pano = exif.ProjectionType == "equirectangular"
			} else if !errors.Is(err, status.NotFound) {
				deps.CtxdLogger().Warn(ctx, "failed to find exif",
					"hash", i.Hash.String(), "error", err.Error())
			}
		}

		out.Images = append(out.Images, img)
	}

	if album.Settings.NewestFirst {
		reverse(out.Images)
	}

	return out, nil
}

func getAlbumContents2(ctx context.Context, deps getAlbumImagesDeps, name string, preview bool) (out getAlbumOutput, err error) {
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
	hashes := make([]uniq.Hash, 0, len(images))
	imgByHash := make(map[uniq.Hash]Image, len(images))

	for _, i := range images {
		// Skip unprocessed images.
		if i.Width == 0 {
			continue
		}

		img := Image{
			Name:        path.Base(i.Path),
			Hash:        i.Hash.String(),
			Width:       i.Width,
			Height:      i.Height,
			BlurHash:    i.BlurHash,
			Description: deps.TxtRenderer().MustRenderLang(ctx, i.Settings.Description),
			size:        i.Size,
		}

		hashes = append(hashes, i.Hash)
		imgByHash[i.Hash] = img
	}

	if !preview {
		gps, err := deps.PhotoGpsFinder().FindByHashes(ctx, hashes...)
		if err == nil {
			for _, v := range gps {
				img := imgByHash[v.Hash]
				img.Gps = &v
				imgByHash[v.Hash] = img
			}
		} else if !errors.Is(err, status.NotFound) {
			deps.CtxdLogger().Warn(ctx, "failed to find gps",
				"hashes", hashes, "error", err.Error())
		}

		exif, err := deps.PhotoExifFinder().FindByHashes(ctx, hashes...)
		if err == nil {
			for _, v := range exif {
				img := imgByHash[v.Hash]
				img.Exif = &v
				imgByHash[v.Hash] = img
			}
		} else if !errors.Is(err, status.NotFound) {
			deps.CtxdLogger().Warn(ctx, "failed to find exif",
				"hashes", hashes, "error", err.Error())
		}
	}

	for _, h := range hashes {
		out.Images = append(out.Images, imgByHash[h])
	}

	if album.Settings.NewestFirst {
		reverse(out.Images)
	}

	return out, nil
}

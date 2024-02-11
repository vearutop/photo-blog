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
	Album        photo.Album `json:"album"`
	Description  string      `json:"description,omitempty"`
	Images       []Image     `json:"images,omitempty"`
	Tracks       []track     `json:"tracks,omitempty"`
	HideOriginal bool        `json:"hide_original"`
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

	out.HideOriginal = privacy.HideOriginal

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

		gpxs, err := deps.PhotoGpxFinder().FindByHashes(ctx, album.Settings.GpxTracksHashes...)
		if err != nil && !errors.Is(err, status.NotFound) {
			return out, err
		}

		for _, gpx := range gpxs {
			s := gpx.Settings.Val

			if s.Name == "" {
				s.Name = path.Base(gpx.Path)
			}

			out.Tracks = append(out.Tracks, track{
				Hash:        gpx.Hash,
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

	imageHashes := make([]uniq.Hash, 0, len(images))
	for _, i := range images {
		// Skip unprocessed images.
		if i.BlurHash == "" {
			continue
		}

		imageHashes = append(imageHashes, i.Hash)
	}

	gpsData := map[uniq.Hash]photo.Gps{}
	exifData := map[uniq.Hash]photo.Exif{}

	if !preview {
		if !privacy.HideGeoPosition {
			gpss, err := deps.PhotoGpsFinder().FindByHashes(ctx, imageHashes...)
			if err != nil && !errors.Is(err, status.NotFound) {
				return out, err
			}

			for _, gps := range gpss {
				gpsData[gps.Hash] = gps
			}
		}

		exifs, err := deps.PhotoExifFinder().FindByHashes(ctx, imageHashes...)
		if err != nil && !errors.Is(err, status.NotFound) {
			return out, err
		}

		for _, exif := range exifs {
			exifData[exif.Hash] = exif
		}
	}

	for _, i := range images {
		// Skip unprocessed images.
		if i.BlurHash == "" {
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

		if !preview {
			if !privacy.HideGeoPosition {
				if gps, ok := gpsData[i.Hash]; ok {
					img.Gps = &gps
				}
			}

			if exif, ok := exifData[i.Hash]; ok {
				if !privacy.HideTechDetails {
					img.Exif = &exif
				}

				img.Is360Pano = exif.ProjectionType == "equirectangular"
			}
		}

		out.Images = append(out.Images, img)
	}

	if album.Settings.NewestFirst {
		reverse(out.Images)
	}

	return out, nil
}

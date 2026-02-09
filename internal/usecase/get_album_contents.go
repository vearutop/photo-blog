package usecase

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/storage"
	"github.com/vearutop/photo-blog/internal/infra/storage/visitor"
	"github.com/vearutop/photo-blog/pkg/txt"
)

type getAlbumImagesDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoGpsFinder() uniq.Finder[photo.Gps]
	PhotoExifFinder() uniq.Finder[photo.Exif]
	PhotoMetaFinder() uniq.Finder[photo.Meta]
	Settings() settings.Values
	PhotoGpxFinder() uniq.Finder[photo.Gpx]
	VisitorStats() *visitor.StatsRepository
	FavoriteRepository() *storage.FavoriteRepository
	DepCache() *dep.Cache
	ImageSelector() *storage.ImageSelector

	service.TxtRendererProvider
}

type getAlbumInput struct {
	Name   string `path:"name"`
	Locale string `cookie:"locale" default:"en-US"`
}

type Image struct {
	Name        string          `json:"name"`
	Hash        string          `json:"hash"`
	Width       int64           `json:"width"`
	Height      int64           `json:"height"`
	BlurHash    string          `json:"blur_hash,omitempty"`
	Gps         *photo.Gps      `json:"gps,omitempty"`
	Exif        *photo.Exif     `json:"exif,omitempty"`
	Description string          `json:"description,omitempty"`
	Is360Pano   bool            `json:"is_360_pano,omitempty"`
	Size        int64           `json:"size,omitempty"`
	Time        time.Time       `json:"time"`
	Meta        *photo.MetaData `json:"meta,omitempty"`
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

		*out, err = getAlbumContents(ctx, deps, imagesFilter{albumName: in.Name}, false)

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

type imagesFilter struct {
	albumName string
	search    string
	lens      string
	camera    string
	list      []uniq.Hash
}

func getAlbumContents(ctx context.Context, deps getAlbumImagesDeps, filter imagesFilter, preview bool) (out getAlbumOutput, err error) {
	return buildAlbumContents(ctx, deps, filter, preview)
}

func buildAlbumContents(ctx context.Context, deps getAlbumImagesDeps, filter imagesFilter, preview bool) (out getAlbumOutput, err error) {
	name := filter.albumName
	albumHash := photo.AlbumHash(filter.albumName)

	var (
		album   photo.Album
		images  []photo.Image
		isAdmin = auth.IsAdmin(ctx)
		query   string
	)

	if strings.HasPrefix(name, "list-") {
		l := strings.TrimPrefix(name, "list-")
		album.Title = "List"
		album.Name = name

		name = "list"
		ll := strings.Split(l, ",")
		hashes := make([]uniq.Hash, 0, len(ll))
		for _, l := range ll {
			var h uniq.Hash

			if err := h.UnmarshalText([]byte(l)); err != nil {
				return getAlbumOutput{}, fmt.Errorf("decode hash: %w", err)
			}

			hashes = append(hashes, h)
		}
		images, err = deps.PhotoImageFinder().FindByHashes(ctx, hashes...)
	}

	if strings.HasPrefix(name, "search:") {
		query = strings.TrimPrefix(name, "search:")
		name = "search"
	}

	switch name {
	case "list":

	case photo.Favorite:
		visitorHash := auth.VisitorFromContext(ctx)
		if visitorHash == 0 {
			return out, status.PermissionDenied
		}

		album.Title = "Favorite Photos"
		album.Name = photo.Favorite
		images, err = deps.FavoriteRepository().FindImages(ctx, visitorHash)

	case "search":
		if !auth.IsAdmin(ctx) {
			return out, status.PermissionDenied
		}
		album.Title = query
		album.Name = "search"
		images, err = deps.PhotoAlbumImageFinder().SearchImages(ctx, query)

	case photo.Orphan:
		if !auth.IsAdmin(ctx) {
			return out, status.PermissionDenied
		}

		album.Title = "Orphan Photos"
		album.Name = photo.Orphan
		images, err = deps.PhotoAlbumImageFinder().FindOrphanImages(ctx)
	case photo.Broken:
		if !isAdmin {
			return out, status.PermissionDenied
		}

		album.Title = "Broken Photos"
		album.Name = photo.Broken
		images, err = deps.PhotoAlbumImageFinder().FindBrokenImages(ctx)
	default:
		album, err = deps.PhotoAlbumFinder().FindByHash(ctx, albumHash)
		if err != nil {
			return out, err
		}
		if preview {
			album.Settings = photo.AlbumSettings{}
			images, err = deps.PhotoAlbumImageFinder().FindPreviewImages(ctx, albumHash, album.CoverImage, 4)
		} else {
			images, err = deps.PhotoAlbumImageFinder().FindImages(ctx, albumHash)
		}
	}

	if err != nil {
		return out, err
	}

	out.Album = album

	if err := out.prepare(ctx, deps, images, preview); err != nil {
		return out, err
	}

	return out, nil
}

func (out *getAlbumOutput) prepare(ctx context.Context, deps getAlbumImagesDeps, images []photo.Image, preview bool) error {
	out.Images = make([]Image, 0, len(images))
	album := out.Album
	albumSettings := album.Settings
	isAdmin := auth.IsAdmin(ctx)

	var privacy settings.Privacy

	// Privacy settings are only enabled for guests.
	if !isAdmin {
		privacy = deps.Settings().Privacy()
	}

	out.HideOriginal = privacy.HideOriginal
	imageHashes := make([]uniq.Hash, 0, len(images))

	for _, i := range images {
		// Skip unprocessed images.
		if i.BlurHash == "" {
			continue
		}

		imageHashes = append(imageHashes, i.Hash)
	}

	var (
		gpsData  = map[uniq.Hash]photo.Gps{}
		exifData = map[uniq.Hash]photo.Exif{}
		metaData = map[uniq.Hash]photo.Meta{}
		imgAlbum map[uniq.Hash][]photo.Album
	)

	if !preview {
		if !privacy.HideGeoPosition {
			gpss, err := deps.PhotoGpsFinder().FindByHashes(ctx, imageHashes...)
			if err != nil && !errors.Is(err, status.NotFound) {
				return err
			}

			for _, gps := range gpss {
				gpsData[gps.Hash] = gps
			}
		}

		exifs, err := deps.PhotoExifFinder().FindByHashes(ctx, imageHashes...)
		if err != nil && !errors.Is(err, status.NotFound) {
			return err
		}

		for _, exif := range exifs {
			exifData[exif.Hash] = exif
		}

		metas, err := deps.PhotoMetaFinder().FindByHashes(ctx, imageHashes...)
		if err != nil && !errors.Is(err, status.NotFound) {
			return err
		}

		for _, meta := range metas {
			metaData[meta.Hash] = meta
		}

		imgAlbum, err = deps.PhotoAlbumImageFinder().FindImageAlbums(ctx, album.Hash, imageHashes...)
		if err != nil {
			return err
		}
	}

	textReplaces := append(deps.Settings().Appearance().TextReplaces, albumSettings.TextReplaces...)

	for _, i := range images {
		// Skip unprocessed images.
		if !i.Ready() {
			continue
		}

		h := i.Hash.String()

		img := Image{
			Name:        strings.TrimSuffix(path.Base(i.Path), "."+h+".jpg"),
			Hash:        h,
			Width:       i.Width,
			Height:      i.Height,
			BlurHash:    i.BlurHash,
			Description: deps.TxtRenderer().MustRenderLang(ctx, i.Settings.Description, textReplaces.Apply),
			Size:        i.Size,
			Time:        i.Time(),
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

			if meta, ok := metaData[i.Hash]; ok {
				img.Meta = &meta.Data.Val
			}

			if albums, ok := imgAlbum[i.Hash]; ok {
				links := ""

				for _, a := range albums {
					if !a.Public && !isAdmin {
						continue
					}

					links += `<br/><a href="/` + a.Name + `"><span class="icon-link film-icon"></span>` +
						deps.TxtRenderer().MustRenderLang(ctx, a.Title, txt.StripTags, textReplaces.Apply) + `</a>`
				}

				if links != "" {
					img.Description += links
				}
			}
		}

		out.Images = append(out.Images, img)
	}

	if albumSettings.NewestFirst {
		reverse(out.Images)
	}

	if albumSettings.DailyRulers {
		dateShift := -time.Second
		if albumSettings.NewestFirst {
			dateShift = time.Second
		}

		prevDate := ""

		for _, i := range out.Images {
			d := i.Time.Format(time.DateOnly)
			if d != prevDate {
				albumSettings.Texts = append(albumSettings.Texts, txt.Chronological{
					Time: i.Time.Add(dateShift),
					Text: "### " + d + "\n",
				})

				prevDate = d
			}
		}
	}

	if !preview {
		gpxs, err := deps.PhotoGpxFinder().FindByHashes(ctx, albumSettings.GpxTracksHashes...)
		if err != nil && !errors.Is(err, status.NotFound) {
			return err
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

		for i, t := range albumSettings.Texts {
			t.Text, err = deps.TxtRenderer().RenderLang(ctx, t.Text, textReplaces.Apply)
			if err != nil {
				return err
			}

			albumSettings.Texts[i] = t
		}

		albumSettings.Description, err = deps.TxtRenderer().RenderLang(ctx, albumSettings.Description, textReplaces.Apply)
		if err != nil {
			return err
		}
	}

	var err error

	album.Title, err = deps.TxtRenderer().RenderLang(ctx, album.Title, txt.StripTags, textReplaces.Apply)
	if err != nil {
		return err
	}

	album.Settings = albumSettings

	out.Album = album

	return nil
}

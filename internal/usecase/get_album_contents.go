package usecase

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"math"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/image/sprite"
	"github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/storage"
	"github.com/vearutop/photo-blog/internal/infra/storage/visitor"
	"github.com/vearutop/photo-blog/pkg/txt"

	"github.com/vearutop/gpxt/geotag"
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
	PersistentCacheStorage() *sqluct.Storage
	AlbumSprites() *sprite.Service

	service.TxtRendererProvider
}

type getAlbumInput struct {
	Name   string `path:"name"`
	Locale string `cookie:"locale" default:"en-US"`
}

type Image struct {
	Name            string          `json:"name"`
	Hash            string          `json:"hash"`
	Width           int64           `json:"width"`
	Height          int64           `json:"height"`
	BlurHash        string          `json:"blur_hash,omitempty"`
	Gps             *photo.Gps      `json:"gps,omitempty"`
	Exif            *photo.Exif     `json:"exif,omitempty"`
	Description     string          `json:"description,omitempty"`
	DescriptionHTML template.HTML   `json:"description_html,omitempty"`
	AISays          string          `json:"ai_says,omitempty"`
	Is360Pano       bool            `json:"is_360_pano,omitempty"`
	Size            int64           `json:"size,omitempty"`
	UTime           int64           `json:"utime"`
	Meta            *photo.MetaData `json:"meta,omitempty"`
}

type track struct {
	Hash uniq.Hash `json:"hash"`
	photo.GpxSettings
}

type getAlbumOutput struct {
	Album         photo.Album                 `json:"album"`
	Description   string                      `json:"description,omitempty"`
	Images        []Image                     `json:"images,omitempty"`
	Tracks        []track                     `json:"tracks,omitempty"`
	ThumbSprites  map[string]*sprite.ViewItem `json:"thumb_sprites,omitempty"`
	MarkerSprites map[string]*sprite.ViewItem `json:"marker_sprites,omitempty"`
	SpriteSheets  map[string]sprite.Sheet     `json:"sprite_sheets,omitempty"`
	HideOriginal  bool                        `json:"hide_original"`
	SkipSprites   bool                        `json:"skip_sprites,omitempty"`

	// Page is the resolved page (chrono-marker split) shown in Images. "" means the unnamed page —
	// not yet opened (oldest-first) or not yet closed out (newest-first); see pageForTime.
	Page string `json:"page,omitempty"`
	// Pages lists every page that has at least one image, in chronological order.
	Pages []string `json:"pages,omitempty"`

	// Timeline is Images and the rendered chrono texts merged into final display order — see
	// buildAlbumTimeline. A client-side renderer should walk this instead of re-deriving order
	// from Images/Album.Settings.Texts, so ordering logic lives in exactly one place.
	Timeline []albumTimelineItem `json:"timeline,omitempty"`
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

func parseListHashes(name string) ([]uniq.Hash, error) {
	l := strings.TrimPrefix(name, "list-")
	ll := strings.Split(l, ",")
	hashes := make([]uniq.Hash, 0, len(ll))

	for _, l := range ll {
		var h uniq.Hash

		if err := h.UnmarshalText([]byte(l)); err != nil {
			return nil, fmt.Errorf("decode hash: %w", err)
		}

		hashes = append(hashes, h)
	}

	return hashes, nil
}

type imagesFilter struct {
	albumName string
	search    string
	lens      string
	camera    string
	list      []uniq.Hash

	// page requests one page (chrono-marker split) of a real album; "" means "use the default page".
	page string
	// pageResolved marks page as already resolved (e.g. from a photo hash permalink), skipping
	// default-page resolution and page-name validation.
	pageResolved bool
}

// pageMarkerRe matches a "[page:NAME]" directive as the first line of a chrono text.
var pageMarkerRe = regexp.MustCompile(`^\s*\[page:([^\]\n]+)\][ \t]*\r?\n?`)

// splitPageMarker extracts a leading "[page:NAME]" directive from a chrono text, returning the
// page name it starts ("" if none) and the remaining text with the directive line removed.
func splitPageMarker(text string) (page, rest string) {
	m := pageMarkerRe.FindStringSubmatchIndex(text)
	if m == nil {
		return "", text
	}

	return strings.TrimSpace(text[m[2]:m[3]]), text[m[1]:]
}

// pageBoundary marks where a named album page begins.
type pageBoundary struct {
	time time.Time
	name string
}

// pageBoundaries extracts page-split markers from an album's chrono texts, sorted ascending by time.
// "[page:*]" is a wildcard: it marks a chrono text as shown on every page and never starts one.
func pageBoundaries(texts []txt.Chronological) []pageBoundary {
	var bounds []pageBoundary

	for _, t := range texts {
		name, _ := splitPageMarker(t.Text)
		if name == "" || name == "*" {
			continue
		}

		bounds = append(bounds, pageBoundary{time: t.Time, name: name})
	}

	sort.Slice(bounds, func(i, j int) bool { return bounds[i].time.Before(bounds[j].time) })

	return bounds
}

// displayBefore is the single source of truth for what an album's display order means: whether
// moment a is shown before moment b. Every place that cares about NewestFirst — page splitting,
// timeline interleaving, finding the "first" image — goes through this one function, so a new
// ordering scheme (or a fix to this one) only has to change in one place.
func displayBefore(a, b time.Time, newestFirst bool) bool {
	if newestFirst {
		return a.After(b)
	}

	return a.Before(b)
}

// displayReached reports whether, by the time display order arrives at moment t, a marker/text
// timestamped at bt has already been passed.
func displayReached(t, bt time.Time, newestFirst bool) bool {
	return !displayBefore(t, bt, newestFirst)
}

// pageForTime resolves which page a moment belongs to. A "[page:NAME]" marker is a pure split
// point — whether it opens the page that follows or closes the page that precedes depends on
// which way the album reads, because that's genuinely which side you haven't gotten to yet:
//
//   - Oldest-first: a marker OPENS the page that follows it going forward in time, like dating the
//     first entry of a new chapter. A moment belongs to the latest marker at or before it; "" is
//     everything before the first marker (not opened yet).
//   - Newest-first: a marker CLOSES the page that precedes it, like stamping "this is 2025" on New
//     Year's Eve for the year that just ended. A moment belongs to the earliest marker at or after
//     it; "" is everything after the last marker (not closed out yet — the current page).
//
// Both read the same way through displayReached: a boundary applies once display order has reached
// it, and bounds is sorted ascending, so oldest-first keeps the latest reached boundary while
// newest-first (walking conceptually from the end) stops at the first one.
func pageForTime(bounds []pageBoundary, t time.Time, newestFirst bool) string {
	if newestFirst {
		for _, b := range bounds {
			if displayReached(t, b.time, newestFirst) {
				return b.name
			}
		}

		return ""
	}

	page := ""

	for _, b := range bounds {
		if !displayReached(t, b.time, newestFirst) {
			break
		}

		page = b.name
	}

	return page
}

func pageForUTime(bounds []pageBoundary, utime int64, newestFirst bool) string {
	return pageForTime(bounds, time.Unix(utime, 0), newestFirst)
}

// defaultPage picks the page containing the first image in display order (respecting newestFirst) —
// what a bare album URL should show.
func defaultPage(images []photo.Image, newestFirst bool, bounds []pageBoundary) string {
	var target *photo.Image

	for i := range images {
		if images[i].BlurHash == "" {
			continue // unprocessed, never displayed
		}

		if target == nil || displayBefore(time.Unix(images[i].UTime, 0), time.Unix(target.UTime, 0), newestFirst) {
			target = &images[i]
		}
	}

	if target == nil {
		return ""
	}

	return pageForUTime(bounds, target.UTime, newestFirst)
}

// filterImagesByPage keeps only the images whose timestamp falls on the given page.
func filterImagesByPage(images []photo.Image, bounds []pageBoundary, page string, newestFirst bool) []photo.Image {
	filtered := images[:0]

	for _, img := range images {
		if pageForUTime(bounds, img.UTime, newestFirst) == page {
			filtered = append(filtered, img)
		}
	}

	return filtered
}

// filterPageTexts strips "[page:NAME]" directives from chrono texts and keeps only the entries
// belonging to the given page. A marker's own trailing text (if any) becomes a normal chrono text
// like any other, positioned in the timeline by its own real timestamp — a marker only decides
// page membership, not display position.
func filterPageTexts(texts []txt.Chronological, page string, newestFirst bool) []txt.Chronological {
	bounds := pageBoundaries(texts)

	filtered := texts[:0]

	for _, t := range texts {
		name, rest := splitPageMarker(t.Text)

		if name != "*" && pageForUTime(bounds, t.Time.Unix(), newestFirst) != page {
			continue
		}

		if name != "" && strings.TrimSpace(rest) == "" {
			continue
		}

		t.Text = rest
		filtered = append(filtered, t)
	}

	return filtered
}

func getAlbumContents(ctx context.Context, deps getAlbumImagesDeps, filter imagesFilter, preview bool) (out getAlbumOutput, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get album output: %w", err)
		}
	}()

	name := filter.albumName
	albumHash := photo.AlbumHash(filter.albumName)

	var (
		album   photo.Album
		images  []photo.Image
		isAdmin = auth.IsAdmin(ctx)
		query   string
	)

	if strings.HasPrefix(name, "list-") {
		album.Title = "List"
		album.Name = name
		out.SkipSprites = true

		var hashes []uniq.Hash

		hashes, err = parseListHashes(name)
		if err != nil {
			return getAlbumOutput{}, err
		}

		name = "list"
		images, err = deps.PhotoImageFinder().FindByHashes(ctx, hashes...)
	}

	if strings.HasPrefix(name, "search:") {
		query = strings.TrimPrefix(name, "search:")
		name = "search"
		out.SkipSprites = true
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
		out.SkipSprites = true

	case "search":
		if !auth.IsAdmin(ctx) {
			return out, status.PermissionDenied
		}
		album.Title = query
		album.Name = "search"
		images, err = deps.PhotoAlbumImageFinder().SearchImages(ctx, query)
		out.SkipSprites = true

	case photo.Orphan:
		if !auth.IsAdmin(ctx) {
			return out, status.PermissionDenied
		}

		album.Title = "Orphan Photos"
		album.Name = photo.Orphan
		images, err = deps.PhotoAlbumImageFinder().FindOrphanImages(ctx)
		out.SkipSprites = true

	case photo.Broken:
		if !isAdmin {
			return out, status.PermissionDenied
		}

		album.Title = "Broken Photos"
		album.Name = photo.Broken
		images, err = deps.PhotoAlbumImageFinder().FindBrokenImages(ctx)
		out.SkipSprites = true

	default:
		album, err = deps.PhotoAlbumFinder().FindByHash(ctx, albumHash)
		if err != nil {
			return out, err
		}

		if preview {
			album.Settings = photo.AlbumSettings{}
			images, err = deps.PhotoAlbumImageFinder().FindPreviewImages(ctx, albumHash, album.CoverImage, 4)
		} else {
			images, err = deps.ImageSelector().Select().ByAlbumName(album.Name).Find(ctx)
			// images, err = deps.PhotoAlbumImageFinder().FindImages(ctx, albumHash)
		}
	}

	if err != nil {
		return out, err
	}

	bounds := pageBoundaries(album.Settings.Texts)

	if len(bounds) > 0 {
		seen := make(map[string]bool, len(bounds)+1)

		for _, img := range images {
			if img.BlurHash == "" {
				continue
			}

			seen[pageForUTime(bounds, img.UTime, album.Settings.NewestFirst)] = true
		}

		for _, b := range bounds {
			if seen[b.name] {
				out.Pages = append(out.Pages, b.name)
			}
		}

		if seen[""] {
			out.Pages = append(out.Pages, "")
		}
	}

	switch {
	case filter.pageResolved:
		images = filterImagesByPage(images, bounds, filter.page, album.Settings.NewestFirst)
		out.Page = filter.page

	case filter.page != "":
		if !slices.Contains(out.Pages, filter.page) {
			return out, status.NotFound
		}

		images = filterImagesByPage(images, bounds, filter.page, album.Settings.NewestFirst)
		out.Page = filter.page

	case len(bounds) > 0:
		page := defaultPage(images, album.Settings.NewestFirst, bounds)
		images = filterImagesByPage(images, bounds, page, album.Settings.NewestFirst)
		out.Page = page
	}

	out.Album = album

	if err := out.prepare(ctx, deps, images, preview); err != nil {
		return out, fmt.Errorf("prepare album: %w", err)
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

	var geoIdx *geotag.Index

	// Prepare GeoTag index.
	if len(album.Settings.GpxTracksHashes) > 0 {
		geoIdx = geotag.NewIndex(geotag.Options{
			Interpolate: true,
		})

		gpxFiles, err := deps.PhotoGpxFinder().FindByHashes(ctx, album.Settings.GpxTracksHashes...)
		if err != nil {
			return fmt.Errorf("find gpx tracks: %w", err)
		}

		for _, gpx := range gpxFiles {
			g, err := gpx.Load()
			if err != nil {
				return fmt.Errorf("load gpx: %w", err)
			}

			geoIdx.AddGPX(g)
		}

		geoIdx.Build()
	}

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
			UTime:       i.UTime,
		}

		if !preview {
			if !privacy.HideGeoPosition {
				if gps, ok := gpsData[i.Hash]; ok {
					img.Gps = &gps
				} else if geoIdx != nil && i.TakenAt != nil {
					if p, ok := geoIdx.Lookup(*i.TakenAt); ok {
						var gps photo.Gps

						gps.Hash = i.Hash
						gps.Latitude = float64(p.Lat)
						gps.Longitude = float64(p.Lon)
						gps.GpsTime = *i.TakenAt

						if !math.IsNaN(float64(p.Alt)) {
							gps.Altitude = float64(p.Alt)
						}

						img.Gps = &gps
					}
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

		if img.Description != "" {
			img.DescriptionHTML = template.HTML(img.Description)
		}

		if img.Meta != nil {
			if !albumSettings.HideAISays {
				img.AISays = buildAISays(img.Meta)
			}

			img.Meta.ImageClassification = nil
			img.Meta.Faces = nil
		}

		out.Images = append(out.Images, img)
	}

	sort.SliceStable(out.Images, func(i, j int) bool {
		return displayBefore(time.Unix(out.Images[i].UTime, 0), time.Unix(out.Images[j].UTime, 0), albumSettings.NewestFirst)
	})

	albumSettings.Texts = filterPageTexts(albumSettings.Texts, out.Page, albumSettings.NewestFirst)

	if albumSettings.DailyRulers {
		dateShift := -time.Second
		if albumSettings.NewestFirst {
			dateShift = time.Second
		}

		prevDate := ""

		for _, i := range out.Images {
			if i.Is360Pano {
				continue
			}

			d := time.Unix(i.UTime, 0).Format(time.DateOnly)
			if d != prevDate {
				albumSettings.Texts = append(albumSettings.Texts, txt.Chronological{
					Time: time.Unix(i.UTime, 0).Add(dateShift),
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
	out.Timeline = buildAlbumTimeline(out.Images, albumSettings.Texts, albumSettings.NewestFirst)

	return nil
}

package usecase

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/bool64/cache"
	"github.com/bool64/ctxd"
	"github.com/docker/go-units"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/image/sprite"
	infraService "github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/pkg/txt"
)

// albumCacheVersion must be bumped whenever getAlbumOutput or albumPageData's JSON shape changes
// in a way that isn't safe for an old cached blob to deserialize into (i.e. anything but adding a
// new field with omitempty that every reader tolerates being zero-value). album-data/album-page
// are FailoverOf caches: they keep serving a stale blob past its TTL while refreshing in the
// background, so an incompatible old shape doesn't error — it silently deserializes into zero
// values, which is exactly what happened twice while pagination was being built (Image and
// Timeline). Bumping this orphans every previously-cached entry at once instead of chasing each
// field that turns out to depend on one.
//
// InvalidatePersistentCache's enum tag can't reference this constant (Go struct tags must be
// literal strings), so its "album-data-N"/"album-page-N" entries must be updated by hand to match
// whenever this changes.
const albumCacheVersion = "2"

// AlbumDataCacheName and AlbumPageCacheName are the persistent cache names for album content and
// rendered album pages. Exported so InvalidatePersistentCache's allow-list can reference the same
// values rather than duplicating the version number.
const (
	AlbumDataCacheName = "album-data-" + albumCacheVersion
	AlbumPageCacheName = "album-page-" + albumCacheVersion
)

type AlbumPageBuilder struct {
	deps               getAlbumImagesDeps
	albumDataCache     *cache.FailoverOf[getAlbumOutput]
	albumDataCacheName string
	albumPageCache     *cache.FailoverOf[albumPageData]
	albumPageCacheName string
}

func NewAlbumPageBuilder(deps getAlbumImagesDeps) *AlbumPageBuilder {
	albumDataCacheName := AlbumDataCacheName
	albumPageCacheName := AlbumPageCacheName

	return &AlbumPageBuilder{
		deps:               deps,
		albumDataCache:     infraService.MakePersistentCacheOf[getAlbumOutput](deps, albumDataCacheName, 30*time.Hour),
		albumDataCacheName: albumDataCacheName,
		albumPageCache:     infraService.MakePersistentCacheOf[albumPageData](deps, albumPageCacheName, 30*time.Hour),
		albumPageCacheName: albumPageCacheName,
	}
}

func (pb *AlbumPageBuilder) getCachedAlbum(ctx context.Context, name string, preview bool, page string, pageResolved bool) (getAlbumOutput, error) {
	cacheKey := []byte(name + "/" + strconv.FormatBool(auth.IsAdmin(ctx)) + "/" + txt.Language(ctx) + "/" +
		strconv.FormatBool(preview) + "/" + page)
	cacheName := pb.albumDataCacheName
	cacheMiss := false

	d, err := pb.albumDataCache.Get(ctx, cacheKey, func(ctx context.Context) (getAlbumOutput, error) {
		cacheMiss = true

		return getAlbumContents(ctx, pb.deps, imagesFilter{albumName: name, page: page, pageResolved: pageResolved}, preview)
	})
	if err != nil {
		return getAlbumOutput{}, fmt.Errorf("cache get: %w, cache miss: %v", err, cacheMiss)
	}

	if cacheMiss {
		// Labels are reset right before (re)registering them, as close to the cache write as
		// possible, so a concurrent AlbumChanged invalidation during the (slow) build above
		// can't be lost by finding no labels to match against.
		if err := pb.deps.DepCache().ResetKey(ctx, cacheName, cacheKey); err != nil {
			return getAlbumOutput{}, fmt.Errorf("reset cache deps: %w", err)
		}

		pb.deps.DepCache().ServiceSettingsDependency(cacheName, cacheKey)
		pb.deps.DepCache().AlbumDependency(cacheName, cacheKey, name)
	}

	return d, nil
}

// resolveHashPage finds which page of albumName contains the image with the given hash, so
// permalinks like /{name}/photo-{hash}.html always open the page the photo actually lives on.
// ok is false when the album has no pages, or the hash/album can't be resolved.
func (pb *AlbumPageBuilder) resolveHashPage(ctx context.Context, albumName string, hash uniq.Hash) (page string, ok bool) {
	album, err := pb.deps.PhotoAlbumFinder().FindByHash(ctx, photo.AlbumHash(albumName))
	if err != nil {
		return "", false
	}

	bounds := pageBoundaries(album.Settings.Texts)
	if len(bounds) == 0 {
		return "", false
	}

	img, err := pb.deps.PhotoImageFinder().FindByHash(ctx, hash)
	if err != nil {
		return "", false
	}

	return pageForUTime(bounds, img.UTime, album.Settings.NewestFirst), true
}

func (pb *AlbumPageBuilder) addSprites(ctx context.Context, d *albumPageData) {
	deps := pb.deps
	album := d.AlbumData.Album

	imageSets := make([][]Image, 0, 1+len(d.SubAlbums))
	imageSets = append(imageSets, d.StrippedAlbumData.Images)
	for _, subAlbum := range d.SubAlbums {
		imageSets = append(imageSets, subAlbum.Images)
	}

	spriteImages := pb.mergeAlbumSpriteImages(imageSets...)

	manifestKey := deps.AlbumSprites().ManifestKey(spriteImages)
	d.SpriteManifestKey = manifestKey

	manifest, err := deps.AlbumSprites().ManifestReady(ctx, manifestKey)
	if err != nil {
		d.spritePendingManifestKey = manifestKey
		d.spritePendingImages = spriteImages
		deps.CtxdLogger().Info(ctx, "album sprite: album page sprite manifest pending",
			"album", album.Name,
			"manifest_key", manifestKey,
			"image_count", len(spriteImages),
			"has_sprite_sheets", false)
		if !errors.Is(err, cache.ErrNotFound) {
			deps.CtxdLogger().Error(ctx, "failed to get album sprite manifest", "album", album.Name, "error", err)
		}

		return
	}

	retirementKey, err := deps.AlbumSprites().TrackAlbum(ctx, spriteImages, album.Hash)
	if err != nil {
		deps.CtxdLogger().Error(ctx, "failed to track album sprite manifest", "album", album.Name, "error", err)
	} else {
		if err := deps.DepCache().ResetKey(ctx, sprite.RetirementCacheName, retirementKey); err != nil {
			deps.CtxdLogger().Error(ctx, "failed to reset album sprite retirement deps", "album", album.Name, "error", err)
		} else {
			labels := make([]string, 0, 1+len(d.SubAlbums))
			labels = append(labels, album.Name)
			for _, subAlbum := range d.SubAlbums {
				labels = append(labels, subAlbum.Album.Name)
			}

			deps.DepCache().AlbumDependency(sprite.RetirementCacheName, retirementKey, labels...)
		}
	}

	items := deps.AlbumSprites().View(manifest)
	d.ThumbSprites = pb.filterThumbSprites(items, d.StrippedAlbumData.Images)
	d.AlbumData.ThumbSprites = d.ThumbSprites
	d.StrippedAlbumData.ThumbSprites = d.ThumbSprites
	d.MarkerSprites = deps.AlbumSprites().MarkerView(manifest)
	d.SpriteSheets = deps.AlbumSprites().CompactSheets(items, d.MarkerSprites)
	d.AlbumData.MarkerSprites = d.MarkerSprites
	d.StrippedAlbumData.MarkerSprites = d.MarkerSprites
	d.AlbumData.SpriteSheets = d.SpriteSheets
	d.StrippedAlbumData.SpriteSheets = d.SpriteSheets

	for i := range d.SubAlbums {
		d.SubAlbums[i].ThumbSprites = pb.filterThumbSprites(items, d.SubAlbums[i].Images)
		d.SubAlbums[i].SpriteSheets = d.AlbumData.SpriteSheets
	}

	deps.CtxdLogger().Info(ctx, "album sprite: album page sprite manifest ready",
		"album", album.Name,
		"manifest_key", manifestKey,
		"image_count", len(spriteImages),
		"sprite_sheet_count", len(d.SpriteSheets),
		"pending_manifest_key", d.spritePendingManifestKey != "")
}

func (pb *AlbumPageBuilder) filterThumbSprites(items map[string]*sprite.ViewItem, images []Image) map[string]*sprite.ViewItem {
	if len(items) == 0 || len(images) == 0 {
		return nil
	}

	res := make(map[string]*sprite.ViewItem, len(images))
	for _, img := range images {
		if item, ok := items[img.Hash]; ok {
			res[img.Hash] = item
		}
	}

	if len(res) == 0 {
		return nil
	}

	return res
}

func (pb *AlbumPageBuilder) mergeAlbumSpriteImages(images ...[]Image) []sprite.Image {
	seen := make(map[string]int)
	merged := make([]sprite.Image, 0)

	for _, set := range images {
		for _, item := range pb.albumSpriteImages(set) {
			key := item.Hash.String()
			if idx, ok := seen[key]; ok {
				if item.HasGPS {
					merged[idx].HasGPS = true
				}

				continue
			}

			seen[key] = len(merged)
			merged = append(merged, item)
		}
	}

	return merged
}

func (pb *AlbumPageBuilder) albumSpriteImages(images []Image) []sprite.Image {
	spriteImages := make([]sprite.Image, 0, len(images))

	for _, img := range images {
		if img.Is360Pano {
			continue
		}

		var h uniq.Hash
		if err := h.UnmarshalText([]byte(img.Hash)); err == nil {
			spriteImages = append(spriteImages, sprite.Image{
				Hash:   h,
				Width:  img.Width,
				Height: img.Height,
				HasGPS: img.Gps != nil,
			})
		}
	}

	return spriteImages
}

func (pb *AlbumPageBuilder) cachedBuild(ctx context.Context, cont getAlbumOutput) (albumPageData, error) {
	cacheKey := []byte("page:" + cont.Album.Name + "/" + strconv.FormatBool(auth.IsAdmin(ctx)) + "/" +
		strconv.FormatBool(auth.IsBot(ctx)) + "/" + txt.Language(ctx) + "/" + cont.Page)
	cacheName := pb.albumPageCacheName
	cacheMiss := false

	d, err := pb.albumPageCache.Get(ctx, cacheKey, func(ctx context.Context) (albumPageData, error) {
		cacheMiss = true

		d, err := pb.build(ctx, cont)
		if err != nil {
			return albumPageData{}, err
		}

		return d, nil
	})
	if err != nil {
		return albumPageData{}, err
	}

	if cacheMiss {
		// Labels are reset right before (re)registering them, as close to the cache write as
		// possible, so a concurrent AlbumChanged invalidation during the (slow) build above
		// can't be lost by finding no labels to match against.
		if err := pb.deps.DepCache().ResetKey(ctx, cacheName, cacheKey); err != nil {
			return albumPageData{}, fmt.Errorf("reset cache deps: %w", err)
		}

		pb.deps.DepCache().ServiceSettingsDependency(cacheName, cacheKey)
		pb.deps.DepCache().AlbumDependency(cacheName, cacheKey, cont.Album.Name)

		if d.spritePendingManifestKey != "" {
			pb.deps.DepCache().SpriteManifestDependency(cacheName, cacheKey, d.spritePendingManifestKey)
		}

		for _, subAlbum := range d.SubAlbums {
			pb.deps.DepCache().AlbumDependency(cacheName, cacheKey, subAlbum.Album.Name)
		}
	}

	pb.deps.CtxdLogger().Info(ctx, "album sprite: album page cache result",
		"album", cont.Album.Name,
		"cache_name", cacheName,
		"cache_key", string(cacheKey),
		"cache_hit", !cacheMiss,
		"manifest_key", d.SpriteManifestKey,
		"manifest_pending", d.spritePendingManifestKey != "",
		"has_sprite_sheets", len(d.SpriteSheets) > 0,
		"sprite_sheet_count", len(d.SpriteSheets))

	if cacheMiss && d.spritePendingManifestKey != "" && len(d.spritePendingImages) > 0 {
		pb.startPendingSpriteBuild(ctx, cont.Album, d.spritePendingManifestKey, d.spritePendingImages)
	}

	return d, nil

}

func (pb *AlbumPageBuilder) build(ctx context.Context, cont getAlbumOutput) (albumPageData, error) {
	deps := pb.deps

	album := cont.Album

	d := albumPageData{}
	d.Title = album.Title

	// Description is only shown on the unnamed page ("" — see pageForTime).
	if cont.Page == "" {
		d.Description = template.HTML(album.Settings.Description)
	}

	d.Page = cont.Page
	d.Name = album.Name
	d.Public = album.Public
	d.Hash = album.Hash.String()
	d.Count = len(cont.Images)
	d.AlbumData = cont
	d.StrippedAlbumData = strippedAlbumData(cont)
	d.AlbumData.Album.Settings.CollabKey = ""
	d.StrippedAlbumData.Album.Settings.CollabKey = ""
	// Computed fresh rather than reused from cont.Timeline: cont comes from the album-data
	// persistent cache, a JSON blob that can predate whatever fields getAlbumOutput currently
	// has — trusting a cached field here silently breaks the page the moment that field didn't
	// exist yet when the entry was cached. Images/Texts are cheap to re-merge and don't have
	// this problem since they're not new.
	d.Timeline = buildAlbumTimeline(cont.Images, cont.Album.Settings.Texts, cont.Album.Settings.NewestFirst)
	d.JsRender = album.Settings.JsRender
	d.Featured = deps.Settings().Appearance().FeaturedAlbumName

	d.fill(ctx, deps.TxtRenderer(), deps.Settings())
	if len(cont.Images) > 1 {
		d.OGTitle = fmt.Sprintf("%s (%d photos)", album.Title, len(cont.Images))
	} else {
		d.OGTitle = album.Title
	}
	d.OGSiteName = deps.TxtRenderer().MustRenderLang(ctx, deps.Settings().Appearance().SiteTitle, func(o *txt.RenderOptions) {
		o.StripTags = true
	})

	d.ShowMap = !album.Settings.HideMap
	d.ShowEXIFPreview = album.Settings.ShowEXIFPreview
	d.ShowAISays = !album.Settings.HideAISays
	d.PreRender = true
	d.HasPanos = false
	d.HasPixelpeep = strings.Contains(string(d.Description), `class="pixelpeep`)

	for _, img := range cont.Images {
		if img.Is360Pano {
			d.HasPanos = true
		}

		if strings.Contains(string(img.DescriptionHTML), `class="pixelpeep`) {
			d.HasPixelpeep = true
		}
	}

	for _, item := range d.Timeline {
		if strings.Contains(string(item.Text), `class="pixelpeep`) {
			d.HasPixelpeep = true
		}
	}

	maps := deps.Settings().Maps()

	d.MapTiles = maps.Tiles
	if maps.Cache {
		d.MapTiles = "/map-tile/{s}/{r}/{z}/{x}/{y}.png"
	}

	if album.Settings.MapTiles != "" {
		d.MapTiles = album.Settings.MapTiles
	}

	d.MapAttribution = maps.Attribution
	if album.Settings.MapAttribution != "" {
		d.MapAttribution = album.Settings.MapAttribution
	}

	// TotalSize controls visibility of batch download button.
	privacy := deps.Settings().Privacy()
	if d.IsAdmin || (!privacy.HideOriginal && !privacy.HideBatchDownload && !album.Settings.HideDownload.True()) {
		var totalSize int64
		for _, img := range cont.Images {
			totalSize += img.Size
		}

		if totalSize > 0 {
			d.TotalSize = units.HumanSize(float64(totalSize))
		}
	}

	if deps.Settings().Visitors().Tag {
		d.EnableFavorite = true
	}

	switch {
	case album.CoverImage != 0:
		d.CoverImage = "/thumb/1200w/" + album.CoverImage.String() + ".jpg"
	case len(cont.Images) > 0:
		d.CoverImage = "/thumb/1200w/" + cont.Images[0].Hash + ".jpg"
	}

	for _, name := range album.Settings.SubAlbumNames {
		a, err := deps.PhotoAlbumFinder().FindByHash(ctx, uniq.StringHash(name))
		if err != nil {
			return d, err
		}

		if a.Hidden && !album.Settings.ShowHiddenSubAlbums {
			continue
		}

		if (!a.Public && !album.Settings.ShowPrivateSubAlbums) || a.Name == "" {
			if !d.IsAdmin {
				continue
			}
		}

		cont, err := pb.getCachedAlbum(ctx, a.Name, true, "", false)

		if len(cont.Images) == 0 && !d.IsAdmin {
			continue
		}

		d.SubAlbums = append(d.SubAlbums, cont)
	}

	if deps.Settings().Appearance().AlbumSpritesEnabled() && !cont.SkipSprites && !cont.Album.Settings.SkipSprites {
		deps.CtxdLogger().Info(ctx, "album sprite: adding album sprites")
		pb.addSprites(ctx, &d)
	}

	return d, nil
}

type albumPageData struct {
	pageCommon

	Description template.HTML
	OGTitle     string
	OGPageURL   string
	OGSiteName  string
	Name        string
	CoverImage  string
	CollabKey   string
	Public      bool
	NewestFirst bool
	Hash        string
	IsPhotoPage bool
	Page        string
	JsRender    bool

	Count          int
	TotalSize      string
	Visits         string
	EnableFavorite bool

	MapTiles       string
	MapAttribution string
	Featured       string

	AlbumData         getAlbumOutput
	StrippedAlbumData getAlbumOutput
	Timeline          []albumTimelineItem

	ShowMap           bool
	ShowEXIFPreview   bool
	ShowAISays        bool
	PreRender         bool
	HasPanos          bool
	HasPixelpeep      bool
	ThumbSprites      map[string]*sprite.ViewItem
	MarkerSprites     map[string]*sprite.ViewItem
	SpriteSheets      map[string]sprite.Sheet
	SpriteManifestKey string

	spritePendingManifestKey string
	spritePendingImages      []sprite.Image
}

func (pb *AlbumPageBuilder) startPendingSpriteBuild(ctx context.Context, album photo.Album, manifestKey string, spriteImages []sprite.Image) {
	go func() {
		ctx = context.WithoutCancel(ctx)
		ctx = ctxd.AddFields(ctx,
			"album", album.Name,
			"album_hash", album.Hash.String())
		pb.deps.CtxdLogger().Info(ctx, "album sprite: start async album sprite build",
			"album", album.Name,
			"manifest_key", manifestKey,
			"image_count", len(spriteImages))
		pb.deps.AlbumSprites().EnsureBuild(ctx, manifestKey, spriteImages)

		if err := pb.deps.DepCache().SpriteManifestChanged(ctx, manifestKey); err != nil {
			pb.deps.CtxdLogger().Error(ctx, "failed to set album sprite manifest changed", "album", album.Name, "error", err)
		}
	}()
}

// albumTimelineItem is either a photo or a chrono text, already placed in final display order —
// see buildAlbumTimeline. Image must round-trip through JSON: the album-page persistent cache
// stores albumPageData (which embeds this) as JSON, and on a cache hit build() never re-runs, so
// whatever the server template needs to render has to survive that exact round-trip, not just
// work when freshly computed in memory. ImageHash is a separate, deliberately minimal key: a
// client-side renderer only ever needs the hash, never the full (and here duplicated) Image object.
type albumTimelineItem struct {
	Image     *Image        `json:"image_full,omitempty"`
	ImageHash string        `json:"image,omitempty"`
	Text      template.HTML `json:"text,omitempty"`
	Ts        int64         `json:"-"`
}

func strippedAlbumData(cont getAlbumOutput) getAlbumOutput {
	res := cont
	res.Images = append([]Image(nil), cont.Images...)

	for i, img := range res.Images {
		img.Description = ""
		img.DescriptionHTML = ""
		img.AISays = ""
		res.Images[i] = img
	}

	return res
}

// appendText adds a text timeline entry, unless its text is blank — there's no use case for
// rendering an empty chrono-text block, so it's dropped here rather than trusted to callers.
func appendText(timeline []albumTimelineItem, t txt.Chronological) []albumTimelineItem {
	if strings.TrimSpace(t.Text) == "" {
		return timeline
	}

	return append(timeline, albumTimelineItem{
		Text: template.HTML(t.Text),
		Ts:   t.Time.Unix(),
	})
}

func buildAlbumTimeline(images []Image, texts []txt.Chronological, newestFirst bool) []albumTimelineItem {
	if len(images) == 0 && len(texts) == 0 {
		return nil
	}

	remaining := make([]txt.Chronological, len(texts))
	copy(remaining, texts)

	timeline := make([]albumTimelineItem, 0, len(images)+len(texts))

	for _, img := range images {
		if img.Is360Pano {
			continue
		}

		if len(remaining) > 0 {
			imgTime := time.Unix(img.UTime, 0)

			next := remaining[:0]
			for _, t := range remaining {
				if !displayReached(imgTime, t.Time, newestFirst) {
					next = append(next, t)
					continue
				}

				timeline = appendText(timeline, t)
			}

			remaining = next
		}

		imgCopy := img
		timeline = append(timeline, albumTimelineItem{
			Image:     &imgCopy,
			ImageHash: img.Hash,
			Ts:        img.UTime,
		})
	}

	for _, t := range remaining {
		timeline = appendText(timeline, t)
	}

	return timeline
}

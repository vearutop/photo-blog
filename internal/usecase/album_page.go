package usecase

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"strconv"
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

type AlbumPageBuilder struct {
	deps               getAlbumImagesDeps
	albumDataCache     *cache.FailoverOf[getAlbumOutput]
	albumDataCacheName string
	albumPageCache     *cache.FailoverOf[albumPageData]
	albumPageCacheName string
}

func NewAlbumPageBuilder(deps getAlbumImagesDeps) *AlbumPageBuilder {
	albumDataCacheName := "album-data"
	albumPageCacheName := "album-page"

	return &AlbumPageBuilder{
		deps:               deps,
		albumDataCache:     infraService.MakePersistentCacheOf[getAlbumOutput](deps, albumDataCacheName, 30*time.Hour),
		albumDataCacheName: albumDataCacheName,
		albumPageCache:     infraService.MakePersistentCacheOf[albumPageData](deps, albumPageCacheName, 30*time.Hour),
		albumPageCacheName: albumPageCacheName,
	}
}

func (pb *AlbumPageBuilder) getCachedAlbum(ctx context.Context, name string, preview bool) (getAlbumOutput, error) {
	cacheKey := []byte(name + "/" + strconv.FormatBool(auth.IsAdmin(ctx)) + "/" + txt.Language(ctx) + "/" + strconv.FormatBool(preview))
	cacheName := pb.albumDataCacheName
	cacheMiss := false

	d, err := pb.albumDataCache.Get(ctx, cacheKey, func(ctx context.Context) (getAlbumOutput, error) {
		cacheMiss = true
		if err := pb.deps.DepCache().ResetKey(ctx, cacheName, cacheKey); err != nil {
			return getAlbumOutput{}, fmt.Errorf("reset cache deps: %w", err)
		}

		return getAlbumContents(ctx, pb.deps, imagesFilter{albumName: name}, preview)
	})
	if err != nil {
		return getAlbumOutput{}, err
	}

	if cacheMiss {
		pb.deps.DepCache().ServiceSettingsDependency(cacheName, cacheKey)
		pb.deps.DepCache().AlbumDependency(cacheName, cacheKey, name)
	}

	return d, nil
}

func (pb *AlbumPageBuilder) addSprites(ctx context.Context, d *albumPageData) {
	deps := pb.deps
	album := d.AlbumData.Album

	imageSets := make([][]Image, 0, 1+len(d.SubAlbums))
	imageSets = append(imageSets, d.AlbumData.Images)
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
	d.ThumbSprites = pb.filterThumbSprites(items, d.AlbumData.Images)
	d.AlbumData.ThumbSprites = d.ThumbSprites
	d.MarkerSprites = deps.AlbumSprites().MarkerView(manifest)
	d.SpriteSheets = deps.AlbumSprites().CompactSheets(items, d.MarkerSprites)
	d.AlbumData.MarkerSprites = d.MarkerSprites
	d.AlbumData.SpriteSheets = d.SpriteSheets

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
		strconv.FormatBool(auth.IsBot(ctx)) + "/" + txt.Language(ctx))
	cacheName := pb.albumPageCacheName
	cacheMiss := false

	d, err := pb.albumPageCache.Get(ctx, cacheKey, func(ctx context.Context) (albumPageData, error) {
		cacheMiss = true
		if err := pb.deps.DepCache().ResetKey(ctx, cacheName, cacheKey); err != nil {
			return albumPageData{}, fmt.Errorf("reset cache deps: %w", err)
		}

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

	d.Description = template.HTML(album.Settings.Description)
	d.Name = album.Name
	d.Public = album.Public
	d.Hash = album.Hash.String()
	d.Count = len(cont.Images)
	d.AlbumData = cont
	d.AlbumData.Images = append([]Image(nil), cont.Images...)
	d.AlbumData.Album.Settings.CollabKey = ""
	d.Timeline = buildAlbumTimeline(cont.Images, cont.Album.Settings.Texts, cont.Album.Settings.NewestFirst)
	d.Featured = deps.Settings().Appearance().FeaturedAlbumName

	// Clear image descriptions from JSON.
	for i, img := range d.AlbumData.Images {
		img.Description = ""

		d.AlbumData.Images[i] = img
	}

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

	for _, img := range cont.Images {
		if img.Is360Pano {
			d.HasPanos = true
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

		cont, err := pb.getCachedAlbum(ctx, a.Name, true)

		if len(cont.Images) == 0 && !d.IsAdmin {
			continue
		}

		d.SubAlbums = append(d.SubAlbums, cont)
	}

	if deps.Settings().Appearance().AlbumSpritesEnabled() {
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

	Count          int
	TotalSize      string
	Visits         string
	EnableFavorite bool

	MapTiles       string
	MapAttribution string
	Featured       string

	AlbumData getAlbumOutput
	Timeline  []albumTimelineItem

	ShowMap           bool
	ShowEXIFPreview   bool
	ShowAISays        bool
	PreRender         bool
	HasPanos          bool
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

type albumTimelineItem struct {
	Image *Image
	Text  template.HTML
	Ts    int64
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
			next := remaining[:0]
			for _, t := range remaining {
				tt := t.Time.Unix()
				if newestFirst {
					if tt < img.UTime {
						next = append(next, t)
						continue
					}
				} else {
					if tt > img.UTime {
						next = append(next, t)
						continue
					}
				}

				timeline = append(timeline, albumTimelineItem{
					Text: template.HTML(t.Text),
					Ts:   tt,
				})
			}

			remaining = next
		}

		imgCopy := img
		timeline = append(timeline, albumTimelineItem{
			Image: &imgCopy,
			Ts:    img.UTime,
		})
	}

	for _, t := range remaining {
		timeline = append(timeline, albumTimelineItem{
			Text: template.HTML(t.Text),
			Ts:   t.Time.Unix(),
		})
	}

	return timeline
}

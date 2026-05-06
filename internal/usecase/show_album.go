package usecase

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/go-units"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/image/sprite"
	infraService "github.com/vearutop/photo-blog/internal/infra/service"
	"github.com/vearutop/photo-blog/pkg/txt"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type showAlbumAtImageInput struct {
	showAlbumInput
	Hash uniq.Hash `path:"hash"`
}

type showAlbumInput struct {
	request.EmbeddedSetter

	Name      string `path:"name"`
	CollabKey string `query:"collab_key" description:"Access key to enable content upload and management."`
	imgHash   uniq.Hash
}

func ShowAlbumAtImage(up usecase.IOInteractorOf[showAlbumInput, web.Page]) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in showAlbumAtImageInput, out *web.Page) error {
		in.imgHash = in.Hash

		return up.Invoke(ctx, in.showAlbumInput, out)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
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

	Images    []Image
	Panoramas []Image

	Count          int
	TotalSize      string
	Visits         string
	EnableFavorite bool

	MapTiles       string
	MapAttribution string
	Featured       string

	AlbumData getAlbumOutput
	Timeline  []albumTimelineItem

	ShowMap         bool
	ShowEXIFPreview bool
	ShowAISays      bool
	PreRender       bool
	HasPanos        bool
	ThumbSprites    map[string]*sprite.ViewItem
	MarkerSprites   map[string]*sprite.ViewItem
	SpriteSheets    map[string]sprite.Sheet
}

func albumSpriteImages(images []Image) []sprite.Image {
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

func mergeAlbumSpriteImages(images ...[]Image) []sprite.Image {
	seen := make(map[string]int)
	merged := make([]sprite.Image, 0)

	for _, set := range images {
		for _, item := range albumSpriteImages(set) {
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

func filterThumbSprites(items map[string]*sprite.ViewItem, images []Image) map[string]*sprite.ViewItem {
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

// ShowAlbum creates use case interactor to show album.
func ShowAlbum(deps interface {
	getAlbumImagesDeps
	showAlbumSpriteDeps
}) usecase.IOInteractorOf[showAlbumInput, web.Page] {
	tmpl, err := static.Template("album.gohtml")
	if err != nil {
		panic(err)
	}

	notFound := NotFound(deps)

	cacheName := "album-data"
	c := infraService.MakePersistentCacheOf[getAlbumOutput](deps, cacheName, time.Hour)

	u := usecase.NewInteractor(func(ctx context.Context, in showAlbumInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_album", 1)
		deps.CtxdLogger().Info(ctx, "showing album", "name", in.Name)

		cacheKey := []byte(in.Name + strconv.FormatBool(auth.IsAdmin(ctx)) + txt.Language(ctx))
		cont, err := c.Get(ctx, cacheKey, func(ctx context.Context) (getAlbumOutput, error) {
			if err := deps.DepCache().ResetKey(ctx, cacheName, cacheKey); err != nil {
				return getAlbumOutput{}, fmt.Errorf("reset cache deps: %w", err)
			}

			deps.DepCache().ServiceSettingsDependency(cacheName, cacheKey)
			deps.DepCache().AlbumDependency(cacheName, cacheKey, in.Name)

			out.ResponseWriter().Header().Set("X-Cache-Miss", "1")

			return getAlbumContents(ctx, deps, imagesFilter{albumName: in.Name}, false)
		})
		if err != nil {
			if errors.Is(err, status.NotFound) {
				return notFound.Invoke(ctx, struct{}{}, out)
			}

			return fmt.Errorf("get album contents: %w", err)
		}

		if in.CollabKey != "" && in.CollabKey != cont.Album.Settings.CollabKey {
			return status.Wrap(errors.New("wrong collab_key"), status.PermissionDenied)
		}

		if cont.Album.Settings.Redirect != "" {
			http.Redirect(out.ResponseWriter(), in.Request(), cont.Album.Settings.Redirect, http.StatusMovedPermanently)
		}

		album := cont.Album

		d := albumPageData{}
		d.Title = album.Title

		d.Description = template.HTML(album.Settings.Description)
		d.Name = album.Name
		d.CollabKey = in.CollabKey
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
			img.DescriptionHTML = ""

			d.AlbumData.Images[i] = img
		}

		d.fill(ctx, deps.TxtRenderer(), deps.Settings())
		if len(cont.Images) > 1 {
			d.OGTitle = fmt.Sprintf("%s (%d photos)", album.Title, len(cont.Images))
		} else {
			d.OGTitle = album.Title
		}
		d.OGPageURL = "https://" + in.Request().Host + in.Request().URL.Path
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

		if d.IsAdmin {
			ps, err := deps.VisitorStats().AlbumViews(ctx, album.Hash)
			if err != nil {
				deps.CtxdLogger().Error(ctx, "failed to get album views", "error", err)
			} else {
				d.Visits = fmt.Sprintf("%d/%d/%d", ps.Uniq, ps.Views, ps.Refers)
			}
		}

		switch {
		case in.imgHash != 0:
			d.CoverImage = "/thumb/1200w/" + in.imgHash.String() + ".jpg"
		case album.CoverImage != 0:
			d.CoverImage = "/thumb/1200w/" + album.CoverImage.String() + ".jpg"
		case len(cont.Images) > 0:
			d.CoverImage = "/thumb/1200w/" + cont.Images[0].Hash + ".jpg"
		}

		for _, name := range album.Settings.SubAlbumNames {
			a, err := deps.PhotoAlbumFinder().FindByHash(ctx, uniq.StringHash(name))
			if err != nil {
				return err
			}

			if a.Hidden && !album.Settings.ShowHiddenSubAlbums {
				continue
			}

			if (!a.Public && !album.Settings.ShowPrivateSubAlbums) || a.Name == "" {
				if !d.IsAdmin {
					continue
				}
			}

			cacheKey := []byte(a.Name + strconv.FormatBool(auth.IsAdmin(ctx)) + txt.Language(ctx) + "::preview")
			cont, err := c.Get(ctx, cacheKey, func(ctx context.Context) (getAlbumOutput, error) {
				if err := deps.DepCache().ResetKey(ctx, cacheName, cacheKey); err != nil {
					return getAlbumOutput{}, fmt.Errorf("reset cache deps: %w", err)
				}

				return getAlbumContents(ctx, deps, imagesFilter{albumName: a.Name}, true)
			})
			if err != nil {
				return err
			}

			if len(cont.Images) == 0 && !d.IsAdmin {
				continue
			}

			d.SubAlbums = append(d.SubAlbums, cont)
			deps.DepCache().AlbumDependency(cacheName, cacheKey, cont.Album.Name)
		}

		if deps.Settings().Appearance().AlbumSpritesEnabled() {
			imageSets := make([][]Image, 0, 1+len(d.SubAlbums))
			imageSets = append(imageSets, cont.Images)
			for _, subAlbum := range d.SubAlbums {
				imageSets = append(imageSets, subAlbum.Images)
			}

			spriteImages := mergeAlbumSpriteImages(imageSets...)
			if manifest, ok, err := deps.AlbumSprites().Ready(ctx, album, auth.IsAdmin(ctx), spriteImages); err != nil {
				deps.CtxdLogger().Error(ctx, "failed to get album sprite manifest", "album", album.Name, "error", err)
			} else if ok {
				items := deps.AlbumSprites().View(manifest)
				d.ThumbSprites = filterThumbSprites(items, cont.Images)
				d.AlbumData.ThumbSprites = d.ThumbSprites
				d.MarkerSprites = deps.AlbumSprites().MarkerView(manifest)
				d.SpriteSheets = deps.AlbumSprites().CompactSheets(items, d.MarkerSprites)
				d.AlbumData.MarkerSprites = d.MarkerSprites
				d.AlbumData.SpriteSheets = d.SpriteSheets

				for i := range d.SubAlbums {
					d.SubAlbums[i].ThumbSprites = filterThumbSprites(items, d.SubAlbums[i].Images)
					d.SubAlbums[i].SpriteSheets = d.AlbumData.SpriteSheets
				}
			}
		}

		return out.Render(tmpl, d)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument, status.PermissionDenied)

	return u
}

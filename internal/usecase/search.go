package usecase

import (
	"context"
	"errors"
	"fmt"
	"html/template"

	"github.com/docker/go-units"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

// SearchImages creates use case interactor to show images for criteria.
func SearchImages(deps getAlbumImagesDeps) usecase.Interactor {
	tmpl, err := static.Template("album.html")
	if err != nil {
		panic(err)
	}

	notFound := NotFound(deps)

	type searchInput struct {
		request.EmbeddedSetter
		Query string `query:"q"`
	}

	type pageData struct {
		pageCommon

		Description template.HTML
		OGTitle     string
		OGPageURL   string
		OGSiteName  string
		Name        string
		CoverImage  string
		IsAdmin     bool
		Public      bool
		Hash        string

		Images    []Image
		Panoramas []Image

		Count     int
		TotalSize string

		MapTiles       string
		MapAttribution string
		Featured       string

		AlbumData getAlbumOutput

		ShowMap bool
	}

	u := usecase.NewInteractor(func(ctx context.Context, in searchInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "search_images", 1)
		deps.CtxdLogger().Info(ctx, "searching images", "query", in.Query)

		cont, err := getAlbumContents(ctx, deps, "search:"+in.Query, false)
		if err != nil {
			if errors.Is(err, status.NotFound) {
				return notFound.Invoke(ctx, struct{}{}, out)
			}

			return fmt.Errorf("get album contents: %w", err)
		}

		album := cont.Album

		d := pageData{}
		d.Title = album.Title

		d.Description = template.HTML(album.Settings.Description)
		d.Name = album.Name
		d.IsAdmin = auth.IsAdmin(ctx)
		d.Public = album.Public
		d.Hash = album.Hash.String()
		d.Count = len(cont.Images)
		d.AlbumData = cont
		d.Featured = deps.Settings().Appearance().FeaturedAlbumName

		d.fill(ctx, deps.TxtRenderer(), deps.Settings())
		d.OGTitle = fmt.Sprintf("%s (%d photos)", album.Title, len(cont.Images))
		d.OGPageURL = "https://" + in.Request().Host + in.Request().URL.Path
		d.OGSiteName = deps.Settings().Appearance().SiteTitle

		d.ShowMap = !album.Settings.HideMap

		maps := deps.Settings().Maps()

		d.MapTiles = maps.Tiles
		if maps.Cache {
			d.MapTiles = "/map-tile/{r}/{z}/{x}/{y}.png"
		}

		d.MapAttribution = maps.Attribution

		// TotalSize controls visibility of batch download button.
		privacy := deps.Settings().Privacy()
		if d.IsAdmin || (!privacy.HideOriginal && !privacy.HideBatchDownload) {
			var totalSize int64
			for _, img := range cont.Images {
				totalSize += img.Size
			}

			d.TotalSize = units.HumanSize(float64(totalSize))
		}

		switch {
		case album.CoverImage != 0:
			d.CoverImage = "/thumb/1200w/" + album.CoverImage.String() + ".jpg"
		case len(cont.Images) > 0:
			d.CoverImage = "/thumb/1200w/" + cont.Images[0].Hash + ".jpg"
		}

		return out.Render(tmpl, d)
	})

	u.SetTags("Search")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

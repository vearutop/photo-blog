package usecase

import (
	"context"
	"fmt"
	"html/template"

	"github.com/docker/go-units"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
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

	//notFound := NotFound(deps)

	type searchInput struct {
		request.EmbeddedSetter
		Query  string  `query:"q"`
		Lens   *string `query:"lens"`
		Camera *string `query:"camera"`
		Offset uint64  `query:"offset"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in searchInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "search_images", 1)
		deps.CtxdLogger().Info(ctx, "searching images", "query", in.Query)

		q := deps.ImageSelector().Select()

		if !auth.IsAdmin(ctx) {
			q.OnlyPublic()
		}

		title := "Search: "

		if in.Query != "" {
			title += in.Query
			q.Search(in.Query)
		}

		if in.Lens != nil {
			title += " Lens: " + *in.Lens
			q.ByLens(*in.Lens)
		}

		if in.Camera != nil {
			title += " Camera: " + *in.Camera
			q.ByCamera(*in.Camera)
		}

		q.Limit(500)
		q.Offset(in.Offset)

		images, err := q.Find(ctx)
		if err != nil {
			return fmt.Errorf("select images: %w", err)
		}

		cont := getAlbumOutput{}
		cont.Album.Title = title
		cont.Album.Name = "search"
		cont.Album.Hash = photo.AlbumHash(title)

		if err := cont.prepare(ctx, deps, images, false); err != nil {
			return fmt.Errorf("prepare album contents: %w", err)
		}

		album := cont.Album

		d := albumPageData{}
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
			d.MapTiles = "/map-tile/{s}/{r}/{z}/{x}/{y}.png"
		}

		d.MapAttribution = maps.Attribution

		// TotalSize controls the visibility of the batch download button.
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

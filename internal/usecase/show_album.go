package usecase

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"github.com/docker/go-units"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
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

// ShowAlbum creates use case interactor to show album.
func ShowAlbum(deps getAlbumImagesDeps) usecase.IOInteractorOf[showAlbumInput, web.Page] {
	tmpl, err := static.Template("album.html")
	if err != nil {
		panic(err)
	}

	notFound := NotFound(deps)

	type pageData struct {
		pageCommon

		Description template.HTML
		OGTitle     string
		OGPageURL   string
		OGSiteName  string
		Name        string
		CoverImage  string
		IsAdmin     bool
		CollabKey   string
		Public      bool
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

		ImageBaseURL string
		ShowMap      bool
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showAlbumInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_album", 1)
		deps.CtxdLogger().Info(ctx, "showing album", "name", in.Name)

		cont, err := getAlbumContents(ctx, deps, in.Name, false)
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

		d := pageData{}
		d.Title = album.Title

		d.Description = template.HTML(album.Settings.Description)
		d.Name = album.Name
		d.IsAdmin = auth.IsAdmin(ctx)
		d.CollabKey = in.CollabKey
		d.Public = album.Public
		d.Hash = album.Hash.String()
		d.Count = len(cont.Images)
		d.AlbumData = cont
		d.AlbumData.Album.Settings.CollabKey = ""
		d.Featured = deps.Settings().Appearance().FeaturedAlbumName

		d.fill(ctx, deps.TxtRenderer(), deps.Settings().Appearance())
		d.OGTitle = fmt.Sprintf("%s (%d photos)", album.Title, len(cont.Images))
		d.OGPageURL = "https://" + in.Request().Host + in.Request().URL.Path
		d.OGSiteName = deps.Settings().Appearance().SiteTitle

		d.ImageBaseURL = album.Settings.ImageBaseURL
		d.ShowMap = !album.Settings.HideMap

		maps := deps.Settings().Maps()

		d.MapTiles = maps.Tiles
		if maps.Cache {
			d.MapTiles = "/map-tile/{r}/{z}/{x}/{y}.png"
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
		if d.IsAdmin || (!privacy.HideOriginal && !privacy.HideBatchDownload) {
			var totalSize int64
			for _, img := range cont.Images {
				totalSize += img.Size
			}

			d.TotalSize = units.HumanSize(float64(totalSize))
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

		return out.Render(tmpl, d)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument, status.PermissionDenied)

	return u
}

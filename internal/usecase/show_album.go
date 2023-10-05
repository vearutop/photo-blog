package usecase

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/docker/go-units"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type showAlbumAtImageInput struct {
	showAlbumInput
	Hash uniq.Hash `path:"hash"`
}

type showAlbumInput struct {
	Name    string `path:"name"`
	hasAuth bool
	imgHash uniq.Hash
}

func (i *showAlbumInput) SetRequest(r *http.Request) {
	if r.Header.Get("Authorization") != "" {
		i.hasAuth = true
	}
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
func ShowAlbum(deps getAlbumImagesDeps, contents usecase.IOInteractorOf[getAlbumInput, getAlbumOutput]) usecase.IOInteractorOf[showAlbumInput, web.Page] {
	tpl, err := static.Assets.ReadFile("album.html")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		panic(err)
	}

	type pageData struct {
		Title       string
		Description template.HTML
		OGTitle     string
		Name        string
		CoverImage  string
		NonAdmin    bool
		Public      bool
		Hash        string

		Images    []Image
		Panoramas []Image

		Count     int
		TotalSize string

		MapTiles       string
		MapAttribution string

		AlbumData getAlbumOutput
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showAlbumInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_album", 1)
		deps.CtxdLogger().Info(ctx, "showing album", "name", in.Name)

		cont := getAlbumOutput{}

		if err := contents.Invoke(ctx, getAlbumInput{
			Name: in.Name,
		}, &cont); err != nil {
			return err
		}

		if len(cont.Images) == 0 {
			return errors.New("no images")
		}

		album := cont.Album

		d := pageData{}
		d.Title = album.Title
		d.Description = template.HTML(strings.ReplaceAll(album.Settings.Description, "\n", "<br />"))
		d.OGTitle = fmt.Sprintf("%s (%d photos)", album.Title, len(cont.Images))
		d.Name = album.Name
		d.NonAdmin = !in.hasAuth
		d.Public = album.Public
		d.Hash = album.Hash.String()
		d.Count = len(cont.Images)
		d.AlbumData = cont

		d.MapTiles = deps.ServiceSettings().MapTiles
		d.MapAttribution = deps.ServiceSettings().MapAttribution

		var totalSize int64
		for _, img := range cont.Images {
			if img.Exif == nil || img.BlurHash == "" {
				continue
			}

			if img.Exif.ProjectionType == "" {
				d.Images = append(d.Images, img)
			} else {
				d.Panoramas = append(d.Panoramas, img)
			}

			totalSize += img.size
		}
		d.TotalSize = units.HumanSize(float64(totalSize))

		switch {
		case in.imgHash != 0:
			d.CoverImage = "/thumb/1200w/" + in.imgHash.String() + ".jpg"
		case album.CoverImage != 0:
			d.CoverImage = "/thumb/1200w/" + album.CoverImage.String() + ".jpg"
		default:
			d.CoverImage = "/thumb/1200w/" + cont.Images[0].Hash + ".jpg"
		}

		return out.Render(tmpl, d)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

package usecase

import (
	"context"
	"errors"
	"html/template"
	"io"
	"net/http"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/resources/static"
)

type albumPage struct {
	Title      string
	Name       string
	CoverImage string
	NonAdmin   bool
	Public     bool

	writer io.Writer
}

func (o *albumPage) SetWriter(w io.Writer) {
	o.writer = w
}

func (o *albumPage) Render(tmpl *template.Template) error {
	return tmpl.Execute(o.writer, o)
}

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

func ShowAlbumAtImage(up usecase.IOInteractorOf[showAlbumInput, albumPage]) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in showAlbumAtImageInput, out *albumPage) error {
		in.imgHash = in.Hash

		return up.Invoke(ctx, in.showAlbumInput, out)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

// ShowAlbum creates use case interactor to show album.
func ShowAlbum(deps getAlbumImagesDeps) usecase.IOInteractorOf[showAlbumInput, albumPage] {
	tpl, err := static.Assets.ReadFile("album.html")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("htmlResponse").Parse(string(tpl))
	if err != nil {
		panic(err)
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showAlbumInput, out *albumPage) error {
		deps.StatsTracker().Add(ctx, "show_album", 1)
		deps.CtxdLogger().Info(ctx, "showing album", "name", in.Name)

		albumHash := photo.AlbumHash(in.Name)

		album, err := deps.PhotoAlbumFinder().FindByHash(ctx, albumHash)
		if err != nil {
			return err
		}

		images, err := deps.PhotoAlbumImageFinder().FindImages(ctx, albumHash)
		if err != nil {
			return err
		}

		if len(images) == 0 {
			return errors.New("no images")
		}

		out.Title = album.Title
		out.Name = album.Name
		out.NonAdmin = !in.hasAuth
		out.Public = album.Public

		switch {
		case in.imgHash != 0:
			out.CoverImage = "/thumb/1200w/" + in.imgHash.String() + ".jpg"
		case album.CoverImage != 0:
			out.CoverImage = "/thumb/1200w/" + album.CoverImage.String() + ".jpg"
		default:
			out.CoverImage = "/thumb/1200w/" + images[0].Hash.String() + ".jpg"
		}

		return out.Render(tmpl)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)

	return u
}

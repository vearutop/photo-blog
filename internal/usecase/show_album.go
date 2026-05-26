package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/swaggest/rest/request"
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
	tmpl, err := static.Template("album.gohtml")
	if err != nil {
		panic(err)
	}

	notFound := NotFound(deps)

	b := NewAlbumPageBuilder(deps)

	u := usecase.NewInteractor(func(ctx context.Context, in showAlbumInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_album", 1)
		deps.CtxdLogger().Debug(ctx, "showing album", "name", in.Name)

		cont, err := b.getCachedAlbum(ctx, in.Name, false)
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

		d, err := b.cachedBuild(ctx, cont)
		if err != nil {
			return err
		}
		d.CollabKey = in.CollabKey
		d.OGPageURL = "https://" + in.Request().Host + in.Request().URL.Path

		if in.imgHash != 0 {
			d.CoverImage = "/thumb/1200w/" + in.imgHash.String() + ".jpg"
		}

		if d.IsAdmin {
			ps, err := deps.VisitorStats().AlbumViews(ctx, d.AlbumData.Album.Hash)
			if err != nil {
				deps.CtxdLogger().Error(ctx, "failed to get album views", "error", err)
			} else {
				d.Visits = fmt.Sprintf("%d/%d/%d", ps.Uniq, ps.Views, ps.Refers)
			}
		}

		return out.Render(tmpl, d)
	})

	u.SetTags("Album")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument, status.PermissionDenied)

	return u
}

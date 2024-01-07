package control

import (
	"context"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/storage"
	uc "github.com/vearutop/photo-blog/internal/usecase"
	"github.com/vearutop/photo-blog/pkg/txt"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type dashboardDeps interface {
	TxtRenderer() *txt.Renderer
	Settings() settings.Values
	Stats() *storage.Stats
}

func Dashboard(deps dashboardDeps) usecase.Interactor {
	tmpl, err := static.Template("dashboard.html")
	if err != nil {
		panic(err)
	}

	type pageData struct {
		uc.PageCommon

		PhotosCount int
	}

	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *web.Page) error {
		d := pageData{}

		d.Title = "Dashboard"
		d.PhotosCount = 123
		d.Fill(ctx, deps.TxtRenderer(), deps.Settings().Appearance())

		return output.Render(tmpl, d)
	})

	return u
}

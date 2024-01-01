package usecase

import (
	"context"
	"html/template"
	"net/http"

	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/pkg/txt"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type notFoundDeps interface {
	Settings() settings.Values
	StatsTracker() stats.Tracker
	TxtRenderer() *txt.Renderer
}

// NotFound creates use case interactor to show Not Found page.
func NotFound(deps notFoundDeps) usecase.IOInteractorOf[struct{}, web.Page] {
	tmpl, err := static.Template("not-found.html")
	if err != nil {
		panic(err)
	}

	type pageData struct {
		pageCommon

		Description template.HTML
	}

	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "not_found", 1)

		d := pageData{}
		d.fill(ctx, deps.TxtRenderer(), deps.Settings().Appearance())

		d.Description = `There is nothing to be shown at this page. Please check the <a href="/">home page</a> instead.`

		out.ResponseWriter().WriteHeader(http.StatusNotFound)

		return out.Render(tmpl, d)
	})

	u.SetTags("Site")

	return u
}

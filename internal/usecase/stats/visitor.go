package stats

import (
	"context"
	"time"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

func ShowVisitor(deps showDailyStatsDeps) usecase.Interactor {
	tmpl, err := static.Template("stats/table.html")
	if err != nil {
		panic(err)
	}

	type showVisitorInput struct {
		Hash uniq.Hash `path:"hash"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in showVisitorInput, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "show_visitor", 1)
		deps.CtxdLogger().Info(ctx, "showing visitor")

		info, err := deps.VisitorStats().VisitorInfo(ctx, in.Hash)
		if err != nil {
			return err
		}

		d := pageData{}
		d.Title = "Visitor"

		d.Tables = append(d.Tables, Table{
			Title: "Info",
			Rows:  infoRows(info),
		})

		type pageVisit struct {
			Date string `json:"date"`
			Page string `json:"page"`
		}

		var pageVisits []pageVisit

		pv, err := deps.VisitorStats().PageVisits(ctx, in.Hash)
		if err != nil {
			return err
		}

		for _, row := range pv {
			pageVisits = append(pageVisits, pageVisit{
				Date: time.Unix(row.Date, 0).Format("2006-01-02"),
				Page: albumLink(ctx, row.Page, deps.PhotoAlbumFinder()),
			})
		}

		d.Tables = append(d.Tables, Table{
			Title: "Visits",
			Rows:  pageVisits,
		})

		return out.Render(tmpl, d)
	})

	return u
}

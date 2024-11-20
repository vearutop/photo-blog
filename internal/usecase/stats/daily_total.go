package stats

import (
	"context"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/storage/visitor"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type showDailyStatsDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	VisitorStats() *visitor.StatsRepository
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoImageFinder() uniq.Finder[photo.Image]
}

func ShowDailyTotal(deps showDailyStatsDeps) usecase.Interactor {
	tmpl, err := static.Template("stats/table.html")
	if err != nil {
		panic(err)
	}

	type dateRow struct {
		Name   string `json:"name"`
		Date   string `json:"date"`
		Uniq   int    `json:"uniq"`
		Views  int    `json:"views"`
		Refers int    `json:"refers"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "daily_total", 1)
		deps.CtxdLogger().Info(ctx, "showing daily total")

		now := time.Now()
		st, err := deps.VisitorStats().DailyTotal(ctx, now.Add(-30*24*time.Hour), now)
		if err != nil {
			return err
		}

		d := pageData{}
		d.Title = "Daily Total"

		var rows []dateRow

		for _, row := range st {
			r := dateRow{}
			r.Date = time.Unix(row.Date, 0).Format("2006-01-02")
			if row.Hash == 0 {
				r.Name = `<a href="/">[main page]</a>`
			} else {
				r.Name = albumLink(ctx, row.Hash, deps.PhotoAlbumFinder())
			}
			r.Views = row.Views
			r.Uniq = row.Uniq
			r.Refers = row.Refers

			rows = append(rows, r)
		}

		d.Tables = append(d.Tables, Table{
			Rows: rows,
		})

		return out.Render(tmpl, d)
	})

	return u
}

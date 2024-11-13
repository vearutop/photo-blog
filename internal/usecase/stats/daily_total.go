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

	type pageData struct {
		Title string    `json:"title"`
		Rows  []dateRow `json:"rows"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "daily_total", 1)
		deps.CtxdLogger().Info(ctx, "showing daily total")

		now := time.Now()
		st, err := deps.VisitorStats().DailyTotal(ctx, now.Add(-30*24*time.Hour), now)
		if err != nil {
			return err
		}

		var hashes []uniq.Hash

		for _, row := range st {
			if row.Hash == 0 {
				continue
			}

			hashes = append(hashes, row.Hash)
		}

		albums, err := deps.PhotoAlbumFinder().FindByHashes(ctx, hashes...)
		if err != nil {
			return err
		}

		nameByHash := make(map[uniq.Hash]string, len(hashes))
		for _, a := range albums {
			nameByHash[a.Hash] = a.Name
		}

		d := pageData{
			Title: "Daily Total",
		}

		for _, row := range st {
			r := dateRow{}
			r.Date = time.Unix(row.Date, 0).Format("2006-01-02")
			if row.Hash == 0 {
				r.Name = `<a href="/">[main page]</a>`
			} else {
				name := nameByHash[row.Hash]
				if name == "" {
					r.Name = "[not found: " + row.Hash.String() + "]"
				} else {
					r.Name = `<a href="/` + name + `/">` + name + `</a>`
				}
			}
			r.Views = row.Views
			r.Uniq = row.Uniq
			r.Refers = row.Refers

			d.Rows = append(d.Rows, r)
		}

		return out.Render(tmpl, d)
	})

	return u
}

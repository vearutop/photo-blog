package stats

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/stats"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/storage/visitor"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

type showDailyStatsDeps interface {
	StatsTracker() stats.Tracker
	CtxdLogger() ctxd.Logger
	Settings() settings.Values

	VisitorStats() *visitor.StatsRepository
	PhotoAlbumFinder() uniq.Finder[photo.Album]
	PhotoImageFinder() uniq.Finder[photo.Image]
	PhotoAlbumImageFinder() photo.AlbumImageFinder
}

func ShowDailyTotal(deps showDailyStatsDeps) usecase.Interactor {
	type dateRow struct {
		Name     string `json:"name"`
		Date     string `json:"date"`
		Uniq     int    `json:"uniq"`
		Views    int    `json:"views"`
		Refers   int    `json:"refers"`
		Visitors string `json:"visitors"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "daily_total", 1)
		deps.CtxdLogger().Info(ctx, "showing daily total")

		now := time.Now()
		st, err := deps.VisitorStats().DailyTotal(ctx, now.Add(-30*24*time.Hour), now)
		if err != nil {
			return err
		}

		d := PageData{}
		d.Title = "Daily Total"

		var rows []dateRow

		for _, row := range st {
			r := dateRow{}
			r.Date = time.Unix(row.Date, 0).Format("2006-01-02")
			r.Name = albumLink(ctx, row.Hash, deps.PhotoAlbumFinder())
			r.Views = row.Views
			r.Uniq = row.Uniq
			r.Refers = row.Refers
			for _, v := range strings.Split(row.Visitors, ",") {
				i64, err := strconv.ParseInt(v, 10, 64)
				if err == nil {
					h := uniq.Hash(i64)

					r.Visitors += `<a href="/stats/visitor/` + h.String() + `.html">` + h.String() + `</a> `
				}
			}

			rows = append(rows, r)
		}

		d.Tables = append(d.Tables, Table{
			Rows: rows,
		})

		return out.Render(static.TableTemplate, d)
	})

	return u
}

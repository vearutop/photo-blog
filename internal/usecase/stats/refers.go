package stats

import (
	"context"
	"time"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

func ShowRefers(deps showDailyStatsDeps) usecase.Interactor {
	tmpl, err := static.Template("stats/table.html")
	if err != nil {
		panic(err)
	}

	type referRow struct {
		Date    string `json:"date"`
		Visitor string `json:"visitor"`
		Referer string `json:"referer"`
		URL     string `json:"url"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *web.Page) error {
		deps.StatsTracker().Add(ctx, "refers", 1)
		deps.CtxdLogger().Info(ctx, "showing refers")

		st, err := deps.VisitorStats().LatestRefers(ctx)
		if err != nil {
			return err
		}

		d := pageData{
			Title: "Latest Refers",
		}

		var rows []referRow

		for _, row := range st {
			h := row.Visitor.String()
			r := referRow{}
			r.Date = time.Unix(row.TS, 0).Format(time.DateTime)
			r.Visitor = `<a href="/stats/visitor/` + h + `.html">` + h + `</a>`
			r.Referer = `<a href="` + row.Referer + `">` + row.Referer + `</a>`
			r.URL = `<a href="` + row.URL + `">` + row.URL + `</a>`

			rows = append(rows, r)
		}

		d.Tables = append(d.Tables, Table{
			Rows: rows,
		})

		return out.Render(tmpl, d)
	})

	return u
}

package stats

import (
	"context"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

func TopImages(deps showDailyStatsDeps) usecase.Interactor {
	tmpl, err := static.Template("stats/table.html")
	if err != nil {
		panic(err)
	}

	type dateRow struct {
		Preview      string  `json:"preview"`
		Hash         string  `json:"hash"`
		Uniq         int     `json:"uniq"`
		Views        int     `json:"views"`
		Zooms        int     `json:"zooms"`
		ViewTime     float64 `json:"view_minutes"`
		ThumbTime    float64 `json:"preview_minutes"`
		ThumbPrtTime float64 `json:"preview_mobile_stripe_minutes"`
	}

	type pageData struct {
		Title string    `json:"title"`
		Rows  []dateRow `json:"rows"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *web.Page) error {
		st, err := deps.VisitorStats().TopImages(ctx)
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

		d := pageData{
			Title: "Top Images",
		}

		for _, row := range st {
			r := dateRow{}
			r.Preview = `<a href="/list-` + row.Hash.String() + `/"><img style="width: 300px" src="/thumb/300w/` + row.Hash.String() + `.jpg" src="/thumb/600w/` + row.Hash.String() + `.jpg 2x"/></a>`
			r.Hash = row.Hash.String()
			r.Views = row.Views
			r.Uniq = row.Uniq
			r.Zooms = row.Zooms
			r.ViewTime = float64(row.ViewMs) / float64(60*1000)
			r.ThumbTime = float64(row.ThumbMs) / float64(60*1000)
			r.ThumbPrtTime = float64(row.ThumbPrtMs) / float64(60*1000)

			d.Rows = append(d.Rows, r)
		}

		return out.Render(tmpl, d)
	})

	return u
}

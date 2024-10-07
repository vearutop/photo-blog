package stats

import (
	"context"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

func TopPages(deps showDailyStatsDeps) usecase.Interactor {
	tmpl, err := static.Template("stats/table.html")
	if err != nil {
		panic(err)
	}

	type dateRow struct {
		Name   string `json:"name"`
		Uniq   int    `json:"uniq"`
		Views  int    `json:"views"`
		Refers int    `json:"refers"`
	}

	type pageData struct {
		Title string    `json:"title"`
		Rows  []dateRow `json:"rows"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *web.Page) error {
		st, err := deps.VisitorStats().TopAlbums(ctx)
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
			Title: "Top Pages",
		}

		for _, row := range st {
			r := dateRow{}
			if row.Hash == 0 {
				r.Name = `<a href="/">[main page]</a>`
			} else {
				name := nameByHash[row.Hash]
				r.Name = `<a href="/` + name + `/">` + name + `</a>`
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

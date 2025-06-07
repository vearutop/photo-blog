package stats

import (
	"context"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

func TopPages(deps showDailyStatsDeps) usecase.Interactor {
	type dateRow struct {
		Name   string `json:"name"`
		Uniq   int    `json:"uniq"`
		Views  int    `json:"views"`
		Refers int    `json:"refers"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *web.Page) error {
		st, err := deps.VisitorStats().TopAlbums(ctx)
		if err != nil {
			return err
		}

		d := PageData{}
		d.Title = "Top Pages"

		var rows []dateRow

		for _, row := range st {
			r := dateRow{}
			r.Name = albumLink(ctx, row.Hash, deps.PhotoAlbumFinder())
			r.Views = row.Views
			r.Uniq = row.Uniq
			r.Refers = row.Refers

			rows = append(rows, r)
		}

		d.Tables = append(d.Tables, Table{
			Rows: rows,
		})

		return out.Render(static.TableTemplate, d)
	})

	return u
}

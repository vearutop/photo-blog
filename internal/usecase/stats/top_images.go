package stats

import (
	"context"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
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

	type topImagesFilter struct {
		AlbumName string `query:"album_name"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in topImagesFilter, out *web.Page) error {
		var imageHashes []uniq.Hash

		if in.AlbumName != "" {
			res, err := deps.PhotoAlbumImageFinder().FindImages(ctx, photo.AlbumHash(in.AlbumName))
			if err != nil {
				return err
			}

			imageHashes = make([]uniq.Hash, len(res))
			for i, img := range res {
				imageHashes[i] = img.Hash
			}
		}

		st, err := deps.VisitorStats().TopImages(ctx, imageHashes...)
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

		d := pageData{}
		d.Title = "Top Images"

		var rows []dateRow

		thumbBase := deps.Settings().Appearance().ThumbBaseURL
		if thumbBase == "" {
			thumbBase = "/thumb"
		}

		ms2min := func(ms int) float64 {
			f := float64(ms) / float64(60*1000)
			return float64(int(100*f)) / 100.0
		}

		for _, row := range st {
			r := dateRow{}
			r.Preview = `<a href="/list-` + row.Hash.String() + `/"><img style="width: 300px" src="` + thumbBase + `/300w/` + row.Hash.String() + `.jpg" src="` + thumbBase + `/600w/` + row.Hash.String() + `.jpg 2x"/></a>`
			r.Hash = row.Hash.String()
			r.Views = row.Views
			r.Uniq = row.Uniq
			r.Zooms = row.Zooms
			r.ViewTime = ms2min(row.ViewMs)
			r.ThumbTime = ms2min(row.ThumbMs)
			r.ThumbPrtTime = ms2min(row.ThumbPrtMs)

			rows = append(rows, r)
		}

		d.Tables = append(d.Tables, Table{
			Rows: rows,
		})

		return out.Render(tmpl, d)
	})

	return u
}

package control

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/bool64/dev/version"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/usecase/stats"
	"github.com/vearutop/photo-blog/pkg/web"
	"github.com/vearutop/photo-blog/resources/static"
)

func Version() usecase.Interactor {
	type GitHubRelease struct {
		TagName     string    `json:"tag_name"`
		Draft       bool      `json:"draft"`
		Prerelease  bool      `json:"prerelease"`
		CreatedAt   time.Time `json:"created_at"`
		PublishedAt time.Time `json:"published_at"`
		Assets      []struct {
			Size               int    `json:"size"`
			BrowserDownloadUrl string `json:"browser_download_url"`
		} `json:"assets"`
	}

	type releaseRow struct {
		Version   string    `json:"version"`
		CreatedAt time.Time `json:"created_at"`
		Update    string    `json:"update"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *web.Page) error {
		repo := "vearutop/photo-blog"

		resp, err := http.Get("https://api.github.com/repos/" + repo + "/releases")
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var releases []GitHubRelease
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return err
		}

		d := stats.PageData{}
		d.Title = "Versions"
		d.Description = "Current Version: " + version.Module("github.com/"+repo).Version + " (" + runtime.GOOS + "-" + runtime.GOARCH + ")"

		var rows []releaseRow

		for _, row := range releases {
			r := releaseRow{}
			r.CreatedAt = row.CreatedAt
			r.Version = row.TagName
			r.Update = `<a href="/settings/self-update?version=` + row.TagName + `">Apply Update</a>`

			rows = append(rows, r)
		}

		d.Tables = append(d.Tables, stats.Table{
			Rows: rows,
		})

		return out.Render(static.TableTemplate, d)
	})

	return u
}

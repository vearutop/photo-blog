package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bool64/ctxd"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/settings"
)

func ServeSitemap(deps interface {
	CtxdLogger() ctxd.Logger
	Settings() settings.Values
	PhotoAlbumFinder() uniq.Finder[photo.Album]
}) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in struct{}, out *response.EmbeddedSetter) error {
		deps.CtxdLogger().Info(ctx, "serving sitemap")

		baseURL := deps.Settings().Appearance().CanonicalBaseURL
		if baseURL == "" {
			return errors.New("canonical base URL is not set")
		}

		baseURL = strings.TrimSuffix(baseURL, "/")

		rw := out.ResponseWriter()

		albums, err := deps.PhotoAlbumFinder().FindAll(ctx)
		if err != nil {
			return err
		}

		rw.Header().Set("Content-Type", "application/xml")
		rw.Header().Set("Cache-Control", "max-age=31536000")

		var lastUpdatedAt time.Time

		var resp []byte

		resp = append(resp, []byte(`<?xml version="1.0" encoding="UTF-8"?>`)...)
		resp = append(resp, []byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)...)

		for _, a := range albums {
			if !a.Public {
				continue
			}

			if a.UpdatedAt.After(lastUpdatedAt) {
				lastUpdatedAt = a.UpdatedAt
			}

			resp = append(resp, []byte(`<url>`)...)
			resp = append(resp, []byte(`<loc>`+baseURL+"/"+a.Name+`/</loc>`)...)
			resp = append(resp, []byte(`<lastmod>`+a.UpdatedAt.Format(time.RFC3339)+`</lastmod>`)...)
			resp = append(resp, []byte(`</url>`)...)
		}

		resp = append(resp, []byte(`<url>`)...)
		resp = append(resp, []byte(`<loc>`+baseURL+`/</loc>`)...)
		resp = append(resp, []byte(`<lastmod>`+lastUpdatedAt.Format(time.RFC3339)+`</lastmod>`)...)
		resp = append(resp, []byte(`</url>`)...)

		resp = append(resp, []byte(`</urlset>`)...)

		_, err = rw.Write(resp)
		if err != nil {
			deps.CtxdLogger().Error(ctx, "write sitemap", "err", err)
			return err
		}

		return nil
	})
	u.SetTags("SEO")

	return u
}

package stats

import (
	"context"
	"errors"

	"github.com/bool64/cache"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

var (
	albumNameCache = cache.NewFailoverOf[string]()
)

func albumLink(ctx context.Context, hash uniq.Hash, finder uniq.Finder[photo.Album]) string {
	if hash == 0 {
		return `<a href="/">[main page]</a>`
	}

	name, err := albumNameCache.Get(ctx, []byte(hash.String()), func(ctx context.Context) (string, error) {
		album, err := finder.FindByHash(ctx, hash)
		if err != nil {
			return "", err
		}

		return album.Name, nil
	})

	if err != nil {
		if errors.Is(err, status.NotFound) {
			return "[not found: " + hash.String() + "]"
		}

		return "[err: " + err.Error() + ": " + hash.String() + "]"
	}

	return `<a href="/` + name + `/">` + name + `</a>`
}

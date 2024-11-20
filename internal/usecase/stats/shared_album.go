package stats

import (
	"context"
	"github.com/bool64/cache"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

var (
	albumNameCache = cache.NewFailoverOf[string]()
)

func albumLink(ctx context.Context, hash uniq.Hash, finder uniq.Finder[photo.Album]) string {
	name, err := albumNameCache.Get(ctx, []byte(hash.String()), func(ctx context.Context) (string, error) {
		album, err := finder.FindByHash(ctx, hash)
		if err != nil {
			return "", err
		}

		return album.Name, nil
	})

	if err != nil {
		return "[err: " + err.Error() + ": " + hash.String() + "]"

	}

	if name == "" {
		return "[not found: " + hash.String() + "]"
	} else {
		return `<a href="/` + name + `/">` + name + `</a>`
	}
}

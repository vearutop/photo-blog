package photo

import (
	"context"
)

type ImageIndexer interface {
	Index(ctx context.Context, image Image) error
}

type ImageEnsurer interface {
	Ensure(ctx context.Context, value ImageData) (Image, error)
}

type ImageUpdater interface {
	Update(ctx context.Context, value ImageData) error
}

type ImageFinder interface {
	FindByHash(ctx context.Context, hash Hash) (Image, error)
}

type Image struct {
	Identity
	Time
	ImageData
}

type ImageData struct {
	Hash   Hash   `db:"hash"`
	Size   int64  `db:"size"`
	Path   string `db:"path"`
	Width  int64  `db:"width"`
	Height int64  `db:"height"`
}

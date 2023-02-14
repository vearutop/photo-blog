package photo

import (
	"context"

	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type ImageIndexer interface {
	Index(ctx context.Context, image Images, flags IndexingFlags) error
}

type ImageEnsurer interface {
	Ensure(ctx context.Context, value ImageData) (Images, error)
}

type ImageUpdater interface {
	Update(ctx context.Context, value ImageData) error
}

type ImageFinder interface {
	FindByHash(ctx context.Context, hash uniq.Hash) (Images, error)
	FindAll(ctx context.Context) ([]Images, error)
}

type IndexingFlags struct {
	RebuildExif bool `formData:"rebuild_exif"`
	RebuildGps  bool `formData:"rebuild_gps"`
}

type Images struct {
	Identity
	uniq.Time
	ImageData
}

type ImageData struct {
	Hash   uniq.Hash `db:"hash"`
	Size   int64     `db:"size"`
	Path   string    `db:"path"`
	Width  int64     `db:"width"`
	Height int64     `db:"height"`
}

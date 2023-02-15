package photo

import (
	"context"

	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type ImageIndexer interface {
	Index(ctx context.Context, image Image, flags IndexingFlags) error
}

type IndexingFlags struct {
	RebuildExif bool `formData:"rebuild_exif"`
	RebuildGps  bool `formData:"rebuild_gps"`
}

type Image struct {
	uniq.Head
	Size     int64  `db:"size"`
	Path     string `db:"path"`
	Width    int64  `db:"width"`
	Height   int64  `db:"height"`
	BlurHash string `db:"blurhash"`
}

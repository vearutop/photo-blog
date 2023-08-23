package photo

import (
	"context"
	"time"

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
	uniq.File
	Width    int64      `db:"width" json:"width"`
	Height   int64      `db:"height" json:"height"`
	BlurHash string     `db:"blurhash" json:"blurhash"`
	TakenAt  *time.Time `db:"taken_at" json:"taken_at"`
	HasAVIF  bool       `db:"has_avif" json:"has_avif"`
}

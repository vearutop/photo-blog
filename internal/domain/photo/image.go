package photo

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type ImageIndexer interface {
	Index(ctx context.Context, image Image, flags IndexingFlags) error
	QueueIndex(ctx context.Context, img Image, flags IndexingFlags)
	QueueCallback(ctx context.Context, cb func(ctx context.Context))
}

type IndexingFlags struct {
	RebuildExif      bool `formData:"rebuild_exif"`
	RebuildGps       bool `formData:"rebuild_gps"`
	RebuildImageSize bool `formData:"rebuild_image_size"`
}

type Image struct {
	uniq.File
	Width     int64         `db:"width" title:"Width, px" json:"width" readOnly:"true"`
	Height    int64         `db:"height" title:"Height, px" json:"height" readOnly:"true"`
	BlurHash  string        `db:"blurhash" title:"BlurHash" json:"blurhash" readOnly:"true"`
	Sharpness *uint8        `db:"sharpness" title:"Sharpness" json:"sharpness" readOnly:"true"`
	PHash     uniq.Hash     `db:"phash" title:"PerceptionHash" json:"phash" readonly:"true"` // uniq.Hash is used JSON accuracy.
	TakenAt   *time.Time    `db:"taken_at" title:"Taken At" json:"taken_at"`
	Settings  ImageSettings `db:"settings" json:"settings" title:"Settings" description:"Additional parameters for an album."`
}

type ImageSettings struct {
	Description string   `json:"description,omitempty" formType:"textarea" title:"Description" description:"Description of an image, can contain HTML."`
	HTTPSources []string `json:"http_sources,omitempty"`
	Rotate      int      `json:"rotate,omitempty"`
}

// TODO: generalize scanner with generics.
func (s *ImageSettings) Scan(src any) error {
	if src == nil {
		return nil
	}

	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("unsupported type %T", src)
	}
}

func (s ImageSettings) Value() (driver.Value, error) {
	j, err := json.Marshal(s)

	return string(j), err
}

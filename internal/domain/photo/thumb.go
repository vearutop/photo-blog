package photo

import (
	"context"
	"fmt"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"io"
	"strconv"
	"strings"

	"github.com/swaggest/jsonschema-go"
)

type Thumbnailer interface {
	Thumbnail(ctx context.Context, image Image, size ThumbSize) (io.ReadSeeker, error)
}

type ThumbFinder interface {
	Find(ctx context.Context, imageID int, width, height uint) (Thumb, error)
}

type ThumbAdder interface {
	Add(ctx context.Context, value ThumbValue) (Thumb, error)
}

type Thumb struct {
	Identity
	uniq.Head
	ThumbValue
}

type ThumbSize string

func (t ThumbSize) PrepareJSONSchema(schema *jsonschema.Schema) error {
	enum := make([]any, 0, len(ThumbSizes))

	for _, s := range ThumbSizes {
		enum = append(enum, s)
	}

	schema.WithEnum(enum...)

	return nil
}

func (t ThumbSize) WidthHeight() (uint, uint, error) {
	s := string(t)
	if strings.HasSuffix(s, "w") {
		w, err := strconv.Atoi(strings.TrimSuffix(s, "w"))
		if err != nil {
			return 0, 0, err
		}

		return uint(w), 0, nil
	}

	if strings.HasSuffix(s, "h") {
		h, err := strconv.Atoi(strings.TrimSuffix(s, "h"))
		if err != nil {
			return 0, 0, err
		}

		return 0, uint(h), nil
	}

	return 0, 0, fmt.Errorf("unexpected size: %s", t)
}

var ThumbSizes = []ThumbSize{"200h", "400h", "600w", "2400w", "300w", "1200w"}

type ThumbValue struct {
	ImageID int    `db:"image_id"`
	Width   uint   `db:"width"`
	Height  uint   `db:"height"`
	Data    []byte `db:"data"`
}

package photo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/swaggest/jsonschema-go"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type Thumbnailer interface {
	Thumbnail(ctx context.Context, image Image, size ThumbSize) (Thumb, error)
}

type Thumb struct {
	uniq.Head
	Width    uint   `db:"width"`
	Height   uint   `db:"height"`
	Data     []byte `db:"data"`
	FilePath string `db:"file_path"`
}

func (t Thumb) ReadSeeker() io.ReadSeeker {
	return bytes.NewReader(t.Data)
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

var ThumbSizes = []ThumbSize{"2400w", "1200w", "600w", "300w", "200h", "400h"}

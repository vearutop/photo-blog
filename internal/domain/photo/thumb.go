package photo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
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
	Width        uint      `db:"width" json:"width"`
	Height       uint      `db:"height" json:"height"`
	Data         []byte    `db:"data" json:"data,omitempty"`
	FilePath     string    `db:"file_path" json:"file_path,omitempty"`
	Format       ThumbSize `db:"-" json:"format,omitempty"`
	SpriteFile   string    `db:"-" json:"sprite_file,omitempty"`
	SpriteOffset int       `db:"-" json:"sprite_offset,omitempty"`
}

func (t Thumb) ReadSeeker() io.ReadSeeker {
	if t.FilePath != "" {
		return nil
	}

	return bytes.NewReader(t.Data)
}

func (t Thumb) Reader() (io.ReadCloser, error) {
	if t.FilePath != "" {
		f, err := os.Open(t.FilePath)
		if err != nil {
			return nil, err
		}

		return f, nil
	}

	return io.NopCloser(bytes.NewReader(t.Data)), nil
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

var (
	ThumbSizes = []ThumbSize{"2400w", "1200w", "600w", "300w", "200h", "400h"}
	ThumbMid   = ThumbSize("1200w")
)

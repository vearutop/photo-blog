package photo

import (
	"fmt"
	"github.com/swaggest/jsonschema-go"
	"strconv"
	"strings"
	"time"
)

// Album describes database mapping.
type Album struct {
	Identity
	Time
	AlbumData
}

type AlbumData struct {
	Title string `db:"title" formData:"title" json:"title"`
	Name  string `db:"name" formData:"name" json:"name"`
}

type Image struct {
	Identity
	Time
	ImageData
}

type ImageData struct {
	Hash   int64  `db:"hash"`
	Size   int64  `db:"size"`
	Path   string `db:"path"`
	Width  int64  `db:"width"`
	Height int64  `db:"height"`
}

func (i ImageData) StringHash() string {
	return strconv.FormatUint(uint64(i.Hash), 36)
}

type Thumb struct {
	Identity
	Time
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

	return 0, 0, fmt.Errorf("unexpected size: %s", t)
}

var ThumbSizes = []ThumbSize{"600w", "2400w", "300w", "1200w"}

type ThumbValue struct {
	ImageID int    `db:"image_id"`
	Width   uint   `db:"width"`
	Height  uint   `db:"height"`
	Data    []byte `db:"data"`
}

type Identity struct {
	ID int `db:"id,omitempty,serialIdentity" json:"id"`
}

type Time struct {
	CreatedAt time.Time `db:"created_at,omitempty" json:"created_at"`
}

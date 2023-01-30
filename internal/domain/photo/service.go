package photo

import (
	"context"
	"io"
)

type Thumbnailer interface {
	Thumbnail(ctx context.Context, image Image, size ThumbSize) (io.ReadSeeker, error)
}

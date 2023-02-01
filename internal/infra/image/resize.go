package image

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"sync"
	"time"

	"github.com/bool64/brick/opencensus"
	"github.com/nfnt/resize"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"go.opencensus.io/trace"
)

func NewThumbnailer() *Thumbnailer {
	return &Thumbnailer{}
}

type Thumbnailer struct {
	mu        sync.Mutex
	lastPath  string
	lastImage image.Image
}

func (t *Thumbnailer) Thumbnail(ctx context.Context, i photo.Image, size photo.ThumbSize) (res io.ReadSeeker, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ctx, finish := opencensus.AddSpan(ctx,
		trace.StringAttribute("path", i.Path),
		trace.StringAttribute("size", string(size)),
	)
	defer finish(&err)

	r := Resizer{
		Quality: 85,
		Interp:  resize.Lanczos2,
	}

	buf := bytes.NewBuffer(nil)
	w, h, err := size.WidthHeight()
	if err != nil {
		return nil, err
	}

	var img image.Image

	if t.lastPath == i.Path {
		img = t.lastImage
	} else {
		t.lastImage = nil
		t.lastPath = ""

		img, err = r.loadJPEG(ctx, i.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to load JPEG: %w", err)
		}

		t.lastPath = i.Path
		t.lastImage = img

		go func() {
			time.Sleep(20 * time.Second)
			t.mu.Lock()
			defer t.mu.Unlock()
			if t.lastPath == i.Path {
				t.lastImage = nil
				t.lastPath = ""
			}
		}()
	}

	if err := r.resizeJPEG(ctx, img, buf, w, h); err != nil {
		return nil, fmt.Errorf("failed to resize: %w", err)
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func (t *Thumbnailer) PhotoThumbnailer() photo.Thumbnailer {
	return t
}

type Resizer struct {
	Quality int
	Interp  resize.InterpolationFunction
}

func (r *Resizer) loadJPEG(ctx context.Context, fn string) (img image.Image, err error) {
	ctx, finish := opencensus.AddSpan(ctx)
	defer finish(&err)

	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	// decode jpeg into image.Image
	return jpeg.Decode(f)
}

func (r *Resizer) resizeJPEG(ctx context.Context, img image.Image, dst io.Writer, width, height uint) (err error) {
	ctx, finish := opencensus.AddSpan(ctx)
	defer finish(&err)

	// image to width 1000 using Lanczos resampling
	// and preserve aspect ratio
	m := resize.Resize(width, height, img, r.Interp)

	o := jpeg.Options{}
	o.Quality = r.Quality

	// write new image to file
	return jpeg.Encode(dst, m, &o)
}

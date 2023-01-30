package image

import (
	"bytes"
	"context"
	"fmt"
	"github.com/nfnt/resize"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"image/jpeg"
	"io"
	"os"
)

func NewThumbnailer() *Thumbnailer {
	return &Thumbnailer{
		sem: make(chan struct{}, 1),
	}
}

type Thumbnailer struct {
	sem chan struct{}
}

func (t *Thumbnailer) Thumbnail(ctx context.Context, image photo.Image, size photo.ThumbSize) (io.ReadSeeker, error) {
	t.sem <- struct{}{}
	defer func() {
		<-t.sem
	}()

	f, err := os.Open(image.Path)
	if err != nil {
		return nil, err
	}

	r := Resizer{
		Quality: 85,
		Interp:  resize.Lanczos2,
	}

	buf := bytes.NewBuffer(nil)
	w, h, err := size.WidthHeight()
	if err != nil {
		return nil, err
	}

	if err := r.ResizeJPEG(f, buf, w, h); err != nil {
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

func (r *Resizer) ResizeJPEG(src io.Reader, dst io.Writer, width, height uint) error {
	// decode jpeg into image.Image
	img, err := jpeg.Decode(src)
	if err != nil {
		return err
	}

	// image to width 1000 using Lanczos resampling
	// and preserve aspect ratio
	m := resize.Resize(width, height, img, r.Interp)

	o := jpeg.Options{}
	o.Quality = r.Quality

	// write new image to file
	return jpeg.Encode(dst, m, &o)
}

package image

import (
	"github.com/nfnt/resize"
	"image/jpeg"
	"io"
)

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

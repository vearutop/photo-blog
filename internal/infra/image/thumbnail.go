package image

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/bool64/brick/opencensus"
	"github.com/bool64/ctxd"
	"github.com/nfnt/resize"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"go.opencensus.io/trace"
)

type ThumbnailerDeps interface {
	CtxdLogger() ctxd.Logger
}

func NewThumbnailer(deps ThumbnailerDeps) *Thumbnailer {
	return &Thumbnailer{deps: deps}
}

type Thumbnailer struct {
	deps ThumbnailerDeps

	mu sync.Mutex
}

type thumbCtxKey struct{}

func LargerThumbToContext(ctx context.Context, th photo.Thumb) context.Context {
	return context.WithValue(ctx, thumbCtxKey{}, &th)
}

func LargerThumbFromContext(ctx context.Context) *photo.Thumb {
	if th, ok := ctx.Value(thumbCtxKey{}).(*photo.Thumb); ok {
		return th
	}

	return nil
}

func makeThumbnail(
	ctx context.Context,
	i photo.Image,
	size photo.ThumbSize,
) (th photo.Thumb, err error) {
	w, h, err := size.WidthHeight()
	if err != nil {
		return th, err
	}

	lt := LargerThumbFromContext(ctx)
	if lt != nil && lt.Format == size {
		th = *lt

		if th.CreatedAt.IsZero() {
			th.CreatedAt = time.Now()
		}

		if th.Width == 0 || lt.Height == 0 {
			i, err := thumbJPEG(ctx, th)
			if err != nil {
				return th, err
			}

			th.Width, th.Height = uint(i.Bounds().Dx()), uint(i.Bounds().Dy())
		}

		return th, nil
	}

	th.Hash = i.Hash
	th.CreatedAt = time.Now()

	r := Resizer{
		Quality: 85,
		Interp:  resize.Lanczos2,
	}

	buf := bytes.NewBuffer(nil)

	img, err := loadImage(ctx, i, w, h)
	if err != nil {
		return th, err
	}

	w, h, err = r.resizeJPEG(ctx, img, buf, w, h)
	if err != nil {
		return th, fmt.Errorf("resize: %w", err)
	}

	th.Width = w
	th.Height = h
	th.Format = size

	th.Data = buf.Bytes()
	th.Size = len(th.Data)

	return th, nil
}

func (t *Thumbnailer) Thumbnail(ctx context.Context, i photo.Image, size photo.ThumbSize) (th photo.Thumb, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	start := time.Now()
	ctx = ctxd.AddFields(ctx, "img", i.Path, "hash", i.Hash, "size", size)
	t.deps.CtxdLogger().Debug(ctx, "starting thumb")

	ctx, finish := opencensus.AddSpan(ctx,
		trace.StringAttribute("path", i.Path),
		trace.StringAttribute("size", string(size)),
	)
	defer finish(&err)

	th.Hash = i.Hash

	dir := "thumb/" + string(size) + "/" + i.Hash.String()[:1] + "/"
	filePath := dir + i.Hash.String() + ".jpg"

	// Check existing thumb file.
	if s, err := os.Lstat(filePath); err == nil && s.Size() > 0 {
		th.FilePath = filePath
		i, err := loadJPEG(ctx, filePath)
		if err != nil {
			return th, err
		}

		th.Width, th.Height = uint(i.Bounds().Dx()), uint(i.Bounds().Dy())
		return th, nil
	}

	th, err = makeThumbnail(ctx, i, size)
	if err != nil {
		return th, err
	}

	if len(th.Data) > 1e5 {
		if err := os.MkdirAll(dir, 0o700); err == nil {
			if err := os.WriteFile(filePath, th.Data, 0o600); err == nil {
				th.FilePath = filePath
				th.Data = nil
			} else {
				t.deps.CtxdLogger().Error(ctx, "failed to write thumb file",
					"error", err, "filePath", filePath)
			}
		} else {
			t.deps.CtxdLogger().Error(ctx, "failed to ensure dir", "error", err, "dir", dir)
		}
	}

	elapsed := time.Since(start)
	t.deps.CtxdLogger().Debug(ctx, "thumb done", "elapsed", elapsed.String())

	return th, nil
}

func rot90(jimg image.Image) *image.RGBA {
	bounds := jimg.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y
	bounds.Max.X, bounds.Max.Y = bounds.Max.Y, bounds.Max.X

	dimg := image.NewRGBA(bounds)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			org := jimg.At(x, y)
			dimg.Set(height-y, x, org)
		}
	}

	return dimg
}

func rot180(jimg image.Image) *image.RGBA {
	bounds := jimg.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	dimg := image.NewRGBA(bounds)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dimg.Set(width-1-x, height-1-y, jimg.At(x, y))
		}
	}

	return dimg
}

func loadImage(ctx context.Context, i photo.Image, w, h uint) (image.Image, error) {
	lt := LargerThumbFromContext(ctx)
	if lt != nil && (lt.Width >= w && lt.Height >= h) {
		img, err := thumbJPEG(ctx, *lt)
		if err != nil {
			return nil, fmt.Errorf("decoding larger thumb: %w", err)
		}

		return img, nil
	}

	var (
		img image.Image
		err error
	)

	if len(i.Settings.HTTPSources) > 0 {
		img, err = loadJPEGFromURL(ctx, i.Settings.HTTPSources[0])
	} else {
		img, err = loadJPEG(ctx, i.Path)
	}

	if err != nil {
		return img, fmt.Errorf("failed to load JPEG: %w", err)
	}

	switch i.Settings.Rotate {
	case 90:
		img = rot90(img)
	case 180:
		img = rot180(img)
	case 270:
		img = rot180(img)
		img = rot90(img)
	}

	return img, nil
}

func (t *Thumbnailer) PhotoThumbnailer() photo.Thumbnailer {
	return t
}

type Resizer struct {
	Quality int
	Interp  resize.InterpolationFunction
}

func loadJPEG(ctx context.Context, fn string) (img image.Image, err error) {
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

func loadJPEGFromURL(ctx context.Context, u string) (img image.Image, err error) {
	ctx, finish := opencensus.AddSpan(ctx)
	defer finish(&err)

	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// decode jpeg into image.Image
	return jpeg.Decode(resp.Body)
}

func thumbJPEG(ctx context.Context, t photo.Thumb) (image.Image, error) {
	if t.FilePath != "" {
		return loadJPEG(ctx, t.FilePath)
	}

	return jpeg.Decode(t.ReadSeeker())
}

func (r *Resizer) resizeJPEG(ctx context.Context, img image.Image, dst io.Writer, width, height uint) (w, h uint, err error) {
	ctx, finish := opencensus.AddSpan(ctx)
	defer finish(&err)

	m := resize.Resize(width, height, img, r.Interp)

	o := jpeg.Options{}
	o.Quality = r.Quality

	w, h = uint(m.Bounds().Dx()), uint(m.Bounds().Dy())

	// write new image to file
	return w, h, jpeg.Encode(dst, m, &o)
}

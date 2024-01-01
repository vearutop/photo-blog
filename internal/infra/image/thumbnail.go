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
	"github.com/bool64/ctxd"
	"github.com/nfnt/resize"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/infra/files"
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

func largerThumbFromContext(ctx context.Context) *photo.Thumb {
	if th, ok := ctx.Value(thumbCtxKey{}).(*photo.Thumb); ok {
		return th
	}

	return nil
}

func (t *Thumbnailer) Thumbnail(ctx context.Context, i photo.Image, size photo.ThumbSize) (th photo.Thumb, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	start := time.Now()
	ctx = ctxd.AddFields(ctx, "img", i.Path, "size", size)
	t.deps.CtxdLogger().Info(ctx, "starting thumb")

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
		return th, err
	}

	img, err := t.loadImage(ctx, i, w, h)
	if err != nil {
		return th, err
	}

	w, h, err = r.resizeJPEG(ctx, img, buf, w, h)
	if err != nil {
		return th, fmt.Errorf("failed to resize: %w", err)
	}

	th.Width = w
	th.Height = h
	th.Hash = i.Hash

	if len(buf.Bytes()) > 1e5 {
		dir := "thumb/" + string(size) + "/" + i.Hash.String()[:1] + "/"
		if err := os.MkdirAll(dir, 0o700); err == nil {
			filePath := dir + i.Hash.String() + ".jpg"

			if err := os.WriteFile(filePath, buf.Bytes(), 0o600); err == nil {
				th.FilePath = filePath
			} else {
				t.deps.CtxdLogger().Error(ctx, "failed to write thumb file",
					"error", err, "filePath", filePath)
			}
		} else {
			t.deps.CtxdLogger().Error(ctx, "failed to ensure dir", "error", err, "dir", dir)
		}
	}

	if th.FilePath == "" {
		th.Data = buf.Bytes()
	}

	elapsed := time.Since(start)
	t.deps.CtxdLogger().Info(ctx, "thumb done", "elapsed", elapsed.String())

	// TODO: add proper concurrency/rate limiter here to avoid resource overuse on limited systems.
	if elapsed > time.Second {
		time.Sleep(100 * time.Millisecond)
	}

	return th, nil
}

func (t *Thumbnailer) loadImage(ctx context.Context, i photo.Image, w, h uint) (image.Image, error) {
	lt := largerThumbFromContext(ctx)
	if lt != nil && (lt.Width > w || lt.Height > h) {
		img, err := thumbJPEG(ctx, *lt)
		if err != nil {
			return nil, fmt.Errorf("decoding larger thumb: %w", err)
		}

		return img, nil
	}

	time.Sleep(time.Second) // To reduce CPU load. TODO: remove?

	img, err := loadJPEG(ctx, files.Path(i.Path))
	if err != nil {
		return img, fmt.Errorf("failed to load JPEG: %w", err)
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

func thumbJPEG(ctx context.Context, t photo.Thumb) (image.Image, error) {
	if t.FilePath != "" {
		return loadJPEG(ctx, files.Path(t.FilePath))
	}

	return jpeg.Decode(t.ReadSeeker())
}

func (r *Resizer) resizeJPEG(ctx context.Context, img image.Image, dst io.Writer, width, height uint) (w, h uint, err error) {
	ctx, finish := opencensus.AddSpan(ctx)
	defer finish(&err)

	// image to width 1000 using Lanczos resampling
	// and preserve aspect ratio
	m := resize.Resize(width, height, img, r.Interp)

	o := jpeg.Options{}
	o.Quality = r.Quality

	w, h = uint(m.Bounds().Dx()), uint(m.Bounds().Dy())

	// write new image to file
	return w, h, jpeg.Encode(dst, m, &o)
}

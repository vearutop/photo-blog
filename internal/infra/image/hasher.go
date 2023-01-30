package image

import (
	"context"
	"github.com/bool64/ctxd"
	"github.com/cespare/xxhash/v2"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"io"
	"os"
)

func NewHasher(upstream photo.ImageEnsurer, log ctxd.Logger) *Hasher {
	return &Hasher{
		upstream: upstream,
		log:      log,
	}
}

type Hasher struct {
	upstream photo.ImageEnsurer
	log      ctxd.Logger
}

func (h *Hasher) PhotoImageEnsurer() photo.ImageEnsurer {
	return h
}

func (h *Hasher) Ensure(ctx context.Context, value photo.ImageData) (photo.Image, error) {
	if value.Hash != 0 {
		return h.upstream.Ensure(ctx, value)
	}

	f, err := os.Open(value.Path)
	if err != nil {
		return photo.Image{}, ctxd.WrapError(ctx, err, "failed to open image",
			"path", value.Path)
	}
	closed := false
	defer func() {
		if !closed {
			err := f.Close()
			if err != nil && h.log != nil {
				h.log.Error(ctx, "failed to close image file after reading",
					"path", value.Path, "error", err)
			}
		}
	}()

	x := xxhash.New()

	value.Size, err = io.Copy(x, f)
	if err != nil {
		return photo.Image{}, err
	}

	value.Hash = int64(x.Sum64())

	closed = true
	if err = f.Close(); err != nil {
		return photo.Image{}, err
	}

	return h.upstream.Ensure(ctx, value)
}

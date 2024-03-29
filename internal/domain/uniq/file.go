package uniq

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/bool64/ctxd"
	"github.com/cespare/xxhash/v2"
)

type File struct {
	Head
	Size int64  `db:"size" title:"File Size" json:"size" readOnly:"true"`
	Path string `db:"path" title:"File Path" json:"path" readOnly:"true"`
}

func (v *File) SetPath(ctx context.Context, path string) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return ctxd.WrapError(ctx, err, "set file path",
			"path", v.Path)
	}
	closed := false
	defer func() {
		if !closed {
			clErr := f.Close()
			if clErr != nil && err == nil {
				err = ctxd.WrapError(ctx, err, "set file path, close file after reading",
					"path", v.Path)
			}
		}
	}()

	x := xxhash.New()

	v.Size, err = io.Copy(x, f)
	if err != nil {
		return err
	}

	v.Path = path
	v.Hash = Hash(x.Sum64())

	closed = true
	if err = f.Close(); err != nil {
		return err
	}

	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now()
	}

	return nil
}

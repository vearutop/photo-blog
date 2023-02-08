package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

const (
	// ThumbsTable is the name of the table.
	ThumbsTable = "thumbs"
)

func NewThumbRepository(storage *sqluct.Storage, upstream photo.Thumbnailer) *ThumbRepository {
	tr := &ThumbRepository{
		upstream: upstream,
	}

	tr.StorageOf = sqluct.Table[photo.Thumb](storage, ThumbsTable)

	return tr
}

// ThumbRepository saves thumbnails to database.
type ThumbRepository struct {
	upstream photo.Thumbnailer
	sqluct.StorageOf[photo.Thumb]
}

func (tr *ThumbRepository) PhotoThumbnailer() photo.Thumbnailer {
	return tr
}

func (tr *ThumbRepository) Thumbnail(ctx context.Context, image photo.Image, size photo.ThumbSize) (io.ReadSeeker, error) {
	w, h, err := size.WidthHeight()
	if err != nil {
		return nil, err
	}

	th, err := tr.Find(ctx, image.ID, w, h)
	if err == nil {
		return bytes.NewReader(th.Data), nil
	}

	t, err := tr.upstream.Thumbnail(ctx, image, size)
	if err != nil {
		return nil, err
	}

	d, err := io.ReadAll(t)
	if err != nil {
		return nil, err
	}

	th, err = tr.Add(ctx, photo.ThumbValue{
		ImageID: image.ID,
		Width:   w,
		Height:  h,
		Data:    d,
	})
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(th.Data), nil
}

func (tr *ThumbRepository) Find(ctx context.Context, imageID int, width, height uint) (photo.Thumb, error) {
	q := tr.SelectStmt().
		Where(tr.Eq(&tr.R.ImageID, imageID))

	if width > 0 {
		q = q.Where(tr.Fmt("%s = %d", &tr.R.Width, width))
	}

	if height > 0 {
		q = q.Where(tr.Fmt("%s = %d", &tr.R.Height, height))
	}

	row, err := tr.Get(ctx, q)
	if err != nil {
		return photo.Thumb{}, fmt.Errorf("find thumb by image id %q and size %dx%d: %w",
			imageID, width, height, augmentErr(err))
	}

	return row, nil
}

func (tr *ThumbRepository) Add(ctx context.Context, value photo.ThumbValue) (photo.Thumb, error) {
	r := photo.Thumb{}
	r.ThumbValue = value
	r.CreatedAt = time.Now()

	if id, err := tr.InsertRow(ctx, r); err != nil {
		return r, ctxd.WrapError(ctx, augmentErr(err), "store thumbnail")
	} else {
		r.ID = int(id)
	}

	return r, nil
}

func (tr *ThumbRepository) PhotoThumbAdder() photo.ThumbAdder {
	return tr
}

package storage

import (
	"context"
	"fmt"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"time"
)

const (
	// ThumbsTable is the name of the table.
	ThumbsTable = "thumbs"
)

func NewThumbRepository(storage *sqluct.Storage) *ThumbRepository {
	tr := &ThumbRepository{}

	tr.storage = storage
	tr.row = &photo.Thumb{}
	tr.rf = storage.Ref()
	tr.rf.AddTableAlias(tr.row, ThumbsTable)

	return tr
}

// ThumbRepository saves thumbnails to database.
type ThumbRepository struct {
	storage *sqluct.Storage
	rf      *sqluct.Referencer
	row     *photo.Thumb
}

func (tr *ThumbRepository) Find(ctx context.Context, imageID int, width, height int) (photo.Thumb, error) {
	row := photo.Thumb{}

	q := tr.storage.SelectStmt(ThumbsTable, row).
		Where(tr.rf.Fmt("%s = %d", &tr.row.ImageID, imageID))

	if width > 0 {
		q = q.Where(tr.rf.Fmt("%s = %d", &tr.row.Width, width))
	}

	if height > 0 {
		q = q.Where(tr.rf.Fmt("%s = %d", &tr.row.Height, height))
	}

	if err := tr.storage.Select(ctx, q, &row); err != nil {
		return photo.Thumb{}, fmt.Errorf("find thumb by image id %q and size %dx%d: %w", imageID, width, height, err)
	}

	return row, nil
}

func (tr *ThumbRepository) Add(ctx context.Context, value photo.ThumbValue) (photo.Thumb, error) {
	r := photo.Thumb{}
	r.ThumbValue = value
	r.CreatedAt = time.Now()

	q := tr.storage.InsertStmt(ThumbsTable, r)

	if res, err := tr.storage.Exec(ctx, q); err != nil {
		return r, ctxd.WrapError(ctx, err, "store thumbnail")
	} else {
		id, err := res.LastInsertId()
		if err != nil {
			return r, ctxd.WrapError(ctx, err, "get created thumb id")
		}

		r.ID = int(id)
	}

	return r, nil
}

func (tr *ThumbRepository) PhotoThumbAdder() photo.ThumbAdder {
	return tr
}

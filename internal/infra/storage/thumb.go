package storage

import (
	"context"
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
	return &ThumbRepository{
		storage: storage,
	}
}

// ThumbRepository saves thumbnails to database.
type ThumbRepository struct {
	storage *sqluct.Storage
}

func (gs *ThumbRepository) Add(ctx context.Context, value photo.ThumbValue) (photo.Thumb, error) {
	r := photo.Thumb{}
	r.ThumbValue = value
	r.CreatedAt = time.Now()

	q := gs.storage.InsertStmt(ThumbsTable, r)

	if res, err := gs.storage.Exec(ctx, q); err != nil {
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

func (gs *ThumbRepository) PhotoThumbAdder() photo.ThumbAdder {
	return gs
}

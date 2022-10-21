package storage

import (
	"context"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"time"
)

const (
	// ImagesTable is the name of the table.
	ImagesTable = "images"
)

func NewImageRepository(storage *sqluct.Storage) *ImageRepository {
	return &ImageRepository{
		storage: storage,
	}
}

// ImageRepository saves images to database.
type ImageRepository struct {
	storage *sqluct.Storage
}

func (gs *ImageRepository) Add(ctx context.Context, value photo.ImageData) (photo.Image, error) {
	r := photo.Image{}
	r.ImageData = value
	r.CreatedAt = time.Now()

	q := gs.storage.InsertStmt(ImagesTable, r)

	if res, err := gs.storage.Exec(ctx, q); err != nil {
		return r, ctxd.WrapError(ctx, err, "store image")
	} else {
		id, err := res.LastInsertId()
		if err != nil {
			return r, ctxd.WrapError(ctx, err, "get created image id")
		}

		r.ID = int(id)
	}

	return r, nil
}

func (gs *ImageRepository) PhotoImageAdder() photo.ImageAdder {
	return gs
}

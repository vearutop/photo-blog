package storage

import (
	"context"
	"errors"
	"github.com/Masterminds/squirrel"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"modernc.org/sqlite"
	"time"
)

const (
	// ImagesTable is the name of the table.
	ImagesTable = "images"
)

func NewImageRepository(storage *sqluct.Storage) *ImageRepository {
	return &ImageRepository{
		StorageOf: sqluct.Table[photo.Image](storage, ImagesTable),
	}
}

// ImageRepository saves images to database.
type ImageRepository struct {
	sqluct.StorageOf[photo.Image]
}

func (ir *ImageRepository) FindByHash(ctx context.Context, hash int64) (photo.Image, error) {
	q := ir.SelectStmt().Where(squirrel.Eq{ir.Ref(&ir.R.Hash): hash})
	return ir.Get(ctx, q)
}

func (ir *ImageRepository) Ensure(ctx context.Context, value photo.ImageData) (photo.Image, error) {
	if value.Hash == 0 {
		return photo.Image{}, errors.New("missing image hash")
	}

	r := photo.Image{}
	r.ImageData = value
	r.CreatedAt = time.Now()

	q := ir.SelectStmt().Where(squirrel.Eq{ir.Ref(&ir.R.Hash): r.Hash})
	if i, err := ir.Get(ctx, q); err == nil {
		return i, nil
	}

	if id, err := ir.InsertRow(ctx, r); err != nil {
		var se *sqlite.Error

		if errors.As(err, &se) {
			if se.Code() == 2067 {
				err = status.Wrap(err, status.AlreadyExists)
			}
		}

		return r, ctxd.WrapError(ctx, err, "store image")
	} else {
		r.ID = int(id)
	}

	return r, nil
}

func (ir *ImageRepository) PhotoImageEnsurer() photo.ImageEnsurer {
	return ir
}

func (ir *ImageRepository) PhotoImageFinder() photo.ImageFinder {
	return ir
}

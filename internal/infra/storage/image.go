package storage

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

const (
	// ImagesTable is the name of the table.
	ImagesTable = "images"

	ErrMissingHash = ctxd.SentinelError("missing image hash")
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

func (ir *ImageRepository) FindByHash(ctx context.Context, hash photo.Hash) (photo.Image, error) {
	q := ir.SelectStmt().Where(squirrel.Eq{ir.Ref(&ir.R.Hash): hash})
	return augmentResErr(ir.Get(ctx, q))
}

func (ir *ImageRepository) Ensure(ctx context.Context, value photo.ImageData) (photo.Image, error) {
	if value.Hash == 0 {
		return photo.Image{}, ErrMissingHash
	}

	r := photo.Image{}
	r.ImageData = value
	r.CreatedAt = time.Now()

	q := ir.SelectStmt().Where(squirrel.Eq{ir.Ref(&ir.R.Hash): r.Hash})
	if i, err := ir.Get(ctx, q); err == nil {
		return i, nil
	}

	if id, err := ir.InsertRow(ctx, r); err != nil {
		return r, ctxd.WrapError(ctx, augmentErr(err), "store image")
	} else {
		r.ID = int(id)
	}

	return r, nil
}

func (ir *ImageRepository) Update(ctx context.Context, value photo.ImageData) error {
	if value.Hash == 0 {
		return ErrMissingHash
	}

	q := ir.UpdateStmt(value).Where(squirrel.Eq{ir.Ref(&ir.R.Hash): value.Hash})
	_, err := q.ExecContext(ctx)

	return augmentErr(err)
}

func (ir *ImageRepository) PhotoImageEnsurer() photo.ImageEnsurer {
	return ir
}

func (ir *ImageRepository) PhotoImageUpdater() photo.ImageUpdater {
	return ir
}

func (ir *ImageRepository) PhotoImageFinder() photo.ImageFinder {
	return ir
}

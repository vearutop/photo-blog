package storage

import (
	"context"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

const (
	// ImagesTable is the name of the table.
	ImagesTable = "images"
)

func NewImagesRepository(storage *sqluct.Storage) *ImagesRepository {
	return &ImagesRepository{
		StorageOf: sqluct.Table[photo.Images](storage, ImagesTable),
	}
}

// ImagesRepository saves images to database.
type ImagesRepository struct {
	sqluct.StorageOf[photo.Images]
}

func (ir *ImagesRepository) FindByHash(ctx context.Context, hash uniq.Hash) (photo.Images, error) {
	q := ir.SelectStmt().Where(ir.Eq(&ir.R.Hash, hash))
	return augmentResErr(ir.Get(ctx, q))
}

func (ir *ImagesRepository) FindAll(ctx context.Context) ([]photo.Images, error) {
	return augmentResErr(ir.List(ctx, ir.SelectStmt()))
}

func (ir *ImagesRepository) Ensure(ctx context.Context, value photo.ImageData) (photo.Images, error) {
	if value.Hash == 0 {
		return photo.Images{}, ErrMissingHash
	}

	r := photo.Images{}
	r.ImageData = value
	r.CreatedAt = time.Now()

	q := ir.SelectStmt().Where(ir.Eq(&ir.R.Hash, r.Hash))
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

func (ir *ImagesRepository) Update(ctx context.Context, value photo.ImageData) error {
	if value.Hash == 0 {
		return ErrMissingHash
	}

	q := ir.UpdateStmt(value).Where(ir.Eq(&ir.R.Hash, value.Hash))
	_, err := q.ExecContext(ctx)

	return augmentErr(err)
}

func (ir *ImagesRepository) PhotoImageEnsurer() photo.ImageEnsurer {
	return ir
}

func (ir *ImagesRepository) PhotoImageUpdater() photo.ImageUpdater {
	return ir
}

func (ir *ImagesRepository) PhotoImageFinder() photo.ImageFinder {
	return ir
}

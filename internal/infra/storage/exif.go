package storage

import (
	"context"
	"github.com/Masterminds/squirrel"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"time"
)

const (
	// ExifTable is the name of the table.
	ExifTable = "exif"
)

func NewExifRepository(storage *sqluct.Storage) *ExifRepository {
	return &ExifRepository{
		StorageOf: sqluct.Table[photo.Exif](storage, ExifTable),
	}
}

// ExifRepository saves images to database.
type ExifRepository struct {
	sqluct.StorageOf[photo.Exif]
}

func (ir *ExifRepository) FindByHash(ctx context.Context, hash photo.Hash) (photo.Exif, error) {
	q := ir.SelectStmt().Where(squirrel.Eq{ir.Ref(&ir.R.Hash): hash})
	return augmentResErr(ir.Get(ctx, q))
}

func (ir *ExifRepository) Ensure(ctx context.Context, value photo.Exif) error {
	if value.Hash == 0 {
		return ErrMissingHash
	}

	r := value

	if _, err := ir.FindByHash(ctx, value.Hash); err == nil {
		// Update.
		if _, err := ir.UpdateStmt(value).Where(squirrel.Eq{ir.Ref(&ir.R.Hash): value.Hash}).ExecContext(ctx); err != nil {
			return ctxd.WrapError(ctx, augmentErr(err), "update exif")
		}
	} else {
		// Insert.
		r.CreatedAt = time.Now()
		if _, err := ir.InsertRow(ctx, r, sqluct.InsertIgnore); err != nil {
			return ctxd.WrapError(ctx, augmentErr(err), "insert exif")
		}
	}

	return nil
}

func (ir *ExifRepository) PhotoExifEnsurer() photo.ExifEnsurer {
	return ir
}

func (ir *ExifRepository) PhotoExifFinder() photo.ExifFinder {
	return ir
}

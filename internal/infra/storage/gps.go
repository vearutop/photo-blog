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
	// GpsTable is the name of the table.
	GpsTable = "gps"
)

func NewGpsRepository(storage *sqluct.Storage) *GpsRepository {
	return &GpsRepository{
		StorageOf: sqluct.Table[photo.Gps](storage, GpsTable),
	}
}

// GpsRepository saves images to database.
type GpsRepository struct {
	sqluct.StorageOf[photo.Gps]
}

func (ir *GpsRepository) FindByHash(ctx context.Context, hash photo.Hash) (photo.Gps, error) {
	q := ir.SelectStmt().Where(squirrel.Eq{ir.Ref(&ir.R.Hash): hash})
	return augmentResErr(ir.Get(ctx, q))
}

func (ir *GpsRepository) Ensure(ctx context.Context, value photo.Gps) error {
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

func (ir *GpsRepository) PhotoGpsEnsurer() photo.GpsEnsurer {
	return ir
}

func (ir *GpsRepository) PhotoGpsFinder() photo.GpsFinder {
	return ir
}

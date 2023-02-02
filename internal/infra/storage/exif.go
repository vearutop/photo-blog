package storage

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"modernc.org/sqlite"
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
	return ir.Get(ctx, q)
}

func (ir *ExifRepository) Ensure(ctx context.Context, value photo.Exif) error {
	if value.Hash == 0 {
		return ErrMissingHash
	}

	r := value
	r.CreatedAt = time.Now()

	if _, err := ir.InsertRow(ctx, r); err != nil {
		var se *sqlite.Error

		if errors.As(err, &se) {
			if se.Code() == 2067 || se.Code() == 1555 {
				err = status.Wrap(err, status.AlreadyExists)
			}
		}

		return ctxd.WrapError(ctx, err, "store exif")
	}

	return nil
}

func (ir *ExifRepository) PhotoExifEnsurer() photo.ExifEnsurer {
	return ir
}

func (ir *ExifRepository) PhotoExifFinder() photo.ExifFinder {
	return ir
}

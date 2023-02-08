package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"modernc.org/sqlite"
)

func augmentErr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return status.Wrap(err, status.NotFound)
	}

	var se *sqlite.Error

	if errors.As(err, &se) {
		if se.Code() == 2067 || se.Code() == 1555 {
			err = status.Wrap(err, status.AlreadyExists)
		}
	}

	return err
}

func augmentReturnErr[V any](_ V, err error) error {
	return augmentErr(err)
}

func augmentResErr[V any](res V, err error) (V, error) {
	return res, augmentErr(err)
}

type hashedRepo[V any, T interface {
	*V
	HashPtr() *photo.Hash
	SetCreatedAt(t time.Time)
}] struct {
	sqluct.StorageOf[V]
}

func (ir *hashedRepo[V, T]) FindByHash(ctx context.Context, hash photo.Hash) (V, error) {
	q := ir.SelectStmt().Where(ir.Eq(T(ir.R).HashPtr(), hash))
	return augmentResErr(ir.Get(ctx, q))
}

func (ir *hashedRepo[V, T]) Ensure(ctx context.Context, value V) error {
	v := T(&value)
	h := *v.HashPtr()

	if h == 0 {
		return ErrMissingHash
	}

	if _, err := ir.FindByHash(ctx, h); err == nil {
		// Update.
		if _, err := ir.UpdateStmt(value).Where(ir.Eq(T(ir.R).HashPtr(), h)).ExecContext(ctx); err != nil {
			return ctxd.WrapError(ctx, augmentErr(err), "update")
		}
	} else {
		// Insert.
		v.SetCreatedAt(time.Now())
		if _, err := ir.InsertRow(ctx, value, sqluct.InsertIgnore); err != nil {
			return ctxd.WrapError(ctx, augmentErr(err), "insert")
		}
	}

	return nil
}

package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"modernc.org/sqlite"
)

const ErrMissingHash = ctxd.SentinelError("missing hash")

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
	HashPtr() *uniq.Hash
	SetCreatedAt(t time.Time)
	GetCreatedAt() time.Time
}] struct {
	sqluct.StorageOf[V]
}

func (ir *hashedRepo[V, T]) FindByHash(ctx context.Context, hash uniq.Hash) (V, error) {
	q := ir.SelectStmt().Where(ir.Eq(T(ir.R).HashPtr(), hash))
	return augmentResErr(ir.Get(ctx, q))
}

func (ir *hashedRepo[V, T]) FindAll(ctx context.Context) ([]V, error) {
	return augmentResErr(ir.List(ctx, ir.SelectStmt()))
}

func (ir *hashedRepo[V, T]) Ensure(ctx context.Context, value V) (V, error) {
	v := T(&value)
	h := *v.HashPtr()

	if h == 0 {
		return value, ErrMissingHash
	}

	if val, err := ir.FindByHash(ctx, h); err == nil {
		// Update.
		vv := T(&val)
		v.SetCreatedAt(vv.GetCreatedAt())

		if _, err := ir.UpdateStmt(value).Where(ir.Eq(T(ir.R).HashPtr(), h)).ExecContext(ctx); err != nil {
			return value, ctxd.WrapError(ctx, augmentErr(err), "update")
		}
	} else {
		// Insert.
		v.SetCreatedAt(time.Now())
		if _, err := ir.InsertRow(ctx, value); err != nil {
			return value, ctxd.WrapError(ctx, augmentErr(err), "insert")
		}
	}

	return value, nil
}

func (ir *hashedRepo[V, T]) Add(ctx context.Context, value V) error {
	v := T(&value)
	h := *v.HashPtr()

	if h == 0 {
		return ErrMissingHash
	}

	v.SetCreatedAt(time.Now())

	return augmentReturnErr(ir.InsertRow(ctx, value))
}

func (ir *hashedRepo[V, T]) Update(ctx context.Context, value V) error {
	v := T(&value)
	h := *v.HashPtr()

	if h == 0 {
		return ErrMissingHash
	}

	return augmentReturnErr(ir.UpdateStmt(value).Where(ir.Eq(T(ir.R).HashPtr(), h)).ExecContext(ctx))
}

func (ir *hashedRepo[V, T]) Delete(ctx context.Context, h uniq.Hash) error {
	return augmentReturnErr(ir.DeleteStmt().Where(ir.Eq(T(ir.R).HashPtr(), h)).ExecContext(ctx))
}

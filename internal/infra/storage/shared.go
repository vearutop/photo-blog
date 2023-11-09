package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
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

func augmentReturnErr[V any](res V, err error) error {
	err = augmentErr(err)
	if err != nil {
		err = fmt.Errorf("find %T: %w", res, err)
	}

	return err
}

func augmentResErr[V any](res V, err error) (V, error) {
	err = augmentErr(err)
	if err != nil {
		err = fmt.Errorf("find %T: %w", res, err)
	}

	return res, err
}

type hashedRepo[V any, T interface {
	*V

	HashPtr() *uniq.Hash
	CreatedAtPtr() *time.Time

	SetCreatedAt(t time.Time)
	GetCreatedAt() time.Time
}] struct {
	sqluct.StorageOf[V]
}

func (ir *hashedRepo[V, T]) hashCol() *uniq.Hash {
	return T(ir.R).HashPtr()
}

func (ir *hashedRepo[V, T]) hashEq(h uniq.Hash) squirrel.Eq {
	return ir.Eq(ir.hashCol(), h)
}

func (ir *hashedRepo[V, T]) Exists(ctx context.Context, hash uniq.Hash) (bool, error) {
	col := ir.Col(ir.hashCol())

	q := ir.SelectStmt(func(options *sqluct.Options) {
		options.Columns = []string{col}
	}).Where(ir.hashEq(hash))

	_, err := ir.Get(ctx, q)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return false, augmentErr(err)
}

func (ir *hashedRepo[V, T]) FindByHash(ctx context.Context, hash uniq.Hash) (V, error) {
	q := ir.SelectStmt().Where(ir.hashEq(hash))
	return augmentResErr(ir.Get(ctx, q))
}

func (ir *hashedRepo[V, T]) FindByHashes(ctx context.Context, hashes ...uniq.Hash) ([]V, error) {
	q := ir.SelectStmt().Where(ir.Eq(ir.hashCol(), hashes))

	return augmentResErr(ir.List(ctx, q))
}

func (ir *hashedRepo[V, T]) findBaseByHash(ctx context.Context, hash uniq.Hash) (V, error) {
	q := ir.SelectStmt(func(options *sqluct.Options) {
		options.Columns = []string{ir.Col(T(ir.R).CreatedAtPtr())}
	}).Where(ir.hashEq(hash))
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

	if val, err := ir.findBaseByHash(ctx, h); err == nil {
		// Update.
		vv := T(&val)
		v.SetCreatedAt(vv.GetCreatedAt())

		if _, err := ir.UpdateStmt(value).Where(ir.hashEq(h)).ExecContext(ctx); err != nil {
			return value, ctxd.WrapError(ctx, augmentErr(err), "update")
		}
	} else {
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return value, ctxd.WrapError(ctx, augmentErr(err), "find")
		}

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

	return augmentReturnErr(ir.UpdateStmt(value).Where(ir.hashEq(h)).ExecContext(ctx))
}

func (ir *hashedRepo[V, T]) Delete(ctx context.Context, h uniq.Hash) error {
	return augmentReturnErr(ir.DeleteStmt().Where(ir.hashEq(h)).ExecContext(ctx))
}

package hashed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/bool64/ctxd"
	"github.com/bool64/dbwrap"
	"github.com/bool64/sqluct"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"modernc.org/sqlite"
)

const ErrMissingHash = ctxd.SentinelError("missing hash")

func AugmentErr(err error) error {
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

func AugmentReturnErr[V any](res V, err error) error {
	err = AugmentErr(err)
	if err != nil {
		err = fmt.Errorf("find %T: %w", res, err)
	}

	return err
}

func AugmentResErr[V any](res V, err error) (V, error) {
	err = AugmentErr(err)
	if err != nil {
		err = fmt.Errorf("find %T: %w", res, err)
	}

	return res, err
}

type Entity interface {
	HashPtr() *uniq.Hash
	CreatedAtPtr() *time.Time

	SetCreatedAt(t time.Time)
	GetCreatedAt() time.Time
}

type Repo[V any, T interface {
	*V
	Entity
}] struct {
	mu sync.Mutex
	sqluct.StorageOf[V]

	// Prepare is optional, it is called on the value to validate/prepare before create/update.
	Prepare func(ctx context.Context, v *V) error
}

func (ir *Repo[V, T]) hashCol() *uniq.Hash {
	return T(ir.R).HashPtr()
}

func (ir *Repo[V, T]) hashEq(h uniq.Hash) squirrel.Eq {
	return ir.Eq(ir.hashCol(), h)
}

func (ir *Repo[V, T]) Exists(ctx context.Context, hash uniq.Hash) (bool, error) {
	var v V
	ctx = dbwrap.WithCaller(ctx, fmt.Sprintf("Exists:%T", v))

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

	return false, AugmentErr(err)
}

func (ir *Repo[V, T]) FindByHash(ctx context.Context, hash uniq.Hash) (V, error) {
	var v V
	ctx = dbwrap.WithCaller(ctx, fmt.Sprintf("FindByHash:%T", v))

	return ir.findByHash(ctx, hash)
}

func (ir *Repo[V, T]) findByHash(ctx context.Context, hash uniq.Hash) (V, error) {
	q := ir.SelectStmt().Where(ir.hashEq(hash))
	return AugmentResErr(ir.Get(ctx, q))
}

func (ir *Repo[V, T]) FindByHashes(ctx context.Context, hashes ...uniq.Hash) ([]V, error) {
	var v V
	ctx = dbwrap.WithCaller(ctx, fmt.Sprintf("FindByHashes:%T", v))

	if len(hashes) == 0 {
		return nil, nil
	}

	q := ir.SelectStmt().Where(ir.Eq(ir.hashCol(), hashes))

	return AugmentResErr(ir.List(ctx, q))
}

func (ir *Repo[V, T]) findBaseByHash(ctx context.Context, hash uniq.Hash) (V, error) {
	q := ir.SelectStmt(func(options *sqluct.Options) {
		options.Columns = []string{ir.Col(T(ir.R).CreatedAtPtr())}
	}).Where(ir.hashEq(hash))
	return AugmentResErr(ir.Get(ctx, q))
}

func (ir *Repo[V, T]) FindAll(ctx context.Context) ([]V, error) {
	var v V
	ctx = dbwrap.WithCaller(ctx, fmt.Sprintf("FindAll:%T", v))

	return AugmentResErr(ir.List(ctx, ir.SelectStmt()))
}

func (ir *Repo[V, T]) Ensure(ctx context.Context, value V, options ...uniq.EnsureOption[V]) (V, error) {
	var x V
	ctx = dbwrap.WithCaller(ctx, fmt.Sprintf("Ensure:%T", x))

	v := T(&value)
	h := *v.HashPtr()

	if h == 0 {
		return value, ErrMissingHash
	}

	ir.mu.Lock()
	defer ir.mu.Unlock()

	var opts []func(o *sqluct.Options)

	if val, err := ir.findByHash(ctx, h); err == nil {
		// Update.
		vv := T(&val)
		v.SetCreatedAt(vv.GetCreatedAt())

		skipUpdate := false

		for _, o := range options {
			if o.OnUpdate != nil {
				opts = append(opts, func(opt *sqluct.Options) {
					o.OnUpdate(ir.StorageOf, opt)
				})
			}

			if o.Prepare != nil {
				skipUpdate = o.Prepare(v, vv)
			}
		}

		if skipUpdate {
			return value, nil
		}

		if ir.Prepare != nil {
			if err := ir.Prepare(ctx, &value); err != nil {
				return value, fmt.Errorf("prepare value: %w", err)
			}
		}

		q := ir.UpdateStmt(value, opts...).Where(ir.hashEq(h))
		stmt, args, err := q.ToSql()
		if err != nil {
			return value, fmt.Errorf("prepare update statement: %w", err)
		}

		ctx = ctxd.AddFields(ctx, "statement", stmt, "args", args)

		if _, err := q.ExecContext(ctx); err != nil {
			return value, ctxd.WrapError(ctx, AugmentErr(err), "update")
		}
	} else {
		if !errors.Is(err, sql.ErrNoRows) {
			return value, ctxd.WrapError(ctx, AugmentErr(err), "find")
		}

		for _, o := range options {
			if o.OnInsert != nil {
				opts = append(opts, func(opt *sqluct.Options) {
					o.OnInsert(ir.StorageOf, opt)
				})
			}

			if o.Prepare != nil {
				o.Prepare(&value, nil)
			}
		}

		// Insert.
		v.SetCreatedAt(time.Now())

		if ir.Prepare != nil {
			if err := ir.Prepare(ctx, &value); err != nil {
				return value, fmt.Errorf("prepare value: %w", err)
			}
		}

		if _, err := ir.InsertRow(ctx, value, opts...); err != nil {
			return value, ctxd.WrapError(ctx, AugmentErr(err), "insert")
		}
	}

	return value, nil
}

func (ir *Repo[V, T]) Add(ctx context.Context, value V) error {
	var x V
	ctx = dbwrap.WithCaller(ctx, fmt.Sprintf("Add:%T", x))

	v := T(&value)
	h := *v.HashPtr()

	if h == 0 {
		return ErrMissingHash
	}

	v.SetCreatedAt(time.Now())

	if ir.Prepare != nil {
		if err := ir.Prepare(ctx, v); err != nil {
			return fmt.Errorf("prepare value: %w", err)
		}
	}

	return AugmentReturnErr(ir.InsertRow(ctx, value))
}

func (ir *Repo[V, T]) Update(ctx context.Context, value V) error {
	var x V
	ctx = dbwrap.WithCaller(ctx, fmt.Sprintf("Update:%T", x))

	v := T(&value)
	h := *v.HashPtr()

	if h == 0 {
		return ErrMissingHash
	}

	if ir.Prepare != nil {
		if err := ir.Prepare(ctx, v); err != nil {
			return fmt.Errorf("prepare value: %w", err)
		}
	}

	return AugmentReturnErr(ir.UpdateStmt(value).Where(ir.hashEq(h)).ExecContext(ctx))
}

func (ir *Repo[V, T]) Delete(ctx context.Context, h uniq.Hash) error {
	var x V
	ctx = dbwrap.WithCaller(ctx, fmt.Sprintf("Delete:%T", x))

	return AugmentReturnErr(ir.DeleteStmt().Where(ir.hashEq(h)).ExecContext(ctx))
}

package storage

import (
	"context"
	"fmt"
	"github.com/bool64/ctxd"
	"github.com/vearutop/photo-blog/internal/domain/text"
	"time"

	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

const (
	// LabelTable is the name of the table.
	LabelTable = "label"
)

func NewLabelRepository(storage *sqluct.Storage) *LabelRepository {
	return &LabelRepository{
		hashedRepo: hashedRepo[text.Label, *text.Label]{
			StorageOf: sqluct.Table[text.Label](storage, LabelTable),
		},
	}
}

// LabelRepository saves images to database.
type LabelRepository struct {
	hashedRepo[text.Label, *text.Label]
}

func (tr *LabelRepository) Ensure(ctx context.Context, value text.Label) (text.Label, error) {
	h := value.Hash

	if h == 0 {
		return value, ErrMissingHash
	}

	if val, err := tr.Find(ctx, value.Locale, h); err == nil && len(val) == 1 {
		// Update.
		vv := val[0]
		value.SetCreatedAt(vv.GetCreatedAt())

		if _, err := tr.UpdateStmt(value).
			Where(tr.Eq(&tr.R.Hash, h)).
			Where(tr.Eq(&tr.R.Locale, value.Locale)).
			ExecContext(ctx); err != nil {
			return value, ctxd.WrapError(ctx, augmentErr(err), "update")
		}
	} else {
		// Insert.
		value.SetCreatedAt(time.Now())
		if _, err := tr.InsertRow(ctx, value); err != nil {
			return value, ctxd.WrapError(ctx, augmentErr(err), "insert")
		}
	}

	return value, nil
}

func (tr *LabelRepository) Find(ctx context.Context, locale string, hashes ...uniq.Hash) ([]text.Label, error) {
	q := tr.SelectStmt().
		Where(tr.Eq(&tr.R.Hash, hashes)).
		Where(tr.Eq(&tr.R.Locale, locale))

	rows, err := tr.List(ctx, q)
	if err != nil {
		return []text.Label{}, fmt.Errorf("find label by hashes %v and locale %s: %w",
			hashes, locale, augmentErr(err))
	}

	return rows, nil
}

func (tr *LabelRepository) Delete(ctx context.Context, locale string, hashes ...uniq.Hash) error {
	q := tr.DeleteStmt().
		Where(tr.Eq(&tr.R.Hash, hashes)).
		Where(tr.Eq(&tr.R.Locale, locale))

	return augmentReturnErr(q.ExecContext(ctx))
}

func (tr *LabelRepository) TextLabelFinder() text.LabelFinder {
	return tr
}

func (tr *LabelRepository) TextLabelEnsurer() text.LabelEnsurer {
	return tr
}

func (tr *LabelRepository) TextLabelDeleter() text.LabelDeleter {
	return tr
}

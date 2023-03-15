package storage

import (
	"context"
	"github.com/Masterminds/squirrel"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"time"
)

const (
	// VisitorTable is the name of the table.
	VisitorTable = "visitor"
)

func NewVisitorRepository(storage *sqluct.Storage) *VisitorRepository {
	return &VisitorRepository{
		hashedRepo: hashedRepo[auth.Visitor, *auth.Visitor]{
			StorageOf: sqluct.Table[auth.Visitor](storage, VisitorTable),
		},
	}
}

// VisitorRepository saves images to database.
type VisitorRepository struct {
	hashedRepo[auth.Visitor, *auth.Visitor]
}

func (ir *VisitorRepository) AddHits(ctx context.Context, hash uniq.Hash, increment int, latest time.Time) error {
	return augmentReturnErr(ir.UpdateStmt(nil).
		Set(ir.Ref(&ir.R.Hits), squirrel.Expr(ir.Ref(&ir.R.Hits)+"+ ?", increment)).
		Set(ir.Ref(&ir.R.Latest), latest).
		Where(ir.Eq(ir.R.Hash, hash)).
		ExecContext(ctx))
}

func (ir *VisitorRepository) AuthVisitorEnsurer() uniq.Ensurer[auth.Visitor] {
	return ir
}

func (ir *VisitorRepository) AuthVisitorFinder() uniq.Finder[auth.Visitor] {
	return ir
}

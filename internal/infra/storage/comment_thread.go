package storage

import (
	"context"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/comment"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

const (
	// ThreadTable is the name of the table.
	ThreadTable = "thread"
)

func NewThreadRepository(storage *sqluct.Storage) *ThreadRepository {
	return &ThreadRepository{
		hashedRepo: hashedRepo[comment.Thread, *comment.Thread]{
			StorageOf: sqluct.Table[comment.Thread](storage, ThreadTable),
		},
	}
}

// ThreadRepository saves images to database.
type ThreadRepository struct {
	hashedRepo[comment.Thread, *comment.Thread]
}

func (ir *ThreadRepository) CommentThreadEnsurer() uniq.Ensurer[comment.Thread] {
	return ir
}

func (ir *ThreadRepository) CommentThreadFinder() uniq.Finder[comment.Thread] {
	return ir
}

func (ir *ThreadRepository) FindChronoMessages(ctx context.Context, hash uniq.Hash) []

func (ir *ThreadRepository) FindMessages(ctx context.Context, thread comment.Thread) {
	ir.R.Type
	ir.SelectStmt().Where()
}

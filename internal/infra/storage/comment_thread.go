package storage

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/comment"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/storage/hashed"
)

const (
	// ThreadTable is the name of the table.
	ThreadTable = "thread"
)

func NewThreadRepository(storage *sqluct.Storage) *ThreadRepository {
	return &ThreadRepository{
		Repo: hashed.Repo[comment.Thread, *comment.Thread]{
			StorageOf: sqluct.Table[comment.Thread](storage, ThreadTable),
		},
	}
}

// ThreadRepository saves images to database.
type ThreadRepository struct {
	hashed.Repo[comment.Thread, *comment.Thread]
}

func (ir *ThreadRepository) CommentThreadEnsurer() uniq.Ensurer[comment.Thread] {
	return ir
}

func (ir *ThreadRepository) CommentThreadFinder() uniq.Finder[comment.Thread] {
	return ir
}

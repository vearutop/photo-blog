package storage

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/comment"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/storage/hashed"
)

const (
	// MessageTable is the name of the table.
	MessageTable = "message"
)

func NewMessageRepository(storage *sqluct.Storage) *MessageRepository {
	return &MessageRepository{
		Repo: hashed.Repo[comment.Message, *comment.Message]{
			StorageOf: sqluct.Table[comment.Message](storage, MessageTable),
		},
	}
}

// MessageRepository saves images to database.
type MessageRepository struct {
	hashed.Repo[comment.Message, *comment.Message]
}

func (ir *MessageRepository) CommentMessageEnsurer() uniq.Ensurer[comment.Message] {
	return ir
}

func (ir *MessageRepository) CommentMessageFinder() uniq.Finder[comment.Message] {
	return ir
}

package usecase

import (
	"context"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/comment"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

type getMessagesDeps interface {
	service.CommentMessageFinderProvider
	service.CommentThreadFinderProvider
}

func GetMessages(deps getMessagesDeps) usecase.Interactor {
	type getMessagesRequest struct {
		Type string    `query:"type"`
		Hash uniq.Hash `query:"hash"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getMessagesRequest, output *o) error {
		thread := comment.Thread{}

		deps.CommentThreadFinder().FindByHash()
	})

	return u
}

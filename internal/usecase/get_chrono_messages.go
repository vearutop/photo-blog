package usecase

import (
	"context"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/comment"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

func GetChronoMessages(deps getMessagesDeps) usecase.Interactor {
	type getMessagesRequest struct {
		Hash uniq.Hash `path:"hash"`
	}

	type chronoMessages struct {
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getMessagesRequest, output *chronoMessages) error {
		thread := comment.Thread{}

		deps.CommentThreadFinder().FindAll()
	})

	return u
}

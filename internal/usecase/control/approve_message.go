package control

import (
	"context"
	"github.com/bool64/sqluct"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/comment"
	"github.com/vearutop/photo-blog/internal/domain/site"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

type approveMessageDeps interface {
	service.SiteVisitorEnsurerProvider
	service.CommentMessageFinderProvider
	service.CommentMessageEnsurerProvider
}

func ApproveMessage(deps approveMessageDeps) usecase.Interactor {
	type approveMessageInput struct {
		MessageHash    uniq.Hash `json:"message_hash"`
		ApproveVisitor bool      `json:"approve_visitor"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input approveMessageInput, output *struct{}) error {
		message, err := deps.CommentMessageFinder().FindByHash(ctx, input.MessageHash)
		if err != nil {
			return err
		}

		if input.ApproveVisitor {
			v := site.Visitor{}
			v.Hash = message.VisitorHash
			v.Approved = true

			_, err = deps.SiteVisitorEnsurer().Ensure(ctx, v, uniq.EnsureOption[site.Visitor]{
				OnUpdate: func(st sqluct.StorageOf[site.Visitor], o *sqluct.Options) {
					o.Columns = []string{st.Col(&st.R.Approved)}
				},
			})

			return err
		}

		message.Approved = true
		_, err = deps.CommentMessageEnsurer().Ensure(ctx, message, uniq.EnsureOption[comment.Message]{
			OnUpdate: func(st sqluct.StorageOf[comment.Message], o *sqluct.Options) {
				o.Columns = []string{st.Col(&st.R.Approved)}
			},
		})

		return err
	})

	return u
}

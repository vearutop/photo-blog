package usecase

import (
	"context"
	"time"

	"github.com/bool64/sqluct"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/internal/domain/comment"
	"github.com/vearutop/photo-blog/internal/domain/site"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

type addMessageDeps interface {
	service.SiteVisitorFinderProvider
	service.SiteVisitorEnsurerProvider
	service.CommentThreadEnsurerProvider
	service.CommentMessageEnsurerProvider
}

func AddMessage(deps addMessageDeps) usecase.Interactor {
	type addMessageRequest struct {
		Name        string     `json:"name"`
		Type        string     `json:"type" enum:"image,album"`
		RelatedHash uniq.Hash  `json:"related_hash"`
		RelatedAt   *time.Time `json:"related_at"`
		Text        string     `json:"text"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input addMessageRequest, output *comment.Message) (err error) {
		thread := comment.Thread{
			Type:        input.Type,
			RelatedHash: input.RelatedHash,
			RelatedAt:   input.RelatedAt,
		}
		th := input.Type + input.RelatedHash.String()
		if input.RelatedAt != nil {
			th += input.RelatedAt.String()
		}
		thread.Hash = uniq.StringHash(th)

		if _, err := deps.CommentThreadEnsurer().Ensure(ctx, thread, uniq.EnsureOption[comment.Thread]{
			Prepare: func(candidate *comment.Thread, existing *comment.Thread) (skipUpdate bool) {
				return true
			},
		}); err != nil {
			return err
		}

		visitor := auth.VisitorFromContext(ctx)
		message := comment.Message{}
		message.Hash = uniq.StringHash(thread.Hash.String() + visitor.String() + input.Text)
		message.ThreadHash = thread.Hash
		message.VisitorHash = visitor
		message.Text = input.Text

		*output, err = deps.CommentMessageEnsurer().Ensure(ctx, message, uniq.EnsureOption[comment.Message]{
			Prepare: func(candidate *comment.Message, existing *comment.Message) (skipUpdate bool) {
				return true
			},
		})
		if err != nil {
			return err
		}

		if input.Name != "" {
			v := site.Visitor{Name: input.Name}
			v.Hash = visitor

			_, err = deps.SiteVisitorEnsurer().Ensure(ctx, v, uniq.EnsureOption[site.Visitor]{
				OnUpdate: func(st sqluct.StorageOf[site.Visitor], o *sqluct.Options) {
					o.Columns = []string{st.Col(&st.R.Name)}
				},
			})
		}

		return err
	})

	return u
}

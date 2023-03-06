package text

import (
	"context"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type LabelEnsurer interface {
	Ensure(ctx context.Context, value Label) (Label, error)
}

type LabelDeleter interface {
	Delete(ctx context.Context, locale string, hashes ...uniq.Hash) error
}

type LabelFinder interface {
	Find(ctx context.Context, locale string, hashes ...uniq.Hash) ([]Label, error)
}

type Label struct {
	uniq.Head

	Locale string `db:"locale" json:"locale"`
	Text   string `db:"text" json:"text"`
}

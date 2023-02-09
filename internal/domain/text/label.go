package text

import (
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type Label struct {
	uniq.Head

	Type string `db:"type"`
	Val  string `db:"val"`
}

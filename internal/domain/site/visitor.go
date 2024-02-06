package site

import "github.com/vearutop/photo-blog/internal/domain/uniq"

type Visitor struct {
	uniq.Head

	Approved bool   `db:"approved" json:"approved"`
	Name     string `db:"name" json:"name"`
}

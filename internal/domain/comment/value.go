package comment

import (
	"time"

	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type Message struct {
	uniq.Head

	ThreadHash  uniq.Hash `db:"thread_hash" json:"thread_hash"`
	VisitorHash uniq.Hash `db:"visitor_hash" json:"visitor_hash"`
	Approved    bool      `db:"approved" json:"approved"`
	Text        string    `db:"text" json:"text"`
}

type Thread struct {
	uniq.Head

	Type        string     `db:"type" json:"type"`
	RelatedHash uniq.Hash  `db:"related_hash" json:"related_hash"`
	RelatedAt   *time.Time `db:"related_at" json:"related_at"`
}

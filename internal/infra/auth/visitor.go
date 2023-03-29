package auth

import (
	"time"

	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type Visitor struct {
	uniq.Head
	Latest      time.Time `db:"latest"`
	Hits        int       `db:"hits"`
	UserAgent   string    `db:"user_agent"`
	Referrer    string    `db:"referrer"`
	Destination string    `db:"destination"`
	RemoteAddr  string    `db:"remote_addr"`
}

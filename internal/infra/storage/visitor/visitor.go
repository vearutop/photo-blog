package visitor

import (
	"time"

	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/storage"
)

func newVisitorRepository(st *sqluct.Storage) *visitorRepository {
	return &visitorRepository{
		HashedRepo: storage.HashedRepo[visitor, *visitor]{
			StorageOf: sqluct.Table[visitor](st, visitorTable),
		},
	}
}

// VisitorRepository saves images to database.
type visitorRepository struct {
	storage.HashedRepo[visitor, *visitor]
}

const visitorTable = "visitor"

type visitor struct {
	uniq.Head
	LastSeen  time.Time `db:"last_seen" description:"Last seen"`
	Lang      string    `db:"lang" description:"Visitor lang"`
	IPAddr    string    `db:"ip_addr" description:"Visitor IP address"`
	UserAgent string    `db:"user_agent" description:"Visitor user agent"`
	Device    string    `db:"device" description:"Device"`
	IsBot     bool      `db:"is_bot" description:"Visitor is bot"`
	IsAdmin   bool      `db:"is_admin" description:"Visitor is admin"`
	Referer   string    `db:"referer" description:"Referer"`

	ScreenHeight int     `db:"scr_h" description:"Visitor screen height"`
	ScreenWidth  int     `db:"scr_w" description:"Visitor screen width"`
	PixelRatio   float64 `db:"px_r" description:"Visitor pixel ratio"`
}

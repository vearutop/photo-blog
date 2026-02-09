package visitor

import (
	"time"

	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/storage/hashed"
)

func newVisitorRepository(st *sqluct.Storage) *visitorRepository {
	return &visitorRepository{
		Repo: hashed.Repo[Visitor, *Visitor]{
			StorageOf: sqluct.Table[Visitor](st, visitorTable),
		},
	}
}

// VisitorRepository saves images to database.
type visitorRepository struct {
	hashed.Repo[Visitor, *Visitor]
}

const visitorTable = "visitor"

type Visitor struct {
	uniq.Head
	LastSeen  time.Time `db:"last_seen" description:"Last seen"`
	Lang      string    `db:"lang" description:"Visitor lang"`
	IPAddr    string    `db:"ip_addr" description:"Visitor IP address"`
	IP        string    `db:"ip" description:"Visitor IP"`
	UserAgent string    `db:"user_agent" description:"Visitor user agent"`
	Device    string    `db:"device" description:"Device"`
	IsBot     bool      `db:"is_bot" description:"Visitor is bot"`
	IsAdmin   bool      `db:"is_admin" description:"Visitor is admin"`
	Referer   string    `db:"referer" description:"Referer"`

	ScreenHeight int     `db:"scr_h" description:"Visitor screen height"`
	ScreenWidth  int     `db:"scr_w" description:"Visitor screen width"`
	PixelRatio   float64 `db:"px_r" description:"Visitor pixel ratio"`

	Country   string  `db:"country" description:"Country"`
	City      string  `db:"city" description:"City"`
	Latitude  float64 `db:"latitude" description:"Latitude"`
	Longitude float64 `db:"longitude" description:"Longitude"`

	Label string `db:"label" description:"Custom label"`
}

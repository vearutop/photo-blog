package photo

import (
	"context"
	"time"

	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type GpsEnsurer interface {
	Ensure(ctx context.Context, value Gps) error
}

type GpsFinder interface {
	FindByHash(ctx context.Context, hash uniq.Hash) (Gps, error)
}

type Gps struct {
	uniq.Head

	Altitude  float64   `db:"altitude" json:"altitude"`
	Longitude float64   `db:"longitude" json:"longitude"`
	Latitude  float64   `db:"latitude" json:"latitude"`
	GpsTime   time.Time `db:"time" json:"time"`
}

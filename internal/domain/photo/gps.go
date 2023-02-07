package photo

import (
	"context"
	"time"
)

type GpsEnsurer interface {
	Ensure(ctx context.Context, value Gps) error
}

type GpsFinder interface {
	FindByHash(ctx context.Context, hash Hash) (Gps, error)
}

type Gps struct {
	HashHead

	Altitude  float64   `db:"altitude" json:"altitude"`
	Longitude float64   `db:"longitude" json:"longitude"`
	Latitude  float64   `db:"latitude" json:"latitude"`
	GpsTime   time.Time `db:"time" json:"time"`
}

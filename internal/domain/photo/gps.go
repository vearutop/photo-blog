package photo

import (
	"time"

	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type Gps struct {
	uniq.Head

	Altitude  float64   `db:"altitude" json:"altitude"`
	Longitude float64   `db:"longitude" json:"longitude"`
	Latitude  float64   `db:"latitude" json:"latitude"`
	GpsTime   time.Time `db:"time" json:"time"`
}

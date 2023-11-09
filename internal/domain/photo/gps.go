package photo

import (
	"time"

	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type Gps struct {
	uniq.Head

	Altitude  float64   `db:"altitude" title:"Altitude" json:"altitude"`
	Latitude  float64   `db:"latitude" title:"Latitude" json:"latitude"`
	Longitude float64   `db:"longitude" title:"Longitude" json:"longitude"`
	GpsTime   time.Time `db:"time" title:"GPS Timestamp" json:"time"`
}

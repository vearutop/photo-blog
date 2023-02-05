package photo

import (
	"context"
	"time"
)

type ExifEnsurer interface {
	Ensure(ctx context.Context, value Exif) error
}

type ExifFinder interface {
	FindByHash(ctx context.Context, hash Hash) (Exif, error)
}

type Exif struct {
	Time
	Hash Hash `db:"hash" json:"hash" description:"image hash"`

	Rating          int        `db:"rating" json:"rating"`
	ExposureTime    string     `db:"exposure_time" json:"exposure_time"`
	ExposureTimeSec float64    `db:"exposure_time_sec" json:"exposure_time_sec"`
	FNumber         float64    `db:"f_number" json:"f_number"`
	FocalLength     float64    `db:"focal_length" json:"focal_length"`
	ISOSpeed        int        `db:"iso_speed" json:"iso_speed"`
	LensModel       string     `db:"lens_model" json:"lens_model"`
	CameraMake      string     `db:"camera_make" json:"camera_make"`
	CameraModel     string     `db:"camera_model" json:"camera_model"`
	Software        string     `db:"software" json:"software"`
	Digitized       *time.Time `db:"digitized" json:"digitized"`
	ProjectionType  string     `db:"projection_type" json:"projection_type"`
}

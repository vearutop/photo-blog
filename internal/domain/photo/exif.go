package photo

import (
	"time"

	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type Exif struct {
	uniq.Head

	Rating          int        `db:"rating" json:"rating" title:"Rating" minimum:"0" maximum:"5"`
	ExposureTime    string     `db:"exposure_time" json:"exposure_time" title:"Exposure"`
	ExposureTimeSec float64    `db:"exposure_time_sec" json:"exposure_time_sec" title:"Exposure (sec.)"`
	FNumber         float64    `db:"f_number" json:"f_number" title:"Aperture"`
	FocalLength     float64    `db:"focal_length" json:"focal_length" title:"Focal length"`
	ISOSpeed        int        `db:"iso_speed" json:"iso_speed" title:"ISO"`
	LensModel       string     `db:"lens_model" json:"lens_model" title:"Lens"`
	CameraMake      string     `db:"camera_make" json:"camera_make" title:"Manufacturer"`
	CameraModel     string     `db:"camera_model" json:"camera_model" title:"Camera"`
	Software        string     `db:"software" json:"software" title:"Software"`
	Digitized       *time.Time `db:"digitized" json:"digitized" title:"Digitized"`
	ProjectionType  string     `db:"projection_type" json:"projection_type" title:"Projection"`
}

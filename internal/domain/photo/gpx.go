package photo

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/bool64/sqluct"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type GpxSettings struct {
	Name   string  `json:"name,omitempty"`
	MinLat float64 `json:"minLat"`
	MinLon float64 `json:"minLon"`
	MaxLat float64 `json:"maxLat"`
	MaxLon float64 `json:"maxLon"`
}

func (s *GpxSettings) Scan(src any) error {
	if src == nil {
		return nil
	}

	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("unsupported type %T", src)
	}
}

func (s GpxSettings) Value() (driver.Value, error) {
	j, err := json.Marshal(s)

	return string(j), err
}

type Gpx struct {
	uniq.File

	Settings sqluct.JSON[GpxSettings] `db:"settings"`
}

func (g *Gpx) Index() error {
	gpxFile, err := gpx.ParseFile(g.Path)
	if err != nil {
		return err
	}

	s := g.Settings.Val

	s.MinLat = 200
	s.MinLon = 200
	s.MaxLat = -200
	s.MaxLon = -200

	// Analyize/manipulate your track data here...
	for _, track := range gpxFile.Tracks {
		for _, segment := range track.Segments {
			for _, point := range segment.Points {
				if point.Longitude > s.MaxLon {
					s.MaxLon = point.Longitude
				}

				if point.Longitude < s.MinLon {
					s.MinLon = point.Longitude
				}

				if point.Latitude > s.MaxLat {
					s.MaxLat = point.Latitude
				}

				if point.Latitude < s.MinLat {
					s.MinLat = point.Latitude
				}
			}
		}
	}

	g.Settings.Val = s

	return nil
}

package service

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Settings struct {
	SiteTitle         string `json:"site_title" title:"Title" description:"The title of this site."`
	MapTiles          string `json:"map_tiles" title:"Map Tiles" description:"URL to custom map tiles." example:"https://retina-tiles.p.rapidapi.com/local/osm{r}/v1/{z}/{x}/{y}.png?rapidapi-key=YOUR-RAPIDAPI-KEY"`
	MapAttribution    string `json:"map_attribution" title:"Map Attribution" description:"Map tiles attribution."`
	UploadStorage     string `split_words:"true" default:"./photo-blog-data/" json:"upload_storage" title:"Upload Storage" description:"Path to directory where uploaded files are stored."`
	FeaturedAlbumName string `split_words:"true" json:"featured_album_name" title:"Featured Album Name" description:"The name of an album to show on the main page"`
}

func (s *Settings) Scan(src any) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("unsupported type %T", src)
	}
}

func (s Settings) Value() (driver.Value, error) {
	j, err := json.Marshal(s)

	return string(j), err
}

func (s Settings) Title() string {
	return "Settings"
}

package service

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Settings struct {
	SiteTitle         string `json:"site_title" title:"Title" description:"The title of this site."`
	MapTiles          string `json:"map_tiles" title:"Map tiles" description:"URL to custom map tiles." example:"https://retina-tiles.p.rapidapi.com/local/osm{r}/v1/{z}/{x}/{y}.png?rapidapi-key=YOUR-RAPIDAPI-KEY"`
	MapAttribution    string `json:"map_attribution" title:"Map attribution" description:"Map tiles attribution."`
	UploadStorage     string `split_words:"true" default:"./photo-blog-data/" json:"upload_storage" title:"Upload storage" description:"Path to directory where uploaded files are stored."`
	WebDAVStorage     string `split_words:"true" json:"webdav_storage" title:"WebDAV storage" description:"Path to directory with WebDAV access."`
	FeaturedAlbumName string `split_words:"true" json:"featured_album_name" title:"Featured album name" description:"The name of an album to show on the main page."`
	AccessLogFile     string `split_words:"true" default:"./photo-blog-access.log" json:"access_log_file" title:"Path to access log" description:"When not empty, requests to albums and photos will be logged."`
	TagVisitors       bool   `split_words:"true" default:"true" json:"tag_visitors" title:"Tag unique visitors" description:"Unique visitors would be tagged with cookies."`
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

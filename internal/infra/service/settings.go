package service

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Settings struct {
	UploadStorage string `split_words:"true" default:"./photo-blog-data/" json:"upload_storage" title:"Upload Storage" description:"Path to directory where uploaded files are stored."`
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

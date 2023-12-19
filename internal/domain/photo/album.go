package photo

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type AlbumImageAdder interface {
	AddImages(ctx context.Context, albumHash uniq.Hash, imageHashes ...uniq.Hash) error
}

type AlbumImageDeleter interface {
	DeleteImages(ctx context.Context, albumHash uniq.Hash, imageHashes ...uniq.Hash) error
}

type AlbumImageFinder interface {
	FindImages(ctx context.Context, albumHash uniq.Hash) ([]Image, error)
	FindPreviewImages(ctx context.Context, albumHash uniq.Hash, coverImage uniq.Hash, limit uint64) ([]Image, error)
}

type ChronoText struct {
	Time time.Time `json:"time" title:"Timestamp" description:"In RFC 3339 format, e.g. 2020-01-01T01:02:03Z"`
	Text string    `json:"text" title:"Text" formType:"textarea" description:"Text, can contain HTML."`
}

type AlbumSettings struct {
	Description     string       `json:"description,omitempty" formType:"textarea" title:"Description" description:"Description of an album, can contain HTML."`
	GpxTracksHashes []uniq.Hash  `json:"gpx_tracks_hashes,omitempty" title:"GPX track hashes"`
	NewestFirst     bool         `json:"newest_first,omitempty" title:"Newest first" description:"Show newest images at the top."`
	Texts           []ChronoText `json:"texts,omitempty" title:"Chronological texts"`
}

func (s *AlbumSettings) Scan(src any) error {
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

func (s AlbumSettings) Value() (driver.Value, error) {
	j, err := json.Marshal(s)

	return string(j), err
}

type Album struct {
	uniq.Head
	Title      string        `db:"title" json:"title" formType:"textarea" title:"Title" description:"Title of an album."`
	Name       string        `db:"name" json:"name" title:"DstAlbumName" required:"true" readOnly:"true" description:"A slug value that is used in album URL."`
	Public     bool          `db:"public" json:"public" inlineTitle:"Album is public." noTitle:"true" title:"Public" description:"Makes album visible in the main page."`
	CoverImage uniq.Hash     `db:"cover_image" json:"cover_image,omitempty" title:"Cover Image" description:"ImageHash Id of image to use as a cover."`
	Settings   AlbumSettings `db:"settings" json:"settings" title:"Settings" description:"Additional parameters for an album."`
	_          struct{}      `title:"The Album"`
}

func AlbumHash(name string) uniq.Hash {
	return uniq.StringHash(name)
}

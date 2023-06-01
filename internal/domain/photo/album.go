package photo

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"

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
}

type AlbumSettings struct {
	GpxTracksHashes []uniq.Hash `json:"gpx_tracks_hashes,omitempty"`
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
	Title      string        `db:"title" json:"title" title:"Title" formType:"textarea" description:"Title of an album."`
	Name       string        `db:"name" json:"name" title:"Name" required:"true" description:"A slug value that is used in album URL."`
	Public     bool          `db:"public" json:"public" inlineTitle:"Album is public." noTitle:"true" title:"Public" description:"Makes album visible in the main page."`
	CoverImage uniq.Hash     `db:"cover_image" json:"cover_image,omitempty" title:"Cover Image" description:"Hash Id of image to use as a cover."`
	Settings   AlbumSettings `db:"settings" json:"settings" title:"Settings" description:"Additional parameters for an album."`
}

func AlbumHash(name string) uniq.Hash {
	return uniq.StringHash(name)
}

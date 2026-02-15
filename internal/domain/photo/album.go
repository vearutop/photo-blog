package photo

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/form"
	"github.com/vearutop/photo-blog/pkg/txt"
)

// Names of special "albums".
const (
	Orphan   = "orphan-photos"
	Broken   = "broken-photos"
	Favorite = "favorite-photos"
)

type AlbumImageAdder interface {
	AddImages(ctx context.Context, albumHash uniq.Hash, imageHashes ...uniq.Hash) error
	SetAlbumImageTimestamp(ctx context.Context, album uniq.Hash, img uniq.Hash, ts int64) error
}

type AlbumImageDeleter interface {
	DeleteImages(ctx context.Context, albumHash uniq.Hash, imageHashes ...uniq.Hash) error
}

type AlbumImageFinder interface {
	FindImages(ctx context.Context, albumHash uniq.Hash) ([]Image, error)
	FindPreviewImages(ctx context.Context, albumHash uniq.Hash, coverImage uniq.Hash, limit uint64) ([]Image, error)
	FindOrphanImages(ctx context.Context) ([]Image, error)
	SearchImages(ctx context.Context, query string) ([]Image, error)
	FindBrokenImages(ctx context.Context) ([]Image, error)
	FindRemoteImages(ctx context.Context) ([]Image, error)
	FindImageAlbums(ctx context.Context, excludeAlbum uniq.Hash, imageHashes ...uniq.Hash) (map[uniq.Hash][]Album, error)
}

type AlbumSettings struct {
	Description     string              `json:"description,omitempty" formType:"textarea" title:"Description" description:"Description of an album, can contain HTML."`
	GpxTracksHashes []uniq.Hash         `json:"gpx_tracks_hashes,omitempty" items.title:"Hash" title:"GPX track hashes"`
	NewestFirst     bool                `json:"newest_first,omitempty" noTitle:"true" inlineTitle:"Newest first" description:"Show newest images at the top."`
	DailyRulers     bool                `json:"daily_rulers,omitempty" noTitle:"true" inlineTitle:"Daily rulers" description:"Show date splits between the photos."`
	Texts           []txt.Chronological `json:"texts,omitempty" title:"Chronological texts"`
	TextReplaces    txt.Replaces        `json:"text_replaces,omitempty" title:"Text replaces"`

	// Deprecated: TODO remove and implement as separate entity.
	Redirect        string     `json:"redirect,omitempty" title:"Relative or absolute URL to redirect to with HTTP 301 status."`
	HideMap         bool       `json:"hide_map,omitempty" noTitle:"true" inlineTitle:"Hide map on album page."`
	HideDownload    form.Bool3 `json:"hide_download,omitzero" title:"Hide download on album page."`
	HideAISays      bool       `json:"hide_ai_says,omitzero" noTitle:"true" inlineTitle:"Hide AI says on album page."`
	ShowEXIFPreview bool       `json:"show_exif_preview,omitzero" noTitle:"true" inlineTitle:"Show EXIF preview on album page."`

	MapTiles       string `json:"tiles" title:"Map tiles" description:"URL to custom map tiles, overrides app default." example:"https://retina-tiles.p.rapidapi.com/local/osm{r}/v1/{z}/{x}/{y}.png?rapidapi-key=YOUR-RAPIDAPI-KEY"`
	MapAttribution string `json:"attribution" title:"Map attribution" description:"Map tiles attribution, overrides app default."`

	MapMinLon float64 `json:"map_min_lon,omitempty" title:"Map min longitude" description:"Overrides map default boundary."`
	MapMaxLon float64 `json:"map_max_lon,omitempty" title:"Map max longitude" description:"Overrides map default boundary."`
	MapMinLat float64 `json:"map_min_lat,omitempty" title:"Map min latitude" description:"Overrides map default boundary."`
	MapMaxLat float64 `json:"map_max_lat,omitempty" title:"Map max latitude" description:"Overrides map default boundary."`

	CollabKey string `json:"collab_key,omitempty" title:"Collaboration key, when provided, user can add/delete album content."`

	ShowPrivateSubAlbums bool     `json:"show_private_sub_albums,omitempty" title:"Show private sub albums"`
	ShowHiddenSubAlbums  bool     `json:"show_hidden_sub_albums,omitempty" title:"Show hidden sub albums"`
	SubAlbumNames        []string `json:"sub_album_names,omitempty" items.title:"Album Name" title:"Sub albums"`
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
	Name       string        `db:"name" json:"name" title:"Name" required:"true" readOnly:"true" description:"A slug value that is used in album URL."`
	Public     bool          `db:"public" json:"public" inlineTitle:"Album is public." noTitle:"true" title:"Public" description:"Makes album visible in the main page."`
	Hidden     bool          `db:"hidden" json:"hidden" inlineTitle:"Album is hidden." noTitle:"true" title:"Hidden" description:"Makes album invisible in the main page list."`
	CoverImage uniq.Hash     `db:"cover_image" json:"cover_image,omitempty" title:"Cover Image" description:"Hash of image to use as a cover."`
	Settings   AlbumSettings `db:"settings" json:"settings" title:"Settings" description:"Additional parameters for an album."`
	_          struct{}      `title:"The Album"`
}

func AlbumHash(name string) uniq.Hash {
	return uniq.StringHash(name)
}

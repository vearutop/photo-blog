package photo

import "time"

// Album describes database mapping.
type Album struct {
	Identity
	Time
	AlbumData
}

type AlbumData struct {
	Title string `db:"title" formData:"title" json:"title"`
	Name  string `db:"name" formData:"name" json:"name"`
}

type Image struct {
	Identity
	Time
	ImageData
}

type ImageData struct {
	Hash int64  `db:"hash"`
	Path string `db:"path"`
}

type Thumb struct {
	Identity
	Time
	ThumbValue
}

type ThumbValue struct {
	ImageID int    `db:"image_id"`
	Width   int    `db:"width"`
	Height  int    `db:"height"`
	Data    []byte `db:"data"`
}

type Identity struct {
	ID int `db:"id,omitempty,identity" json:"id"`
}

type Time struct {
	CreatedAt time.Time `db:"created_at,omitempty" json:"created_at"`
}

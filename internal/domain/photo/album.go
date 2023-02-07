package photo

import "context"

type AlbumAdder interface {
	Add(ctx context.Context, data AlbumData) (Album, error)
	AddImages(ctx context.Context, albumID int, imageIDs ...int) error // TODO: migrate to image hashes.
}

type AlbumDeleter interface {
	DeleteImages(ctx context.Context, albumID int, imageIDs ...int) error
}

type AlbumFinder interface {
	FindByName(ctx context.Context, name string) (Album, error)
	FindAll(ctx context.Context) ([]Album, error)
	FindImages(ctx context.Context, albumID int) ([]Image, error)
}

// Album describes database mapping.
type Album struct {
	Identity
	Time
	AlbumData
}

type AlbumData struct {
	Title  string `db:"title" formData:"title" json:"title"`
	Name   string `db:"name" formData:"name" json:"name"`
	Public bool   `db:"public" formData:"public" json:"public"`
}

package photo

import (
	"context"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type AlbumAdder interface {
	Add(ctx context.Context, data AlbumData) (Album, error)
	AddImages(ctx context.Context, albumID int, imageIDs ...int) error // TODO: migrate to image hashes.
}

type AlbumUpdater interface {
	Update(ctx context.Context, id int, data AlbumData) error
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
	uniq.Head
	AlbumData
}

type AlbumData struct {
	Title      string    `db:"title" formData:"title" json:"title"`
	Name       string    `db:"name" formData:"name" json:"name" description:"Name value is immutable, it can be deleted with whole record."`
	Public     bool      `db:"public" formData:"public" json:"public"`
	CoverImage uniq.Hash `db:"cover_image" formData:"cover_image" json:"cover_image"`
}

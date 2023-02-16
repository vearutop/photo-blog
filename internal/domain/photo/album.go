package photo

import (
	"context"

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

type Album struct {
	uniq.Head
	Title      string    `db:"title" formData:"title" json:"title"`
	Name       string    `db:"name" formData:"name" json:"name"`
	Public     bool      `db:"public" formData:"public" json:"public"`
	CoverImage uniq.Hash `db:"cover_image" formData:"cover_image" json:"cover_image"`
}

func AlbumHash(name string) uniq.Hash {
	return uniq.StringHash(name)
}

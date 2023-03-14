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
	Title      string    `db:"title" formData:"title" json:"title" title:"Title" formType:"textarea" description:"Title of an album."`
	Name       string    `db:"name" formData:"name" json:"name" title:"Name" description:"A slug value that is used in album URL."`
	Public     bool      `db:"public" formData:"public" json:"public" inlineTitle:"Album is public." noTitle:"true" title:"Public" description:"Makes album visible in the main page."`
	CoverImage uniq.Hash `db:"cover_image" formData:"cover_image" json:"cover_image,omitempty" title:"Cover Image" description:"Hash Id of image to use as a cover."`
}

func AlbumHash(name string) uniq.Hash {
	return uniq.StringHash(name)
}

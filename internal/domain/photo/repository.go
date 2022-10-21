package photo

import "context"

type AlbumAdder interface {
	Add(ctx context.Context, data AlbumData) (Album, error)
	AddImages(ctx context.Context, albumID int, imageIDs ...int) error
}

type AlbumFinder interface {
	FindByName(ctx context.Context, name string) (Album, error)
}

type ImageAdder interface {
	Add(ctx context.Context, value ImageData) (Image, error)
}

type ThumbAdder interface {
	Add(ctx context.Context, value ThumbValue) (Thumb, error)
}

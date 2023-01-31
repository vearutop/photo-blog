package photo

import "context"

type AlbumAdder interface {
	Add(ctx context.Context, data AlbumData) (Album, error)
	AddImages(ctx context.Context, albumID int, imageIDs ...int) error
}

type AlbumFinder interface {
	FindByName(ctx context.Context, name string) (Album, error)
	FindImages(ctx context.Context, albumID int) ([]Image, error)
}

type ImageEnsurer interface {
	Ensure(ctx context.Context, value ImageData) (Image, error)
}

type ImageUpdater interface {
	Update(ctx context.Context, value ImageData) error
}

type ImageFinder interface {
	FindByHash(ctx context.Context, hash int64) (Image, error)
}

type ThumbFinder interface {
	Find(ctx context.Context, imageID int, width, height uint) (Thumb, error)
}

type ThumbAdder interface {
	Add(ctx context.Context, value ThumbValue) (Thumb, error)
}

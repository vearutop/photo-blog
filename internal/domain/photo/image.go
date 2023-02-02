package photo

import (
	"context"
	"strconv"
)

type ImageIndexer interface {
	Index(ctx context.Context, image Image) error
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

type Image struct {
	Identity
	Time
	ImageData
}

type ImageData struct {
	Hash   int64  `db:"hash"`
	Size   int64  `db:"size"`
	Path   string `db:"path"`
	Width  int64  `db:"width"`
	Height int64  `db:"height"`
}

func (i ImageData) StringHash() string {
	return strconv.FormatUint(uint64(i.Hash), 36)
}

func StringHashToInt64(hash string) (int64, error) {
	u, err := strconv.ParseUint(hash, 36, 64)
	return int64(u), err
}

package storage

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

const (
	// ExifTable is the name of the table.
	ExifTable = "exif"
)

func NewExifRepository(storage *sqluct.Storage) *ExifRepository {
	return &ExifRepository{
		HashedRepo: HashedRepo[photo.Exif, *photo.Exif]{
			StorageOf: sqluct.Table[photo.Exif](storage, ExifTable),
		},
	}
}

// ExifRepository saves images to database.
type ExifRepository struct {
	HashedRepo[photo.Exif, *photo.Exif]
}

func (ir *ExifRepository) PhotoExifEnsurer() uniq.Ensurer[photo.Exif] {
	return ir
}

func (ir *ExifRepository) PhotoExifFinder() uniq.Finder[photo.Exif] {
	return ir
}

package storage

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

const (
	// ExifTable is the name of the table.
	ExifTable = "exif"
)

func NewExifRepository(storage *sqluct.Storage) *ExifRepository {
	return &ExifRepository{
		hashedRepo: hashedRepo[photo.Exif, *photo.Exif]{
			StorageOf: sqluct.Table[photo.Exif](storage, ExifTable),
		},
	}
}

// ExifRepository saves images to database.
type ExifRepository struct {
	hashedRepo[photo.Exif, *photo.Exif]
}

func (ir *ExifRepository) PhotoExifEnsurer() photo.ExifEnsurer {
	return ir
}

func (ir *ExifRepository) PhotoExifFinder() photo.ExifFinder {
	return ir
}

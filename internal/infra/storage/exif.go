package storage

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/storage/hashed"
)

const (
	// ExifTable is the name of the table.
	ExifTable = "exif"
)

func NewExifRepository(storage *sqluct.Storage) *ExifRepository {
	return &ExifRepository{
		Repo: hashed.Repo[photo.Exif, *photo.Exif]{
			StorageOf: sqluct.Table[photo.Exif](storage, ExifTable),
		},
	}
}

// ExifRepository saves images to database.
type ExifRepository struct {
	hashed.Repo[photo.Exif, *photo.Exif]
}

func (ir *ExifRepository) PhotoExifEnsurer() uniq.Ensurer[photo.Exif] {
	return ir
}

func (ir *ExifRepository) PhotoExifFinder() uniq.Finder[photo.Exif] {
	return ir
}

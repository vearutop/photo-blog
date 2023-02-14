package storage

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

const (
	// ImageTable is the name of the table.
	ImageTable = "image"
)

func NewImageRepository(storage *sqluct.Storage) *ImageRepository {
	return &ImageRepository{
		hashedRepo: hashedRepo[photo.Image, *photo.Image]{
			StorageOf: sqluct.Table[photo.Image](storage, ImageTable),
		},
	}
}

// ImageRepository saves images to database.
type ImageRepository struct {
	hashedRepo[photo.Image, *photo.Image]
}

func (ir *ImageRepository) PhotoImageEnsurer() uniq.Ensurer[photo.Image] {
	return ir
}

func (ir *ImageRepository) PhotoImageFinder() uniq.Finder[photo.Image] {
	return ir
}

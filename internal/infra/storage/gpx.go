package storage

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

const (
	// GpxTable is the name of the table.
	GpxTable = "gpx"
)

func NewGpxRepository(storage *sqluct.Storage) *GpxRepository {
	return &GpxRepository{
		HashedRepo: HashedRepo[photo.Gpx, *photo.Gpx]{
			StorageOf: sqluct.Table[photo.Gpx](storage, GpxTable),
		},
	}
}

// GpxRepository saves images to database.
type GpxRepository struct {
	HashedRepo[photo.Gpx, *photo.Gpx]
}

func (ir *GpxRepository) PhotoGpxEnsurer() uniq.Ensurer[photo.Gpx] {
	return ir
}

func (ir *GpxRepository) PhotoGpxFinder() uniq.Finder[photo.Gpx] {
	return ir
}

func (ir *GpxRepository) PhotoGpxUpdater() uniq.Updater[photo.Gpx] {
	return ir
}

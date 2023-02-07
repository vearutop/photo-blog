package storage

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

const (
	// GpsTable is the name of the table.
	GpsTable = "gps"
)

func NewGpsRepository(storage *sqluct.Storage) *GpsRepository {
	return &GpsRepository{
		hashedRepo: hashedRepo[photo.Gps, *photo.Gps]{
			StorageOf: sqluct.Table[photo.Gps](storage, GpsTable),
		},
	}
}

// GpsRepository saves images to database.
type GpsRepository struct {
	hashedRepo[photo.Gps, *photo.Gps]
}

func (ir *GpsRepository) PhotoGpsEnsurer() photo.GpsEnsurer {
	return ir
}

func (ir *GpsRepository) PhotoGpsFinder() photo.GpsFinder {
	return ir
}

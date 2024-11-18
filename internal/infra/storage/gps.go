package storage

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/storage/hashed"
)

const (
	// GpsTable is the name of the table.
	GpsTable = "gps"
)

func NewGpsRepository(storage *sqluct.Storage) *GpsRepository {
	return &GpsRepository{
		Repo: hashed.Repo[photo.Gps, *photo.Gps]{
			StorageOf: sqluct.Table[photo.Gps](storage, GpsTable),
		},
	}
}

// GpsRepository saves images to database.
type GpsRepository struct {
	hashed.Repo[photo.Gps, *photo.Gps]
}

func (ir *GpsRepository) PhotoGpsEnsurer() uniq.Ensurer[photo.Gps] {
	return ir
}

func (ir *GpsRepository) PhotoGpsFinder() uniq.Finder[photo.Gps] {
	return ir
}

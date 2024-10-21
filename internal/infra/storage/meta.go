package storage

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

const (
	// MetaTable is the name of the table.
	MetaTable = "meta"
)

func NewMetaRepository(storage *sqluct.Storage) *MetaRepository {
	return &MetaRepository{
		HashedRepo: HashedRepo[photo.Meta, *photo.Meta]{
			StorageOf: sqluct.Table[photo.Meta](storage, MetaTable),
		},
	}
}

// MetaRepository saves meta data of hashed entities to database.
type MetaRepository struct {
	HashedRepo[photo.Meta, *photo.Meta]
}

func (ir *MetaRepository) PhotoMetaEnsurer() uniq.Ensurer[photo.Meta] {
	return ir
}

func (ir *MetaRepository) PhotoMetaFinder() uniq.Finder[photo.Meta] {
	return ir
}

func (ir *MetaRepository) PhotoMetaUpdater() uniq.Updater[photo.Meta] {
	return ir
}

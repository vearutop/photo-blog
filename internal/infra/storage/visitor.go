package storage

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/site"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

const (
	// VisitorTable is the name of the table.
	VisitorTable = "visitor"
)

func NewVisitorRepository(storage *sqluct.Storage) *VisitorRepository {
	return &VisitorRepository{
		HashedRepo: HashedRepo[site.Visitor, *site.Visitor]{
			StorageOf: sqluct.Table[site.Visitor](storage, VisitorTable),
		},
	}
}

// VisitorRepository saves images to database.
type VisitorRepository struct {
	HashedRepo[site.Visitor, *site.Visitor]
}

func (ir *VisitorRepository) SiteVisitorEnsurer() uniq.Ensurer[site.Visitor] {
	return ir
}

func (ir *VisitorRepository) SiteVisitorFinder() uniq.Finder[site.Visitor] {
	return ir
}

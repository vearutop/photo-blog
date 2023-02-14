package storage

import (
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

const (
	// AlbumTable is the name of the table.
	AlbumTable = "album"

	// AlbumImageTable is the name of the table.
	AlbumImageTable = "album_image"
)

// AlbumImage describes database mapping.
type AlbumImage struct {
	AlbumID int `db:"album_hash"`
	ImageID int `db:"image_hash"`
	Weight  int `db:"weight"`
}

func NewAlbumRepository(storage *sqluct.Storage) *AlbumRepository {
	return &AlbumRepository{
		hashedRepo: hashedRepo[photo.Album, *photo.Album]{
			StorageOf: sqluct.Table[photo.Album](storage, AlbumTable),
		},
		ai: sqluct.Table[AlbumImage](storage, AlbumImageTable),
	}
}

// AlbumRepository saves images to database.
type AlbumRepository struct {
	hashedRepo[photo.Album, *photo.Album]
	ai sqluct.StorageOf[AlbumImage]
}

func (ir *AlbumRepository) PhotoAlbumEnsurer() uniq.Ensurer[photo.Album] {
	return ir
}

func (ir *AlbumRepository) PhotoAlbumFinder() uniq.Finder[photo.Album] {
	return ir
}

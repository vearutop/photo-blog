package storage

import (
	"context"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"time"
)

const (
	// AlbumsTable is the name of the table.
	AlbumsTable = "albums"

	// AlbumImagesTable is the name of the table.
	AlbumImagesTable = "album_images"
)

// AlbumImage describes database mapping.
type AlbumImage struct {
	AlbumID int `db:"album_id"`
	ImageID int `db:"image_id"`
}

func NewAlbumRepository(storage *sqluct.Storage) *AlbumRepository {
	ar := &AlbumRepository{}
	ar.s = sqluct.Table[photo.Album](storage, AlbumsTable)

	return ar
}

// AlbumRepository saves albums to database.
type AlbumRepository struct {
	s sqluct.StorageOf[photo.Album]
}

func (ar *AlbumRepository) Add(ctx context.Context, data photo.AlbumData) (photo.Album, error) {
	r := photo.Album{}
	r.AlbumData = data
	r.CreatedAt = time.Now()

	if id, err := ar.s.Insert(ctx, r); err != nil {
		return r, fmt.Errorf("add album: %w", err)
	} else {
		r.ID = int(id)
		return r, nil
	}
}

func (ar *AlbumRepository) FindByName(ctx context.Context, name string) (photo.Album, error) {
	q := ar.s.SelectStmt(AlbumsTable, ar.s.R).
		Where(squirrel.Eq{ar.s.Ref(&ar.s.R.Name): name}).
		Limit(1)

	return ar.s.Get(ctx, q)
}

func (ar *AlbumRepository) AddImages(ctx context.Context, albumID int, imageIDs ...int) error {
	rows := make([]AlbumImage, 0, len(imageIDs))

	for _, imageID := range imageIDs {
		ai := AlbumImage{}
		ai.ImageID = imageID
		ai.AlbumID = albumID

		rows = append(rows, ai)
	}

	q := ar.s.InsertStmt(AlbumImagesTable, rows)

	if _, err := ar.s.Exec(ctx, q); err != nil {
		return ctxd.WrapError(ctx, err, "store album images")
	}

	return nil
}

func (ar *AlbumRepository) PhotoAlbumAdder() photo.AlbumAdder {
	return ar
}

func (ar *AlbumRepository) PhotoAlbumFinder() photo.AlbumFinder {
	return ar
}

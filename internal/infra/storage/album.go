package storage

import (
	"context"
	"fmt"
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
	ar.storage = storage

	ar.row = &photo.Album{}
	ar.rf = storage.Ref()
	ar.rf.AddTableAlias(ar.row, AlbumsTable)

	return ar
}

// AlbumRepository saves albums to database.
type AlbumRepository struct {
	storage *sqluct.Storage
	rf      *sqluct.Referencer
	row     *photo.Album
}

func (gs *AlbumRepository) Add(ctx context.Context, data photo.AlbumData) (photo.Album, error) {
	r := photo.Album{}
	r.AlbumData = data
	r.CreatedAt = time.Now()

	q := gs.storage.InsertStmt(AlbumsTable, r)

	if res, err := gs.storage.Exec(ctx, q); err != nil {
		return r, ctxd.WrapError(ctx, err, "store album")
	} else {
		id, err := res.LastInsertId()
		if err != nil {
			return r, ctxd.WrapError(ctx, err, "get created album id")
		}

		r.ID = int(id)
	}

	return r, nil
}

func (gs *AlbumRepository) FindByName(ctx context.Context, name string) (photo.Album, error) {
	row := photo.Album{}

	q := gs.storage.SelectStmt(AlbumsTable, row).
		Where(gs.rf.Fmt("%s = %s", &gs.row.Name, name))

	if err := gs.storage.Select(ctx, q, &row); err != nil {
		return photo.Album{}, fmt.Errorf("find album by name %q: %w", name, err)
	}

	return row, nil
}

func (gs *AlbumRepository) AddImages(ctx context.Context, albumID int, imageIDs ...int) error {
	rows := make([]AlbumImage, 0, len(imageIDs))

	for _, imageID := range imageIDs {
		ai := AlbumImage{}
		ai.ImageID = imageID
		ai.AlbumID = albumID

		rows = append(rows, ai)
	}

	q := gs.storage.InsertStmt(AlbumImagesTable, rows)

	if _, err := gs.storage.Exec(ctx, q); err != nil {
		return ctxd.WrapError(ctx, err, "store album images")
	}

	return nil
}

func (gs *AlbumRepository) PhotoAlbumAdder() photo.AlbumAdder {
	return gs
}

func (gs *AlbumRepository) PhotoAlbumFinder() photo.AlbumFinder {
	return gs
}

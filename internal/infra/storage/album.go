package storage

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
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
	ar.st = storage
	ar.s = sqluct.Table[photo.Album](storage, AlbumsTable)
	ar.sai = sqluct.Table[AlbumImage](storage, AlbumImagesTable)
	ar.si = sqluct.Table[photo.Image](storage, ImagesTable)

	// Adding AlbumImagesTable to ImagesTable referencer.
	ar.si.Referencer.AddTableAlias(ar.sai.R, AlbumImagesTable)

	return ar
}

// AlbumRepository saves albums to database.
type AlbumRepository struct {
	st  *sqluct.Storage
	s   sqluct.StorageOf[photo.Album]
	sai sqluct.StorageOf[AlbumImage]
	si  sqluct.StorageOf[photo.Image]
}

func (ar *AlbumRepository) FindImages(ctx context.Context, albumID int) ([]photo.Image, error) {
	q := ar.si.SelectStmt().
		InnerJoin(
			ar.si.Fmt("%s ON %s = %s AND %s = ?",
				ar.sai.R, &ar.si.R.ID, &ar.sai.R.ImageID, &ar.sai.R.AlbumID),
			albumID,
		).OrderByClause(ar.si.Ref(&ar.si.R.Path))

	return augmentResErr(ar.si.List(ctx, q))
}

func (ar *AlbumRepository) Add(ctx context.Context, data photo.AlbumData) (photo.Album, error) {
	r := photo.Album{}
	r.AlbumData = data
	r.CreatedAt = time.Now()

	if id, err := ar.s.InsertRow(ctx, r); err != nil {
		return augmentResErr(r, err)
	} else {
		r.ID = int(id)
		return r, nil
	}
}

func (ar *AlbumRepository) FindByName(ctx context.Context, name string) (photo.Album, error) {
	q := ar.s.SelectStmt().
		Where(squirrel.Eq{ar.s.Ref(&ar.s.R.Name): name}).
		Limit(1)

	return augmentResErr(ar.s.Get(ctx, q))
}

func (ar *AlbumRepository) DeleteImages(ctx context.Context, albumID int, imageIDs ...int) error {
	_, err := ar.sai.DeleteStmt().Where(squirrel.Eq{
		ar.sai.Ref(&ar.sai.R.AlbumID): albumID,
		ar.sai.Ref(&ar.sai.R.ImageID): imageIDs,
	}).ExecContext(ctx)

	return augmentErr(err)
}

func (ar *AlbumRepository) AddImages(ctx context.Context, albumID int, imageIDs ...int) error {
	rows := make([]AlbumImage, 0, len(imageIDs))

	for _, imageID := range imageIDs {
		ai := AlbumImage{}
		ai.ImageID = imageID
		ai.AlbumID = albumID

		rows = append(rows, ai)
	}

	if _, err := ar.sai.InsertRows(ctx, rows, sqluct.InsertIgnore); err != nil {
		return ctxd.WrapError(ctx, augmentErr(err), "store album images", "rows", rows)
	}

	return nil
}

func (ar *AlbumRepository) PhotoAlbumAdder() photo.AlbumAdder {
	return ar
}

func (ar *AlbumRepository) PhotoAlbumFinder() photo.AlbumFinder {
	return ar
}

func (ar *AlbumRepository) PhotoAlbumDeleter() photo.AlbumDeleter {
	return ar
}

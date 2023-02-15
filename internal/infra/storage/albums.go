package storage

import (
	"context"
	"time"

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

// AlbumImages describes database mapping.
type AlbumImages struct {
	AlbumID int `db:"album_id"`
	ImageID int `db:"image_id"`
}

func NewAlbumsRepository(storage *sqluct.Storage) *AlbumsRepository {
	ar := &AlbumsRepository{}
	ar.st = storage
	ar.s = sqluct.Table[photo.Albums](storage, AlbumsTable)
	ar.sai = sqluct.Table[AlbumImages](storage, AlbumImagesTable)
	ar.si = sqluct.Table[photo.Images](storage, ImagesTable)

	// Adding AlbumImagesTable to ImagesTable referencer.
	ar.si.Referencer.AddTableAlias(ar.sai.R, AlbumImagesTable)

	return ar
}

// AlbumsRepository saves albums to database.
type AlbumsRepository struct {
	st  *sqluct.Storage
	s   sqluct.StorageOf[photo.Albums]
	sai sqluct.StorageOf[AlbumImages]
	si  sqluct.StorageOf[photo.Images]
}

func (ar *AlbumsRepository) FindImages(ctx context.Context, albumID int) ([]photo.Images, error) {
	q := ar.si.SelectStmt().
		InnerJoin(
			ar.si.Fmt("%s ON %s = %s AND %s = ?",
				ar.sai.R, &ar.si.R.ID, &ar.sai.R.ImageID, &ar.sai.R.AlbumID),
			albumID,
		).OrderByClause(ar.si.Ref(&ar.si.R.Path))

	return augmentResErr(ar.si.List(ctx, q))
}

func (ar *AlbumsRepository) Add(ctx context.Context, data photo.AlbumData) (photo.Albums, error) {
	r := photo.Albums{}
	r.AlbumData = data
	r.CreatedAt = time.Now()

	if id, err := ar.s.InsertRow(ctx, r); err != nil {
		return augmentResErr(r, err)
	} else {
		r.ID = int(id)
		return r, nil
	}
}

func (ar *AlbumsRepository) Update(ctx context.Context, id int, data photo.AlbumData) error {
	return augmentReturnErr(ar.s.UpdateStmt(data).Where(ar.s.Eq(&ar.s.R.ID, id)).ExecContext(ctx))
}

func (ar *AlbumsRepository) FindAll(ctx context.Context) ([]photo.Albums, error) {
	return augmentResErr(ar.s.List(ctx, ar.s.SelectStmt()))
}

func (ar *AlbumsRepository) FindByName(ctx context.Context, name string) (photo.Albums, error) {
	q := ar.s.SelectStmt().
		Where(ar.s.Eq(&ar.s.R.Name, name)).
		Limit(1)

	return augmentResErr(ar.s.Get(ctx, q))
}

func (ar *AlbumsRepository) DeleteImages(ctx context.Context, albumID int, imageIDs ...int) error {
	_, err := ar.sai.DeleteStmt().
		Where(ar.sai.Eq(&ar.sai.R.AlbumID, albumID)).
		Where(ar.sai.Eq(&ar.sai.R.ImageID, imageIDs)).
		ExecContext(ctx)

	return augmentErr(err)
}

func (ar *AlbumsRepository) AddImages(ctx context.Context, albumID int, imageIDs ...int) error {
	rows := make([]AlbumImages, 0, len(imageIDs))

	for _, imageID := range imageIDs {
		ai := AlbumImages{}
		ai.ImageID = imageID
		ai.AlbumID = albumID

		rows = append(rows, ai)
	}

	if _, err := ar.sai.InsertRows(ctx, rows, sqluct.InsertIgnore); err != nil {
		return ctxd.WrapError(ctx, augmentErr(err), "store album images", "rows", rows)
	}

	return nil
}

func (ar *AlbumsRepository) PhotoAlbumAdder() photo.AlbumAdder {
	return ar
}

func (ar *AlbumsRepository) PhotoAlbumUpdater() photo.AlbumUpdater {
	return ar
}

func (ar *AlbumsRepository) PhotoAlbumFinderOld() photo.AlbumFinder {
	return ar
}

func (ar *AlbumsRepository) PhotoAlbumDeleter() photo.AlbumDeleter {
	return ar
}

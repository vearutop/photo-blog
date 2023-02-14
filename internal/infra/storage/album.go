package storage

import (
	"context"
	"github.com/bool64/ctxd"
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
	AlbumHash uniq.Hash `db:"album_hash"`
	ImageHash uniq.Hash `db:"image_hash"`
	Weight    int       `db:"weight"`
}

func NewAlbumRepository(storage *sqluct.Storage, ir *ImageRepository) *AlbumRepository {
	return &AlbumRepository{
		hashedRepo: hashedRepo[photo.Album, *photo.Album]{
			StorageOf: sqluct.Table[photo.Album](storage, AlbumTable),
		},
		ai: sqluct.Table[AlbumImage](storage, AlbumImageTable),
		i:  ir,
	}
}

// AlbumRepository saves images to database.
type AlbumRepository struct {
	hashedRepo[photo.Album, *photo.Album]
	ai sqluct.StorageOf[AlbumImage]
	i  *ImageRepository
}

func (r *AlbumRepository) FindImages(ctx context.Context, albumHash uniq.Hash) ([]photo.Image, error) {
	q := r.i.SelectStmt().
		InnerJoin(
			r.i.Fmt("%s ON %s = %s AND %s = ?",
				r.ai.R, &r.ai.R.ImageHash, &r.i.R.Hash, &r.ai.R.AlbumHash),
			albumHash,
		).OrderByClause(r.i.Ref(&r.i.R.Path))

	return augmentResErr(r.i.List(ctx, q))
}

func (r *AlbumRepository) FindByName(ctx context.Context, name string) (photo.Album, error) {
	q := r.SelectStmt().
		Where(r.Eq(&r.R.Name, name)).
		Limit(1)

	return augmentResErr(r.Get(ctx, q))
}

func (r *AlbumRepository) DeleteImages(ctx context.Context, albumHash uniq.Hash, imageHashes ...uniq.Hash) error {
	return augmentReturnErr(r.ai.DeleteStmt().
		Where(r.ai.Eq(&r.ai.R.AlbumHash, albumHash)).
		Where(r.ai.Eq(&r.ai.R.ImageHash, imageHashes)).
		ExecContext(ctx))
}

func (r *AlbumRepository) AddImages(ctx context.Context, albumHash uniq.Hash, imageHashes ...uniq.Hash) error {
	rows := make([]AlbumImage, 0, len(imageHashes))

	for _, imageHash := range imageHashes {
		ai := AlbumImage{}
		ai.ImageHash = imageHash
		ai.AlbumHash = albumHash

		rows = append(rows, ai)
	}

	if _, err := r.ai.InsertRows(ctx, rows, sqluct.InsertIgnore); err != nil {
		return ctxd.WrapError(ctx, augmentErr(err), "store album images", "rows", rows)
	}

	return nil
}

func (r *AlbumRepository) PhotoAlbumEnsurer() uniq.Ensurer[photo.Album] {
	return r
}

func (r *AlbumRepository) PhotoAlbumFinder() uniq.Finder[photo.Album] {
	return r
}

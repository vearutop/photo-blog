package storage

import (
	"context"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

const (
	// FavoriteImageTable is the name of the table.
	FavoriteImageTable = "favorite_image"
)

// FavoriteImage describes database mapping.
type FavoriteImage struct {
	VisitorHash uniq.Hash `db:"visitor_hash"`
	ImageHash   uniq.Hash `db:"image_hash"`
	CreatedAt   time.Time `db:"created_at"`
}

func NewFavoriteRepository(storage *sqluct.Storage) *FavoriteRepository {
	fr := &FavoriteRepository{
		st: storage,
	}

	fr.fi = sqluct.Table[FavoriteImage](storage, FavoriteImageTable)
	fr.r = fr.fi.Referencer

	fr.ai = sqluct.Table[AlbumImage](storage, AlbumImageTable)
	fr.r.AddTableAlias(fr.ai.R, AlbumImageTable)

	fr.i = sqluct.Table[photo.Image](storage, ImageTable)
	fr.r.AddTableAlias(fr.i.R, ImageTable)

	return fr
}

// FavoriteRepository saves images to database.
type FavoriteRepository struct {
	st *sqluct.Storage

	r *sqluct.Referencer

	fi sqluct.StorageOf[FavoriteImage]
	ai sqluct.StorageOf[AlbumImage]
	i  sqluct.StorageOf[photo.Image]
}

func (r *FavoriteRepository) FindImages(ctx context.Context, visitorHash uniq.Hash) ([]photo.Image, error) {
	q := r.i.SelectStmt().
		InnerJoin(
			r.r.Fmt("%s ON %s = %s AND %s = ?", r.fi.R, &r.fi.R.ImageHash, &r.i.R.Hash, &r.fi.R.VisitorHash),
			visitorHash,
		).
		OrderByClause(r.r.Fmt("%s DESC", &r.fi.R.CreatedAt))

	return augmentResErr(r.i.List(ctx, q))
}

func (r *FavoriteRepository) FindAlbumImages(ctx context.Context, visitorHash, albumHash uniq.Hash) ([]photo.Image, error) {
	q := r.i.SelectStmt().
		InnerJoin(
			r.r.Fmt("%s ON %s = %s AND %s = ?", r.fi.R, &r.fi.R.ImageHash, &r.i.R.Hash, &r.fi.R.VisitorHash),
			visitorHash,
		).
		InnerJoin(
			r.r.Fmt("%s ON %s = %s AND %s = ?", r.ai.R, &r.ai.R.ImageHash, &r.i.R.Hash, &r.ai.R.AlbumHash),
			albumHash,
		).
		OrderByClause(r.r.Fmt("%s DESC", &r.fi.R.CreatedAt))

	return augmentResErr(r.i.List(ctx, q))
}

func (r *FavoriteRepository) FindImageHashes(ctx context.Context, visitorHash uniq.Hash, albumHash uniq.Hash) ([]uniq.Hash, error) {
	q := r.fi.SelectStmt().Where(r.r.Fmt("%s = ?", &r.fi.R.VisitorHash), visitorHash)

	if albumHash != 0 {
		q = q.InnerJoin(
			r.r.Fmt("%s ON %s = %s AND %s = ?", r.ai.R, &r.fi.R.ImageHash, &r.ai.R.ImageHash, &r.ai.R.AlbumHash),
			albumHash,
		)
	}

	rows, err := augmentResErr(r.fi.List(ctx, q))
	if err != nil {
		return nil, err
	}

	res := make([]uniq.Hash, len(rows))
	for _, row := range rows {
		res = append(res, row.ImageHash)
	}

	return res, nil
}

func (r *FavoriteRepository) DeleteImages(ctx context.Context, visitorHash uniq.Hash, imageHashes ...uniq.Hash) error {
	return augmentReturnErr(r.fi.DeleteStmt().
		Where(r.r.Eq(&r.fi.R.VisitorHash, visitorHash)).
		Where(r.r.Eq(&r.fi.R.ImageHash, imageHashes)).
		ExecContext(ctx))
}

func (r *FavoriteRepository) AddImages(ctx context.Context, visitorHash uniq.Hash, imageHashes ...uniq.Hash) error {
	rows := make([]FavoriteImage, 0, len(imageHashes))

	for _, imageHash := range imageHashes {
		fi := FavoriteImage{}
		fi.ImageHash = imageHash
		fi.VisitorHash = visitorHash

		rows = append(rows, fi)
	}

	if _, err := r.fi.InsertRows(ctx, rows, sqluct.InsertIgnore); err != nil {
		return ctxd.WrapError(ctx, augmentErr(err), "store favorite images", "rows", rows)
	}

	return nil
}

func (r *FavoriteRepository) FavoriteRepository() *FavoriteRepository {
	return r
}

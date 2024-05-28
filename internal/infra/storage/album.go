package storage

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/Masterminds/squirrel"
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
	AlbumHash uniq.Hash  `db:"album_hash"`
	ImageHash uniq.Hash  `db:"image_hash"`
	Timestamp *time.Time `db:"timestamp"`
}

func NewAlbumRepository(storage *sqluct.Storage, ir *ImageRepository, mr *MetaRepository) *AlbumRepository {
	ar := &AlbumRepository{
		st: storage,
		hashedRepo: hashedRepo[photo.Album, *photo.Album]{
			StorageOf: sqluct.Table[photo.Album](storage, AlbumTable),
		},
	}

	ar.ai = sqluct.Table[AlbumImage](storage, AlbumImageTable)
	ar.i = ir
	ar.m = mr

	ar.Referencer.AddTableAlias(ar.ai.R, AlbumImageTable)
	ar.Referencer.AddTableAlias(ar.m.R, MetaTable)
	ar.Referencer.AddTableAlias(ar.i.R, ImageTable)

	ar.hashedRepo.prepare = func(ctx context.Context, v *photo.Album) error {
		t := v.Settings.Texts
		if len(t) > 0 {
			sort.Slice(t, func(i, j int) bool {
				return t[i].Time.Before(t[j].Time)
			})
		}

		return nil
	}

	return ar
}

// AlbumRepository saves images to database.
type AlbumRepository struct {
	st *sqluct.Storage
	hashedRepo[photo.Album, *photo.Album]

	ai sqluct.StorageOf[AlbumImage]
	i  *ImageRepository
	m  *MetaRepository
}

func (r *AlbumRepository) orderImages(q squirrel.SelectBuilder) squirrel.SelectBuilder {
	return q.OrderByClause(r.Fmt("COALESCE(%s, %s), %s", &r.i.R.TakenAt, &r.i.R.CreatedAt, &r.i.R.Path))
}

func (r *AlbumRepository) FindImages(ctx context.Context, albumHash uniq.Hash) ([]photo.Image, error) {
	q := r.i.SelectStmt().
		InnerJoin(
			r.Fmt("%s ON %s = %s AND %s = ?", r.ai.R, &r.ai.R.ImageHash, &r.i.R.Hash, &r.ai.R.AlbumHash),
			albumHash,
		)

	q = r.orderImages(q)

	return augmentResErr(r.i.List(ctx, q))
}

func (r *AlbumRepository) FindImageAlbums(ctx context.Context, excludeAlbum uniq.Hash, imageHashes ...uniq.Hash) (map[uniq.Hash][]photo.Album, error) {
	q := r.ai.SelectStmt().Columns(r.Cols(r.R)...).
		InnerJoin(r.Fmt("%s ON %s = %s", r.R, &r.ai.R.AlbumHash, &r.R.Hash)).
		Where(r.Eq(&r.ai.R.ImageHash, imageHashes))

	if excludeAlbum != 0 {
		q = q.Where(squirrel.NotEq(r.Eq(&r.ai.R.AlbumHash, excludeAlbum)))
	}

	type row struct {
		photo.Album
		AlbumImage
	}

	var rows []row

	if err := r.st.Select(ctx, q, &rows); err != nil {
		return nil, augmentErr(err)
	}

	res := make(map[uniq.Hash][]photo.Album)
	for _, i := range rows {
		res[i.ImageHash] = append(res[i.ImageHash], i.Album)
	}

	return res, nil
}

func (r *AlbumRepository) FindOrphanImages(ctx context.Context) ([]photo.Image, error) {
	// SELECT image.hash, image.path FROM image
	//                                       LEFT JOIN album_image ON image.hash = album_image.image_hash
	//                                       LEFT JOIN album on album.hash = album_image.album_hash
	// WHERE album.hash is NULL;

	q := r.i.SelectStmt().
		LeftJoin(
			r.Fmt("%s ON %s = %s", r.ai.R, &r.ai.R.ImageHash, &r.i.R.Hash),
		).
		Where(r.Fmt("%s IS NULL", &r.ai.R.AlbumHash)).
		GroupBy(r.Ref(&r.i.R.Hash)).
		OrderByClause(r.Fmt("COALESCE(%s, %s), %s", &r.i.R.TakenAt, &r.i.R.CreatedAt, &r.i.R.Path))

	return augmentResErr(r.i.List(ctx, q))
}

func (r *AlbumRepository) SearchImages(ctx context.Context, query string) ([]photo.Image, error) {
	// SELECT `image`.`width`, `image`.`height`, `image`.`blurhash`, `image`.`phash`, `image`.`taken_at`, `image`.`settings`, `image`.`size`, `image`.`path`, `image`.`hash`, `image`.`created_at`
	// FROM `image`
	// LEFT JOIN `album_image` ON `album_image`.`image_hash` = `image`.`hash`
	// LEFT JOIN `meta` ON `meta`.`hash` = `image`.`hash`
	// WHERE `meta`.`data` LIKE ? GROUP BY `image`.`hash`
	// ORDER BY COALESCE(`image`.`taken_at`, `image`.`created_at`), `image`.`path`

	q := r.i.SelectStmt().
		LeftJoin(r.Fmt("%s ON %s = %s", r.ai.R, &r.ai.R.ImageHash, &r.i.R.Hash)).
		LeftJoin(r.Fmt("%s ON %s = %s", r.m.R, &r.m.R.Hash, &r.i.R.Hash)).
		Where(r.Fmt("%s LIKE ?", &r.m.R.Data), "%"+query+"%").
		GroupBy(r.Ref(&r.i.R.Hash)).
		OrderByClause(r.Fmt("COALESCE(%s, %s), %s", &r.i.R.TakenAt, &r.i.R.CreatedAt, &r.i.R.Path))

	return augmentResErr(r.i.List(ctx, q))
}

func (r *AlbumRepository) SearchImages2(ctx context.Context, query string) ([]photo.Image, error) {
	q := r.i.SelectStmt(func(options *sqluct.Options) {
		options.PrepareColumn = func(col string) string {
			return "image." + col
		}
	}).
		LeftJoin(r.Fmt("%s ON %s = %s", r.ai.R, &r.ai.R.ImageHash, &r.i.R.Hash)).
		LeftJoin(r.Fmt("meta as m ON m.hash=%s", &r.i.R.Hash)).
		Where("m.data like '%" + query + "%'").
		GroupBy(r.Ref(&r.i.R.Hash)).
		OrderByClause(r.Fmt("COALESCE(%s, %s), %s", &r.i.R.TakenAt, &r.i.R.CreatedAt, &r.i.R.Path))

	return augmentResErr(r.i.List(ctx, q))
}

func (r *AlbumRepository) FindBrokenImages(ctx context.Context) ([]photo.Image, error) {
	q := r.i.SelectStmt().
		OrderByClause(r.i.Fmt("COALESCE(%s, %s), %s", &r.i.R.TakenAt, &r.i.R.CreatedAt, &r.i.R.Path))

	l, err := r.i.List(ctx, q)
	if err != nil {
		return nil, err
	}

	var broken []photo.Image
	for _, i := range l {
		i.Settings.Description += "\n\nPath: " + i.Path

		s, err := os.Lstat(i.Path)
		if err != nil {
			i.Settings.Description += "\n\n" + "Error: " + err.Error()
			broken = append(broken, i)

			continue
		}

		if s.Size() != i.Size {
			i.Settings.Description += "\n\n" + fmt.Sprintf("Wrong size: %d on disk, %d in DB", s.Size(), i.Size)
			broken = append(broken, i)
		}
	}

	return broken, nil
}

func (r *AlbumRepository) FindPreviewImages(ctx context.Context, albumHash uniq.Hash, coverImage uniq.Hash, limit uint64) ([]photo.Image, error) {
	q := r.i.SelectStmt().
		InnerJoin(
			r.Fmt("%s ON %s = %s AND %s = ?", r.ai.R, &r.ai.R.ImageHash, &r.i.R.Hash, &r.ai.R.AlbumHash),
			albumHash,
		)

	// Take cover image first.
	if coverImage != 0 {
		q = q.OrderByClause(r.i.Fmt("%s = ? DESC", &r.i.R.Hash), coverImage)
	}

	// Take 3:2 aspect ratio first.
	q = q.OrderByClause(r.Fmt("ROUND(ABS(100.0*%s/%s-150)) ASC", &r.i.R.Width, &r.i.R.Height))

	q = q.OrderByClause(r.Fmt("COALESCE(%s, %s), %s", &r.i.R.TakenAt, &r.i.R.CreatedAt, &r.i.R.Path))
	q = q.Limit(limit)

	q = q.Where(r.Fmt("%s != ''", &r.i.R.BlurHash))

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
		Where(r.Eq(&r.ai.R.AlbumHash, albumHash)).
		Where(r.Eq(&r.ai.R.ImageHash, imageHashes)).
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

func (r *AlbumRepository) SetAlbumImageTimestamp(ctx context.Context, album uniq.Hash, img uniq.Hash, ts time.Time) error {
	v := AlbumImage{
		AlbumHash: album,
		ImageHash: img,
		Timestamp: &ts,
	}

	return augmentReturnErr(r.ai.UpdateStmt(v).Where(squirrel.Eq{
		r.Ref(&r.ai.R.AlbumHash): album,
		r.Ref(&r.ai.R.ImageHash): img,
	}).ExecContext(ctx))
}

func (r *AlbumRepository) PhotoAlbumImageAdder() photo.AlbumImageAdder {
	return r
}

func (r *AlbumRepository) PhotoAlbumImageDeleter() photo.AlbumImageDeleter {
	return r
}

func (r *AlbumRepository) PhotoAlbumImageFinder() photo.AlbumImageFinder {
	return r
}

func (r *AlbumRepository) PhotoAlbumEnsurer() uniq.Ensurer[photo.Album] {
	return r
}

func (r *AlbumRepository) PhotoAlbumFinder() uniq.Finder[photo.Album] {
	return r
}

func (r *AlbumRepository) PhotoAlbumUpdater() uniq.Updater[photo.Album] {
	return r
}

func (r *AlbumRepository) PhotoAlbumDeleter() uniq.Deleter[photo.Album] {
	return r
}

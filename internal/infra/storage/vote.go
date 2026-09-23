package storage

import (
	"context"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/storage/hashed"
)

const (
	// AlbumVoteTable is the name of the table.
	AlbumVoteTable = "album_vote"
)

// AlbumVote describes database mapping.
type AlbumVote struct {
	AlbumHash   uniq.Hash `db:"album_hash"`
	VisitorHash uniq.Hash `db:"visitor_hash"`
	ImageHash   uniq.Hash `db:"image_hash"`
	CreatedAt   time.Time `db:"created_at"`
}

// VoteCount is a number of picks of an image.
type VoteCount struct {
	ImageHash uniq.Hash `db:"image_hash" json:"hash"`
	Count     int       `db:"cnt" json:"count"`
}

func NewVoteRepository(storage *sqluct.Storage) *VoteRepository {
	return &VoteRepository{
		st: storage,
		v:  sqluct.Table[AlbumVote](storage, AlbumVoteTable),
	}
}

// VoteRepository stores visitor picks of album images.
type VoteRepository struct {
	st *sqluct.Storage
	v  sqluct.StorageOf[AlbumVote]
}

// Picks returns images picked by visitor in album.
func (r *VoteRepository) Picks(ctx context.Context, albumHash, visitorHash uniq.Hash) ([]uniq.Hash, error) {
	rows, err := hashed.AugmentResErr(r.v.List(ctx, r.v.SelectStmt().
		Where(r.v.Eq(&r.v.R.AlbumHash, albumHash)).
		Where(r.v.Eq(&r.v.R.VisitorHash, visitorHash)).
		OrderByClause(r.v.Fmt("%s", &r.v.R.CreatedAt))))
	if err != nil {
		return nil, err
	}

	res := make([]uniq.Hash, 0, len(rows))
	for _, row := range rows {
		res = append(res, row.ImageHash)
	}

	return res, nil
}

// Results returns picked images of album, most picked first, and the number of distinct voters.
func (r *VoteRepository) Results(ctx context.Context, albumHash uniq.Hash) ([]VoteCount, int, error) {
	var res []VoteCount

	// Ties go to the image that was picked first.
	q := r.st.QueryBuilder().Select("image_hash", "COUNT(*) AS cnt").From(AlbumVoteTable).
		Where("album_hash = ?", albumHash).
		GroupBy("image_hash").
		OrderBy("cnt DESC", "MIN(created_at)")

	if err := r.st.Select(ctx, q, &res); err != nil {
		return nil, 0, hashed.AugmentErr(err)
	}

	var voters []int

	q = r.st.QueryBuilder().Select("COUNT(DISTINCT visitor_hash)").From(AlbumVoteTable).
		Where("album_hash = ?", albumHash)

	if err := r.st.Select(ctx, q, &voters); err != nil {
		return nil, 0, hashed.AugmentErr(err)
	}

	return res, voters[0], nil
}

// Add stores a pick, it is a no-op if the pick already exists.
func (r *VoteRepository) Add(ctx context.Context, albumHash, visitorHash, imageHash uniq.Hash) error {
	row := AlbumVote{AlbumHash: albumHash, VisitorHash: visitorHash, ImageHash: imageHash, CreatedAt: time.Now()}

	if _, err := r.v.InsertRow(ctx, row, sqluct.InsertIgnore); err != nil {
		return ctxd.WrapError(ctx, hashed.AugmentErr(err), "store vote", "row", row)
	}

	return nil
}

// Delete removes a pick.
func (r *VoteRepository) Delete(ctx context.Context, albumHash, visitorHash, imageHash uniq.Hash) error {
	return hashed.AugmentReturnErr(r.v.DeleteStmt().
		Where(r.v.Eq(&r.v.R.AlbumHash, albumHash)).
		Where(r.v.Eq(&r.v.R.VisitorHash, visitorHash)).
		Where(r.v.Eq(&r.v.R.ImageHash, imageHash)).
		ExecContext(ctx))
}

func (r *VoteRepository) VoteRepository() *VoteRepository {
	return r
}

// InAlbum checks if image belongs to album.
func (r *VoteRepository) InAlbum(ctx context.Context, albumHash, imageHash uniq.Hash) (bool, error) {
	var cnt []int

	q := r.st.QueryBuilder().Select("COUNT(*)").From(AlbumImageTable).
		Where("album_hash = ? AND image_hash = ?", albumHash, imageHash)

	if err := r.st.Select(ctx, q, &cnt); err != nil {
		return false, hashed.AugmentErr(err)
	}

	return cnt[0] > 0, nil
}

package visitor

import (
	"context"
	"net/http"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type Stats struct {
	l  ctxd.Logger
	st *sqluct.Storage

	ref *sqluct.Referencer
	is  *imageStats
}

func NewStats(st *sqluct.Storage, l ctxd.Logger) *Stats {
	s := &Stats{
		l:  l,
		st: st,
	}

	s.is = &imageStats{}

	s.ref = st.MakeReferencer()
	s.ref.AddTableAlias(s.is, "")

	return s
}

func (s *Stats) CollectVisitor(h uniq.Hash, r *http.Request) {
	// TODO: implement
}

func (s *Stats) CollectMain(ctx context.Context, visitor uniq.Hash) {
	// TODO: implement
}

const imageStatsTable = "image_stats"

type imageStats struct {
	Hash    uniq.Hash `db:"hash" description:"Image hash"`
	ViewMs  int       `db:"view_ms" description:"Total focused view time in ms."`
	ThumbMs int       `db:"thumb_ms" description:"Total thumb on screen time, in ms."`
	Views   int       `db:"views" description:"Total focused views count."`
	Zooms   int       `db:"zooms" description:"Total zoom in count."`
	Uniq    int       `db:"uniq" description:"Total unique focused viewers count."`
}

func (s *Stats) CollectImage(ctx context.Context, visitor, image uniq.Hash, viewTimeMs int, zoomedIn bool) {
	is := imageStats{
		Hash:   image,
		Views:  1,
		ViewMs: viewTimeMs,
	}

	if zoomedIn {
		is.Zooms = 1
	}

	_, err := s.st.InsertStmt(imageStatsTable, is).Suffix(s.ref.Fmt("ON CONFLICT(%s) "+
		"DO UPDATE SET "+
		"%s = %s + 1, "+
		"%s = %s + excluded.%s, "+
		"%s = %s + excluded.%s",
		&s.is.Hash,
		&s.is.Views, &s.is.Views,
		&s.is.ViewMs, &s.is.ViewMs, &s.is.ViewMs,
		&s.is.Zooms, &s.is.Zooms, &s.is.Zooms,
	)).ExecContext(ctx)
	if err != nil {
		s.l.Error(ctx, "failed to collect image stats", "error", err)
	}
}

func (s *Stats) CollectThumbs(ctx context.Context, visitor uniq.Hash, thumbs map[uniq.Hash]int) {
	// TODO: implement
}

func (s *Stats) CollectAlbum(ctx context.Context, visitor, album uniq.Hash) {
	// TODO: implement
}

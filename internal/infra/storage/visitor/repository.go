package visitor

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

type StatsRepository struct {
	l  ctxd.Logger
	st *sqluct.Storage

	ref *sqluct.Referencer
	is  *imageStats
	iv  *imageVisitor
	ps  *PageStats
	dps *DailyPageStats
	pv  *pageVisitor
	v   *visitor

	collectImageSuffix  string
	collectThumbsSuffix string

	collectPageSuffix      string
	collectDailyPageSuffix string

	mu             sync.Mutex
	recentVisitors map[uniq.Hash]visitor
	isAdmin        map[uniq.Hash]bool
}

func NewStats(st *sqluct.Storage, l ctxd.Logger) (*StatsRepository, error) {
	s := &StatsRepository{
		l:              l,
		st:             st,
		recentVisitors: make(map[uniq.Hash]visitor),
		isAdmin:        make(map[uniq.Hash]bool),
	}

	s.is = &imageStats{}
	s.iv = &imageVisitor{}
	s.ps = &PageStats{}
	s.dps = &DailyPageStats{}
	s.pv = &pageVisitor{}
	s.v = &visitor{}

	s.ref = st.MakeReferencer()
	s.ref.AddTableAlias(s.is, "")
	s.ref.AddTableAlias(s.iv, "")
	s.ref.AddTableAlias(s.ps, "")
	s.ref.AddTableAlias(s.dps, "")
	s.ref.AddTableAlias(s.pv, "")
	s.ref.AddTableAlias(s.v, "")

	s.collectImageSuffix = s.ref.Fmt(
		"ON CONFLICT(%s) "+
			"DO UPDATE SET "+
			"%s = %s + 1, "+
			"%s = %s + excluded.%s, "+
			"%s = %s + excluded.%s",
		&s.is.Hash,
		&s.is.Views, &s.is.Views,
		&s.is.ViewMs, &s.is.ViewMs, &s.is.ViewMs,
		&s.is.Zooms, &s.is.Zooms, &s.is.Zooms,
	)

	s.collectThumbsSuffix = s.ref.Fmt(
		"ON CONFLICT(%s) "+
			"DO UPDATE SET "+
			"%s = %s + excluded.%s, "+
			"%s = %s + excluded.%s",
		&s.is.Hash,
		&s.is.ThumbMs, &s.is.ThumbMs, &s.is.ThumbMs,
		&s.is.ThumbPrtMs, &s.is.ThumbPrtMs, &s.is.ThumbPrtMs,
	)

	s.collectPageSuffix = s.ref.Fmt(
		"ON CONFLICT(%s) "+
			"DO UPDATE SET "+
			"%s = %s + excluded.%s, "+
			"%s = %s + 1",
		&s.ps.Hash,
		&s.ps.Refers, &s.ps.Refers, &s.ps.Refers,
		&s.ps.Views, &s.ps.Views,
	)

	s.collectDailyPageSuffix = s.ref.Fmt(
		"ON CONFLICT(%s, %s) "+
			"DO UPDATE SET "+
			"%s = %s + excluded.%s, "+
			"%s = %s + 1",
		&s.dps.Hash, &s.dps.Date,
		&s.dps.Refers, &s.dps.Refers, &s.dps.Refers,
		&s.dps.Views, &s.dps.Views,
	)

	if err := s.populateAdmins(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *StatsRepository) IsAdmin(v uniq.Hash) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.isAdmin[v]
}

func (s *StatsRepository) populateAdmins() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	q := s.st.SelectStmt(visitorTable, visitor{}).Where(squirrel.Eq{s.ref.Ref(&s.v.IsAdmin): 1})

	var rows []visitor
	if err := s.st.Select(context.Background(), q, &rows); err != nil {
		return err
	}

	for _, row := range rows {
		s.isAdmin[row.Hash] = true
	}

	return nil
}

func (s *StatsRepository) CollectMain(ctx context.Context, visitor uniq.Hash, referer string, date time.Time) {
	s.CollectAlbum(ctx, visitor, 0, referer, date)
}

const (
	imageStatsTable    = "image_stats"
	imageVisitorsTable = "image_visitors"
)

type imageStats struct {
	Hash       uniq.Hash `db:"hash" description:"Image hash"`
	ViewMs     int       `db:"view_ms" description:"Total focused view time in ms."`
	ThumbMs    int       `db:"thumb_ms" description:"Total thumb on screen time (desktop or landscape), in ms."`
	ThumbPrtMs int       `db:"thumb_prt_ms" description:"Total thumb on screen time (mobile and portrait), in ms."`
	Views      int       `db:"views" description:"Total focused views count."`
	Zooms      int       `db:"zooms" description:"Total zoom in count."`
	Uniq       int       `db:"uniq" description:"Total unique focused viewers count."`
}

type imageVisitor struct {
	Visitor uniq.Hash `db:"visitor" description:"Visitor"`
	Image   uniq.Hash `db:"image" description:"Image hash"`
}

func (s *StatsRepository) CollectImage(ctx context.Context, visitor, image uniq.Hash, viewTimeMs int, zoomedIn bool) {
	is := imageStats{
		Hash:   image,
		Views:  1,
		ViewMs: viewTimeMs,
	}

	if zoomedIn {
		is.Zooms = 1
	}

	_, err := s.st.InsertStmt(imageStatsTable, is).Suffix(s.collectImageSuffix).ExecContext(ctx)
	if err != nil {
		s.l.Error(ctx, "failed to collect image stats", "error", err)
	}

	_, err = s.st.InsertStmt(imageVisitorsTable, imageVisitor{
		Visitor: visitor,
		Image:   image,
	}, func(o *sqluct.Options) {
		o.InsertIgnore = true
	}).ExecContext(ctx)
	if err != nil {
		s.l.Error(ctx, "failed to collect image visitor", "error", err)
	}

	s.updateImageUniq(ctx, image)
}

func (s *StatsRepository) updateImageUniq(ctx context.Context, hashes ...uniq.Hash) {
	// INSERT INTO image_stats (hash, uniq)
	// SELECT image, count(distinct visitor) as uniq
	// FROM image_visitors
	// WHERE image in (-2775344103693285314)
	// GROUP BY image
	// ON CONFLICT DO UPDATE SET uniq = excluded.uniq;

	sel := s.st.QueryBuilder().
		Select(s.ref.Ref(&s.iv.Image), s.ref.Fmt("count(distinct %s) as uniq", &s.iv.Visitor)).
		From(imageVisitorsTable).
		// LeftJoin(s.ref.Fmt(visitorTable+" ON %s = %s", &s.iv.Visitor, &s.v.Hash)).
		Where(squirrel.Eq{
			s.ref.Ref(&s.iv.Image): hashes,
		}).
		// Where(s.ref.Fmt("%s = 0 AND %s = 0", &s.v.IsAdmin, &s.v.IsBot)).
		GroupBy(s.ref.Ref(&s.iv.Image))

	q := s.st.QueryBuilder().Insert(imageStatsTable).
		Columns(s.ref.Ref(&s.is.Hash), s.ref.Ref(&s.is.Uniq)).
		Select(sel).
		Suffix(s.ref.Fmt("ON CONFLICT DO UPDATE SET %s = excluded.%s", &s.is.Uniq, &s.is.Uniq))

	_, err := q.ExecContext(ctx)
	if err != nil {
		s.l.Error(ctx, "failed to update image uniq", "error", err)
	}
}

func (s *StatsRepository) CollectThumbs(ctx context.Context, visitor uniq.Hash, mobilePortraitMode bool, thumbs map[uniq.Hash]int) {
	is := make([]imageStats, 0, len(thumbs))
	if mobilePortraitMode {
		for h, t := range thumbs {
			is = append(is, imageStats{
				Hash:       h,
				ThumbPrtMs: t,
			})
		}
	} else {
		for h, t := range thumbs {
			is = append(is, imageStats{
				Hash:    h,
				ThumbMs: t,
			})
		}
	}

	_, err := s.st.InsertStmt(imageStatsTable, is).Suffix(s.collectThumbsSuffix).ExecContext(ctx)
	if err != nil {
		s.l.Error(ctx, "failed to collect thumbs stats", "error", err)
	}
}

const (
	pageVisitorsTable   = "page_visitors"
	pageStatsTable      = "page_stats"
	dailyPageStatsTable = "daily_page_stats"
)

type pageVisitor struct {
	Visitor uniq.Hash `db:"visitor" description:"Visitor"`
	Page    uniq.Hash `db:"page" description:"Album hash or 0 for main page"`
	Date    int64     `db:"date" description:"Date as truncated unix timestamp"`
}

type PageStats struct {
	Hash   uniq.Hash `db:"hash" description:"Album hash or 0 for main page"`
	Views  int       `db:"views" description:"Total views count."`
	Uniq   int       `db:"uniq" description:"Total unique viewers count."`
	Refers int       `db:"refers" description:"Total referer views count."`
}

type DailyPageStats struct {
	PageStats
	Date int64 `db:"date" description:"Date as truncated unix timestamp"`
}

func dateTs(t time.Time) int64 {
	return t.Truncate(24 * time.Hour).Unix()
}

func (s *StatsRepository) CollectAlbum(ctx context.Context, visitor, album uniq.Hash, referer string, date time.Time) {
	ps := PageStats{
		Hash:  album,
		Views: 1,
	}

	if referer != "" {
		ps.Refers = 1
	}

	_, err := s.st.InsertStmt(pageStatsTable, ps).Suffix(s.collectPageSuffix).ExecContext(ctx)
	if err != nil {
		s.l.Error(ctx, "failed to collect page stats", "error", err)
	}

	d := dateTs(date)

	dps := DailyPageStats{}
	dps.PageStats = ps
	dps.Date = d

	_, err = s.st.InsertStmt(dailyPageStatsTable, dps).Suffix(s.collectDailyPageSuffix).ExecContext(ctx)
	if err != nil {
		s.l.Error(ctx, "failed to collect daily page stats", "error", err)
	}

	_, err = s.st.InsertStmt(pageVisitorsTable, pageVisitor{
		Visitor: visitor,
		Page:    album,
		Date:    d,
	}, func(o *sqluct.Options) {
		o.InsertIgnore = true
	}).ExecContext(ctx)
	if err != nil {
		s.l.Error(ctx, "failed to collect page visitor", "error", err)
	}

	s.updatePageUniq(ctx, album)
	s.updateDailyPageUniq(ctx, d, album)
}

func (s *StatsRepository) updatePageUniq(ctx context.Context, hashes ...uniq.Hash) {
	sel := s.st.QueryBuilder().
		Select(s.ref.Ref(&s.pv.Page), s.ref.Fmt("count(distinct %s) as uniq", &s.pv.Visitor)).
		From(pageVisitorsTable).
		// LeftJoin(s.ref.Fmt(visitorTable+" ON %s = %s", &s.pv.Visitor, &s.v.Hash)).
		Where(squirrel.Eq{s.ref.Ref(&s.pv.Page): hashes}).
		// Where(s.ref.Fmt("%s = 0 AND %s = 0", &s.v.IsAdmin, &s.v.IsBot)).
		GroupBy(s.ref.Ref(&s.pv.Page))

	q := s.st.QueryBuilder().Insert(pageStatsTable).
		Columns(s.ref.Ref(&s.ps.Hash), s.ref.Ref(&s.ps.Uniq)).
		Select(sel).
		Suffix(s.ref.Fmt("ON CONFLICT DO UPDATE SET %s = excluded.%s", &s.ps.Uniq, &s.ps.Uniq))

	_, err := q.ExecContext(ctx)
	if err != nil {
		s.l.Error(ctx, "failed to update page uniq", "error", err)
	}
}

func (s *StatsRepository) updateDailyPageUniq(ctx context.Context, d int64, hashes ...uniq.Hash) {
	sel := s.st.QueryBuilder().
		Select(s.ref.Ref(&s.pv.Page), s.ref.Ref(&s.pv.Date), s.ref.Fmt("count(distinct %s) as uniq", &s.pv.Visitor)).
		From(pageVisitorsTable).
		// LeftJoin(s.ref.Fmt(visitorTable+" ON %s = %s", &s.pv.Visitor, &s.v.Hash)).
		Where(squirrel.Eq{s.ref.Ref(&s.pv.Page): hashes}).
		Where(squirrel.Eq{s.ref.Ref(&s.pv.Date): d}).
		// Where(s.ref.Fmt("%s = 0 AND %s = 0", &s.v.IsAdmin, &s.v.IsBot)).
		GroupBy(s.ref.Ref(&s.pv.Page))

	q := s.st.QueryBuilder().Insert(dailyPageStatsTable).
		Columns(s.ref.Ref(&s.dps.Hash), s.ref.Ref(&s.dps.Date), s.ref.Ref(&s.dps.Uniq)).
		Select(sel).
		Suffix(s.ref.Fmt("ON CONFLICT DO UPDATE SET %s = excluded.%s", &s.dps.Uniq, &s.dps.Uniq))

	_, err := q.ExecContext(ctx)
	if err != nil {
		s.l.Error(ctx, "failed to update daily page uniq", "error", err)
	}
}

const refersTable = "refers"

type refer struct {
	TS      int64     `db:"ts" description:"Timestamp as unix timestamp"`
	Visitor uniq.Hash `db:"visitor" description:"Visitor"`
	Referer string    `db:"referer" description:"Referer URL"`
	URL     string    `db:"url" description:"Target URL"`
}

func (s *StatsRepository) CollectRefer(ctx context.Context, visitor uniq.Hash, ts time.Time, referer, url string) {
	r := refer{
		TS:      ts.Unix(),
		Visitor: visitor,
		Referer: referer,
		URL:     url,
	}

	_, err := s.st.InsertStmt(refersTable, r).ExecContext(ctx)
	if err != nil {
		s.l.Error(ctx, "failed to collect referer visitor", "error", err)
	}
}

const visitorTable = "visitor"

type visitor struct {
	Hash      uniq.Hash `db:"hash" description:"Visitor hash"`
	CreatedAt time.Time `db:"created_at" description:"Timestamp created"`
	Lang      string    `db:"lang" description:"Visitor lang"`
	IPAddr    string    `db:"ip_addr" description:"Visitor IP address"`
	UserAgent string    `db:"user_agent" description:"Visitor user agent"`
	Device    string    `db:"device" description:"Device"`
	IsBot     bool      `db:"is_bot" description:"Visitor is bot"`
	IsAdmin   bool      `db:"is_admin" description:"Visitor is admin"`
	Referer   string    `db:"referer" description:"Referer"`

	ScreenHeight int     `db:"scr_h" description:"Visitor screen height"`
	ScreenWidth  int     `db:"scr_w" description:"Visitor screen width"`
	PixelRatio   float64 `db:"px_r" description:"Visitor pixel ratio"`
}

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func atof(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func (s *StatsRepository) CollectVisitor(h uniq.Hash, isBot, isAdmin bool, r *http.Request) {
	hd := r.Header
	ua := r.UserAgent()
	v := visitor{
		Hash:      h,
		CreatedAt: time.Now(),
		Lang:      hd.Get("Accept-Language"),
		IPAddr:    hd.Get("X-Forwarded-For"),
		UserAgent: ua,
		Device: strings.TrimSpace(
			strings.Trim(hd.Get("Sec-Ch-Ua-Model"), `"`) + " " +
				strings.Trim(hd.Get("Sec-Ch-Ua-Platform"), `"`) + " " +
				strings.Trim(hd.Get("Sec-Ch-Ua-Platform-Version"), `"`),
		),
		IsBot:   isBot,
		IsAdmin: isAdmin,
		Referer: hd.Get("Referer"),

		ScreenWidth:  atoi(r.URL.Query().Get("sw")),
		ScreenHeight: atoi(r.URL.Query().Get("sh")),
		PixelRatio:   atof(r.URL.Query().Get("px")),
	}

	ctx := r.Context()

	if rv, recent := s.isRecentVisitor(v); recent {
		var columns []string

		if !rv.IsBot && v.IsBot {
			columns = append(columns, s.ref.Col(&s.v.IsBot))
		}

		if !rv.IsAdmin && v.IsAdmin {
			columns = append(columns, s.ref.Col(&s.v.IsAdmin))
		}

		if rv.ScreenWidth == 0 && v.ScreenWidth != 0 {
			columns = append(columns, s.ref.Col(&s.v.ScreenWidth), s.ref.Col(&s.v.ScreenHeight), s.ref.Col(&s.v.PixelRatio))
		}

		if len(columns) == 0 {
			return
		}

		_, err := s.st.UpdateStmt(visitorTable, v, func(options *sqluct.Options) {
			options.Columns = columns
		}).Where(squirrel.Eq{s.ref.Ref(&s.v.Hash): v.Hash}).ExecContext(ctx)
		if err != nil {
			s.l.Error(ctx, "failed to update visitor", "error", err)
		}

		s.mu.Lock()
		defer s.mu.Unlock()
		s.recentVisitors[v.Hash] = v

		return
	}

	_, err := s.st.InsertStmt(visitorTable, v, func(options *sqluct.Options) {
		options.InsertIgnore = true
	}).ExecContext(ctx)
	if err != nil {
		s.l.Error(ctx, "failed to collect visitor", "error", err)
	}
}

func (s *StatsRepository) isRecentVisitor(v visitor) (visitor, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if v.IsAdmin && !s.isAdmin[v.Hash] {
		s.isAdmin[v.Hash] = true
	}

	rv, ok := s.recentVisitors[v.Hash]
	if !ok {
		// TODO: eviction.
		s.recentVisitors[v.Hash] = v

		return v, false
	}

	return rv, ok
}

/////

func (s *StatsRepository) DailyTotal(ctx context.Context, minDate, maxDate time.Time) ([]DailyPageStats, error) {
	q := s.st.SelectStmt(dailyPageStatsTable, DailyPageStats{}).
		Where(squirrel.GtOrEq{s.ref.Ref(&s.dps.Date): dateTs(minDate)}).
		Where(squirrel.LtOrEq{s.ref.Ref(&s.dps.Date): dateTs(maxDate)}).
		OrderByClause(s.ref.Fmt("%s DESC, %s DESC, %s DESC, %s != 0 ASC ", &s.dps.Date, &s.dps.Uniq, &s.dps.Views, &s.dps.Hash))

	var res []DailyPageStats
	err := s.st.Select(ctx, q, &res)

	return res, err
}

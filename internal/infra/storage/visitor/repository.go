package visitor

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/pkg/webstats"
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

	visitorRepository *visitorRepository

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

	s.visitorRepository = newVisitorRepository(st)
	s.v = s.visitorRepository.R

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

func (s *StatsRepository) DB() *sql.DB {
	return s.st.DB().DB
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

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func atof(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func (s *StatsRepository) CollectVisitor(h uniq.Hash, isBot, isAdmin bool, ts time.Time, r *http.Request) {
	hd := r.Header
	ua := r.UserAgent()
	v := visitor{
		LastSeen:  ts,
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

	v.Hash = h
	v.CreatedAt = ts

	ctx := r.Context()

	skipUpdate := func(candidate *visitor, existing *visitor) (skipUpdate bool) {
		skipUpdate = true

		if existing.IsAdmin {
			candidate.IsAdmin = true
		}

		if existing.IsBot {
			candidate.IsBot = true
		}

		if candidate.Device == "" {
			candidate.Device = existing.Device
		}

		if candidate.Lang == "" {
			candidate.Lang = existing.Lang
		}

		if candidate.Referer == "" {
			candidate.Referer = existing.Referer
		}

		if candidate.IPAddr == "" {
			candidate.IPAddr = existing.IPAddr
		} else {
			if !strings.Contains(existing.IPAddr, candidate.IPAddr) && len(existing.IPAddr) < 240 {
				skipUpdate = false
				candidate.IPAddr = strings.TrimPrefix(",", existing.IPAddr+","+candidate.IPAddr)
			}
		}

		if skipUpdate && candidate.IsBot && !existing.IsBot {
			skipUpdate = false
		}

		if skipUpdate && candidate.IsAdmin && !existing.IsAdmin {
			skipUpdate = false
		}

		if skipUpdate && candidate.Device != "" && existing.Device == "" {
			skipUpdate = false
		}

		if skipUpdate && candidate.Lang != "" && existing.Lang == "" {
			skipUpdate = false
		}

		if skipUpdate && candidate.Referer != "" && existing.Referer == "" {
			skipUpdate = false
		}

		if skipUpdate && candidate.ScreenWidth != 0 && existing.ScreenWidth == 0 {
			skipUpdate = false
		}

		if skipUpdate && candidate.LastSeen.Sub(existing.LastSeen) > time.Minute {
			skipUpdate = false
		}

		return skipUpdate
	}

	// Nothing to update.
	if rv, ok := s.recentVisitor(v.Hash); ok && skipUpdate(&v, &rv) {
		s.l.Info(ctx, "skip cached visitor", "visitor", v)

		return
	}

	s.l.Info(ctx, "collect visitor", "visitor", v)

	ev, err := s.visitorRepository.Ensure(ctx, v, uniq.EnsureOption[visitor]{
		Prepare: skipUpdate,
	})
	if err != nil {
		s.l.Error(ctx, "failed to ensure visitor", "error", err)
		return
	}

	s.setRecentVisitor(ev)
}

func (s *StatsRepository) recentVisitor(hash uniq.Hash) (visitor, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.recentVisitors[hash]

	return v, ok
}

func (s *StatsRepository) setRecentVisitor(v visitor) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.recentVisitors[v.Hash] = v

	if v.IsAdmin && !s.isAdmin[v.Hash] {
		s.isAdmin[v.Hash] = true
	}
}

func (s *StatsRepository) CollectRequest(ctx context.Context, input CollectStats, ts time.Time) {
	if webstats.IsBot(input.Request().UserAgent()) || s.IsAdmin(input.Visitor) {
		return
	}

	switch {
	// /stats?main=1&sw=1920&sh=1080&px=2&v=k...g
	case input.Main:
		s.CollectMain(ctx, input.Visitor, input.Referer, ts)

	// /stats?album=featured&img=3d3kr8ydrb8l4&time=4169&w=1475&h=983&mw=1620&mh=1080&sw=1920&sh=1080&px=2&v=k...g
	case input.Image != 0:
		zoomedIn := float64(input.MaxWidth)/float64(input.Width) > 1.1 // At least 10% zoom in.
		s.CollectImage(ctx, input.Visitor, input.Image, input.Time, zoomedIn)

	// /stats?album=2024-07-13-aloevera&sw=1280&sh=800&px=2&v=qtuf2cgx08i4
	case input.Album != "":
		s.CollectAlbum(ctx, input.Visitor, photo.AlbumHash(input.Album), input.Referer, ts)

	// /stats?thumb=%7B%2234suxvlfx0lz8%22%3A36704%2C%221z4zoegvmke8n%22%3A36704%2C%223b45tgt52cnms%22%3A36704%2C%221d2ujpqi6nbb4%22%3A36704%2C%221shlwpftv8av4%22%3A36704%7D&sw=1792&sh=1120&px=2&v=1...w
	case len(input.Thumb) > 0:
		s.CollectThumbs(ctx, input.Visitor, input.MobilePortraitMode, input.Thumb)
	}
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

func (s *StatsRepository) TopAlbums(ctx context.Context) ([]PageStats, error) {
	var res []PageStats
	q := s.st.SelectStmt(pageStatsTable, res).OrderByClause(s.ref.Fmt("%s DESC", &s.ps.Uniq))

	err := s.st.Select(ctx, q, &res)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return res, err
	}

	return res, nil
}

func (s *StatsRepository) TopImages(ctx context.Context) ([]imageStats, error) {
	var res []imageStats
	q := s.st.SelectStmt(imageStatsTable, res).OrderByClause(s.ref.Fmt("%s DESC", &s.is.Uniq)).Limit(300)

	err := s.st.Select(ctx, q, &res)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return res, err
	}

	return res, nil
}

func (s *StatsRepository) AlbumViews(ctx context.Context, hash uniq.Hash) (PageStats, error) {
	res := PageStats{}
	q := s.st.SelectStmt(pageStatsTable, res).Where(squirrel.Eq{s.ref.Ref(&s.ps.Hash): hash})

	err := s.st.Select(ctx, q, &res)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return res, err
	}

	return res, nil
}

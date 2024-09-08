// Package main provides a tool to populate stats.sqlite from access.log.
//
// rm photo-blog-data/stats.sqlite && catp photo-blog-data/access.log.zst | go run ./cmd/log2stats
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bool64/brick/database"
	"github.com/bool64/stats"
	"github.com/bool64/zapctxd"
	"github.com/swaggest/rest/request"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/storage"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite"
	"github.com/vearutop/photo-blog/internal/infra/storage/sqlite_stats"
	"github.com/vearutop/photo-blog/internal/infra/storage/visitor"
	"github.com/vearutop/photo-blog/pkg/webstats"
	_ "modernc.org/sqlite" // SQLite3 driver.
)

func main() {
	l := zapctxd.New(zapctxd.Config{})
	cfg := database.Config{}
	cfg.DriverName = "sqlite"
	cfg.MaxOpen = 1
	cfg.MaxIdle = 1
	cfg.DSN = "photo-blog-data/stats.sqlite" + "?_time_format=sqlite"
	cfg.ApplyMigrations = true

	st, err := database.SetupStorageDSN(cfg, l.CtxdLogger(), stats.NoOp{}, sqlite_stats.Migrations)
	if err != nil {
		log.Fatal(err)
	}

	cfg.DSN = "photo-blog-data/db.sqlite" + "?_time_format=sqlite"
	stm, err := database.SetupStorageDSN(cfg, l.CtxdLogger(), stats.NoOp{}, sqlite.Migrations)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	as := storage.NewAlbumRepository(stm, storage.NewImageRepository(stm), storage.NewMetaRepository(stm))
	al, err := as.FindAll(ctx)
	if err != nil {
		log.Fatal(err)
	}

	albums := map[string]int{}
	for _, a := range al {
		albums[a.Name] = 0
	}

	vs, err := visitor.NewStats(st, l)
	if err != nil {
		log.Fatal(err)
	}

	_ = vs

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1e6), 1e6)

	cnt := 0

	type Row struct {
		Ts           string    `json:"time"`
		Time         time.Time `json:"-"`
		Visitor      uniq.Hash `json:"visitor"`
		Host         string    `json:"host"`
		URL          string    `json:"url"`
		UserAgent    string    `json:"user_agent"`
		Device       string    `json:"device"`
		Referer      string    `json:"referer"`
		ForwardedFor string    `json:"forwarded_for"`
		Admin        bool      `json:"admin"`
		Lang         string    `json:"lang"`
	}

	botCnt := 0
	admins := map[uniq.Hash]bool{}
	adminCnt := 0
	refCnt := 0
	refHosts := map[string]int{}
	statsCnt := 0

	dec := request.NewDecoderFactory().MakeDecoder(http.MethodGet, visitor.CollectStats{}, nil)

	for scanner.Scan() {
		cnt++

		var row Row

		err := json.Unmarshal(scanner.Bytes(), &row)
		if err != nil {
			println(err.Error())
			continue
		}

		row.Time, err = time.Parse("2006-01-02T15:04:05Z0700", row.Ts)
		if err != nil {
			println(err.Error())
			continue
		}

		if webstats.IsBot(row.UserAgent) {
			botCnt++
			continue
		}

		if row.Admin && !admins[row.Visitor] {
			admins[row.Visitor] = true
		}

		req, err := http.NewRequest(http.MethodGet, row.URL, nil)
		if err != nil {
			log.Fatal(err.Error())
		}
		req.Header.Set("User-Agent", row.UserAgent)
		req.Header.Set("Accept-Language", row.Lang)
		req.Header.Set("X-Forwarded-For", row.ForwardedFor)
		req.Header.Set("Referer", row.Referer)
		req.Header.Set("Sec-Ch-Ua-Model", row.Device)

		vs.CollectVisitor(row.Visitor, false, admins[row.Visitor], row.Time, req)

		if admins[row.Visitor] {
			adminCnt++

			continue
		}

		if strings.HasPrefix(row.URL, "/stats?") {
			statsCnt++

			var inp visitor.CollectStats

			err = dec.Decode(req, &inp, nil)
			if err != nil {
				log.Fatal(err.Error())
			}

			vs.CollectRequest(ctx, inp, row.Time)
		}

		extRef := false
		if row.Referer != "" {
			ru, err := url.Parse(row.Referer)
			if err == nil {
				h := ru.Hostname()

				if h != row.Host {
					if h == "vearutop.p1cs.art" || h == "p1cs.1337.cx" || h == "mildly.photogenic.hk" ||
						h == "gen.ix.tc" || h == "bigpi.cc" || h == "127.0.0.1" || h == "144.24.188.96" {
					} else {
						refHosts[h]++
						refCnt++
						extRef = true

						vs.CollectRefer(ctx, row.Visitor, row.Time, row.Referer, row.URL)
					}
				}
			}
		}

		// Support for legacy logs.
		if row.Ts < "2024-02-10T02:08:27" {
			if !extRef {
				row.Referer = ""
			}

			if row.URL == "/" {
				vs.CollectMain(ctx, row.Visitor, row.Referer, row.Time)
			}

			p := strings.Split(row.URL, "/")
			if len(p) == 3 {
				if _, ok := albums[p[1]]; ok { // Album exists.
					albums[p[1]]++
					vs.CollectAlbum(ctx, row.Visitor, photo.AlbumHash(p[1]), row.Referer, row.Time)
				}
			}
		}

	}

	println("cnt", cnt)
	println("statsCnt", statsCnt)
	println("botCnt", botCnt)
	println("adminCnt", adminCnt)
	println("admins", len(admins))
	println("refCnt", refCnt)
	fmt.Println(refHosts)
	fmt.Println("ALBUMS", albums)

	if scanner.Err() != nil {
		log.Fatal(scanner.Err())
	}
}

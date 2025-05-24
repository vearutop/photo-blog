package main

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/oschwald/geoip2-golang"
	"github.com/swaggest/assertjson"
)

func main() {
	// https://github.com/femueller/cloud-ip-ranges

	if len(os.Args) < 2 {
		println("usage: " + os.Args[0] + " <IP>")
		return
	}

	// https://github.com/P3TERX/GeoLite.mmdb

	_, err := os.Stat(path.Join(os.TempDir() + "GeoIP2-City.mmdb"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			req, err := http.NewRequest(http.MethodGet, "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb", nil)
			if err != nil {
				log.Fatal(err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Fatal(err)
			}

			defer resp.Body.Close()

			f, err := os.Create(path.Join(os.TempDir() + "GeoIP2-City.mmdb"))
			if err != nil {
				log.Fatal(err)
			}

			if _, err := io.Copy(f, resp.Body); err != nil {
				log.Fatal(err)
			}
		}
	}

	db, err := geoip2.Open(path.Join(os.TempDir() + "GeoIP2-City.mmdb"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// If you are using strings that may be invalid, check that ip is not nil
	ip := net.ParseIP(os.Args[1])
	record, err := db.City(ip)
	if err != nil {
		log.Fatal(err)
	}
	j, _ := assertjson.MarshalIndentCompact(record, "", " ", 120)
	println(string(j))
}

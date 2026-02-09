package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/oschwald/maxminddb-golang"
	"github.com/swaggest/assertjson"
)

func main() {
	// https://github.com/femueller/cloud-ip-ranges

	var mmdbPath string
	flag.StringVar(&mmdbPath, "mmdbPath", path.Join(os.TempDir()+"GeoIP2-City.mmdb"), "path to mmdb")

	flag.Parse()

	if flag.NArg() < 1 {
		println("usage: " + os.Args[0] + " <IP>")
		return
	}

	// https://github.com/P3TERX/GeoLite.mmdb

	println(mmdbPath)

	_, err := os.Stat(mmdbPath)
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

			f, err := os.Create(mmdbPath)
			if err != nil {
				log.Fatal(err)
			}

			if _, err := io.Copy(f, resp.Body); err != nil {
				log.Fatal(err)
			}
		}
	}

	db, err := maxminddb.Open(mmdbPath)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()
	// If you are using strings that may be invalid, check that ip is not nil
	ip := net.ParseIP(flag.Arg(0))

	var res any

	err = db.Lookup(ip, &res)
	if err != nil {
		log.Fatal(err)
	}

	j, _ := assertjson.MarshalIndentCompact(res, "", " ", 120)
	println(string(j))
}

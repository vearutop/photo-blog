// Package main serves 2-4 JPEG images side by side with synchronized pan/zoom,
// so you can pixel-peep the same crop across a set of similarly-framed shots.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vearutop/photo-blog/pkg/pixelpeep"
	"github.com/vearutop/photo-blog/resources/static"
	"github.com/vearutop/statigz"
)

func main() {
	addr := flag.String("addr", ":8765", "address to listen on")
	height := flag.Int("height", 0, "viewport height in px (0 fills the screen height); overridden by \"height\" in -config")
	config := flag.String("config", "", `JSON file like {"height":0,"zoom":1,"offsetX":0,"offsetY":0,"items":[{"url":...,"label":...,"zoom":...,"offsetX":...,"offsetY":...}, ...]} (2-4 items), overrides positional args`)
	flag.Parse()

	cfg, err := loadConfig(*config, flag.Args())
	if err != nil {
		fmt.Fprintln(flag.CommandLine.Output(), err)
		fmt.Fprintln(flag.CommandLine.Output(), "Compare 2-4 JPEG images with synchronized pan/zoom viewports.")
		fmt.Fprintln(flag.CommandLine.Output(), "Usage:\n  pixelpeep [flags] <image1.jpg> <image2.jpg> [image3.jpg] [image4.jpg]\n  pixelpeep [flags] -config config.json")
		flag.PrintDefaults()

		return
	}

	if cfg.Height == 0 {
		cfg.Height = *height
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static", statigz.FileServer(static.Assets)))

	for i, im := range cfg.Items {
		if strings.HasPrefix(im.URL, "http://") || strings.HasPrefix(im.URL, "https://") {
			continue
		}

		localPath := im.URL
		cfg.Items[i].URL = "/img/" + strconv.Itoa(i)
		mux.HandleFunc(cfg.Items[i].URL, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, localPath)
		})
	}

	tpl, err := static.Template("pixelpeep.html")
	if err != nil {
		log.Fatal(err)
	}

	// "Copy link" in the editor points back at this same server's "/" with a
	// "?config=" override, so a link is shareable with anyone who can reach
	// this local server (e.g. over the LAN) without restarting it.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqCfg := cfg

		if shared := r.URL.Query().Get("config"); shared != "" {
			if err := json.Unmarshal([]byte(shared), &reqCfg); err != nil {
				http.Error(w, "invalid config: "+err.Error(), http.StatusBadRequest)
				return
			}

			if err := reqCfg.Validate(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		configJSON, err := json.Marshal(reqCfg)
		if err != nil {
			log.Print(err)
			return
		}

		data := struct {
			ConfigJSON template.JS
			Editor     bool
			ShareBase  string
		}{ConfigJSON: template.JS(configJSON), Editor: true}

		if err := tpl.Execute(w, data); err != nil {
			log.Print(err)
		}
	})

	log.Printf("pixelpeep: open http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func loadConfig(config string, paths []string) (pixelpeep.Config, error) {
	var cfg pixelpeep.Config

	if config != "" {
		b, err := os.ReadFile(config)
		if err != nil {
			return cfg, err
		}

		if err := json.Unmarshal(b, &cfg); err != nil {
			return cfg, fmt.Errorf("parsing %s: %w", config, err)
		}
	} else {
		for _, p := range paths {
			cfg.Items = append(cfg.Items, pixelpeep.Image{URL: p, Label: filepath.Base(p)})
		}
	}

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

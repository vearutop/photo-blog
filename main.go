// Package main provides photo-blog web service.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/bool64/brick"
	"github.com/vearutop/photo-blog/internal/infra"
	"github.com/vearutop/photo-blog/internal/infra/nethttp"
	"github.com/vearutop/photo-blog/internal/infra/service"
)

func main() {
	var (
		cfg         service.Config
		migrate     = flag.Bool("migrate", false, "Run migrations and exit.")
		storagePath = flag.String("storage-path", "", "Optional path to data storage, defaults to './photo-blog-data/'.")
		listen      = flag.String("listen", "127.0.0.1:8008", "Address and port to listen to.")
	)

	brick.Start(&cfg, func(docsMode bool) (*brick.BaseLocator, http.Handler) {
		cfg.HTTPListenAddr = *listen

		// Initialize application resources.
		sl, err := infra.NewServiceLocator(cfg, docsMode)
		if err != nil {
			log.Fatalf("failed to init service: %v", err)
		}

		return sl.BaseLocator, nethttp.NewRouter(sl)
	}, func(o *brick.StartOptions) {
		if migrate != nil && *migrate {
			o.NoHTTP = true
		}

		if storagePath != nil && *storagePath != "" {
			cfg.StoragePath = *storagePath
		}
	})
}

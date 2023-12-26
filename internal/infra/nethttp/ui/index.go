// Package ui provides application web user interface.
package ui

import (
	"io/fs"
	"net/http"
	"os"

	"github.com/vearutop/photo-blog/resources/static"
	"github.com/vearutop/statigz"
)

// Static serves static assets.
var Static *statigz.Server

//nolint:gochecknoinits
func init() {
	if _, err := os.Stat("./resources/static"); err == nil {
		// path/to/whatever exists
		Static = statigz.FileServer(os.DirFS("./resources/static").(fs.ReadDirFS))
	} else {
		Static = statigz.FileServer(static.Assets)
	}
}

// Index serves index page of the application.
func Index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Static.ServeHTTP(w, r)
	})
}

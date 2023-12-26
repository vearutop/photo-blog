// Package help provides embedded static assets.
package help

import (
	"embed"
)

// Assets provides embedded static assets for web application.
//
//go:embed *.md *.png *.jpg
var Assets embed.FS

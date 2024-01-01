// Package ui provides application web user interface.
package ui

import (
	"github.com/vearutop/photo-blog/resources/static"
	"github.com/vearutop/statigz"
)

// Static serves static assets.
var Static = statigz.FileServer(static.Assets)

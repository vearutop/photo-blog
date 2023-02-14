// Package sqlite_thumbs provides migrations.
package sqlite_thumbs

import (
	"embed"
)

// Migrations provide database migrations.
//
//go:embed *.sql
var Migrations embed.FS

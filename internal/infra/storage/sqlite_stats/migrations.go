// Package sqlite_stats provides migrations.
package sqlite_stats

import (
	"embed"
)

// Migrations provide database migrations.
//
//go:embed *.sql
var Migrations embed.FS

// Package sqlite_cache provides app-specific migrations of persistent cache DB.
package sqlite_cache

import (
	"embed"
)

// Migrations provide database migrations.
//
//go:embed *.sql
var Migrations embed.FS

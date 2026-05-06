package invalidation

import "embed"

// Migrations provide database migrations.
//
//go:embed *.sql
var Migrations embed.FS

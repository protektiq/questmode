package migrations

import "embed"

// SQL holds embedded migration files (inferred schema; reconcile with v1.0 when available).
//
//go:embed *.sql
var SQL embed.FS

// Package migrations exposes immutable SQL migrations to the runtime migrator.
package migrations

import "embed"

// FS contains every numbered SQL migration.
//
//go:embed *.sql
var FS embed.FS

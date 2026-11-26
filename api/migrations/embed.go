// Package migrations embeds the goose SQL migrations so the binary applies
// them at startup without files on disk.
package migrations

import "embed"

// FS holds every migration, in goose's timestamped naming.
//
//go:embed *.sql
var FS embed.FS

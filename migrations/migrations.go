// Package migrations embeds the SQL migration files so uptime-server ships them
// inside the single static binary (no external migrations dir needed at runtime).
// Files here follow the Goose format and naming: NNNNN_description.sql.
package migrations

import "embed"

// FS holds all *.sql migrations, applied by goose (see internal/database).
//
//go:embed *.sql
var FS embed.FS

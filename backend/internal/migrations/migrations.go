// Package migrations embeds the SQL migrations so the API binary can apply
// them with goose at startup, without shipping .sql files in the image.
package migrations

import "embed"

// Only versioned files (00001_name.sql); helper scripts like verify.sql stay out.
//
//go:embed [0-9]*_*.sql
var FS embed.FS

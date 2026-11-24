// Package migrations embeds the SQL migrations so the API binary can apply
// them with goose at startup, without shipping .sql files in the image.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS

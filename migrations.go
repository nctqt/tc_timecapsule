package casechronicle

import "embed"

// exposes the embedded SQL schema files from the root context.
//
//go:embed sql/schema/*.sql
var EmbedMigrations embed.FS

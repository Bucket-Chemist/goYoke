package defaults

import "embed"

//go:embed all:agents all:conventions all:rules all:schemas all:skills routing-schema.json CLAUDE.md settings-template.json
var FS embed.FS

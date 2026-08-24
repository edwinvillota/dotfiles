// Package themes embeds the palette definitions so the built binary can
// apply themes without a repo checkout. Palettes are fetched verbatim from
// terminalcolors.com; see each palette.toml header for its source URL.
package themes

import "embed"

//go:embed */palette.toml
var FS embed.FS

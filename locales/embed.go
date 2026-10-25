// Package locales embeds the Gettext catalogs shared by backend builds.
package locales

import "embed"

// LocaleFiles contains backend and template catalogs for each translated language.
// English source strings need no catalog.
//
//go:embed */LC_MESSAGES
var LocaleFiles embed.FS

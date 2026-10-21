package domaincheck

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version is embedded from the repository's calendar-version source of truth.
var Version = strings.TrimSpace(versionFile)

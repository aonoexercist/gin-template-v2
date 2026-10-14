// Package version holds build metadata injected at link time via -ldflags.
package version

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

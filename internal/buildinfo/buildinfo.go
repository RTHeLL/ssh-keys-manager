// Package buildinfo holds version metadata injected at link time (GoReleaser / -ldflags).
package buildinfo

// These are overridden by -X at build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

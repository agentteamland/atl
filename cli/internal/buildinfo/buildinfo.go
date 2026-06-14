// Package buildinfo holds version metadata. The release build injects real
// values via -ldflags -X (see .goreleaser.yaml); a plain `go build` / dev run
// keeps the defaults.
package buildinfo

// These are overwritten at release build time with -X linker flags.
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

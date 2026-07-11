// Package version holds zentag's release version.
package version

// Version lands in the "zentag" tag on written files and in --version.
// goreleaser stamps release builds via -ldflags -X; non-release builds
// report "dev".
var Version = "dev"

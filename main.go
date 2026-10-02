package main

import (
	"regexp"
	"runtime/debug"

	"github.com/fabiodrneles/go-release-manager/cmd"
)

// Set by GoReleaser through -ldflags "-X main.version=... -X main.commit=...".
var (
	version = "dev"
	commit  = ""
)

func main() {
	cmd.Execute(resolveVersion(version, debug.ReadBuildInfo), commit)
}

// releaseVersion matches tagged versions and rejects the pseudo-versions
// (v0.0.0-20260101000000-abcdef123456) and "+dirty" builds that Go stamps
// on local builds, so those keep reporting "dev" (spec 002 FR-1).
var releaseVersion = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`)

var pseudoVersion = regexp.MustCompile(`\d{14}-[0-9a-f]{12}`)

// resolveVersion falls back to the module version recorded by `go install
// module@vX.Y.Z` when the release build did not embed one.
func resolveVersion(v string, info func() (*debug.BuildInfo, bool)) string {
	if v != "dev" {
		return v
	}
	if bi, ok := info(); ok && releaseVersion.MatchString(bi.Main.Version) && !pseudoVersion.MatchString(bi.Main.Version) {
		return bi.Main.Version
	}
	return v
}

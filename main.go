package main

import "go-release-manager/cmd"

// Set by GoReleaser through -ldflags "-X main.version=... -X main.commit=...".
var (
	version = "dev"
	commit  = ""
)

func main() {
	cmd.Execute(version, commit)
}

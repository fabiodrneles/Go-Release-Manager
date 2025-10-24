package main

import (
	"go-release-manager/cmd"
)

var (
	version string
	commit  string
)

func main() {
	if version == "" {
		version = "dev"
	}
	if commit == "" {
		commit = "none"
	}

	cmd.SetVersionInfo(version, commit)
	cmd.Execute()
}

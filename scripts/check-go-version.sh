#!/bin/sh
# Fails when a workflow or the Dockerfile pins a Go version instead of
# following go.mod, which broke the v0.11.0 release (issue #14).
set -eu
want=$(sed -n 's/^go \([0-9]*\.[0-9]*\).*/\1/p' go.mod)
bad=0
# The sdd-kit engine (sdd-*.yml) is synced from the kit and only installs tools with
# the stable Go; it never builds this module, so it is not held to go.mod.
for wf in .github/workflows/*.yml; do
	case "$wf" in */sdd-*.yml) continue ;; esac
	if grep -Hn "go-version:" "$wf"; then
		echo "use go-version-file: go.mod nos workflows" >&2
		bad=1
	fi
done
if grep -n "FROM golang:" Dockerfile | grep -v "golang:$want" ; then
	echo "Dockerfile deve usar golang:$want (go.mod)" >&2
	bad=1
fi
[ "$bad" -eq 0 ] && echo "check-go-version: workflows e Dockerfile seguem o go.mod ($want)"
exit "$bad"

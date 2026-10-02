#!/bin/sh
# Fails when a workflow or the Dockerfile pins a Go version instead of
# following go.mod, which broke the v0.11.0 release (issue #14).
set -eu
want=$(sed -n 's/^go \([0-9]*\.[0-9]*\).*/\1/p' go.mod)
bad=0
if grep -n "go-version:" .github/workflows/*.yml; then
	echo "use go-version-file: go.mod nos workflows" >&2
	bad=1
fi
if grep -n "FROM golang:" Dockerfile | grep -v "golang:$want" ; then
	echo "Dockerfile deve usar golang:$want (go.mod)" >&2
	bad=1
fi
[ "$bad" -eq 0 ] && echo "check-go-version: workflows e Dockerfile seguem o go.mod ($want)"
exit "$bad"

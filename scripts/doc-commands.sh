#!/bin/sh
# Runs every ```bash block of README.md against a sample repository with the
# freshly built binary on PATH (spec 002 AC-5). Commands that change state
# (create without --dry-run) belong in ```text blocks.
set -eu
root=$(pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
go build -o "$tmp/bin/go-release-manager" .
repo="$tmp/repo"
mkdir -p "$repo"
cd "$repo"
git init -q
git -c user.name=t -c user.email=t@t commit -q --allow-empty -m "feat: first"
git tag v1.2.0
git -c user.name=t -c user.email=t@t commit -q --allow-empty -m "feat: second"
blocks=$(awk '/^```bash$/{b=1; n++; next} /^```$/{b=0} b{print > ("'"$tmp"'/block" n ".sh")} END{print n+0}' "$root/README.md")
i=1
while [ "$i" -le "$blocks" ]; do
	echo "== README bloco bash $i"
	PATH="$tmp/bin:$PATH" NO_COLOR=1 GITHUB_TOKEN='' sh -eu "$tmp/block$i.sh"
	i=$((i + 1))
done
echo "doc-commands: $blocks bloco(s) ok"

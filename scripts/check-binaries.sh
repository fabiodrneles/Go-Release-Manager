#!/bin/sh
# Fails when a compiled binary is tracked by git (spec 003 AC-3).
set -eu
found=0
for f in $(git ls-files); do
	[ -f "$f" ] || continue
	magic=$(head -c 4 "$f" | od -An -tx1 | tr -d ' \n')
	case "$magic" in
	7f454c46 | 4d5a* | cffaedfe | cefaedfe | feedface | feedfacf | cafebabe)
		echo "binário versionado: $f" >&2
		found=1
		;;
	esac
done
[ "$found" -eq 0 ] && echo "check-binaries: nenhum binário versionado"
exit "$found"

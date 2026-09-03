#!/usr/bin/env bash
# Step 2 of 2: run the benchmark against the generated corpus.
#
#   ./step2.sh            # archmage struct targets C#  (default)
#   ./step2.sh go         # archmage struct targets Go
#
# Anything after the language is passed through to `go test`, e.g.
#   ./step2.sh cs -benchtime=10x
#   ./step2.sh -benchtime=10x     # the language may be omitted
#
# Five iterations per case, three rounds, so the numbers can be fed to
# benchstat. A summary lands in bench-report.md next to the output directory.
# -v turns on progress lines; the benchmark result lines stay intact.
set -euo pipefail
cd "$(dirname "$0")"
lang=cs
case "${1:-}" in
cs | go)
	lang="$1"
	shift
	;;
"" | -*) ;;
*)
	echo "step2.sh: unknown language \"$1\", want cs or go" >&2
	exit 2
	;;
esac
exec go test -v -run='^$' -bench=. -benchtime=5x -count=3 -timeout=0 -language="$lang" "$@"

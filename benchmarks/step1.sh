#!/usr/bin/env bash
# Step 1 of 2: generate the benchmark corpus.
#
# Writes 100 config files (60 xlsx, 10 csv, 15 yaml, 15 json) of 3000 config
# entries and 20 logical fields each, plus the enum definitions and the shared
# l10n table. The corpus lands in /Volumes/Fuzz/archmage_bench/testdata when
# that volume is mounted, and in ./testdata otherwise.
#
# The seed is fixed, so the corpus is identical on every run. Pass flags
# through for a smaller one, e.g.  ./step1.sh -files=10 -rows=100
set -euo pipefail
cd "$(dirname "$0")"
exec go run . "$@"

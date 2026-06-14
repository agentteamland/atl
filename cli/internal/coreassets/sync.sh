#!/usr/bin/env bash
#
# Mirror core/ (rules + skills) into this package's embed/ dir so the atl binary
# can ship them via go:embed. go:embed cannot reach outside the cli/ module
# (no `..`), so the monorepo's core/ is copied here as the build input.
#
# Run after changing anything under core/. The TestEmbedMatchesCore test fails
# if embed/ has drifted from core/, so CI/local tests catch a stale mirror.

set -euo pipefail
cd "$(dirname "$0")" # cli/internal/coreassets/

CORE="../../../core"
DEST="embed"

rm -rf "$DEST"
mkdir -p "$DEST"
cp -R "$CORE/rules" "$DEST/rules"
cp -R "$CORE/skills" "$DEST/skills"

echo "coreassets: synced core/{rules,skills} -> $DEST/"

#!/usr/bin/env bash
# needs: none
#
# pin — a pinned project file is held back from promote; unpin re-enables it.
source /e2e/lib.sh

fresh
write_test_index
cd "$PROJ" || exit 2
atl install --global  agentteamland/atl-e2e-team >/dev/null || bad "global install errored"
atl install --project agentteamland/atl-e2e-team >/dev/null || bad "project install errored"

GLOB="$HOME/.claude/agents/e2e-agent/agent.md"
REL="agents/e2e-agent/agent.md"
printf '\n<!-- e2e: project-only tweak -->\n' >> "$PROJ/.claude/$REL"

( cd "$PROJ" && atl pin "$REL" >/dev/null ) || bad "pin errored"
BEFORE=$(sha256sum "$GLOB" | cut -d' ' -f1)
( cd "$PROJ" && atl promote >/dev/null )
[ "$BEFORE" = "$(sha256sum "$GLOB" | cut -d' ' -f1)" ] && ok "pinned file not promoted" || bad "pinned file leaked to global"

( cd "$PROJ" && atl unpin "$REL" >/dev/null ) || bad "unpin errored"
( cd "$PROJ" && atl promote >/dev/null )
grep -q "project-only tweak" "$GLOB" && ok "unpinned file now promoted" || bad "unpin did not re-enable promote"

finish

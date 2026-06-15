#!/usr/bin/env bash
# needs: none
#
# promote — a project-layer gain (modified file + new child) lifts to global,
# and a second pass is a no-op (idempotent).
source /e2e/lib.sh

fresh
write_test_index
cd "$PROJ" || exit 2
atl install --global  agentteamland/atl-e2e-team >/dev/null || bad "global install errored"
atl install --project agentteamland/atl-e2e-team >/dev/null || bad "project install errored"

GLOB="$HOME/.claude/agents/e2e-agent/agent.md"
printf '\n<!-- e2e: learned prefer-X -->\n' >> "$PROJ/.claude/agents/e2e-agent/agent.md"
mkdir -p "$PROJ/.claude/agents/e2e-agent/children"
echo "e2e new knowledge" > "$PROJ/.claude/agents/e2e-agent/children/e2e-learned.md"

( cd "$PROJ" && atl promote >/dev/null ) || bad "promote errored"
grep -q "learned prefer-X" "$GLOB" && ok "modified agent promoted to global" || bad "agent gain not promoted"
[ -f "$HOME/.claude/agents/e2e-agent/children/e2e-learned.md" ] && ok "new child promoted to global" || bad "new child not promoted"

OUT=$( cd "$PROJ" && atl promote )
echo "$OUT" | grep -qi "nothing to lift" && ok "second promote is a no-op" || bad "promote not idempotent ($OUT)"

finish

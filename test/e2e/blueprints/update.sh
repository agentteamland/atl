#!/usr/bin/env bash
# needs: none
#
# update — a global-layer change fans out into an unmodified project copy.
source /e2e/lib.sh

fresh
write_test_index
cd "$PROJ" || exit 2
atl install --global  agentteamland/atl-e2e-team >/dev/null || bad "global install errored"
atl install --project agentteamland/atl-e2e-team >/dev/null || bad "project install errored"

GA="$HOME/.claude/agents/e2e-agent/agent.md"
printf '\n<!-- e2e: global-side update -->\n' >> "$GA"
( cd "$PROJ" && atl update >/dev/null 2>&1 )
grep -q "global-side update" "$PROJ/.claude/agents/e2e-agent/agent.md" \
  && ok "unmodified project file refreshed from global" || bad "fan-out did not refresh"

finish

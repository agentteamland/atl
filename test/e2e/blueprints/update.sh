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
out=$( cd "$PROJ" && atl update 2>&1 )
grep -q "global-side update" "$PROJ/.claude/agents/e2e-agent/agent.md" \
  && ok "unmodified project file refreshed from global" || bad "fan-out did not refresh"

# F4: the edit above made the global copy diverge from the published version, so
# the throttled network pass (which re-fetches) should surface a publish
# suggestion for that gain — exercised here against the real GitHub fixture.
echo "$out" | grep -q "atl-e2e-team not yet upstream" \
  && ok "network pass surfaced a publish suggestion for the global gain (F4)" \
  || bad "no publish suggestion for the global gain (F4)"

finish

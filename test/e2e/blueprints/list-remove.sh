#!/usr/bin/env bash
# needs: none
#
# list-remove — list shows the installed team; remove deletes its files +
# manifest at the scope.
source /e2e/lib.sh

fresh
write_test_index
cd "$PROJ" || exit 2
atl install --project agentteamland/atl-e2e-team >/dev/null || bad "install errored"

out="$(cd "$PROJ" && atl list 2>&1)"
echo "$out" | grep -q "atl-e2e-team" && ok "list shows the installed team" || bad "list missing team -- [$out]"

( cd "$PROJ" && atl remove agentteamland/atl-e2e-team >/dev/null ) || bad "remove errored"
[ ! -e "$PROJ/.claude/agents/e2e-agent" ] && ok "remove deleted project files" || bad "remove left files"
if ls "$PROJ/.atl/installed/"*.json >/dev/null 2>&1; then bad "remove left project manifest"; else ok "remove dropped project manifest"; fi

finish

#!/usr/bin/env bash
# needs: none
#
# doctor — a deleted installed file is detected as drift and self-healed from
# the pinned source.
source /e2e/lib.sh

fresh
write_test_index
cd "$PROJ" || exit 2
atl install --project agentteamland/atl-e2e-team >/dev/null || bad "install errored"

VICTIM="$PROJ/.claude/agents/e2e-agent/agent.md"
rm -f "$VICTIM"
[ ! -f "$VICTIM" ] || bad "could not delete victim file"

( cd "$PROJ" && atl doctor >/dev/null 2>&1 )
[ -f "$VICTIM" ] && ok "doctor restored the deleted file" || bad "doctor did not restore"

finish

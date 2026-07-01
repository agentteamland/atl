#!/usr/bin/env bash
# needs: none
#
# gc — orphaned assets (files no manifest owns) are reported (dry-run), soft-
# deleted (--apply) while owned + core files stay intact, and restored (--undo).
# The whole point is reversibility, so the round-trip is the assertion.
source /e2e/lib.sh

fresh
write_test_index
cd "$PROJ" || exit 2
atl install --project agentteamland/atl-e2e-team >/dev/null || bad "install errored"

OWNED="$PROJ/.claude/agents/e2e-agent/agent.md"
[ -f "$OWNED" ] && ok "install wrote an owned agent file" || bad "no owned file after install"

# Two orphans: a gain beside an installed unit, and a wholly-unowned dir.
GAIN="$PROJ/.claude/agents/e2e-agent/children/orphan-gain.md"
ROGUE="$PROJ/.claude/skills/rogue/SKILL.md"
mkdir -p "$(dirname "$GAIN")" "$(dirname "$ROGUE")"
echo gain  > "$GAIN"
echo rogue > "$ROGUE"

# Dry-run reports both and touches nothing.
OUT=$(cd "$PROJ" && atl gc 2>&1)
{ echo "$OUT" | grep -q "orphan-gain.md" && echo "$OUT" | grep -q "rogue"; } \
  && ok "dry-run reports both orphans" || bad "dry-run missed an orphan -- [$OUT]"
{ [ -f "$GAIN" ] && [ -f "$ROGUE" ]; } && ok "dry-run touched nothing" || bad "dry-run removed a file"

# Apply soft-deletes the orphans but preserves the manifest-owned file.
( cd "$PROJ" && atl gc --apply >/dev/null 2>&1 )
{ [ ! -f "$GAIN" ] && [ ! -f "$ROGUE" ]; } && ok "apply soft-deleted both orphans" || bad "apply left an orphan on disk"
[ -f "$OWNED" ] && ok "apply preserved the manifest-owned file" || bad "apply removed an owned file"

# Undo restores them from the trash.
( cd "$PROJ" && atl gc --undo >/dev/null 2>&1 )
{ [ -f "$GAIN" ] && [ -f "$ROGUE" ]; } && ok "undo restored both orphans" || bad "undo did not restore"

finish

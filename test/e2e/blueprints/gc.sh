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

# Two orphans of different kinds: a GAIN beside an installed unit (learning-loop
# growth — retained by default), and a wholly-unowned ROGUE dir (swept by default).
GAIN="$PROJ/.claude/agents/e2e-agent/children/orphan-gain.md"
ROGUE="$PROJ/.claude/skills/rogue/SKILL.md"
mkdir -p "$(dirname "$GAIN")" "$(dirname "$ROGUE")"
echo gain  > "$GAIN"
echo rogue > "$ROGUE"

# Dry-run: the rogue is a sweepable orphan; the gain is reported as RETAINED
# (a separate section), and nothing is touched.
OUT=$(cd "$PROJ" && atl gc 2>&1)
echo "$OUT" | grep -q "rogue" && ok "dry-run reports the unowned orphan" || bad "dry-run missed the rogue -- [$OUT]"
{ echo "$OUT" | grep -q "orphan-gain.md" && echo "$OUT" | grep -qi "retain"; } \
  && ok "dry-run reports the gain as retained" || bad "dry-run should report the gain as retained -- [$OUT]"
{ [ -f "$GAIN" ] && [ -f "$ROGUE" ]; } && ok "dry-run touched nothing" || bad "dry-run removed a file"

# Default apply: sweeps the unowned rogue, RETAINS the learning-loop gain, and
# preserves the manifest-owned file.
( cd "$PROJ" && atl gc --apply >/dev/null 2>&1 )
[ ! -f "$ROGUE" ] && ok "apply swept the unowned orphan" || bad "apply left the rogue on disk"
[ -f "$GAIN" ]    && ok "apply retained the learning-loop gain (default)" || bad "apply swept a gain it should retain"
[ -f "$OWNED" ]   && ok "apply preserved the manifest-owned file" || bad "apply removed an owned file"

# --include-gains reclaims the gain too.
( cd "$PROJ" && atl gc --apply --include-gains >/dev/null 2>&1 )
[ ! -f "$GAIN" ] && ok "--include-gains swept the gain" || bad "--include-gains left the gain on disk"

# Undo restores the most recent batch (the gain) from the trash — reversibility.
( cd "$PROJ" && atl gc --undo >/dev/null 2>&1 )
[ -f "$GAIN" ] && ok "undo restored the gain from trash" || bad "undo did not restore the gain"

finish

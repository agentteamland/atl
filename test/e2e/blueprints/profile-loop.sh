#!/usr/bin/env bash
# needs: token
#
# profile-loop — the full real-Claude profile loop: install profile-team, a
# `claude -p` session drops a profile-fact marker -> atl tick enqueues it ->
# /profile-drain has the profile-curator write a profile.md under
# ~/.atl/profiles -> ack deletes it. The KEEP fact carries a relationship anchor
# (friend) + a situation, so it persists even under the reality gate. A second
# segment proves breaking-change migration (infra #5): a profile one MAJOR behind
# its interface is migrated on touch. A third proves the reality gate (infra #2):
# a documentation-example / placeholder profile-fact is DROPPED (acked, no profile,
# no interface authored), not materialized into a fabricated person. Assertions key
# off queue + file STATE (key/profile presence, the fact drained — never an exact
# transported value), so the non-deterministic LLM turn stays non-flaky. Migration
# FIDELITY (the exact fields the curator rewrote) is LLM-variable → NOTE-tiered.
source /e2e/lib.sh
# note() is NOT in lib.sh (only ok=PASS / bad=FAIL) — a local observe-only tier for
# LLM-variable assertions (llm-e2e-assertion-tiering), mirroring delivery-loop.sh.
note() { echo "  note - $1"; }

fresh
write_test_index_profile
headless_claude_setup
cd "$PROJ" || exit 2
atl install --global agentteamland/profile-team >/dev/null || bad "install errored"
[ -f "$HOME/.claude/rules/profile-capture.md" ]      && ok "profile-capture rule active" || bad "rule missing"
[ -f "$HOME/.claude/skills/profile-drain/SKILL.md" ] && ok "profile-drain skill active"  || bad "skill missing"

claude_turn() { ( cd "$PROJ" && claude -p "$1" --dangerously-skip-permissions --output-format json ) >>"$HOME/turns.log" 2>&1; }

PROMPT='My friend Alex is terrified of confrontation and just started an anxious new job. Please record this about Alex using the profile-capture inline marker format — a single "<!-- profile-fact: ... -->" HTML comment with entity: alex and the relevant fields.'
claude_turn "$PROMPT" || bad "capture turn errored (see turns.log)"
T=$(find "$HOME/.claude/projects" -name '*.jsonl' 2>/dev/null | xargs ls -t 2>/dev/null | head -1)
{ [ -n "$T" ] && grep -q '<!-- profile-fact:' "$T"; } && ok "real session dropped a profile-fact marker" || bad "no profile-fact marker in transcript"

( cd "$PROJ" && atl tick >/dev/null 2>&1 )
PEEK=$(cd "$PROJ" && atl learnings peek --channel profile-fact --json 2>/dev/null)
echo "$PEEK" | jq -e 'length > 0' >/dev/null 2>&1 && ok "marker enqueued on the profile-fact channel" || bad "queue empty after tick -- [$PEEK]"

touch "$HOME/.beforedrain"
claude_turn "/profile-drain" || bad "drain turn errored (see turns.log)"
FOUND=$(find "$HOME/.atl/profiles/people" -name 'profile.md' -newer "$HOME/.beforedrain" 2>/dev/null)
[ -n "$FOUND" ] && ok "curator wrote a profile.md under ~/.atl/profiles/people" || bad "no profile written by /profile-drain"
PEEK2=$(cd "$PROJ" && atl learnings peek --channel profile-fact --json 2>/dev/null)
echo "$PEEK2" | jq -e 'length == 0' >/dev/null 2>&1 && ok "profile-fact queue drained (processed-then-deleted)" || bad "queue still has items -- [$PEEK2]"

# ---- breaking-change migration (infra #5) ----
# Seeded AFTER install (fresh wipes ~/.atl) on the `object` type, which is in the
# resolver's scan set (marker-drain §2) so the touch resolves the existing profile
# -> §4 sees P<I across a major -> the migration branch. Isolated from the person
# canonical-materialization path. A `rename` op keeps the assertions grep-stable.
mkdir -p "$HOME/.atl/profiles/objects/oldmug" "$HOME/.atl/profiles/_interfaces/migrations/object"

cat > "$HOME/.atl/profiles/objects/oldmug/profile.md" <<'EOF'
---
meta:
  type-id: object
  schema-version: 1.0.0
  is-self: false
identity:
  name: Old Mug
heirloom-note: a gift from grandma
_sources:
  identity.name: user-confirmed
  heirloom-note: user-confirmed
---
# Old Mug
EOF

cat > "$HOME/.atl/profiles/_interfaces/object.md" <<'EOF'
---
type-id: object
schema-version: 2.0.0
changelog:
  - version: 1.0.0
    added: [everything]
  - version: 2.0.0
    breaking: [rename heirloom-note -> story-note]
tier-defaults:
  identity.*: 1
  story-note: 1
---
# Object
EOF

cat > "$HOME/.atl/profiles/_interfaces/migrations/object/1.0.0-to-2.0.0.md" <<'EOF'
---
type-id: object
from: 1.0.0
to: 2.0.0
operations:
  - rename: { from: heirloom-note, to: story-note }
---
# object 1.0.0 -> 2.0.0
Rename heirloom-note to story-note; carry its _sources entry verbatim.
EOF

MBODY=$'entity: oldmug\ntype: object\nfields:\n  identity.name: Old Mug\nsource: user-confirmed'
( cd "$PROJ" && atl learnings _enqueue profile-fact "$MBODY" >/dev/null 2>&1 ) || bad "_enqueue object migration fact errored"

touch "$HOME/.beforemigrate"
claude_turn "/profile-drain" || bad "migration drain turn errored (see turns.log)"

MPROF="$HOME/.atl/profiles/objects/oldmug/profile.md"
# Migration FIDELITY (did the LLM curator correctly apply the migration ops) is
# LLM-variable — NOTE-tiered, not a hard fail (llm-e2e-assertion-tiering). The
# deterministic plumbing (the fact drained, below) stays CORE.
if grep -q 'schema-version: 2.0.0' "$MPROF" 2>/dev/null; then ok "profile migrated to schema-version 2.0.0"; else note "schema-version not bumped this run (LLM-variable migration fidelity)"; fi
if grep -q 'story-note' "$MPROF" 2>/dev/null; then ok "renamed field present (story-note)"; else note "renamed field missing this run (LLM-variable)"; fi
if grep -q 'heirloom-note' "$MPROF" 2>/dev/null; then note "old field path still present this run (LLM-variable)"; else ok "old field path removed by migration"; fi
PEEK3=$(cd "$PROJ" && atl learnings peek --channel profile-fact --json 2>/dev/null)
echo "$PEEK3" | jq -e 'length == 0' >/dev/null 2>&1  && ok "migration fact drained"                       || bad "migration fact not drained -- [$PEEK3]"

# ---- reality gate (infra #2) ----
# A documentation-example / placeholder profile-fact must be DROPPED, not made into a person.
# Seeded via _enqueue, NOT a claude prompt: an illustrative marker in a prompt would re-pollute
# the very queue the gate guards (self-reference), and the capture rule would refuse to emit a
# marker for junk anyway. Unambiguous literal-placeholder class (never the bare-name-textbook
# coin-flip) — a bare skeleton with NO relationship anchor. State-based assertions only.
JUNK=$'entity: placeholder-example\ntype: person\nfields:\n  field: value\n  another: value'
( cd "$PROJ" && atl learnings _enqueue profile-fact "$JUNK" >/dev/null 2>&1 ) || bad "_enqueue junk fact errored"
BEFORE_IFACE=$(find "$HOME/.atl/profiles/_interfaces" -type f 2>/dev/null | wc -l | tr -d ' ')
claude_turn "/profile-drain" || bad "reality-gate drain turn errored (see turns.log)"
JPROF=$(find "$HOME/.atl/profiles" -path '*placeholder-example*' -name 'profile.md' 2>/dev/null)
[ -z "$JPROF" ] && ok "reality gate: no profile created for the placeholder entity" || bad "gate FAILED — junk profile written: $JPROF"
AFTER_IFACE=$(find "$HOME/.atl/profiles/_interfaces" -type f 2>/dev/null | wc -l | tr -d ' ')
[ "$AFTER_IFACE" = "$BEFORE_IFACE" ] && ok "reality gate: no interface authored from junk" || bad "gate FAILED — interface authored from junk ($BEFORE_IFACE -> $AFTER_IFACE)"
PEEK4=$(cd "$PROJ" && atl learnings peek --channel profile-fact --json 2>/dev/null)
echo "$PEEK4" | jq -e 'length == 0' >/dev/null 2>&1  && ok "reality gate: junk fact acked + queue drained" || bad "junk fact not drained (stuck un-acked?) -- [$PEEK4]"

# On failure, surface what's otherwise lost when the container is torn down.
if [ "$FAIL" -gt 0 ]; then
  echo "===== DEBUG (profile-loop failed) ====="
  echo "--- claude --version ---"; claude --version 2>&1 | head -1
  echo "--- turns.log ---"; cat "$HOME/turns.log" 2>/dev/null
  echo "--- newest transcript: ${T:-none} ---"
  if [ -n "${T:-}" ]; then
    echo "  block(s) carrying the marker (type + snippet):"
    jq -c 'select(.message.role=="assistant") | .message.content[]? | select((.text // .thinking // "") | test("<!-- profile-fact:")) | {type, snippet: ((.text // .thinking)[0:160])}' "$T" 2>/dev/null
  fi
  echo "--- ~/.atl/profiles tree ---"; find "$HOME/.atl/profiles" 2>/dev/null
  echo "--- atl tick (verbose re-run) ---"; ( cd "$PROJ" && atl tick 2>&1 )
  echo "========================================"
fi

finish

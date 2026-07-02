#!/usr/bin/env bash
# needs: token
#
# profile-loop — the full real-Claude profile loop: install profile-team, a
# `claude -p` session drops a profile-fact marker -> atl tick enqueues it ->
# /profile-drain has the profile-curator write a profile.md under
# ~/.atl/profiles -> ack deletes it. A second segment proves breaking-change
# migration (infra #5): a profile one MAJOR behind its interface is migrated on
# touch, not add-only-filled. Assertions key off queue + file STATE (schema
# version, key presence — never an exact transported value), so the
# non-deterministic LLM turn stays non-flaky.
source /e2e/lib.sh

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
grep -q 'schema-version: 2.0.0' "$MPROF" 2>/dev/null && ok "profile migrated to schema-version 2.0.0"     || bad "schema-version not bumped to 2.0.0"
grep -q 'story-note' "$MPROF" 2>/dev/null            && ok "renamed field present (story-note)"           || bad "new field path missing after migration"
grep -q 'heirloom-note' "$MPROF" 2>/dev/null         && bad "old field path still present (heirloom-note)" || ok "old field path removed by migration"
PEEK3=$(cd "$PROJ" && atl learnings peek --channel profile-fact --json 2>/dev/null)
echo "$PEEK3" | jq -e 'length == 0' >/dev/null 2>&1  && ok "migration fact drained"                       || bad "migration fact not drained -- [$PEEK3]"

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

#!/usr/bin/env bash
#
# Layer A — deterministic golden-path e2e for atl. No Claude Code session needed;
# this drives the `atl` binary directly and asserts on files + manifests. It is
# the always-on regression backbone.
#
# Runs inside the container as `testuser`. Network is required once (install
# fetches a real, ref-pinned team from the index); every scenario then restores
# a local snapshot, so the suite is both isolated and fast.

set -uo pipefail

PASS=0
FAIL=0
ok()  { echo "  ok   - $1"; PASS=$((PASS + 1)); }
bad() { echo "  FAIL - $1"; FAIL=$((FAIL + 1)); }
hashf() { sha256sum "$1" 2>/dev/null | cut -d' ' -f1; }

# A small, ref-pinned team (v0.8.1) -> deterministic content.
TEAM="agentteamland/design-system-team"

snapshot() { rm -rf "$HOME"/.snap; mkdir -p "$HOME"/.snap
  cp -a "$HOME/.claude" "$HOME/.snap/claude"
  cp -a "$HOME/.atl"    "$HOME/.snap/atl"
  cp -a "$HOME/proj"    "$HOME/.snap/proj"; }
restore() { rm -rf "$HOME/.claude" "$HOME/.atl" "$HOME/proj"
  cp -a "$HOME/.snap/claude" "$HOME/.claude"
  cp -a "$HOME/.snap/atl"    "$HOME/.atl"
  cp -a "$HOME/.snap/proj"   "$HOME/proj"; }

echo "== setup: install at both scopes (one real network install) =="
rm -rf "$HOME/.claude" "$HOME/.atl" "$HOME/proj"
mkdir -p "$HOME/proj"
cd "$HOME/proj" || exit 2
atl install --global  "$TEAM" || bad "global install errored"
atl install --project "$TEAM" || bad "project install errored"
[ -d "$HOME/.claude/agents" ]      && ok "global agents reflected"  || bad "global agents missing"
[ -d "$HOME/proj/.claude/agents" ] && ok "project agents reflected" || bad "project agents missing"
ls "$HOME/.atl/installed/"*.json      >/dev/null 2>&1 && ok "global manifest written"  || bad "global manifest missing"
ls "$HOME/proj/.atl/installed/"*.json >/dev/null 2>&1 && ok "project manifest written" || bad "project manifest missing"
# core (platform rules + skills) ships in the binary and reflects to the global
# layer on install
[ -f "$HOME/.claude/rules/communication-style.md" ] && ok "core rules reflected to global"  || bad "core rules missing"
[ -f "$HOME/.claude/skills/drain/SKILL.md" ]        && ok "core skills reflected to global" || bad "core skills missing"

# Pick a project agent file to evolve (relative to the .claude dir).
AGENT=$(find "$HOME/proj/.claude/agents" -name '*.md' | head -1)
REL=${AGENT#"$HOME"/proj/.claude/}
if [ -z "$AGENT" ]; then bad "no agent file to work with"; echo "Layer A: $PASS passed, $FAIL failed"; exit 1; fi
ok "working agent: $REL"
GLOB="$HOME/.claude/$REL"
snapshot

echo "== 1. promote: a project gain lifts to global =="
restore
printf '\n<!-- e2e: learned prefer-X -->\n' >> "$HOME/proj/.claude/$REL"
mkdir -p "$(dirname "$HOME/proj/.claude/$REL")/children"
echo "e2e new knowledge" > "$(dirname "$HOME/proj/.claude/$REL")/children/e2e-learned.md"
( cd "$HOME/proj" && atl promote >/dev/null ) || bad "promote errored"
grep -q "learned prefer-X" "$GLOB" && ok "modified agent promoted to global" || bad "agent gain not promoted"
[ -f "$(dirname "$GLOB")/children/e2e-learned.md" ] && ok "new child promoted to global" || bad "new child not promoted"

echo "== 2. promote is idempotent =="
OUT=$( cd "$HOME/proj" && atl promote )
echo "$OUT" | grep -qi "nothing to lift" && ok "second promote is a no-op" || bad "promote not idempotent ($OUT)"

echo "== 3. pin keeps a gain project-only; unpin re-enables it =="
restore
printf '\n<!-- e2e: project-only tweak -->\n' >> "$HOME/proj/.claude/$REL"
( cd "$HOME/proj" && atl pin "$REL" >/dev/null ) || bad "pin errored"
BEFORE=$(hashf "$GLOB")
( cd "$HOME/proj" && atl promote >/dev/null )
[ "$BEFORE" = "$(hashf "$GLOB")" ] && ok "pinned file not promoted" || bad "pinned file leaked to global"
( cd "$HOME/proj" && atl unpin "$REL" >/dev/null ) || bad "unpin errored"
( cd "$HOME/proj" && atl promote >/dev/null )
grep -q "project-only tweak" "$GLOB" && ok "unpinned file now promoted" || bad "unpin did not re-enable promote"

echo "== 4. doctor self-heals a deleted installed file =="
restore
VICTIM=$(find "$HOME/proj/.claude/agents" -name '*.md' | head -1)
rm -f "$VICTIM"
[ ! -f "$VICTIM" ] || bad "could not delete victim"
( cd "$HOME/proj" && atl doctor >/dev/null 2>&1 )
[ -f "$VICTIM" ] && ok "doctor restored the deleted file" || bad "doctor did not restore"

echo "== 5. update fans an unmodified project file out from global =="
restore
GA=$(find "$HOME/.claude/agents" -name '*.md' | head -1)
GREL=${GA#"$HOME"/.claude/}
printf '\n<!-- e2e: global-side update -->\n' >> "$GA"
( cd "$HOME/proj" && atl update >/dev/null 2>&1 )
grep -q "global-side update" "$HOME/proj/.claude/$GREL" \
  && ok "unmodified project file refreshed from global" || bad "fan-out did not refresh"

echo "== 6. list shows installed teams; remove deletes one scope =="
restore
list_out="$(cd "$HOME/proj" && atl list 2>&1)"
echo "$list_out" | grep -q "design-system-team" && ok "list shows the installed team" || bad "list missing team -- got: [$list_out]"
( cd "$HOME/proj" && atl remove agentteamland/design-system-team >/dev/null ) || bad "remove errored"
[ ! -e "$HOME/proj/.claude/agents/ds-architect-agent" ] && ok "remove deleted project files" || bad "remove left files"
if ls "$HOME/proj/.atl/installed/"*.json >/dev/null 2>&1; then bad "remove left project manifest"; else ok "remove dropped project manifest"; fi
[ -d "$HOME/.claude/agents/ds-architect-agent" ] && ok "global untouched by project-scoped remove" || bad "global wrongly removed"

echo ""
echo "Layer A: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]

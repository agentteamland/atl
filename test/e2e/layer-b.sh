#!/usr/bin/env bash
#
# Layer B — full-loop e2e: a real `claude -p` session drives the v2 learning loop
# end to end. Proves the loop is genuinely wired — a learning marker dropped in a
# real Claude session flows: marker -> atl tick -> durable queue -> /drain skill
# -> knowledge base, and the processed item is deleted (the v1 re-report bug
# class stays structurally dead).
#
# Needs auth (unlike Layer A): CLAUDE_CODE_OAUTH_TOKEN (from `claude setup-token`)
# or ANTHROPIC_API_KEY, injected by run.sh at `docker run` time. Runs inside the
# container as `testuser`. One linear scenario (no snapshot/restore) — each real
# `claude -p` turn costs tokens, so there's nothing to repeat.

set -uo pipefail

PASS=0
FAIL=0
ok()  { echo "  ok   - $1"; PASS=$((PASS + 1)); }
bad() { echo "  FAIL - $1"; FAIL=$((FAIL + 1)); }

TEAM="agentteamland/design-system-team"
PROJ="$HOME/proj"
LOG="$HOME/claude-turns.log"

# --- 0. auth guard (defense in depth; run.sh also checks on the host) ---
if [ -z "${CLAUDE_CODE_OAUTH_TOKEN:-}" ] && [ -z "${ANTHROPIC_API_KEY:-}" ]; then
  echo "!! Layer B needs CLAUDE_CODE_OAUTH_TOKEN or ANTHROPIC_API_KEY in the env" >&2
  exit 2
fi

# Fresh-container headless UX: a brand-new ~/.claude would otherwise prompt for
# trust/theme/onboarding and hang a non-interactive `claude -p`. The token covers
# auth; this seed covers first-run prompts; --dangerously-skip-permissions covers
# the permission prompt at call time.
mkdir -p "$HOME/.claude"
printf '{ "hasCompletedOnboarding": true }\n' > "$HOME/.claude.json"

# A real headless turn. -p (hooks + skills still fire — deliberately NOT --bare);
# skip permission prompts; JSON output so failures show is_error. Always cd to
# PROJ (queue + transcript discovery are cwd-keyed). Tee to a log for debugging.
claude_turn() { # $1 = label, $2 = prompt
  echo "---- claude turn: $1 ----" >> "$LOG"
  ( cd "$PROJ" && claude -p "$2" --dangerously-skip-permissions --output-format json ) \
    >> "$LOG" 2>&1
  local rc=$?
  echo "(exit $rc)" >> "$LOG"
  return $rc
}

echo "== setup: project install (rule + drain skill + hooks) =="
rm -rf "$HOME/.claude/agents" "$HOME/.claude/rules" "$HOME/.claude/skills" \
       "$HOME/.claude/settings.json" "$HOME/.atl" "$PROJ"
mkdir -p "$PROJ"
cd "$PROJ" || exit 2
atl install --project "$TEAM" || bad "project install errored"
[ -f "$HOME/.claude/rules/learning-capture.md" ] && ok "learning-capture rule reflected" || bad "learning-capture rule missing"
[ -f "$HOME/.claude/skills/drain/SKILL.md" ]      && ok "drain skill reflected"          || bad "drain skill missing"
grep -q 'session-start' "$HOME/.claude/settings.json" 2>/dev/null && ok "session-start hook bound" || bad "session-start hook missing"
grep -q 'atl tick'      "$HOME/.claude/settings.json" 2>/dev/null && ok "tick hook bound"          || bad "tick hook missing"

echo "== 1. capture: a real claude session drops a learning marker =="
MARKER_PROMPT='We just decided to cache user sessions in Redis with a 24-hour TTL, because most users return within a day and it sharply cuts database load. Please capture this as a learning using the inline marker format from the learning-capture rule — a single "<!-- learning: ... -->" HTML comment that includes the WHY.'
claude_turn "capture" "$MARKER_PROMPT" || bad "capture turn errored (see $LOG)"

# Assert the marker actually landed in the transcript — isolates "did the LLM
# cooperate" from "did the plumbing carry it through" (the next assertion).
TRANSCRIPT=$(find "$HOME/.claude/projects" -name '*.jsonl' 2>/dev/null | xargs ls -t 2>/dev/null | head -1)
if [ -n "$TRANSCRIPT" ] && grep -q '<!-- learning:' "$TRANSCRIPT"; then
  ok "marker present in transcript"
else
  bad "no learning marker in transcript ($TRANSCRIPT) — capture prompt produced none"
fi

echo "== 2. atl tick drains the marker into the durable queue =="
( cd "$PROJ" && atl tick >/dev/null 2>&1 )
# Assert on QUEUE STATE, not tick's output: the drain session's own hooks may
# already drain it (dedup makes that safe), so "0 new" from tick is fine as long
# as the item is in the queue.
PEEK=$(cd "$PROJ" && atl learnings peek --channel learning --json 2>/dev/null)
if echo "$PEEK" | jq -e 'length > 0' >/dev/null 2>&1; then
  ok "marker enqueued (learning channel non-empty)"
else
  bad "learning queue empty after tick — got: $PEEK"
fi

echo "== 3. /drain folds the queue into the knowledge base, then acks =="
touch "$HOME/.beforedrain"
claude_turn "drain" "/drain" || bad "drain turn errored (see $LOG)"

# A KB file appeared in one of the three destinations (don't assert which —
# /drain infers the destination from the payload's shape).
FOUND=$(find "$PROJ/.atl/wiki" "$PROJ/.atl/journal" "$PROJ/.claude/agents" "$HOME/.claude/agents" \
        -type f -newer "$HOME/.beforedrain" 2>/dev/null)
if [ -n "$FOUND" ]; then
  ok "drain wrote a knowledge-base file"
  echo "    -> $(echo "$FOUND" | head -3 | tr '\n' ' ')"
else
  bad "no KB file written by /drain"
fi

echo "== 4. the processed item was acked (deleted from the queue) =="
PEEK2=$(cd "$PROJ" && atl learnings peek --channel learning --json 2>/dev/null)
if echo "$PEEK2" | jq -e 'length == 0' >/dev/null 2>&1; then
  ok "learning queue drained (processed-then-deleted)"
else
  bad "queue still has items after /drain — got: $PEEK2"
fi

echo ""
echo "Layer B: $PASS passed, $FAIL failed"
echo "(claude turn log: $LOG)"
[ "$FAIL" -eq 0 ]

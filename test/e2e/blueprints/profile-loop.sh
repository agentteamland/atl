#!/usr/bin/env bash
# needs: token
#
# profile-loop — the full real-Claude profile loop: install profile-team, a
# `claude -p` session drops a profile-fact marker -> atl tick enqueues it ->
# /profile-drain has the profile-curator write a profile.md under
# ~/.atl/profiles -> ack deletes it. Assertions key off queue + file STATE
# (never an exact filename), so the non-deterministic LLM turn stays non-flaky.
source /e2e/lib.sh

fresh
write_test_index_profile
headless_claude_setup
cd "$PROJ" || exit 2
atl install --global agentteamland/profile-team >/dev/null || bad "install errored"
[ -f "$HOME/.claude/rules/profile-capture.md" ]      && ok "profile-capture rule active" || bad "rule missing"
[ -f "$HOME/.claude/skills/profile-drain/skill.md" ] && ok "profile-drain skill active"  || bad "skill missing"

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

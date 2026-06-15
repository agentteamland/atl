#!/usr/bin/env bash
# needs: token
#
# learning-loop — the full real-Claude loop: a `claude -p` session drops a
# learning marker -> atl tick enqueues it -> /drain folds it into the KB -> ack
# deletes it. Assertions key off queue + file STATE (never an exact filename or a
# command's "did work" message), so a non-deterministic LLM turn stays non-flaky.
source /e2e/lib.sh

fresh
write_test_index
headless_claude_setup
cd "$PROJ" || exit 2
atl install --project agentteamland/atl-e2e-team >/dev/null || bad "install errored"
[ -f "$HOME/.claude/rules/learning-capture.md" ] && ok "learning-capture rule active" || bad "rule missing"
[ -f "$HOME/.claude/skills/drain/SKILL.md" ]     && ok "drain skill active"           || bad "drain skill missing"

claude_turn() { ( cd "$PROJ" && claude -p "$1" --dangerously-skip-permissions --output-format json ) >>"$HOME/turns.log" 2>&1; }

MARKER='We just decided to cache user sessions in Redis with a 24-hour TTL, because most users return within a day and it sharply cuts database load. Please capture this as a learning using the inline marker format from the learning-capture rule — a single "<!-- learning: ... -->" HTML comment that includes the WHY.'
claude_turn "$MARKER" || bad "capture turn errored (see turns.log)"
T=$(find "$HOME/.claude/projects" -name '*.jsonl' 2>/dev/null | xargs ls -t 2>/dev/null | head -1)
{ [ -n "$T" ] && grep -q '<!-- learning:' "$T"; } && ok "real session dropped a marker" || bad "no marker in transcript"

( cd "$PROJ" && atl tick >/dev/null 2>&1 )
PEEK=$(cd "$PROJ" && atl learnings peek --channel learning --json 2>/dev/null)
echo "$PEEK" | jq -e 'length > 0' >/dev/null 2>&1 && ok "marker enqueued in the durable queue" || bad "queue empty after tick -- [$PEEK]"

touch "$HOME/.beforedrain"
claude_turn "/drain" || bad "drain turn errored (see turns.log)"
FOUND=$(find "$PROJ/.atl/wiki" "$PROJ/.atl/journal" "$PROJ/.claude/agents" -type f -newer "$HOME/.beforedrain" 2>/dev/null)
[ -n "$FOUND" ] && ok "drain wrote a knowledge-base file" || bad "no KB file written by /drain"
PEEK2=$(cd "$PROJ" && atl learnings peek --channel learning --json 2>/dev/null)
echo "$PEEK2" | jq -e 'length == 0' >/dev/null 2>&1 && ok "queue drained (processed-then-deleted)" || bad "queue still has items -- [$PEEK2]"

finish

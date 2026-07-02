#!/usr/bin/env bash
# needs: none
#
# profile-install — a clean user installs profile-team globally: the curator
# agent + /profile-drain skill + profile-capture rule reflect to ~/.claude, and
# the profile-fact plumbing works end-to-end WITHOUT a Claude turn — enqueue a
# fact, peek it on its channel, and confirm `atl session-start` surfaces the
# pending-profile-fact signal (the deterministic half of the loop; the full
# marker->profile.md closure is the token-gated profile-loop blueprint).
source /e2e/lib.sh

fresh
write_test_index_profile
cd "$PROJ" || exit 2

atl install --global agentteamland/profile-team || bad "global install errored"

[ -f "$HOME/.claude/agents/profile-curator/agent.md" ]                 && ok "curator agent reflected"      || bad "curator agent missing"
[ -f "$HOME/.claude/agents/profile-curator/children/marker-drain.md" ] && ok "curator children reflected"   || bad "curator children missing"
[ -f "$HOME/.claude/agents/profile-curator/children/animal-interface.md" ]     && ok "multi-type interface reflected (animal)" || bad "type interfaces missing"
[ -f "$HOME/.claude/agents/profile-curator/children/interface-creation.md" ]   && ok "auto-creation logic reflected"           || bad "interface-creation missing"
[ -f "$HOME/.claude/skills/profile-drain/SKILL.md" ]                   && ok "profile-drain skill reflected" || bad "profile-drain skill missing"
[ -f "$HOME/.claude/rules/profile-capture.md" ]                        && ok "profile-capture rule reflected" || bad "profile-capture rule missing"
ls "$HOME/.atl/installed/"*profile-team*.json >/dev/null 2>&1          && ok "install manifest written"     || bad "manifest missing"

# profile-fact channel plumbing, no Claude turn: enqueue directly, peek, signal.
BODY=$'entity: alex\nkind: friend\nfields:\n  identity.name: Alex Doe\n  traits.fears: [confrontation]\nsource: user-confirmed'
atl learnings _enqueue profile-fact "$BODY" >/dev/null 2>&1 || bad "_enqueue profile-fact errored"
atl learnings peek --channel profile-fact --json 2>/dev/null | jq -e 'length > 0' >/dev/null 2>&1 \
  && ok "profile-fact enqueued on its channel" || bad "profile-fact not in queue"

# The PR-3 Go signal: session-start must surface the pending profile-fact so
# Claude knows to run /profile-drain (mirrors the learning signal).
atl session-start 2>&1 | grep -q 'profile-fact(s) pending' \
  && ok "session-start signals pending profile-facts" || bad "no profile-fact signal at session start"

finish

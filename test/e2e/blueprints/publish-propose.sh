#!/usr/bin/env bash
# needs: gh
#
# publish-propose — a team you DON'T own: a global-layer gain is proposed
# upstream as a real fork + PR against agentteamland/atl-e2e-team, then the PR is
# closed + the fork branch deleted (GitHub-state cleanup).
source /e2e/lib.sh

gh auth setup-git >/dev/null 2>&1 || true
fresh
write_test_index
cd "$PROJ" || exit 2
atl install --global agentteamland/atl-e2e-team >/dev/null || bad "install errored"

printf '\n<!-- e2e propose-upstream gain -->\n' >> "$HOME/.claude/agents/e2e-agent/agent.md"
echo "e2e: propose-upstream test contribution from real usage." > "$HOME/body.md"

OUT=$(cd "$PROJ" && atl publish agentteamland/atl-e2e-team --apply --body-file "$HOME/body.md" 2>&1)
echo "$OUT"
URL=$(echo "$OUT" | grep -o 'https://github.com/agentteamland/atl-e2e-team/pull/[0-9]*' | head -1)
[ -n "$URL" ] && ok "propose-upstream opened a PR" || bad "no PR url in output"

if [ -n "$URL" ]; then
  PR=$(basename "$URL")
  STATE=$(gh pr view "$PR" --repo agentteamland/atl-e2e-team --json state -q .state 2>/dev/null)
  [ "$STATE" = "OPEN" ] && ok "PR is open on the upstream repo" || bad "PR not open (state=$STATE)"
  gh pr diff "$PR" --repo agentteamland/atl-e2e-team 2>/dev/null | grep -q "propose-upstream gain" \
    && ok "PR carries the gain" || bad "PR diff missing the gain"
  gh pr close "$PR" --repo agentteamland/atl-e2e-team >/dev/null 2>&1 \
    && ok "cleanup: PR closed" || bad "cleanup failed"
fi

finish

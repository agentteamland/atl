#!/usr/bin/env bash
# needs: gh
#
# publish-own — a team you OWN: a global-layer gain re-publishes to your repo as
# a real commit + version bump + tag against <login>/atl-e2e-owned. The repo is
# force-reset to the fixture baseline first, so the run is repeatable.
source /e2e/lib.sh

gh auth setup-git >/dev/null 2>&1 || true
LOGIN=$(gh_login)
[ -n "$LOGIN" ] || { bad "gh not authenticated"; finish; exit 1; }
reset_owned_repo "$LOGIN" && ok "reset owned repo to baseline" || bad "could not reset owned repo"

fresh
write_test_index "$LOGIN"
cd "$PROJ" || exit 2
atl install --global "$LOGIN/atl-e2e-owned" >/dev/null || bad "install errored"

printf '\n<!-- e2e own-team gain -->\n' >> "$HOME/.claude/agents/e2e-agent/agent.md"
echo "chore: re-publish e2e owned-team gains" > "$HOME/msg.md"

OUT=$(cd "$PROJ" && atl publish "$LOGIN/atl-e2e-owned" --apply --body-file "$HOME/msg.md" 2>&1)
echo "$OUT"
TAG=$(echo "$OUT" | grep -o 'as v[0-9][0-9.]*' | awk '{print $2}')
[ -n "$TAG" ] && ok "own-team re-published as $TAG" || bad "no tag in output"

if [ -n "$TAG" ]; then
  git ls-remote --tags "https://github.com/$LOGIN/atl-e2e-owned.git" "$TAG" 2>/dev/null | grep -q "$TAG" \
    && ok "tag $TAG pushed to the owned repo" || bad "tag not on remote"
  curl -fsSL "https://github.com/$LOGIN/atl-e2e-owned/raw/main/team.json" 2>/dev/null | grep -q '"version": "0.1.1"' \
    && ok "team.json version bumped on main" || bad "version not bumped on main"
fi

finish

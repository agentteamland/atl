#!/usr/bin/env bash
#
# Shared helpers, sourced by every e2e blueprint. The runner (run.sh) executes
# each blueprint in a FRESH container, so scenarios never share state; within a
# blueprint these helpers handle setup, assertions, and GitHub-state reset.

set -uo pipefail

PROJ="$HOME/proj"
PASS=0
FAIL=0
ok()  { echo "  ok   - $1"; PASS=$((PASS + 1)); }
bad() { echo "  FAIL - $1"; FAIL=$((FAIL + 1)); }

# finish prints the tally and sets the blueprint's exit status.
finish() {
  echo
  echo "${BLUEPRINT:-blueprint}: $PASS passed, $FAIL failed"
  [ "$FAIL" -eq 0 ]
}

# fresh wipes the simulated user's machine for a clean install.
fresh() {
  rm -rf "$HOME/.claude" "$HOME/.atl" "$PROJ"
  mkdir -p "$PROJ"
}

# headless_claude_setup seeds the onboarding flag so `claude -p` doesn't block in
# a brand-new container (the token handles auth; this handles first-run UX).
headless_claude_setup() {
  mkdir -p "$HOME/.claude"
  printf '{ "hasCompletedOnboarding": true }\n' > "$HOME/.claude.json"
}

# write_test_index seeds ~/.atl/index.json with the e2e fixture team(s) so
# `atl install` resolves them offline (index.Resolve prefers the cache). The
# propose-upstream fixture is always present; pass an owner login to also add the
# own-team fixture (<login>/atl-e2e-owned).
write_test_index() {
  local owned_login="${1:-}"
  mkdir -p "$HOME/.atl"
  local teams
  teams='[{"handle":"agentteamland","name":"atl-e2e-team","version":"0.1.0","description":"e2e fixture (propose-upstream).","keywords":["e2e"],"scope":"global","verified":true,"source":{"repo":"agentteamland/atl-e2e-team","subpath":"","ref":"main"}}]'
  if [ -n "$owned_login" ]; then
    teams=$(echo "$teams" | jq --arg l "$owned_login" '. + [{handle:$l,name:"atl-e2e-owned",version:"0.1.0",description:"e2e fixture (own-team).",keywords:["e2e"],scope:"global",verified:false,source:{repo:($l+"/atl-e2e-owned"),subpath:"",ref:"main"}}]')
  fi
  jq -n --argjson teams "$teams" '{schemaVersion:1,generatedAt:"2026-06-15T00:00:00Z",teams:$teams}' > "$HOME/.atl/index.json"
}

# gh_login echoes the authenticated GitHub login (GH_TOKEN is passed through by
# the runner); empty if gh isn't authenticated.
gh_login() {
  gh api user -q .login 2>/dev/null || true
}

# reset_owned_repo force-restores <login>/atl-e2e-owned to the fixture baseline
# and deletes every remote tag, so the own-team blueprint starts clean even if a
# prior run left a bump commit + tag behind. Uses gh's git credential helper.
reset_owned_repo() {
  local login="$1"
  local tmp; tmp=$(mktemp -d)
  git clone -q "https://github.com/$login/atl-e2e-owned.git" "$tmp" || { rm -rf "$tmp"; return 1; }
  (
    cd "$tmp" || exit 1
    find . -mindepth 1 -maxdepth 1 -not -name '.git' -exec rm -rf {} +
    cp -R /e2e/fixtures/owned-team/. .
    git add -A
    git -c user.email=e2e@atl.local -c user.name=atl-e2e commit -q -m "reset: e2e baseline" --allow-empty
    git push -q -f origin HEAD:main
    for t in $(git ls-remote --tags origin | awk '{print $2}' | sed 's|refs/tags/||' | grep -v '\^{}$'); do
      git push -q --delete origin "$t" 2>/dev/null || true
    done
  )
  rm -rf "$tmp"
}

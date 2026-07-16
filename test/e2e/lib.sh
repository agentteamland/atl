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

# write_test_index_profile seeds ~/.atl/index.json with the first-party
# profile-team entry, pointing at the monorepo subpath on `main`. profile-team
# is a real monorepo team (not a standalone fixture repo), so `atl install`
# fetches teams/profile-team from the atl repo tarball over public HTTPS — the
# blueprint stays hermetic (no dedicated fixture repo) and auth-free.
write_test_index_profile() {
  mkdir -p "$HOME/.atl"
  jq -n '{schemaVersion:1,generatedAt:"2026-07-02T00:00:00Z",teams:[{handle:"agentteamland",name:"profile-team",version:"1.0.0",description:"profile-team e2e (monorepo subpath).",keywords:["profile"],scope:"global",verified:true,source:{repo:"agentteamland/atl",subpath:"teams/profile-team",ref:"main"}}]}' > "$HOME/.atl/index.json"
}

# write_test_index_delivery seeds ~/.atl/index.json with the first-party
# delivery-team entry, pointing at the monorepo subpath on `main` (delivery-team
# is not yet in the published catalog, so the e2e injects it the same way
# write_test_index_profile does). It is project-scope, so the blueprint installs
# it into the project, not globally. The ceremony CONTENT comes from `main`; the
# mock MCP server + this blueprint come from the image (COPY test/e2e/), so a
# delivery-team change on the branch still tests against the merged ceremonies.
write_test_index_delivery() {
  mkdir -p "$HOME/.atl"
  jq -n '{schemaVersion:1,generatedAt:"2026-07-06T00:00:00Z",teams:[{handle:"agentteamland",name:"delivery-team",version:"0.1.0",description:"delivery-team e2e (monorepo subpath).",keywords:["delivery"],scope:"project",verified:true,source:{repo:"agentteamland/atl",subpath:"teams/delivery-team",ref:"main"}}]}' > "$HOME/.atl/index.json"
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

# reset_delivery_repo force-restores <login>/atl-e2e-delivery to the fixture
# baseline and rebuilds the two-branch flow, so the github-delivery-loop blueprint
# starts from a clean repo even after a prior run left issues, PRs, feature
# branches, or tags behind. The twin of reset_owned_repo, for the delivery fixture.
reset_delivery_repo() {
  local login="$1"
  local repo="$login/atl-e2e-delivery"
  # Delete every prior issue — idempotency labels would otherwise converge onto them,
  # and (unlike PRs) issues CAN be removed, so a truly empty baseline is achievable.
  for n in $(gh issue list --repo "$repo" --state all --limit 200 --json number -q '.[].number' 2>/dev/null); do
    gh issue delete "$n" --repo "$repo" --yes 2>/dev/null || gh issue close "$n" --repo "$repo" 2>/dev/null || true
  done
  # A clean baseline is a PRECONDITION, not best-effort: `gh issue delete` needs a
  # triage/admin token, and if it silently fell back to `close`, stale CLOSED issues
  # would false-pass the `--state all` count assertions. Fail loudly instead.
  local remaining
  remaining=$(gh issue list --repo "$repo" --state all --limit 200 --json number -q 'length' 2>/dev/null || echo 999)
  [ "$remaining" = 0 ] || return 1
  local tmp; tmp=$(mktemp -d)
  git clone -q "https://github.com/$repo.git" "$tmp" || { rm -rf "$tmp"; return 1; }
  local rc=0
  (
    cd "$tmp" || exit 1
    find . -mindepth 1 -maxdepth 1 -not -name '.git' -exec rm -rf {} +
    cp -R /e2e/fixtures/delivery-repo/. .
    git add -A
    git -c user.email=e2e@atl.local -c user.name=atl-e2e commit -q -m "reset: e2e baseline" --allow-empty
    # rebuild the two-branch delivery flow off the fresh main (deleting a PR's head
    # branch below auto-closes any open PR, so no separate pr-close pass is needed)
    git push -q -f origin HEAD:main    || exit 1
    git push -q -f origin HEAD:dev     || exit 1
    git push -q -f origin HEAD:release || exit 1
    for b in $(git ls-remote --heads origin | awk '{print $2}' | sed 's|refs/heads/||' | grep -vE '^(main|dev|release)$'); do
      git push -q --delete origin "$b" 2>/dev/null || true
    done
    for t in $(git ls-remote --tags origin | awk '{print $2}' | sed 's|refs/tags/||' | grep -v '\^{}$'); do
      git push -q --delete origin "$t" 2>/dev/null || true
    done
  ) || rc=1
  rm -rf "$tmp"
  return $rc
}

# reset_delivery_project deletes any prior "atl-e2e-delivery" GitHub Project for the
# owner, creates a fresh one, sets up the autonomous-loop custom fields, and echoes
# its NUMBER (its only stdout — all gh diagnostics go to /dev/null). Status exists by
# default (Todo/In Progress/Done); Iteration is UI-only (gh cannot create it), so the
# blueprint tolerates its absence. Requires the `project` token scope.
reset_delivery_project() {
  local login="$1"
  local existing
  existing=$(gh project list --owner "$login" --format json --limit 100 -q '.projects[] | select(.title=="atl-e2e-delivery") | .number' 2>/dev/null | head -1)
  [ -n "$existing" ] && gh project delete "$existing" --owner "$login" >/dev/null 2>&1 || true
  local num
  num=$(gh project create --owner "$login" --title "atl-e2e-delivery" --format json -q '.number' 2>/dev/null)
  [ -z "$num" ] && return 1
  gh project field-create "$num" --owner "$login" --name "Story Points" --data-type NUMBER >/dev/null 2>&1 || true
  gh project field-create "$num" --owner "$login" --name "Priority" --data-type SINGLE_SELECT --single-select-options "P0,P1,P2,P3" >/dev/null 2>&1 || true
  echo "$num"
}

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

# claude_turn runs ONE headless `claude -p` turn, bounded by a wall-clock
# timeout. Extra args (e.g. --mcp-config) are passed through after the prompt.
#
# Why the bound: `claude -p` has no per-turn timeout of its own, so an API stall
# hangs the blueprint forever -- it dies with no FAIL line AND no "N passed, M
# failed" summary, which makes a transient stall look like an unexplained crash.
# On expiry the turn NAMES ITSELF on stdout and returns non-zero, so the caller's
# `|| bad "..."` fires, finish() still prints the tally, and the blueprint FAILs
# loudly instead of dying silently.
#
# CLAUDE_TURN_TIMEOUT (seconds) overrides the default. -k sends KILL if the turn
# ignores TERM (a wedged MCP grandchild can hold the pipes open).
#
# What lands in turns.log, and why it is written in three parts: `--output-format
# json` prints ONE envelope, and only when the turn COMPLETES. So a turn killed by
# the timeout above contributes NOTHING -- the log reads as though that turn never
# ran, which is exactly when the failure debug needs it most. And the envelope's
# human-readable half is `.result` alone, one JSON-escaped field on a very long
# line, so a `tail` of it is unreadable and a grep matches escaping rather than
# what the model said. Hence: a header BEFORE the turn (lands whatever happens, so
# every turn is attributable), the raw envelope (machine detail), then `.result`
# as plain text (readable + greppable).
CLAUDE_TURN_TIMEOUT="${CLAUDE_TURN_TIMEOUT:-900}"
claude_turn() {
  local prompt="$1"; shift
  local raw="$HOME/.turn.json"
  printf '===== turn: %.120s\n' "$prompt" >>"$HOME/turns.log"
  ( cd "$PROJ" && timeout -k 30s "$CLAUDE_TURN_TIMEOUT" \
      claude -p "$prompt" "$@" --dangerously-skip-permissions --output-format json \
  ) >"$raw" 2>&1
  local rc=$?
  cat "$raw" >>"$HOME/turns.log"
  jq -r '.result // empty' "$raw" >>"$HOME/turns.log" 2>/dev/null
  if [ "$rc" -eq 124 ] || [ "$rc" -eq 137 ]; then
    echo "  turn timed out after ${CLAUDE_TURN_TIMEOUT}s (see turns.log)" | tee -a "$HOME/turns.log"
  fi
  return "$rc"
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
# A note on generatedAt below: index.Resolve UNION-merges the binary's embedded
# seed with this cache and, for a team present in BOTH (any first-party team),
# the entry with the NEWER generatedAt wins. These test indices must therefore
# out-date the embedded seed, or the seed's (stale, released-tag) version of a
# first-party team wins and the blueprint silently tests the wrong content. A
# fixed past date rots as each release bumps the seed's generatedAt; a far-future
# date keeps the cache authoritative forever. (Invisible for teams with no drift
# vs the seed; it bit delivery-team once backends/ diverged from the v2.6.0 seed.)
write_test_index() {
  local owned_login="${1:-}"
  mkdir -p "$HOME/.atl"
  local teams
  teams='[{"handle":"agentteamland","name":"atl-e2e-team","version":"0.1.0","description":"e2e fixture (propose-upstream).","keywords":["e2e"],"scope":"global","verified":true,"source":{"repo":"agentteamland/atl-e2e-team","subpath":"","ref":"main"}}]'
  if [ -n "$owned_login" ]; then
    teams=$(echo "$teams" | jq --arg l "$owned_login" '. + [{handle:$l,name:"atl-e2e-owned",version:"0.1.0",description:"e2e fixture (own-team).",keywords:["e2e"],scope:"global",verified:false,source:{repo:($l+"/atl-e2e-owned"),subpath:"",ref:"main"}}]')
  fi
  jq -n --argjson teams "$teams" '{schemaVersion:1,generatedAt:"2099-01-01T00:00:00Z",teams:$teams}' > "$HOME/.atl/index.json"
}

# write_test_index_profile seeds ~/.atl/index.json with the first-party
# profile-team entry, pointing at the monorepo subpath on `main`. profile-team
# is a real monorepo team (not a standalone fixture repo), so `atl install`
# fetches teams/profile-team from the atl repo tarball over public HTTPS — the
# blueprint stays hermetic (no dedicated fixture repo) and auth-free.
write_test_index_profile() {
  mkdir -p "$HOME/.atl"
  jq -n '{schemaVersion:1,generatedAt:"2099-01-01T00:00:00Z",teams:[{handle:"agentteamland",name:"profile-team",version:"1.0.0",description:"profile-team e2e (monorepo subpath).",keywords:["profile"],scope:"global",verified:true,source:{repo:"agentteamland/atl",subpath:"teams/profile-team",ref:"main"}}]}' > "$HOME/.atl/index.json"
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
  jq -n '{schemaVersion:1,generatedAt:"2099-01-01T00:00:00Z",teams:[{handle:"agentteamland",name:"delivery-team",version:"0.1.0",description:"delivery-team e2e (monorepo subpath).",keywords:["delivery"],scope:"project",verified:true,source:{repo:"agentteamland/atl",subpath:"teams/delivery-team",ref:"main"}}]}' > "$HOME/.atl/index.json"
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

# reset_delivery_repo force-restores <owner>/atl-e2e-delivery to the fixture baseline
# and rebuilds the two-branch flow, so the github-delivery-loop blueprint starts from a
# clean repo even after a prior run left issues, PRs, feature branches, or tags behind.
# The owner is the org (agentteamland by default). The twin of reset_owned_repo.
reset_delivery_repo() {
  local owner="$1"
  local repo="$owner/atl-e2e-delivery"
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
    # branch below auto-closes any open FEATURE-branch PR, so no separate pr-close pass
    # is needed for those). A dev->release promotion PR is the exception: its head is
    # `dev`, which is never deleted, so one left open by a failed run survives the reset
    # — harmless, because /sprint-review's step 6a is open-or-find and reuses it, and the
    # force-push gives it a fresh head that no prior approval record can match.
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
  local owner="$1"
  local existing
  existing=$(gh project list --owner "$owner" --format json --limit 100 -q '.projects[] | select(.title=="atl-e2e-delivery") | .number' 2>/dev/null | head -1)
  [ -n "$existing" ] && gh project delete "$existing" --owner "$owner" >/dev/null 2>&1 || true
  local num
  num=$(gh project create --owner "$owner" --title "atl-e2e-delivery" --format json -q '.number' 2>/dev/null)
  [ -z "$num" ] && return 1
  gh project field-create "$num" --owner "$owner" --name "Story Points" --data-type NUMBER >/dev/null 2>&1 || true
  gh project field-create "$num" --owner "$owner" --name "Priority" --data-type SINGLE_SELECT --single-select-options "P0,P1,P2,P3" >/dev/null 2>&1 || true
  echo "$num"
}

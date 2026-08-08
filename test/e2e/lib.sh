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
    return "$rc"
  fi

  # ONE retry on a transient API failure. A dropped connection has cost a
  # 29-minute blueprint outright: the ceremony did its work -- the Epic, the
  # Feature and the [Technical Analysis] comment all landed, and the assertions
  # after it passed -- but the envelope never arrived, so the turn returned
  # non-zero and the blueprint failed on a network event.
  #
  # Retrying is safe because of a property the turns already guarantee, not
  # because a second attempt is cheap. Every blueprint that calls this — measured
  # 2026-08-08: learning-loop, profile-loop, profile-backup-restore — writes its
  # durable state through the marker queue (content-hash dedup, tombstoned on
  # ack) or through a git snapshot with mirror semantics, so a re-run CONVERGES
  # instead of duplicating. That is the whole reason this is one line rather than
  # a rollback. Do not extend this to a step without that guarantee.
  #
  # Only genuinely transient shapes, and only once: a real failure must stay a
  # failure, and a retry loop would hide a systematic error behind a longer wait.
  if [ "$rc" -ne 0 ] && grep -qiE 'API Error|Connection closed|overloaded_error|rate.?limit|502 Bad Gateway|503 Service' "$raw" 2>/dev/null; then
    echo "  transient API failure — retrying the turn once (this turn's writes converge)" | tee -a "$HOME/turns.log"
    printf '===== RETRY: %.120s\n' "$prompt" >>"$HOME/turns.log"
    ( cd "$PROJ" && timeout -k 30s "$CLAUDE_TURN_TIMEOUT" \
        claude -p "$prompt" "$@" --dangerously-skip-permissions --output-format json \
    ) >"$raw" 2>&1
    rc=$?
    cat "$raw" >>"$HOME/turns.log"
    jq -r '.result // empty' "$raw" >>"$HOME/turns.log" 2>/dev/null
    if [ "$rc" -eq 0 ]; then
      echo "  retry succeeded" | tee -a "$HOME/turns.log"
    else
      echo "  retry also failed (rc=$rc) — treating as a real failure" | tee -a "$HOME/turns.log"
    fi
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
# vs the seed; it bit a first-party team once its content diverged from the seed.)
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
# profile-team entry, pointing at the monorepo subpath on ATL_E2E_TEAM_REF (the
# current branch — never `main`: a pin to main means an edited team is never
# loaded, every assertion passes on main's copy, and the run reports green on
# content it never saw). profile-team is a real monorepo team (not a standalone
# fixture repo), so `atl install` fetches teams/profile-team from the atl repo
# tarball over public HTTPS — the blueprint stays hermetic (no dedicated fixture
# repo) and auth-free.
write_test_index_profile() {
  mkdir -p "$HOME/.atl"
  jq -n --arg ref "${ATL_E2E_TEAM_REF:-main}" '{schemaVersion:1,generatedAt:"2099-01-01T00:00:00Z",teams:[{handle:"agentteamland",name:"profile-team",version:"1.0.0",description:"profile-team e2e (monorepo subpath).",keywords:["profile"],scope:"global",verified:true,source:{repo:"agentteamland/atl",subpath:"teams/profile-team",ref:$ref}}]}' > "$HOME/.atl/index.json"
}

# write_test_index_advisory seeds ~/.atl/index.json with BOTH first-party teams
# the transitive-install path needs: personal-advisory-team (the one installed)
# and profile-team (the dependency it declares in team.json `dependencies`).
#
# Both are monorepo subpaths on ATL_E2E_TEAM_REF, for the reason above: the
# container fetches team content from that ref and mounts nothing from the
# working tree.
#
# The dependency is listed here as a BARE name in personal-advisory-team's
# manifest ("profile-team"), which `resolveDep` resolves via `LookupByName`,
# preferring a verified publisher — hence `verified: true` on both.
#
# Why profile-team is in this index is NOT "otherwise the dependency cannot be
# resolved" — that was the first explanation and it is measurably wrong. Removing
# it from here changes nothing observable: `index.Resolve` UNION-merges this cache
# with the binary's EMBEDDED SEED, which carries every first-party team, so the
# recursion resolves it either way (verified — the blueprint stayed 29/29 with
# this entry deleted).
#
# It is here for the generatedAt reason above: the entry the merge
# keeps is the one with the newer `generatedAt`, so the far-future stamp makes
# THIS entry authoritative and the dependency is fetched from ATL_E2E_TEAM_REF.
# Drop it and the dependency silently installs from the seed's released TAG — a
# branch editing teams/profile-team would then be exercised for the consumer and
# not for the dependency, and every assertion would still pass. Same pinned-ref
# trap, one level down the dependency edge.
write_test_index_advisory() {
  mkdir -p "$HOME/.atl"
  jq -n --arg ref "${ATL_E2E_TEAM_REF:-main}" '{schemaVersion:1,generatedAt:"2099-01-01T00:00:00Z",teams:[
    {handle:"agentteamland",name:"personal-advisory-team",version:"0.1.0",description:"personal-advisory-team e2e (monorepo subpath).",keywords:["advisory"],scope:"global",verified:true,source:{repo:"agentteamland/atl",subpath:"teams/personal-advisory-team",ref:$ref}},
    {handle:"agentteamland",name:"profile-team",version:"1.0.0",description:"profile-team e2e (monorepo subpath).",keywords:["profile"],scope:"global",verified:true,source:{repo:"agentteamland/atl",subpath:"teams/profile-team",ref:$ref}}
  ]}' > "$HOME/.atl/index.json"
}

# ---- attributable failures --------------------------------------------------
#
# A blueprint-ending assertion must be able to name its own cause. The pattern
# these helpers replace is `X=$(gh ... 2>/dev/null)` followed by
# `[ -n "$X" ] || { bad "could not do the thing"; finish; exit 1; }` — where the
# only sentence that could explain the red was written to stderr and thrown away.
#
# That cost a real release gate: a GitHub blueprint died 40s in with
# `FAIL - could not seed the PBI` as its entire output, and the cause (most
# likely GitHub's secondary rate limit, which is invisible in /rate_limit) had to
# be reconstructed by hand afterwards from a two-hour run that already had the
# answer. Suppressing stderr converts a loud failure into a silent skip.
#
# Scope note: this is deliberately NOT applied to every suppressed call in the
# suite. A non-fatal `|| true` best-effort call is noise when it fails; only the
# assertions that END a blueprint are worth the extra sentence, because those are
# the ones whose missing cause costs a re-run to recover.
LAST_ERR="$HOME/.last-stderr"

# gh_try runs a command with its stderr CAPTURED to $LAST_ERR rather than
# discarded, and passes its stdout through unchanged, so a caller can keep the
# `X=$(gh_try gh ...)` shape and still recover the diagnostic via `why`.
gh_try() {
  : > "$LAST_ERR"
  "$@" 2>"$LAST_ERR"
}

# why echoes the last captured stderr as ONE bounded line, for interpolation into
# a `bad` message. Flattened because a `bad` line is grepped and tallied by the
# watcher, and bounded because a stack-trace-length assertion message is as
# unreadable as no message at all. Says so explicitly when there was no stderr —
# "the command failed and said nothing" is itself a finding, and is otherwise
# indistinguishable from having forgotten to capture it.
why() {
  local msg
  msg=$(tr '\n\t' '  ' < "$LAST_ERR" 2>/dev/null | tr -s ' ' | sed 's/^ *//; s/ *$//')
  if [ -n "$msg" ]; then printf '%.400s' "$msg"; else printf 'no stderr captured'; fi
}

# gh_login echoes the authenticated GitHub login (GH_TOKEN is passed through by
# the runner); empty if the call fails. Routed through gh_try because an empty
# login has more than one cause — an unauthenticated token, an expired one, and
# a 5xx from the API all return the same empty string, and every caller reports
# the first of those. `why` is what tells them apart.
gh_login() {
  gh_try gh api user -q .login || true
}

# gh_seed_issue creates ONE seeded issue and echoes its number.
#
#   gh_seed_issue <repo> <title> <body-file> [label ...]
#
# The retry follows the precedent claude_turn sets above, including its boundary:
# retrying is safe because of a convergence property, never because a second
# attempt is cheap. `gh issue create` has no such property of its own — a create
# that succeeded but whose response was lost would be DUPLICATED by a naive
# retry — so the convergence is supplied here, by checking first for an issue
# with this exact title before the second attempt. Found -> adopt it; not found
# -> create. That is the check-first-by-stable-key contract, with the title as
# the key.
#
# One retry, not a loop: a real failure must stay a failure, and a retry loop
# would hide a systematic error behind a longer wait. The backoff is sized for
# the failure actually observed — GitHub's secondary (content-creation) rate
# limit, which clears in about a minute — so a lost two-hour gate becomes a
# one-minute pause.
GH_SEED_BACKOFF="${GH_SEED_BACKOFF:-60}"
gh_seed_issue() {
  local repo="$1" title="$2" bodyfile="$3"; shift 3
  local labels=() l
  for l in "$@"; do labels+=(--label "$l"); done

  local attempt url num
  for attempt in 1 2; do
    if [ "$attempt" -eq 2 ]; then
      # Converge, don't duplicate: attempt 1 may have landed the issue and lost
      # the response. Diagnostics to stderr so they reach the run log without
      # polluting this function's stdout (its number is read by $(...)).
      num=$(gh_find_issue_by_title "$repo" "$title")
      if [ -n "$num" ]; then
        echo "  seed response was lost but the issue exists (#$num) — adopting it" >&2
        echo "$num"; return 0
      fi
      echo "  seed failed, retrying once in ${GH_SEED_BACKOFF}s: $(why)" >&2
      sleep "$GH_SEED_BACKOFF"
    fi
    url=$(gh_try gh issue create --repo "$repo" --title "$title" \
            ${labels[@]+"${labels[@]}"} --body-file "$bodyfile")
    num=$(echo "$url" | grep -oE '[0-9]+$')
    [ -n "$num" ] && { echo "$num"; return 0; }
  done
  return 1
}

# gh_find_issue_by_title echoes the number of an existing issue with this EXACT
# title, or nothing. Exact rather than a `--search` match: search is fuzzy and
# eventually-consistent, and an idempotency check that matches the wrong issue is
# worse than one that finds none.
gh_find_issue_by_title() {
  gh issue list --repo "$1" --state all --limit 100 --json number,title 2>/dev/null \
    | jq -r --arg t "$2" '[.[] | select(.title == $t) | .number] | first // empty'
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


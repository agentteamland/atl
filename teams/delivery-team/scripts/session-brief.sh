#!/usr/bin/env bash
# session-brief.sh — the delivery drive loop's session briefing (drive-loop D5).
#
# Declared by team.json as `capabilities.delivery.sessionScript`. `atl
# session-start` runs every declared session script and forwards its stdout into
# the session's context; core learns neither this team nor which backend it runs
# on, so everything backend-shaped is decided HERE, from .delivery/config.json.
#
# What it answers, for a human resuming work: the branch under you is a delivery
# branch — which card is that, what state is it in, which sprint does it carry,
# and has the code moved past the board? The last is the drift nudge: a MERGED PR
# under a card that is still open is `/work-sync`'s first finding, and the moment
# it is cheapest to fix is the session you come back in.
#
# THE OUTPUT CONTRACT IS SILENCE. Core forwards stdout only on a zero exit, so
# every "nothing to say" path here exits 0 with no output, and every failure path
# does the same rather than reporting on itself: a session start is not the place
# to explain that a board read failed. That is why nothing below is fatal and why
# stderr is discarded — but it is also why a genuinely broken declaration would be
# invisible, which is what `atl skills check` and `atl doctor` exist to catch.
#
# NOT `set -e`: an ordinary miss here (a grep that matches nothing, a card with no
# sprint label) is a normal branch of the answer, and aborting on it would silently
# drop the rest of a briefing that was fine.

set -u

# --- cheap local gates: cost nothing off a delivery branch --------------------

config=".delivery/config.json"
[ -r "$config" ] || exit 0

branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null) || exit 0
# The engine's own grammar: delivery/<sprint-slug>/<work-item-id> (dispatch's
# BranchName). Anything else is not a unit of work and has nothing to brief.
case "$branch" in
  delivery/*/*) ;;
  *) exit 0 ;;
esac
id=${branch##*/}
case "$id" in
  ''|*[!0-9]*) exit 0 ;;  # not a work-item id — say nothing rather than guess
esac

# --- backend arm -------------------------------------------------------------

# Read the backend without a jq dependency: `gh --jq` is not available here (the
# value decides whether gh is even the right tool), and jq itself is not a
# delivery-team prerequisite on any platform. A single sed over the one field is
# enough, and a config we cannot read resolves to silence like everything else.
backend=$(sed -n 's/.*"backend"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$config" | head -n 1)
[ -n "$backend" ] || backend="azure"  # the schema default (workerenv.ActiveBackend)

# AZURE IS AN EXPLICIT SILENT ARM, not an oversight.
#
# Azure's board is reached through an MCP surface — an LLM-side tool with no
# shell client and nothing to shell out to, the way GitHub has `gh`. A shell
# script cannot read an Azure work item at all, and the local proxies do not
# carry the answer: plan.json records that a plan was materialised, not what
# state the card is in. Inventing a transport here (raw REST against a PAT this
# script would have to resolve itself) is a second credential path for one
# briefing line, and guessing from local state would report card states that are
# not the board's. Both are worse than the honest outcome, which is that an Azure
# project gets no session briefing until the read has a real transport.
[ "$backend" = "github" ] || exit 0

command -v gh >/dev/null 2>&1 || exit 0

owner=$(sed -n 's/.*"owner"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$config" | head -n 1)
repo=$(sed -n 's/.*"repo"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$config" | head -n 1)
[ -n "$owner" ] && [ -n "$repo" ] || exit 0
nwo="$owner/$repo"

# --- the board reads ---------------------------------------------------------

# ONE call for the card, projected to a single TSV row. Not three calls for
# three fields: this runs on the session-start path, where every round trip is
# latency a human waits through. `--jq` is gh's own built-in, so nothing here
# needs jq on PATH. `@tsv` escapes tabs inside a value, so a title containing one
# cannot shift the fields.
#
# `state` is the ISSUE state, and on this backend that is load-bearing rather
# than a proxy: the adapter defines Done as the issue CLOSED plus the Projects
# Status set to Done, so an open issue is definitively not Done — no project read
# is needed to establish it.
card=$(gh issue view "$id" --repo "$nwo" --json title,state,labels --jq \
  '[.title, .state, ([.labels[].name | select(startswith("sprint:"))] | join(" "))] | @tsv' 2>/dev/null) || exit 0
[ -n "$card" ] || exit 0
IFS=$(printf '\t') read -r title state sprint <<EOF
$card
EOF
[ -n "${title:-}" ] || exit 0

# The Projects v2 Status field, BEST EFFORT and deliberately last. It is the
# state the drive loop actually moves (Todo / In Progress / Done), but reading it
# needs project scope on the token, which the issue read above does not — so a
# token without it must cost the Status line, never the briefing.
# shellcheck disable=SC2016  # $owner/$repo/$number are GraphQL variables bound by -F, not shell expansions
status=$(gh api graphql -F owner="$owner" -F repo="$repo" -F number="$id" -f query='
  query($owner:String!, $repo:String!, $number:Int!) {
    repository(owner:$owner, name:$repo) {
      issue(number:$number) {
        projectItems(first:5) { nodes {
          fieldValueByName(name:"Status") {
            ... on ProjectV2ItemFieldSingleSelectValue { name }
          }
        } }
      }
    }
  }' --jq '[.data.repository.issue.projectItems.nodes[].fieldValueByName.name // empty] | first // empty' 2>/dev/null)

# The drift read: does this branch already have a MERGED PR? `--state all`
# because a merged PR is not an open one, and the whole question is about a
# branch whose work has landed.
pr_state=$(gh pr list --repo "$nwo" --head "$branch" --state all \
  --json state,number --jq 'sort_by(.number) | last | .state // empty' 2>/dev/null)

# --- the briefing ------------------------------------------------------------

# Written as `if` blocks rather than `[ cond ] && assign`: that one-liner is an
# expression whose exit code is the condition's, so as the last statement before
# an `exit` it decides the script's exit status — and on this contract a non-zero
# exit throws the whole briefing away.
line="atl delivery: on $branch — #$id \"$title\""
if [ -n "${status:-}" ]; then line="$line, status $status"; fi
if [ -n "${sprint:-}" ]; then line="$line, $sprint"; fi
if [ -n "${state:-}" ]; then
  line="$line (issue $(printf '%s' "$state" | tr '[:upper:]' '[:lower:]'))"
fi
printf '%s\n' "$line"

# The drift nudge. Stated as a finding plus the command that resolves it, and
# NOT dispatched: moving a card is a write, and `/work-move` reads the board back
# to confirm. The condition is deliberately the conservative one — merged PR AND
# the issue still open — because on this backend a closed issue is a precondition
# of Done, so this cannot fire against a card that is genuinely finished.
if [ "${pr_state:-}" = "MERGED" ] && [ "${state:-}" = "OPEN" ]; then
  printf '%s\n' "atl delivery: drift — this branch's PR is MERGED while #$id is still open, so the board is behind the code. /work-move $id done closes it out (it re-checks the merge before writing)."
fi

exit 0

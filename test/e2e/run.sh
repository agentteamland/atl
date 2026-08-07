#!/usr/bin/env bash
#
# Host runner for the atl e2e harness. Builds the image once, then runs each
# selected blueprint in a FRESH container (docker run --rm — kill + recreate per
# scenario, so blueprints never share state).
#
#   test/e2e/run.sh                       # every blueprint (auth-gated; missing-auth ones skip)
#   test/e2e/run.sh install publish-own   # named blueprints only
#   test/e2e/run.sh --changed             # only what the diff can reach
#   test/e2e/run.sh --lanes               # print the lane partition and stop
#
# Blueprints run in LANES. A blueprint that force-resets a shared external fixture
# declares it on a `# fixture:` line; blueprints sharing one run serially, and lanes
# run concurrently. Parallelism is therefore DERIVED from the declaration rather than
# chosen, so two blueprints can only overlap if they named different fixtures — the
# unsafe case is unreachable rather than merely avoided.
#
# Auth is passed into the container only when present on the host:
#   - gh:     GH_TOKEN (from your `gh auth token`)            — publish blueprints
#   - Claude: CLAUDE_CODE_OAUTH_TOKEN or ANTHROPIC_API_KEY    — the learning-loop blueprint
# Each blueprint declares its need on a `# needs: none|gh|token|gh+token` line; a
# blueprint whose auth is absent is skipped (so this same script is CI-safe and
# local-full). `gh+token` (github-delivery-loop) needs BOTH a GH_TOKEN and a Claude
# token — the real GitHub-backend Layer-B loop.

set -euo pipefail
cd "$(dirname "$0")/../.."   # atl repo root (build context)
BPDIR="test/e2e/blueprints"
RUNLOG="test/e2e/.last-run.log"

# Mirror everything to a STABLE path so the watcher can be armed without being handed a
# path — guessing one (`ls -t` over a task directory, say) is how you end up watching the
# wrong file and trusting a green that belongs to something else.
#
# Re-exec through a pipe rather than `exec > >(tee …)`: a process substitution is not
# waited for, so the final tally — the one line that matters most — can be lost when the
# script exits first. The pipeline is waited for, and PIPESTATUS[0] carries the real
# status through unchanged.
if [ -z "${ATL_E2E_TEED:-}" ]; then
  export ATL_E2E_TEED=1
  bash "$BPDIR/../run.sh" "$@" 2>&1 | tee "$RUNLOG"
  exit "${PIPESTATUS[0]}"
fi

# Refuse an unrecognised flag rather than reinterpreting it — and do it HERE,
# above the ref resolution, because everything below this line costs network
# fetches and then a docker build.
#
# Both routes an invented flag can take end somewhere the caller did not ask for.
# A bare `--dry-run` becomes a blueprint NAME and dies with a confusing "no such
# blueprint"; `--changed --dry-run` is forwarded to select.sh, where it is read
# as a base REF and silently falls back to main — so an invented flag starts a
# real two-hour run, image build and all, while the caller believes they asked
# for a preview. Neither a blueprint name nor a git ref begins with `-`, so this
# rejects exactly the mistaken case and nothing else.
for arg in "$@"; do
  case "$arg" in
    --lanes|--changed) ;;
    -*) echo "!! unknown flag: $arg" >&2
        echo "!! usage: run.sh [--lanes] [--changed [<base-ref>] | <blueprint>...]" >&2
        exit 2 ;;
  esac
done

# Resolve the ref the CONTAINER installs TEAM content from (host side).
#
# The test index pins `source.ref` and the container fetches that ref from
# GitHub — it mounts NOTHING from this working tree. So an edit under teams/ is
# exercised only once it is pushed to the ref being installed. A branch that
# edits a ceremony and runs against `main` reports green on content the test
# never loaded, which is silent by construction: every assertion still passes,
# because they are all passing on main's copy. That has cost a full suite run.
#
# Default to the current branch, then verify what the container WILL fetch
# matches what is on disk. Override with ATL_E2E_TEAM_REF to pin a ref by hand.
ATL_E2E_TEAM_REF="${ATL_E2E_TEAM_REF:-$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo main)}"
if [ "$ATL_E2E_TEAM_REF" = "HEAD" ]; then ATL_E2E_TEAM_REF=main; fi   # detached
# ^ an `[ … ] && x` one-liner would return non-zero on the common (non-detached)
#   path and `set -e` would kill the runner before a single blueprint ran.
git fetch -q origin main 2>/dev/null || true

if ! git fetch -q origin "$ATL_E2E_TEAM_REF" 2>/dev/null; then
  # Not on origin. Harmless ONLY if this branch changes no team content at all.
  if git diff --quiet origin/main -- teams/ 2>/dev/null; then
    echo ">> ref '$ATL_E2E_TEAM_REF' is not on origin, but teams/ matches main — using main"
    ATL_E2E_TEAM_REF=main
  else
    echo "!! '$ATL_E2E_TEAM_REF' changes teams/ but is not pushed to origin." >&2
    echo "!! The container installs team content from GitHub and mounts nothing" >&2
    echo "!! from here, so this run would exercise main's copy and report green" >&2
    echo "!! on a change it never loaded. Push the branch, or set ATL_E2E_TEAM_REF." >&2
    exit 2
  fi
elif ! git diff --quiet FETCH_HEAD -- teams/; then
  echo "!! local teams/ differs from origin/$ATL_E2E_TEAM_REF — the container installs" >&2
  echo "!! team content from that ref, so those edits would NOT be exercised and a" >&2
  echo "!! green run would mean nothing. Commit and push them first." >&2
  exit 2
fi
echo ">> team content from ref: $ATL_E2E_TEAM_REF"
echo ">> logging to $RUNLOG — a full run takes ~2h and reports only at the end, so watch it:"
echo ">>   test/e2e/watch.sh          (under the Monitor tool, persistent)"

# Pick the blueprints BEFORE the image build: --changed can legitimately select
# nothing, and paying a docker build to discover that is the same "expensive setup
# above the precondition" mistake the ref check above exists to avoid.
SHOW_LANES=0
if [ "${1:-}" = "--lanes" ]; then SHOW_LANES=1; shift; fi
export SHOW_LANES
if [ "${1:-}" = "--changed" ]; then
  shift
  names="$(test/e2e/select.sh "$@" | tr '\n' ' ')"
  names="${names% }"
  if [ -z "$names" ]; then
    echo ">> --changed selected no blueprints — nothing to run"
    exit 0
  fi
  echo ">> --changed selected: $names"
elif [ "$#" -gt 0 ]; then
  names="$*"
  echo ">> running selected blueprints: $names"
else
  names="$(ls "$BPDIR"/*.sh | xargs -n1 basename | sed 's/\.sh$//' | sort)"
  echo ">> running ALL blueprints (full suite — the default; pass names to run a subset)"
fi

SUITE_START=$SECONDS
echo ">> building atl-e2e image"
docker build -f test/e2e/Dockerfile -t atl-e2e . >/dev/null
BUILD_SECS=$((SECONDS - SUITE_START))
echo ">> image ready (${BUILD_SECS}s)"

# Resolve auth once (host side). Prefer gh's configured token; fall back to a
# GH_TOKEN env so CI can pass a PAT (there is no interactive `gh auth login`).
GH_TOKEN_VAL="$(gh auth token 2>/dev/null || true)"
: "${GH_TOKEN_VAL:=${GH_TOKEN:-}}"
CLAUDE_TOK="${CLAUDE_CODE_OAUTH_TOKEN:-}"
API_KEY="${ANTHROPIC_API_KEY:-}"

# Per-blueprint wall clock. Without it the suite's cost is invisible: the LLM-tier
# blueprints run many minutes each while the auth-free ones finish in seconds, so the
# progress fraction misleads badly -- a LONGER run can mean the suite got FURTHER, not
# that it slowed down -- and "is this normal?" can only be answered by hand-timing it.
pass=0; fail=0; skip=0; ratelimited=0
timings=()

# ---- lane partitioning ------------------------------------------------------
#
# Blueprints that force-reset the SAME external fixture cannot overlap: the six
# GitHub delivery blueprints each restore a fixture repo to a baseline AND delete
# and recreate a Project of a fixed title, so two at once is not slow, it is
# destructive — the loser sees a board that vanished mid-run and fails an
# assertion that says nothing about concurrency.
#
# So parallelism is DERIVED from a declaration rather than chosen: a blueprint
# names the fixture it mutates on a `# fixture:` line, blueprints sharing one run
# serially in a lane, and lanes run concurrently. Two blueprints can only overlap
# if they declared different fixtures, which makes the unsafe case unreachable
# rather than merely avoided. A blueprint that declares none joins the `free`
# lane, which is also serial — parallelism is bounded by the number of declared
# fixtures, so it cannot grow into N concurrent LLM sessions by accident.
declare -a LANE_KEYS=()
lane_of() { sed -n 's/^# fixture: //p' "$1" | head -1; }

runnable=""
for name in $names; do
  bp="$BPDIR/$name.sh"
  if [ ! -f "$bp" ]; then echo "!! no such blueprint: $name" >&2; exit 2; fi
  needs="$(sed -n 's/^# needs: //p' "$bp" | head -1)"; needs="${needs:-none}"
  case "$needs" in gh|gh+token)
    [ -z "$GH_TOKEN_VAL" ] && { echo ">> skip $name (needs gh — run 'gh auth login')"; skip=$((skip + 1)); continue; } ;;
  esac
  case "$needs" in token|gh+token)
    { [ -z "$CLAUDE_TOK" ] && [ -z "$API_KEY" ]; } && { echo ">> skip $name (needs a Claude token — CLAUDE_CODE_OAUTH_TOKEN)"; skip=$((skip + 1)); continue; } ;;
  esac
  runnable="$runnable $name"
done

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# Clear every lane log before writing the ones this run needs. A lane that ran
# LAST time and not this time would otherwise leave a file nobody is writing, and
# watch.sh globs them all to measure per-lane liveness — so a stale log never
# grows, and after the stall threshold the watcher reports a stall on a lane that
# is not running. A watcher that cries wolf is worse than none: the next real
# stall is the one nobody believes.
rm -f "${RUNLOG%.log}".lane-*.log
attempted=0
for name in $runnable; do
  key="$(lane_of "$BPDIR/$name.sh")"; key="${key:-free}"
  echo "$name" >> "$WORK/lane.$key"
  case " ${LANE_KEYS[*]-} " in *" $key "*) ;; *) LANE_KEYS+=("$key") ;; esac
  attempted=$((attempted + 1))
done

if [ "$attempted" -gt 0 ]; then
  echo ">> ${#LANE_KEYS[@]} lane(s): $(for k in "${LANE_KEYS[@]}"; do printf '%s(%s) ' "$k" "$(wc -l < "$WORK/lane.$k" | tr -d ' ')"; done)"
fi

# --lanes prints the partition and stops. Cheap enough to run before every real
# suite, which matters because a wrong declaration is not a slow run — it is two
# lanes force-resetting one fixture, and the failure surfaces as an unrelated
# assertion in whichever lane lost.
# GraphQL budget preflight — only when a delivery lane is actually selected.
#
# The budget only matters to the GitHub delivery blueprints; a subset of auth-free
# ones has nothing to do with it, and an unconditional check blocks work that could
# not spend a point. (Caught by using it: a `needs: none` blueprint was refused.)
#
# The first full parallel run exhausted the 5000/hr budget at ~30 minutes and took
# down two blueprints in the FREE lane that never touch Projects at all — with
# assertion messages ("no tag in output") that say nothing about rate limits.
#
# Measured afterwards, because the obvious culprit was wrong: the fixture reset
# (delete + create a Project, two field-creates) costs 15 points, i.e. 90 across a
# whole suite — 1.8% of the budget. The consumption is in the CEREMONY TURNS, where
# each `claude -p` agent queries and mutates the board. That is the work under test,
# so it cannot be optimised away.
#
# Which gives the real bound: a full suite costs approximately ONE HOURLY QUOTA, so
# its runtime cannot be compressed far below an hour however many lanes there are.
# The requirement scales with what was SELECTED, not with what a full suite costs.
# A flat full-suite threshold refuses a single delivery blueprint because the budget
# is mid-recovery — the same over-reach as checking at all for a `needs: none`
# subset, one level down.
#
# 400 per delivery blueprint, and where that number comes from matters. The first
# estimate was 800, drawn from the run that EXHAUSTED the budget — a corrupted
# three-lane run whose draw included its own re-runs. It then refused the FULL
# SUITE at 4891/5000: the guard blocking the exact case it exists to protect.
#
# Grounded on the clean two-lane run instead: 4659 -> 2397 across its first twenty
# minutes for seven delivery blueprints, so a peak draw near 2300, ~325 each. 400
# rounds that up without inventing headroom.
#
# Note this is PEAK draw, not total. A run longer than an hour crosses a reset and
# is refilled while it works, which is why what a suite CONSUMES is not what a
# preflight should ask for — asking for the total is what produced the 800.
delivery_n=0
for k in ${LANE_KEYS[@]+"${LANE_KEYS[@]}"}; do
  [ "$k" = free ] && continue
  delivery_n=$((delivery_n + $(wc -l < "$WORK/lane.$k")))
done
if [ "$delivery_n" -gt 0 ] && command -v gh >/dev/null 2>&1; then
  need=$((delivery_n * 400))
  gql_left="$(gh api rate_limit --jq '.resources.graphql.remaining' 2>/dev/null || echo "")"
  if [ -n "$gql_left" ] && [ "$gql_left" -lt "$need" ] 2>/dev/null; then
    echo "!! GraphQL budget is ${gql_left}/5000 and $delivery_n delivery blueprint(s) need about ${need}" >&2
    echo "   and the failures surface on UNRELATED blueprints. Wait for the hourly reset," >&2
    echo "   or run a subset that excludes the github-delivery-* lanes." >&2
    [ "${ATL_E2E_IGNORE_BUDGET:-0}" = 1 ] || exit 4
  fi
fi

if [ "${SHOW_LANES:-0}" = 1 ]; then
  for k in ${LANE_KEYS[@]+"${LANE_KEYS[@]}"}; do
    echo "lane $k:"; sed 's/^/  /' "$WORK/lane.$k"
  done
  exit 0
fi

# Emitting a finished blueprint's output takes a mutex so a block from one lane is
# never interleaved into another's. The watcher greps whole `<< name VERDICT`
# lines and the `========= blueprint:` header, and a torn block would break both.
LOCK="$WORK/emit.lock"
emit() {
  while ! mkdir "$LOCK" 2>/dev/null; do sleep 0.2; done
  cat "$1"
  rmdir "$LOCK"
}

run_lane() {
  local key="$1" name bp needs bp_start bp_secs verdict rc
  local lanelog="${RUNLOG%.log}.lane-$key.log"
  : > "$lanelog"
  while read -r name; do
    bp="$BPDIR/$name.sh"
    needs="$(sed -n 's/^# needs: //p' "$bp" | head -1)"; needs="${needs:-none}"
    local secret_env=()
    case "$needs" in gh|gh+token)    secret_env+=(-e "GH_TOKEN=$GH_TOKEN_VAL") ;; esac
    case "$needs" in token|gh+token) secret_env+=(-e "CLAUDE_CODE_OAUTH_TOKEN=$CLAUDE_TOK" -e "ANTHROPIC_API_KEY=$API_KEY") ;; esac
    # The lane key IS the fixture name, so a lane can only ever touch its own.
    local fixture_env=()
    [ "$key" != "free" ] && fixture_env+=(-e "ATL_E2E_DELIVERY_REPO=$key")

    {
      echo ""
      echo "========= blueprint: $name (needs: $needs, lane: $key) ========="
    } | tee -a "$lanelog" > "$WORK/out.$name"
    bp_start=$SECONDS
    rc=0
    # Live to the lane log, buffered to the temp file. Both halves are load-bearing
    # and they pull in opposite directions:
    #
    #   - the AGGREGATE must stay block-contiguous, or a block torn between two
    #     lanes breaks the watcher's `<< name VERDICT` and `========= blueprint:`
    #     greps — hence the buffer, flushed under a mutex when the blueprint ends.
    #   - the LANE log must grow CONTINUOUSLY, or liveness is unmeasurable: with
    #     output buffered to the end, a lane holding a 34-minute blueprint looks
    #     silent for 34 minutes and the stall detector false-fires at 20. Growth
    #     only at blueprint boundaries is not a heartbeat, it is a completion
    #     signal wearing one.
    #
    # tee gives both from one stream. PIPESTATUS carries docker's real status
    # through — $? would report tee's.
    # errexit off across the pipeline. A FAILING blueprint is the normal case this
    # loop exists to record, but under `set -e` a non-zero pipeline kills the whole
    # lane before its result line is written — so one red blueprint would silently
    # take every later blueprint in its lane with it. `|| true` cannot be used
    # here: it makes PIPESTATUS describe `true` rather than the pipeline.
    set +e
    docker run --rm \
      -e BLUEPRINT="$name" \
      -e ATL_E2E_TEAM_REF="$ATL_E2E_TEAM_REF" \
      ${secret_env[@]+"${secret_env[@]}"} \
      ${fixture_env[@]+"${fixture_env[@]}"} \
      atl-e2e bash "/e2e/blueprints/$name.sh" 2>&1 | tee -a "$lanelog" >> "$WORK/out.$name"
    rc="${PIPESTATUS[0]}"
    set -e
    bp_secs=$((SECONDS - bp_start))
    if [ "$rc" -eq 0 ]; then
      verdict=PASSED
    elif grep -qi 'rate limit' "$WORK/out.$name"; then
      # Name it. A rate-limited run is not evidence about the product, and an
      # assertion message like "no tag in output" reads exactly like one.
      verdict=RATELIMIT
    else
      verdict=FAILED
    fi
    echo "<< $name $verdict (${bp_secs}s)" | tee -a "$lanelog" >> "$WORK/out.$name"
    # The lane log is the watcher's per-lane liveness surface: with several lanes
    # running, "no output for N seconds" is meaningless on the aggregate, because
    # a stalled lane is hidden by a busy one. It was written live above; the
    # aggregate gets the whole block now, atomically.
    emit "$WORK/out.$name"
    echo "$bp_secs|$name|$verdict|$needs" >> "$WORK/results"
  done < "$WORK/lane.$key"
  # Mark the lane finished. Without this a lane that has COMPLETED is
  # indistinguishable from one that is wedged: both stop growing, and the watcher
  # reports a stall on work that is already done.
  echo "===== lane $key complete =====" >> "$lanelog"
}

: > "$WORK/results"
for key in ${LANE_KEYS[@]+"${LANE_KEYS[@]}"}; do
  run_lane "$key" &
done
wait

# No silent truncation. A lane that died without reporting would otherwise shrink
# the tally to something that still reads clean — the failure direction a runner
# must never take, because a short green run is indistinguishable from a correct
# one. (Same discipline the --changed selector is built on.)
got=$(wc -l < "$WORK/results" | tr -d ' ')
if [ "$got" -ne "$attempted" ]; then
  echo "!! harness bug: $attempted blueprint(s) attempted but $got result(s) recorded — a lane died silently" >&2
  exit 3
fi
while IFS='|' read -r s n v d; do
  timings+=("$s|$n|$v|$d")
  case "$v" in
    PASSED)    pass=$((pass + 1)) ;;
    RATELIMIT) ratelimited=$((ratelimited + 1)) ;;
    *)         fail=$((fail + 1)) ;;
  esac
done < "$WORK/results"

SUITE_SECS=$((SECONDS - SUITE_START))

# Slowest first: the point of the table is to show WHERE the time goes, and the
# distribution is extremely lopsided -- the blueprints that never invoke `claude` finish
# in seconds while a single LLM-tier one can run twenty minutes on its own.
if [ "${#timings[@]}" -gt 0 ]; then
  echo ""
  echo "===== timing (slowest first; image build ${BUILD_SECS}s) ====="
  printf '%s\n' "${timings[@]}" | sort -t'|' -k1,1nr | while IFS='|' read -r s n v d; do
    printf '  %6ss  %-38s %-6s %s\n' "$s" "$n" "$v" "$d"
  done
fi

echo ""
rl_note=""
if [ "$ratelimited" -gt 0 ]; then
  rl_note=", $ratelimited RATE-LIMITED"
  echo ""
  echo "!! $ratelimited blueprint(s) hit the GitHub rate limit. Those verdicts say NOTHING about"
  echo "   the product — re-run them after the hourly reset before drawing any conclusion."
fi
echo ""
echo "===== harness: $pass passed, $fail failed$rl_note, $skip skipped — $((SUITE_SECS / 60))m$((SUITE_SECS % 60))s total ====="
# A rate-limited blueprint fails the suite: an unverified result is not a pass.
[ "$fail" -eq 0 ] && [ "$ratelimited" -eq 0 ]

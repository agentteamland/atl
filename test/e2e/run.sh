#!/usr/bin/env bash
#
# Host runner for the atl e2e harness. Builds the image once, then runs each
# selected blueprint in a FRESH container (docker run --rm — kill + recreate per
# scenario, so blueprints never share state).
#
#   test/e2e/run.sh                       # every blueprint (auth-gated; missing-auth ones skip)
#   test/e2e/run.sh install publish-own   # named blueprints only
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
pass=0; fail=0; skip=0
timings=()
for name in $names; do
  bp="$BPDIR/$name.sh"
  if [ ! -f "$bp" ]; then echo "!! no such blueprint: $name" >&2; exit 2; fi
  needs="$(sed -n 's/^# needs: //p' "$bp" | head -1)"
  needs="${needs:-none}"

  # A blueprint may need gh, a Claude token, or (github-delivery-loop) BOTH.
  case "$needs" in gh|gh+token)
    [ -z "$GH_TOKEN_VAL" ] && { echo ">> skip $name (needs gh — run 'gh auth login')"; skip=$((skip + 1)); continue; } ;;
  esac
  case "$needs" in token|gh+token)
    { [ -z "$CLAUDE_TOK" ] && [ -z "$API_KEY" ]; } && { echo ">> skip $name (needs a Claude token — CLAUDE_CODE_OAUTH_TOKEN)"; skip=$((skip + 1)); continue; } ;;
  esac

  echo ""
  echo "========= blueprint: $name (needs: $needs) ========="
  # Least-privilege: inject ONLY the secret this blueprint declares it needs, so a
  # token-tier autonomous `claude -p` blueprint never receives the GH_TOKEN (and a
  # gh blueprint never receives the Claude token). Nothing is exported into a
  # blueprint that declared `needs: none`.
  secret_env=()
  case "$needs" in gh|gh+token)    secret_env+=(-e "GH_TOKEN=$GH_TOKEN_VAL") ;; esac
  case "$needs" in token|gh+token) secret_env+=(-e "CLAUDE_CODE_OAUTH_TOKEN=$CLAUDE_TOK" -e "ANTHROPIC_API_KEY=$API_KEY") ;; esac
  bp_start=$SECONDS
  if docker run --rm \
      -e BLUEPRINT="$name" \
      -e ATL_E2E_TEAM_REF="$ATL_E2E_TEAM_REF" \
      ${secret_env[@]+"${secret_env[@]}"} \
      atl-e2e bash "/e2e/blueprints/$name.sh"; then
    bp_secs=$((SECONDS - bp_start)); verdict=PASSED; pass=$((pass + 1))
  else
    bp_secs=$((SECONDS - bp_start)); verdict=FAILED; fail=$((fail + 1))
  fi
  echo "<< $name $verdict (${bp_secs}s)"
  timings+=("$bp_secs|$name|$verdict|$needs")
done

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
echo "===== harness: $pass passed, $fail failed, $skip skipped — $((SUITE_SECS / 60))m$((SUITE_SECS % 60))s total ====="
[ "$fail" -eq 0 ]

#!/usr/bin/env bash
#
# select_test.sh — pins test/e2e/select.sh.
#
# Host-side, not a blueprint: the selector reads the git tree and the blueprint
# files, neither of which exists inside a run container. Runs in well under a
# second, so there is no reason not to run it before touching the selector.
#
#   test/e2e/select_test.sh
#
# Why it exists at all: a selector fails in the one direction nothing surfaces —
# by running LESS than it should. A missing blueprint looks exactly like a fast,
# clean run. Every case below therefore asserts the full expected set, not
# "contains", so an under-selection and an over-selection both fail.

set -uo pipefail
cd "$(dirname "$0")/../.."
SEL="test/e2e/select.sh"

pass=0; fail=0

# want is the exact expected set, space-separated and sorted.
check() {
  local name="$1" changed="$2" want="$3"
  local got
  got="$(ATL_E2E_SELECT_CHANGED="$changed" bash "$SEL" 2>/dev/null | sort | tr '\n' ' ' | sed 's/ *$//')"
  if [ "$got" = "$want" ]; then
    echo "  ok   - $name"; pass=$((pass + 1))
  else
    echo "  FAIL - $name"
    echo "         want: $want"
    echo "         got:  $got"
    fail=$((fail + 1))
  fi
}

echo "select_test"

# The change that motivated the whole mechanism: two hours of delivery blueprints
# for a diff none of them can reach.
check "a guard change runs only the guard blueprint" \
  'cli/internal/guard/guard.go' \
  'guard'

# Core rules reflect into every project, but what PROVES the reflection is
# install/update; learning-loop and profile-loop exercise the rules' behaviour.
# The delivery blueprints deliberately do not declare core/ — see their headers.
check "a core-rule change runs the reflection + loop blueprints" \
  'core/rules/communication-style.md' \
  'install learning-loop profile-loop update'

check "the engine runs the dispatch-dependent blueprints" \
  'cli/internal/dispatch/dag.go' \
  'delivery-loop github-delivery-engine github-delivery-full-chain github-delivery-loop work-dispatch'

check "team content runs the ceremony blueprints" \
  'teams/delivery-team/skills/sprint-plan/SKILL.md' \
  'delivery-loop github-delivery-autonomous-refine github-delivery-full-chain github-delivery-loop'

# The hook-binding half of install-deps-hooks: `internal/settings` is what binds
# them, and guard declares the same package because its matcher is written there.
check "a settings change runs the hook-binding blueprints" \
  'cli/internal/settings/settings.go' \
  'guard install-deps-hooks'

# Rule 2 — a blueprint is its own consumer.
check "editing a blueprint runs that blueprint" \
  'test/e2e/blueprints/gc.sh' \
  'gc'

# Rule 3 — the shared harness. This is the case the first implementation got
# wrong: it answered "nothing to run" for a change to the machinery that runs
# everything.
want_all="$(for b in test/e2e/blueprints/*.sh; do basename "$b" .sh; done | sort | tr '\n' ' ' | sed 's/ *$//')"
check "a lib.sh change runs everything" 'test/e2e/lib.sh' "$want_all"
check "a Dockerfile change runs everything" 'test/e2e/Dockerfile' "$want_all"

# An empty selection is a legitimate answer, and must be empty rather than
# accidentally everything.
check "a docs-only change runs nothing" \
  'docs/site/cli/guard.md' \
  ''

# A prefix must match as a literal path prefix, not as a regex or a substring.
check "a path that merely contains a declared prefix does not match" \
  'vendor/cli/internal/guard/x.go' \
  ''

echo
echo "select_test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]

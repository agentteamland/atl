#!/usr/bin/env bash
#
# select.sh — print the blueprints a change can actually affect.
#
# The full suite takes ~2 hours, and most of that is three GitHub delivery
# blueprints running the whole ceremony chain. A change to `internal/guard` or to
# a core rule cannot reach any of them, so paying two hours to learn that is pure
# tax — and a tax that big eventually gets skipped, which is worse than a slower
# gate.
#
#   test/e2e/select.sh                 # vs origin/main
#   test/e2e/select.sh v2.15.0         # vs a tag
#   test/e2e/run.sh --changed          # the caller
#
# Each blueprint declares what it exercises on a `# touches:` line — a
# comma-separated list of repo-relative path prefixes. This intersects those with
# what the tree actually changed, committed and uncommitted alike (a selector
# that read only commits would skip the blueprint for the edit you are about to
# test).
#
# THREE FAIL-SAFE RULES, all erring toward running too much:
#   1. A blueprint with no `# touches:` line always runs. Forgetting the line
#      costs time, never coverage — the only direction a selector may err.
#   2. Editing a blueprint runs that blueprint. It is its own consumer, and no
#      declaration would ever say so.
#   3. Touching the shared harness runs everything. Without this the selector
#      answers "nothing to run" for a change to the machinery that runs
#      everything — which is exactly what it did on its first invocation.
#
# This does NOT replace the full suite as the pre-tag release gate. The two
# answer different questions — "can this change have broken anything?" versus
# "is the tree as a whole still green?" — and the second one earns its keep: a
# full run has caught a regression that a hand-picked subset missed.

set -uo pipefail
cd "$(dirname "$0")/../.."
BPDIR="test/e2e/blueprints"

BASE="${1:-origin/main}"
git rev-parse --verify -q "$BASE" >/dev/null 2>&1 || BASE="main"

# ATL_E2E_SELECT_CHANGED overrides the change list with a newline-separated set.
# It exists so the selector is testable: an under-selecting selector fails exactly
# the way this whole mechanism is meant to prevent — silently, by running less
# than it should — and that cannot be pinned against a live working tree.
if [ -n "${ATL_E2E_SELECT_CHANGED:-}" ]; then
  changed="$(printf '%s\n' "$ATL_E2E_SELECT_CHANGED" | sed '/^$/d' | sort -u)"
else
  changed="$(
    { git diff --name-only "$BASE"...HEAD
      git diff --name-only
      git diff --cached --name-only
      git ls-files --others --exclude-standard
    } 2>/dev/null | sort -u
  )"
fi

if [ -z "$changed" ]; then
  echo "select: no changes vs $BASE — nothing to run" >&2
  exit 0
fi

all_blueprints() { for b in "$BPDIR"/*.sh; do basename "$b" .sh; done | sort; }

# Rule 3 — the shared harness.
if printf '%s\n' "$changed" |
   grep -qE '^test/e2e/(lib|run|select)\.sh$|^test/e2e/Dockerfile$|^test/e2e/(fixtures|mock-mcp)/'; then
  echo "select: shared harness changed — every blueprint" >&2
  all_blueprints
  exit 0
fi

# matches_prefix <declaration> — true when any declared prefix starts a changed path.
matches_prefix() {
  local pfx
  for pfx in ${1//,/ }; do
    [ -z "$pfx" ] && continue
    # Escape regex metacharacters so a path is compared as a literal prefix.
    if printf '%s\n' "$changed" | grep -q "^$(printf '%s' "$pfx" | sed 's/[].[^$*\\]/\\&/g')"; then
      return 0
    fi
  done
  return 1
}

selected=""
for bp in "$BPDIR"/*.sh; do
  name="$(basename "$bp" .sh)"
  decl="$(sed -n 's/^# touches: //p' "$bp" | head -1)"

  if printf '%s\n' "$changed" | grep -qx "$BPDIR/$name.sh"; then   # rule 2
    selected+="$name"$'\n'; continue
  fi
  if [ -z "$decl" ]; then                                          # rule 1
    selected+="$name"$'\n'; continue
  fi
  if matches_prefix "$decl"; then
    selected+="$name"$'\n'
  fi
done

selected="$(printf '%s' "$selected" | sed '/^$/d' | sort -u)"

if [ -z "$selected" ]; then
  # An empty selection is a decision, not a run that quietly did nothing. Say so,
  # so a caller (or a human) never reads silence as "everything passed".
  echo "select: no blueprint exercises the changed paths — nothing to run" >&2
  exit 0
fi

printf '%s\n' "$selected"

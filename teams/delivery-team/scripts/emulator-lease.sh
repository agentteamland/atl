#!/usr/bin/env bash
# emulator-lease.sh — the delivery-team's single-slot mobile-emulator lease
# (testing-surfaces.md §3 "Mobile", detail-spec #13).
#
# One booted emulator can't run N parallel mobile suites, so mobile verification
# SERIALIZES: a worker reaching its emulator gate (developer self-test step 4, or
# tester Level-2) runs its mobile command THROUGH this lease — acquire the single
# slot, run, release. Non-mobile work keeps running at the full `atl work dispatch`
# concurrency; only the emulator gate serializes. The Go orchestrator stays out of
# it (the lease is worker-coordinated durable state, not engine memory) — this is
# the same "team owns the mechanism, run by the worker" split as az-attach.sh.
#
# Portable (macOS + Linux) by an atomic `mkdir` lock, deliberately NOT `flock`:
# `flock` is util-linux and is absent on stock macOS, where the iOS simulator must
# run. A crashed lease-holder is detected by its recorded PID and the lock is
# reclaimed, so one dead worker can't wedge the whole mobile lane for the sprint.
#
#   emulator-lease.sh <command> [args...]
#
# Compose it with the preflight health-check so a run is serialized AND on a live
# device (testing-surfaces.md §3):
#   emulator-lease.sh bash -c 'emulator-preflight.sh ios && <pack mobile test cmd>'
#
# Exit: the wrapped command's exit code on a clean run; NONZERO if the single slot
# can't be acquired within the timeout — the mobile gate then "did NOT run", which
# is a BLOCK, never a silent pass (testing-surfaces.md "block, never silent-pass").
#
# Env:
#   DELIVERY_EMULATOR_LOCK   lock dir (default .delivery/emulator.lock)
#   EMULATOR_LEASE_TIMEOUT   max seconds to wait for the slot (default 1800)
set -euo pipefail

die() { echo "emulator-lease: $1" >&2; exit 2; }

[ "$#" -ge 1 ] || die "usage: emulator-lease.sh <command> [args...]"
LOCK="${DELIVERY_EMULATOR_LOCK:-.delivery/emulator.lock}"
TIMEOUT="${EMULATOR_LEASE_TIMEOUT:-1800}"
[[ "$TIMEOUT" =~ ^[0-9]+$ ]] || die "EMULATOR_LEASE_TIMEOUT must be a number of seconds, got: $TIMEOUT"

mkdir -p "$(dirname "$LOCK")"

acquire() {
  local waited=0 holder
  while :; do
    # `mkdir` is atomic on POSIX: exactly one waiter wins the create; the rest fail.
    if mkdir "$LOCK" 2>/dev/null; then
      echo "$$" >"$LOCK/pid"
      return 0
    fi
    # Held — reclaim only if the recorded holder is truly gone (a crashed worker).
    holder="$(cat "$LOCK/pid" 2>/dev/null || true)"
    if [ -n "$holder" ] && ! kill -0 "$holder" 2>/dev/null; then
      # Reclaim ATOMICALLY: rename the stale lock aside — only one waiter's `mv`
      # succeeds. A plain `rm -rf "$LOCK"` here races (two waiters both see the same
      # stale holder, and the second's rm can delete a lock the first just recreated
      # → two concurrent holders). The mv loser's rename fails and it loops to re-read.
      if mv "$LOCK" "$LOCK.stale.$$" 2>/dev/null; then
        echo "emulator-lease: reclaiming stale lock (holder pid $holder is gone)" >&2
        rm -rf "$LOCK.stale.$$"
      fi
      continue
    fi
    if [ "$waited" -ge "$TIMEOUT" ]; then
      die "single-slot lease not acquired within ${TIMEOUT}s (held by pid ${holder:-unknown}) — mobile gate did NOT run; block, never silent-pass"
    fi
    sleep 2
    waited=$((waited + 2))
  done
}

release() { rm -rf "$LOCK"; }

acquire
trap release EXIT INT TERM

# Run the mobile command while holding the single slot; propagate its EXACT exit
# code (a fail releases the slot just as promptly as a pass — see the trap).
set +e
"$@"
rc=$?
set -e
exit "$rc"

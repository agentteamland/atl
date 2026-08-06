#!/usr/bin/env bash
#
# watch.sh — report an e2e suite's problems WHILE it runs, not two hours later.
#
# The full suite takes ~2h. Without a watcher its failures are invisible until the very
# end: the harness reports once, at exit. That is the whole cost — a blueprint that goes
# red four minutes in is diagnosable four minutes in, and instead sits unread for the rest
# of the run while later blueprints burn LLM budget on a tree you already know is broken.
#
# Designed to be driven by the Monitor tool: EVERY LINE THIS SCRIPT PRINTS TO STDOUT
# BECOMES ONE NOTIFICATION. So it prints only what someone would act on — a failed
# blueprint, a stall, and the final tally — never routine progress.
#
#   test/e2e/watch.sh                       # watch test/e2e/.last-run.log
#   test/e2e/watch.sh /path/to/run.log      # watch a specific log
#   ATL_E2E_WATCH_INTERVAL=60 test/e2e/watch.sh
#
# Exit: 0 suite finished clean · 1 suite finished with failures · 2 stalled · 3 log never
# appeared. Under Monitor the exit ends the watch and its code is surfaced.

set -uo pipefail

LOG="${1:-$(cd "$(dirname "$0")" && pwd)/.last-run.log}"
INTERVAL="${ATL_E2E_WATCH_INTERVAL:-300}"

# Silence longer than this means something outside a turn is wedged. Sized off the real
# bound rather than a guess: lib.sh caps a single `claude -p` turn at CLAUDE_TURN_TIMEOUT
# (default 900s) with a 30s kill grace, and the blueprint prints its assertions between
# turns — so no legitimate quiet stretch reaches 20 minutes. Docker hanging, a wedged MCP
# grandchild holding the pipes, or a dead reaper all show up here and nowhere else: a hung
# turn produces no FAIL line AND no summary line, so without this check it is
# indistinguishable from "still working".
STALL="${ATL_E2E_WATCH_STALL:-1200}"

# Wait for the log to appear — the watcher is usually armed a beat before the suite writes.
waited=0
while [ ! -f "$LOG" ]; do
  sleep 5; waited=$((waited + 5))
  if [ "$waited" -ge 300 ]; then
    echo "e2e-watch: no log at $LOG after 5m — was the suite started?"
    exit 3
  fi
done

reported=""          # blueprint names already reported, so a failure notifies ONCE
declare -A last_size=() quiet=()

# The suite runs its blueprints in LANES — blueprints that force-reset the same
# external fixture share one and run serially, and lanes run concurrently. That
# makes the aggregate log useless for stall detection: a wedged lane produces no
# output while a busy one keeps the file growing, so "no output for N seconds" on
# the aggregate can never fire, and if it did it would name whichever blueprint
# happened to start last in ANY lane.
#
# So liveness is measured PER LANE, off the per-lane logs the runner writes
# beside the aggregate. Falls back to the aggregate when there are none (a serial
# run, or an older log), which keeps this script correct against both shapes.
lane_logs() {
  local base="${LOG%.log}"
  local found=0 f
  for f in "$base".lane-*.log; do
    [ -e "$f" ] || continue
    echo "$f"; found=1
  done
  [ "$found" -eq 1 ] || echo "$LOG"
}

# mtime is not enough on its own: some editors/filesystems touch without appending, and a
# suite that is truly wedged may still have its file touched. Growth in BYTES is the
# honest liveness signal — the same reasoning as measuring a worker's heartbeat from the
# OS rather than from something the worker writes about itself.
size_of() { wc -c < "$1" 2>/dev/null | tr -d ' ' || echo 0; }

while :; do

  # --- failures, one notification each, as they land -------------------------------
  while IFS= read -r line; do
    name=$(printf '%s' "$line" | sed -E 's/^<< ([^ ]+) FAILED.*/\1/')
    case " $reported " in *" $name "*) continue ;; esac
    reported="$reported $name"
    echo "e2e FAILED: $line"
    # The failing assertions for that blueprint only, so the notification is diagnosable
    # on its own rather than a pointer to a 40k-line log.
    awk -v n="$name" '
      $0 ~ "^========= blueprint: "n" " {inb=1}
      inb && /^  FAIL /                 {print "   " $0}
      inb && $0 ~ "^"n": "              {print "   " $0; inb=0}
    ' "$LOG" | head -25
  done < <(grep -E '^<< [^ ]+ FAILED' "$LOG" 2>/dev/null || true)

  # --- the suite finished ------------------------------------------------------------
  if grep -q '^===== harness:' "$LOG" 2>/dev/null; then
    echo "e2e finished: $(grep '^===== harness:' "$LOG" | tail -1)"
    grep -q ' 0 failed,' "$LOG" && exit 0
    exit 1
  fi

  # --- stalled, per lane ---------------------------------------------------------------
  while IFS= read -r lane; do
    lcur=$(size_of "$lane")
    if [ "${last_size[$lane]:-0}" = "$lcur" ] && [ "$lcur" != 0 ]; then
      quiet[$lane]=$(( ${quiet[$lane]:-0} + INTERVAL ))
      if [ "${quiet[$lane]}" -ge "$STALL" ]; then
        echo "e2e STALLED: lane $(basename "$lane") silent for ${quiet[$lane]}s (threshold ${STALL}s), last blueprint: $(grep -E '^========= blueprint' "$lane" | tail -1 | sed 's/.*blueprint: //;s/ (needs.*//')"
        echo "   Other lanes may still be running — the aggregate log keeps growing, which is"
        echo "   exactly why this is measured per lane. A hung turn leaves no FAIL line and no"
        echo "   summary, so this is the only tell."
        echo "   Check: docker ps, and the tail of $lane"
        exit 2
      fi
    else
      quiet[$lane]=0
    fi
    last_size[$lane]="$lcur"
  done < <(lane_logs)

  sleep "$INTERVAL"
done

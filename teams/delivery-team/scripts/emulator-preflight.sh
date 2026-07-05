#!/usr/bin/env bash
# emulator-preflight.sh — the delivery-team's mobile-emulator bootability probe
# (testing-surfaces.md §3 "Mobile", detail-spec #13).
#
# The emulator is a DECLARED infra prerequisite, not an assumed one. /sprint-start
# runs this before dispatching any mobile work-unit (fail fast — refuse the sprint
# rather than let each worker discover a dead device mid-flight), and a worker can
# re-run it under the lease as a mid-run health check (idempotent: it boots the
# device only if it isn't already up). It probes for a bootable device and boots it,
# gated on the platform's real readiness signal — iOS `simctl bootstatus`, Android
# `adb ... sys.boot_completed` — bounded by a poll loop, NEVER a fixed `sleep`
# (iOS boot is 30-90s+ and variable; a fixed sleep either wastes time or races).
#
#   emulator-preflight.sh [ios|android]     # auto-detects the available tooling when omitted
#
# Exit 0: a device is booted and responsive. NONZERO: the exact missing prerequisite
# (no simulator/AVD, Xcode license unaccepted, no GUI session, boot timeout) on stderr
# — /sprint-start then REFUSES to start and surfaces it; a mid-run failure marks the
# mobile gate blocked, never silent-passed (testing-surfaces.md "block, never silent-pass").
#
# `timeout(1)` is deliberately NOT used — it is util-linux, absent on stock macOS
# where the iOS simulator runs; the bounded wait is a portable poll loop instead.
#
# Env:
#   EMULATOR_BOOT_TIMEOUT   max seconds to wait for boot (default 180)
#   IOS_SIMULATOR           simctl device name or udid (default: first available)
#   ANDROID_AVD             AVD name (default: first `emulator -list-avds`)
set -euo pipefail

die() { echo "emulator-preflight: $1" >&2; exit 2; }

PLATFORM="${1:-auto}"
BOOT_TIMEOUT="${EMULATOR_BOOT_TIMEOUT:-180}"
[[ "$BOOT_TIMEOUT" =~ ^[0-9]+$ ]] || die "EMULATOR_BOOT_TIMEOUT must be a number of seconds, got: $BOOT_TIMEOUT"

# wait_bounded <max-seconds> <cmd...> — run cmd, return its status if it finishes in
# time, 124 (timed out) if it exceeds max. Portable: polls the process, no `timeout(1)`,
# no fixed post-boot sleep — the readiness command itself is the signal.
wait_bounded() {
  local max="$1"; shift
  "$@" &
  local pid=$! waited=0
  while kill -0 "$pid" 2>/dev/null; do
    if [ "$waited" -ge "$max" ]; then
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
      return 124
    fi
    sleep 2
    waited=$((waited + 2))
  done
  wait "$pid"
}

preflight_ios() {
  command -v xcrun >/dev/null || die "no Xcode tooling (xcrun not found — install Xcode + \`xcode-select --install\`)"
  command -v jq    >/dev/null || die "jq is required to parse the simulator list"
  # simctl fails if the Xcode license hasn't been accepted or there is no usable Xcode.
  xcrun simctl help >/dev/null 2>&1 || die "simctl unavailable (accept the Xcode license: \`sudo xcodebuild -license accept\`)"

  local udid="${IOS_SIMULATOR:-}"
  if [ -z "$udid" ]; then
    udid="$(xcrun simctl list devices available -j 2>/dev/null \
      | jq -r 'first(.devices[][] | select(.isAvailable == true) | .udid) // empty' 2>/dev/null || true)"
  fi
  [ -n "$udid" ] || die "no available iOS simulator (create one in Xcode, or set IOS_SIMULATOR)"

  # `bootstatus -b` boots the device if needed and BLOCKS until boot completes — the
  # real readiness signal. Bounded by wait_bounded so a hung boot (e.g. no GUI session)
  # fails cleanly instead of hanging the whole sprint.
  if ! wait_bounded "$BOOT_TIMEOUT" xcrun simctl bootstatus "$udid" -b >/dev/null 2>&1; then
    die "iOS simulator $udid did not reach a booted, responsive state within ${BOOT_TIMEOUT}s (a hung boot, or no GUI session)"
  fi
  echo "emulator-preflight: iOS simulator $udid is booted and responsive"
}

preflight_android() {
  command -v emulator >/dev/null || die "no Android SDK emulator on PATH (install the Android SDK + an AVD)"
  command -v adb      >/dev/null || die "no adb on PATH (install Android platform-tools)"

  local avd="${ANDROID_AVD:-}"
  if [ -z "$avd" ]; then
    avd="$(emulator -list-avds 2>/dev/null | head -n1 || true)"
  fi
  [ -n "$avd" ] || die "no Android AVD configured (create one with \`avdmanager\`, or set ANDROID_AVD)"

  # Boot headless in the background if nothing is online yet, then wait on the real
  # signals: adb sees the device, then sys.boot_completed flips to 1.
  if ! adb devices 2>/dev/null | grep -qE '\bdevice$'; then
    ( emulator -avd "$avd" -no-window -no-audio -no-snapshot >/dev/null 2>&1 & )
  fi
  if ! wait_bounded "$BOOT_TIMEOUT" adb wait-for-device; then
    die "Android AVD $avd was not seen by adb within ${BOOT_TIMEOUT}s"
  fi
  local waited=0
  until [ "$(adb shell getprop sys.boot_completed 2>/dev/null | tr -d '\r')" = "1" ]; do
    [ "$waited" -ge "$BOOT_TIMEOUT" ] && die "Android AVD $avd did not finish booting (sys.boot_completed) within ${BOOT_TIMEOUT}s"
    sleep 2
    waited=$((waited + 2))
  done
  echo "emulator-preflight: Android AVD $avd is booted and responsive"
}

case "$PLATFORM" in
  ios)     preflight_ios ;;
  android) preflight_android ;;
  auto)
    if command -v xcrun >/dev/null && xcrun simctl help >/dev/null 2>&1; then
      preflight_ios
    elif command -v emulator >/dev/null; then
      preflight_android
    else
      die "no mobile emulator tooling found (need Xcode/simctl for iOS or the Android SDK; pass ios|android to be explicit)"
    fi ;;
  *) die "usage: emulator-preflight.sh [ios|android]" ;;
esac

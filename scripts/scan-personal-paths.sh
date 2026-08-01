#!/usr/bin/env bash
# scan-personal-paths.sh — pre-push scan for personal/local-machine identifiers.
#
# Run before pushing any commit that touches files in a public agentteamland repo.
# Catches two classes of leak:
#
#   1. OS-personal paths (universal, hardcoded in this script):
#      - macOS:   /Users/<name>/...
#      - Linux:   /home/<name>/...
#      - Windows: C:\Users\<name>\...
#
#   2. User-specific patterns (from ~/.claude/scan-personal-strings.conf):
#      - Personal project names (the maintainer's own private codenames)
#      - Personal hostnames, email-prefix usernames, scratch-project codenames
#      - Anything else the maintainer wants kept out of public commits
#
# The user config file is intentionally NOT checked into any repo — it lives in
# the maintainer's home, expressing personal-string knowledge that mustn't leak.
# The script in this public repo only carries universal OS-path patterns.
#
# Usage:
#   ./scripts/scan-personal-paths.sh                  # scan staged changes (default — fast, forward-flow only)
#   ./scripts/scan-personal-paths.sh --diff <ref>     # scan diff vs a ref (e.g., origin/main)
#   ./scripts/scan-personal-paths.sh --all            # scan entire working tree (slow — historical sweep)
#
# IMPORTANT — diff-mode blind spot:
#
#   Both --staged (default) and --diff scan ONLY added lines in the diff.
#   Personal info already on `main` and untouched by your current change
#   passes through silently. The check fires only when YOUR edit's added
#   lines contain the pattern.
#
#   Concrete past incident (2026-04-26): a Turkish-content cleanup PR was
#   accepted by --staged because the personal-name references on lines 39
#   and 68-78 of an older file were not in the diff. They had been on
#   `main` for weeks. Only --all (or a one-off `--diff <much-older-ref>`)
#   would have surfaced them.
#
#   When auditing a repo for accumulated leaks (rather than just gating
#   the next push), use --all explicitly. The default scan is for
#   FORWARD-FLOW protection, not historical sweep.
#
# Discovered via 2026-04-25 session: a workspace skill leaked the maintainer's
# absolute home path (/Users/<name>/projects/...) into a public repo PR. CI didn't
# catch it; the user did, by reading the diff. This script automates that catch.

set -euo pipefail

CONFIG="$HOME/.claude/scan-personal-strings.conf"
MODE="staged"
DIFF_REF=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --staged) MODE="staged"; shift ;;
    --diff)   MODE="diff"; DIFF_REF="${2:-}"; shift 2 ;;
    --all)    MODE="all"; shift ;;
    --self-test) MODE="self-test"; shift ;;
    -h|--help)
      sed -n '2,46p' "$0"
      exit 0
      ;;
    *) echo "Unknown arg: $1 (try --help)" >&2; exit 2 ;;
  esac
done

# File globs to scan. Skip binaries and anything explicitly excluded.
INCLUDE_GLOBS=(
  '*.md' '*.txt' '*.sh' '*.bash' '*.zsh' '*.fish'
  '*.json' '*.yaml' '*.yml' '*.toml' '*.ini' '*.conf'
  '*.tmpl' '*.template'
  '*.go' '*.py' '*.rb' '*.js' '*.ts' '*.tsx' '*.jsx'
  '*.html' '*.css' '*.dart' '*.kt' '*.swift' '*.rs'
  # This file exempts ITSELF, and every repo's thin delegate of the same name. A matcher's
  # fixtures have to contain strings that MATCH — `--self-test` cannot pin "a real home
  # path is a leak" without holding a real-looking home path — so the scanner would
  # otherwise flag its own test data forever, and the standard escape from that is to
  # weaken the fixtures until they stop matching, which quietly removes the test. Narrow
  # by construction: one filename, and nothing in it is content this repo publishes.
  ':(exclude,glob)**/scan-personal-paths.sh'
  ':(exclude)scan-personal-paths.sh'
)

# Built-in OS-path patterns. Universal — safe to live in this public file.
# Each is an extended-regex (ERE).
declare -a BUILTIN_PATTERNS=(
  '/Users/[A-Za-z][A-Za-z0-9_.-]+/'                  # macOS
  '/home/[a-z_][a-z0-9_-]+/'                          # Linux
  # Windows. TWO backslashes in this single-quoted string, not four: single quotes are
  # literal, so `\\` reaches grep -E as an escaped backslash and matches ONE. The
  # original four made the ERE require a DOUBLED backslash, which occurs only in
  # already-escaped text (JSON, a shell literal) — so this pattern matched no real
  # Windows path and the Windows third of the guard never fired at all. Fixed here;
  # `--self-test` now pins it.
  'C:\\Users\\[A-Za-z][A-Za-z0-9_.-]+\\'
)

# Read user-specific patterns from the external config (if it exists).
USER_PATTERNS=()
if [[ -f "$CONFIG" ]]; then
  while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip blank lines and comments
    [[ -z "${line// }" ]] && continue
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    USER_PATTERNS+=("$line")
  done < "$CONFIG"
fi

# Get the diff or content to scan.
case "$MODE" in
  staged)
    DIFF=$(git diff --cached --unified=0 -- "${INCLUDE_GLOBS[@]}" 2>/dev/null || true)
    ;;
  diff)
    if [[ -z "$DIFF_REF" ]]; then
      echo "✗ --diff requires a ref (e.g., --diff origin/main)" >&2
      exit 2
    fi
    DIFF=$(git diff --unified=0 "$DIFF_REF" -- "${INCLUDE_GLOBS[@]}" 2>/dev/null || true)
    ;;
  self-test)
    # No repo needed — the self-test block below swaps in its own fixture per case.
    DIFF="(self-test)"
    ;;
  all)
    # Full working-tree scan: list tracked files matching globs, concatenate.
    DIFF=""
    while IFS= read -r f; do
      [[ -f "$f" ]] || continue
      DIFF+="$(printf '+++ b/%s\n' "$f")"$'\n'
      DIFF+="$(sed 's/^/+/' "$f")"$'\n' || true
    done < <(git ls-files -- "${INCLUDE_GLOBS[@]}" 2>/dev/null || true)
    ;;
esac

if [[ -z "$DIFF" ]]; then
  echo "ℹ Nothing to scan ($MODE mode)."
  exit 0
fi

violations=0
report=""

# Home-directory names that are never a person: container and CI conventions. A real
# leak is a real human's account name, so these can be dropped without weakening the
# check — and they have to be, or the gate is unusable in a repo that documents its own
# test containers. Measured on agentteamland/workspace: 17 hits, all of them
# `/home/testuser/` and `/home/linuxbrew/` from container fixtures and the wiki pages
# describing them. A gate that fails 17 times on correct content gets switched off in a
# week, which is the failure mode `denylist-fp-safety-does-not-transfer` describes.
#
# Deliberately NOT here: placeholder names from documentation examples (`foo`, `bar`).
# Allowlisting those would let a real account of that name through. Write examples with
# the angle-bracket form this file's own header uses — `/Users/<name>/...` — which no
# pattern matches.
declare -a NONPERSONAL_HOMES=(testuser linuxbrew runner node vscode ubuntu)

# True when the line still holds a home path belonging to somebody real. Judged per
# OCCURRENCE, never per line: `/home/testuser/x and /home/realname/y` on one line must
# still report, so a line is dropped only when EVERY match on it is non-personal.
line_has_personal_home() {
  local line="$1" pat="$2" occ name a allow
  while IFS= read -r occ; do
    [[ -z "$occ" ]] && continue
    name="${occ%[/\\]}"; name="${name##*[/\\]}"
    allow=0
    for a in "${NONPERSONAL_HOMES[@]}"; do [[ "$name" == "$a" ]] && { allow=1; break; }; done
    [[ "$allow" -eq 0 ]] && return 0
  done < <(printf '%s' "$line" | grep -oE -- "$pat" 2>/dev/null || true)
  return 1
}

# Helper: scan DIFF against a single ERE pattern, collect added-line matches.
# Added lines start with '+' (but not '+++ b/' file-header).
scan_pattern() {
  local label="$1"
  local pattern="$2"
  local mode="$3"  # 'regex' or 'fixed'
  local allowlist="${4:-no}"  # 'yes' only for the built-in OS-path patterns
  local matches
  if [[ "$mode" == "fixed" ]]; then
    # Fixed-string match on added lines only (exclude '+++ b/' file headers)
    matches=$(printf '%s\n' "$DIFF" | grep -n -F -- "$pattern" 2>/dev/null | grep -E '^[0-9]+:\+' | grep -vE '^[0-9]+:\+\+\+ ' || true)
  else
    matches=$(printf '%s\n' "$DIFF" | grep -nE -- "$pattern" 2>/dev/null | grep -E '^[0-9]+:\+' | grep -vE '^[0-9]+:\+\+\+ ' || true)
  fi
  # The allowlist applies ONLY to the built-in OS-path patterns. A user pattern is a
  # personal string the maintainer named on purpose; nothing filters those.
  if [[ "$allowlist" == "yes" && -n "$matches" ]]; then
    local kept=""
    while IFS= read -r line; do
      [[ -z "$line" ]] && continue
      line_has_personal_home "$line" "$pattern" && kept+="$line"$'\n'
    done <<< "$matches"
    matches="${kept%$'\n'}"
  fi
  if [[ "${SELFTEST_CAPTURE:-0}" == "1" ]]; then SELFTEST_OUT+="$matches"$'\n'; return 0; fi
  if [[ -n "$matches" ]]; then
    report+=$'\n'"  ✗ $label"$'\n'
    while IFS= read -r line; do
      report+="      $line"$'\n'
    done <<< "$matches"
    violations=$((violations + 1))
  fi
}

# --self-test: pin the built-in patterns and the allowlist against fixtures.
#
# This exists because the Windows pattern was WRONG from the day it was written and
# nobody noticed: it required a doubled backslash, so it matched no real Windows path
# and that third of the guard never fired. A scanner reports "clean" identically whether
# it looked and found nothing or never looked at all, so the failure is invisible by
# construction — exactly the case `untested-guard-is-as-fragile-as-what-it-guards`
# describes. Run it after touching BUILTIN_PATTERNS or NONPERSONAL_HOMES.
if [[ "$MODE" == "self-test" ]]; then
  st_pass=0; st_fail=0
  # want=hit → the line must be reported · want=miss → it must not be
  st() { # <want> <description> <line>
    local want="$1" desc="$2" line="$3" got=miss
    SELFTEST_CAPTURE=1 SELFTEST_OUT=""
    local d="+++ b/f.md"$'\n'"+$line"
    local saved="$DIFF"; DIFF="$d"; SELFTEST_OUT=""
    for pat in "${BUILTIN_PATTERNS[@]}"; do scan_pattern "x" "$pat" "regex" "yes"; done
    DIFF="$saved"; SELFTEST_CAPTURE=0
    [[ -n "${SELFTEST_OUT//[$'\n' ]/}" ]] && got=hit
    if [[ "$got" == "$want" ]]; then st_pass=$((st_pass+1)); echo "  ok   - $desc"
    else st_fail=$((st_fail+1)); echo "  FAIL - $desc (wanted $want, got $got)"; fi
  }
  echo "scan-personal-paths --self-test"
  st hit  "macOS home is a leak"            '/Users/somebody/projects/x'
  st hit  "linux home is a leak"            '/home/somebody/projects/x'
  st hit  "windows home is a leak"          'C:\Users\Someone\Desktop'
  st miss "macOS <name> placeholder exempt" '/Users/<name>/projects/...'
  st miss "linux <name> placeholder exempt" '/home/<name>/projects/...'
  st miss "container home allowlisted"      '/home/testuser/proj and /home/linuxbrew/bin'
  st miss "CI home allowlisted"             'C:\Users\runner\work'
  st hit  "MIXED line still reports"        '/home/testuser/x plus /home/realname/y'
  st hit  "allowlist is exact, not prefix"  '/home/testuser2/x'
  echo "self-test: $st_pass passed, $st_fail failed"
  [[ "$st_fail" -eq 0 ]]; exit $?
fi

# Run built-in OS-path patterns (allowlist on — see NONPERSONAL_HOMES)
for pat in "${BUILTIN_PATTERNS[@]}"; do
  scan_pattern "OS personal path: $pat" "$pat" "regex" "yes"
done

# Run user-defined patterns. Each line is a fixed string unless it starts with "regex:".
# Guard against empty array under set -u.
if [[ ${#USER_PATTERNS[@]} -gt 0 ]]; then
  for entry in "${USER_PATTERNS[@]}"; do
    if [[ "$entry" == regex:* ]]; then
      pat="${entry#regex:}"
      scan_pattern "User pattern (regex): $pat" "$pat" "regex"
    else
      scan_pattern "User pattern: $entry" "$entry" "fixed"
    fi
  done
fi

if [[ $violations -gt 0 ]]; then
  echo "✗ scan-personal-paths.sh — $violations violation(s) in $MODE:"
  printf '%s' "$report"
  echo ""
  echo "If a hit is a false positive (e.g., legitimate documentation example),"
  echo "either reword the example or skip this scan with explicit justification."
  echo "Refusing to push by exit code 1."
  exit 1
fi

if [[ ${#USER_PATTERNS[@]} -eq 0 && ! -f "$CONFIG" ]]; then
  echo "✓ scan-personal-paths.sh — no OS-path leaks in $MODE."
  echo "  ℹ Add user-specific patterns to $CONFIG (one per line) for personal"
  echo "    project names, hostnames, etc. Lines starting with 'regex:' are"
  echo "    treated as ERE. Lines starting with '#' are comments."
else
  echo "✓ scan-personal-paths.sh — clean ($MODE; checked ${#BUILTIN_PATTERNS[@]} built-in + ${#USER_PATTERNS[@]} user pattern(s))."
fi

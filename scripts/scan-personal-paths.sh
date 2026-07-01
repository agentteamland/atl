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
)

# Built-in OS-path patterns. Universal — safe to live in this public file.
# Each is an extended-regex (ERE).
declare -a BUILTIN_PATTERNS=(
  '/Users/[A-Za-z][A-Za-z0-9_.-]+/'                  # macOS
  '/home/[a-z_][a-z0-9_-]+/'                          # Linux
  'C:\\\\Users\\\\[A-Za-z][A-Za-z0-9_.-]+\\\\'        # Windows (escaped backslashes)
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

# Helper: scan DIFF against a single ERE pattern, collect added-line matches.
# Added lines start with '+' (but not '+++ b/' file-header).
scan_pattern() {
  local label="$1"
  local pattern="$2"
  local mode="$3"  # 'regex' or 'fixed'
  local matches
  if [[ "$mode" == "fixed" ]]; then
    # Fixed-string match on added lines only (exclude '+++ b/' file headers)
    matches=$(printf '%s\n' "$DIFF" | grep -n -F -- "$pattern" 2>/dev/null | grep -E '^[0-9]+:\+' | grep -vE '^[0-9]+:\+\+\+ ' || true)
  else
    matches=$(printf '%s\n' "$DIFF" | grep -nE -- "$pattern" 2>/dev/null | grep -E '^[0-9]+:\+' | grep -vE '^[0-9]+:\+\+\+ ' || true)
  fi
  if [[ -n "$matches" ]]; then
    report+=$'\n'"  ✗ $label"$'\n'
    while IFS= read -r line; do
      report+="      $line"$'\n'
    done <<< "$matches"
    violations=$((violations + 1))
  fi
}

# Run built-in OS-path patterns
for pat in "${BUILTIN_PATTERNS[@]}"; do
  scan_pattern "OS personal path: $pat" "$pat" "regex"
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

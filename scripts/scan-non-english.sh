#!/usr/bin/env bash
# scan-non-english.sh — fail if committed files contain non-English (Turkish) text.
#
# The rule: every committed artifact in a public AgentTeamLand repo — Markdown,
# code, comments — is English-only. The SOLE exception is explicit localization
# mirrors under a `/tr/` path (the Turkish docs mirror), which are excluded here.
#
# It flags Turkish-specific letters (ç ğ ı İ ö ş ü …) — characters absent from
# English, so a hit is a strong signal of non-English prose. Proper-name or
# example leakage (e.g. "Yılmaz") is caught too and should be reworded.
#
# Usage:
#   ./scripts/scan-non-english.sh            # scan the whole working tree (audit)
#   ./scripts/scan-non-english.sh --staged   # scan staged changes only (pre-push gate)
#
# Exit 0 = clean, 1 = violations found, 2 = bad usage.

set -euo pipefail

MODE="all"
case "${1:-}" in
  ""|--all) MODE="all" ;;
  --staged) MODE="staged" ;;
  -h|--help) sed -n '2,18p' "$0"; exit 0 ;;
  *) echo "Unknown arg: $1 (try --help)" >&2; exit 2 ;;
esac

# Turkish-specific letters (absent from English).
PATTERN='ç|ğ|ı|İ|ö|ş|ü|â|î|û|Ç|Ğ|Ö|Ş|Ü|Â|Î|Û'

# File types to scan.
GLOBS=(':(glob)**/*.md' ':(glob)**/*.txt' ':(glob)**/*.sh' ':(glob)**/*.go'
       ':(glob)**/*.ts' ':(glob)**/*.tsx' ':(glob)**/*.js' ':(glob)**/*.jsx'
       ':(glob)**/*.py' ':(glob)**/*.json' ':(glob)**/*.yaml' ':(glob)**/*.yml')

# Exclude localization mirrors, build output, and the i18n VitePress config (its
# `locales.tr` block holds the Turkish UI strings that render the /tr/ mirror).
exclude_path() {
  case "$1" in
    # This script necessarily carries the Turkish-letter pattern itself.
    */scan-non-english.sh) return 0 ;;
    */tr/*|*/node_modules/*|*/dist/*|*/.vitepress/dist/*|*/.vitepress/cache/*|*/.vitepress/config.*) return 0 ;;
    *) return 1 ;;
  esac
}

if [ "$MODE" = "staged" ]; then
  files=$(git diff --cached --name-only --diff-filter=ACM -- "${GLOBS[@]}" 2>/dev/null || true)
else
  files=$(git ls-files -- "${GLOBS[@]}" 2>/dev/null || true)
fi

violations=0
report=""
while IFS= read -r f; do
  [ -z "$f" ] && continue
  [ -f "$f" ] || continue
  exclude_path "$f" && continue
  hits=$(grep -nE "$PATTERN" "$f" 2>/dev/null || true)
  if [ -n "$hits" ]; then
    report+=$'\n'"  ✗ $f"$'\n'
    while IFS= read -r line; do report+="      $line"$'\n'; done <<< "$hits"
    violations=$((violations + 1))
  fi
done <<< "$files"

if [ "$violations" -gt 0 ]; then
  echo "✗ scan-non-english.sh — Turkish characters found in $violations file(s):"
  printf '%s' "$report"
  echo ""
  echo "Every committed artifact must be English-only (except /tr/ localization mirrors)."
  echo "Translate the content, or — for a genuine localized mirror — place it under /tr/."
  exit 1
fi

echo "✓ scan-non-english.sh — clean ($MODE): no non-English text outside /tr/ mirrors."

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
# Each blueprint declares its need on a `# needs: none|gh|token` line; a blueprint
# whose auth is absent is skipped (so this same script is CI-safe and local-full).

set -euo pipefail
cd "$(dirname "$0")/../.."   # atl repo root (build context)
BPDIR="test/e2e/blueprints"

echo ">> building atl-e2e image"
docker build -f test/e2e/Dockerfile -t atl-e2e . >/dev/null
echo ">> image ready"

if [ "$#" -gt 0 ]; then
  names="$*"
else
  names="$(ls "$BPDIR"/*.sh | xargs -n1 basename | sed 's/\.sh$//' | sort)"
fi

# Resolve auth once (host side).
GH_TOKEN_VAL="$(gh auth token 2>/dev/null || true)"
CLAUDE_TOK="${CLAUDE_CODE_OAUTH_TOKEN:-}"
API_KEY="${ANTHROPIC_API_KEY:-}"

pass=0; fail=0; skip=0
for name in $names; do
  bp="$BPDIR/$name.sh"
  if [ ! -f "$bp" ]; then echo "!! no such blueprint: $name" >&2; exit 2; fi
  needs="$(sed -n 's/^# needs: //p' "$bp" | head -1)"
  needs="${needs:-none}"

  if [ "$needs" = "gh" ] && [ -z "$GH_TOKEN_VAL" ]; then
    echo ">> skip $name (needs gh — run 'gh auth login')"; skip=$((skip + 1)); continue
  fi
  if [ "$needs" = "token" ] && [ -z "$CLAUDE_TOK" ] && [ -z "$API_KEY" ]; then
    echo ">> skip $name (needs a Claude token — CLAUDE_CODE_OAUTH_TOKEN)"; skip=$((skip + 1)); continue
  fi

  echo ""
  echo "========= blueprint: $name (needs: $needs) ========="
  if docker run --rm \
      -e BLUEPRINT="$name" \
      -e GH_TOKEN="$GH_TOKEN_VAL" \
      -e CLAUDE_CODE_OAUTH_TOKEN="$CLAUDE_TOK" \
      -e ANTHROPIC_API_KEY="$API_KEY" \
      atl-e2e bash "/e2e/blueprints/$name.sh"; then
    echo "<< $name PASSED"; pass=$((pass + 1))
  else
    echo "<< $name FAILED"; fail=$((fail + 1))
  fi
done

echo ""
echo "===== harness: $pass passed, $fail failed, $skip skipped ====="
[ "$fail" -eq 0 ]

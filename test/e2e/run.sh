#!/usr/bin/env bash
#
# Host runner for the atl e2e harness. Builds the image (binary compiled inside
# the container — no host Go needed) and runs a layer.
#
#   test/e2e/run.sh            # Layer A (deterministic, auth-free)
#   test/e2e/run.sh --layer-b  # Layer B (full loop) — needs CLAUDE_CODE_OAUTH_TOKEN
#
# Run from anywhere; it resolves the repo root itself.

set -euo pipefail

cd "$(dirname "$0")/../.."   # atl repo root (build context)

echo ">> building atl-e2e image"
docker build -f test/e2e/Dockerfile -t atl-e2e .

if [ "${1:-}" = "--layer-b" ]; then
  if [ -z "${CLAUDE_CODE_OAUTH_TOKEN:-}" ] && [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    echo "!! Layer B needs CLAUDE_CODE_OAUTH_TOKEN (from 'claude setup-token') or ANTHROPIC_API_KEY" >&2
    exit 2
  fi
  echo ">> running Layer B (full loop, real Claude)"
  docker run --rm \
    -e CLAUDE_CODE_OAUTH_TOKEN="${CLAUDE_CODE_OAUTH_TOKEN:-}" \
    -e ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-}" \
    atl-e2e bash /e2e/layer-b.sh
else
  echo ">> running Layer A (deterministic)"
  docker run --rm atl-e2e bash /e2e/layer-a.sh
fi

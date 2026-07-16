# syntax=docker/dockerfile:1
#
# Containerized e2e test image for atl.
# Build context is the repo root (the build stage needs cli/):
#
#   docker build -f test/e2e/Dockerfile -t atl-e2e .
#
# Two-layer harness (see the harness decision in the maintainer workspace):
#   Layer A — deterministic, no Claude  -> bash /e2e/layer-a.sh
#   Layer B — full loop, real Claude    -> needs CLAUDE_CODE_OAUTH_TOKEN (planned)

# --- Stage 1: build the atl binary in-container (host-arch-agnostic) ---
FROM golang:1.26 AS build
WORKDIR /src/cli
COPY cli/go.mod cli/go.sum ./
RUN go mod download
COPY cli/ ./
RUN CGO_ENABLED=0 go build -o /out/atl ./cmd/atl

# --- Stage 2: runtime — a brand-new user's machine ---
FROM node:22-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends git ca-certificates jq curl \
 && rm -rf /var/lib/apt/lists/* \
 && npm install -g @anthropic-ai/claude-code
# gh CLI (arch-aware) — the publish + github-delivery blueprints drive real
# fork/PR/commit/tag + Projects v2 (field-create/item-edit need a modern gh).
RUN ARCH=$(dpkg --print-architecture) \
 && curl -fsSL "https://github.com/cli/cli/releases/download/v2.86.0/gh_2.86.0_linux_${ARCH}.tar.gz" | tar xz -C /tmp \
 && mv /tmp/gh_*/bin/gh /usr/local/bin/gh \
 && rm -rf /tmp/gh_*
COPY --from=build /out/atl /usr/local/bin/atl
COPY test/e2e/ /e2e/
# a non-root user with a clean home simulates a first-time install
RUN useradd -m -s /bin/bash testuser
USER testuser
# a real user has a git identity configured; simulate it so publish's commits work
RUN git config --global user.email "e2e@atl.local" \
 && git config --global user.name "atl-e2e"
WORKDIR /home/testuser
CMD ["bash"]

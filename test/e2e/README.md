# atl end-to-end test harness

Container-based e2e tests that exercise `atl` on a simulated brand-new user's
machine. The image builds `atl` from source (multi-stage, no host Go needed) and
installs Claude Code, so the container is a faithful stand-in for a real user.

## Layer A — deterministic (no Claude session)

Drives the `atl` binary over the golden paths and asserts on files + manifests:

- install at both scopes
- **promote** a project gain to global (modified file + new child)
- promote idempotency
- **pin** keeps a gain project-only; **unpin** re-enables it
- **doctor** self-heals a deleted installed file
- **update** fans an unmodified project file out from global

Auth-free; this is the always-on regression backbone. Run from anywhere:

```bash
test/e2e/run.sh
```

Network is required once per run (install fetches a real, ref-pinned team —
`design-system-team@v0.8.1` — from the index). Each scenario then restores a
local snapshot, so the suite is isolated and fast.

## Layer B — full loop (real Claude session) · planned

Real `claude -p` (headless) sessions so hooks fire, `/drain` runs, and the full
capture → queue → drain → KB → promote → fan-out loop closes for real. Needs a
Claude Code token in the container:

```bash
claude setup-token                                    # once, where you can log in
CLAUDE_CODE_OAUTH_TOKEN=<token> test/e2e/run.sh --layer-b
```

Subscription OAuth cannot run inside a container, so `setup-token` (a 1-year
token) is the supported path; `ANTHROPIC_API_KEY` also works. `layer-b.sh` lands
in a follow-up.

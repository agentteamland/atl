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

## Layer B — full loop (real Claude session)

Real `claude -p` (headless) sessions so hooks fire, `/drain` runs, and the
capture → queue → drain → KB loop closes for real. Asserts:

- install reflects the `learning-capture` rule + the `/drain` skill and binds the hooks
- **capture** — a real session drops a `<!-- learning: … -->` marker (verified in the transcript)
- **tick** — `atl tick` enqueues the marker into the durable queue
- **drain** — the `/drain` skill folds the queue into the KB (wiki / journal / agent KB)
- **ack** — the processed item is deleted from the queue (the v1 re-report bug class stays dead)

Needs a Claude Code token in the container:

```bash
claude setup-token                                    # once, where you can log in
CLAUDE_CODE_OAUTH_TOKEN=<token> test/e2e/run.sh --layer-b
```

Subscription OAuth cannot run inside a container, so `setup-token` (a 1-year
token) is the supported path; `ANTHROPIC_API_KEY` also works. The assertions key
off queue + file **state**, never a command's "did work" message or an exact KB
filename, so the non-determinism of a real LLM turn doesn't make them flaky.

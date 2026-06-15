# atl end-to-end test harness

Container-based e2e tests. Each **blueprint** (one scenario) runs in a FRESH
container (`docker run --rm` — kill + recreate per scenario, so blueprints never
share state). The image builds `atl` from source (multi-stage, no host Go needed)
and installs Claude Code + gh, so the container is a faithful stand-in for a real
user's machine.

## Run

```bash
test/e2e/run.sh                       # every blueprint (auth-gated; missing-auth ones skip)
test/e2e/run.sh install publish-own   # named blueprints only
```

Auth is passed into the container only when present on the host:

- **gh** — `GH_TOKEN` (from your `gh auth token`) — the publish blueprints
- **Claude** — `CLAUDE_CODE_OAUTH_TOKEN` (from `claude setup-token`) or `ANTHROPIC_API_KEY` — the learning-loop blueprint

A blueprint whose auth is absent is skipped, so the same script is CI-safe (only
the auth-free core runs) and local-full (everything runs when you're authed).

## Blueprints

Each lives in `blueprints/<name>.sh`, declares its auth need on a `# needs:` line,
sources `lib.sh`, and asserts on file / manifest / queue **state** (never an exact
filename or a command's "did work" message — so the non-deterministic publish +
learning blueprints stay non-flaky).

| Blueprint | needs | What it proves |
|---|---|---|
| `install` | none | install at both scopes; assets + manifests + embedded core reflect |
| `promote` | none | a project gain lifts to global; second pass is a no-op |
| `pin` | none | a pinned file is held back from promote; unpin re-enables it |
| `doctor` | none | a deleted installed file is self-healed from the pinned source |
| `update` | none | a global change fans out to an unmodified project copy |
| `list-remove` | none | list shows the team; remove deletes its files + manifest |
| `learning-loop` | token | real `claude -p`: marker → tick → queue → /drain → KB → ack |
| `publish-propose` | gh | a gain in a team you don't own → real fork + PR (then cleanup) |
| `publish-own` | gh | a team you own → real commit + version bump + tag |

## Fixtures

`fixtures/` holds two minimal teams; two real GitHub repos mirror them so the
publish blueprints exercise actual GitHub:

- `agentteamland/atl-e2e-team` — propose-upstream upstream (not owned by the tester)
- `<your-login>/atl-e2e-owned` — own-team re-publish target (the `publish-own`
  blueprint force-resets it to the fixture baseline each run, so it's repeatable)

The blueprints inject a test-only `~/.atl/index.json` (via `write_test_index` in
`lib.sh`) so `atl install` resolves the fixtures offline — the production index is
never touched.

## CI

Not wired into CI yet (atl's CI runs Go build/vet/test only). The auth-free
blueprints are CI-ready as-is; the publish + learning-loop blueprints need
secrets (a gh token, a Claude token) — a follow-on.

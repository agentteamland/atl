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
- **both** (`gh+token`) — `github-delivery-loop` needs a `GH_TOKEN` **and** a Claude token; the token also needs the `project` scope (Projects v2) — `gh auth refresh -s project`

A blueprint whose auth is absent is skipped, so the same script is CI-safe (only
the auth-free core runs) and local-full (everything runs when you're authed).

### Run only what the change can reach

A full run is ~2 hours, and three GitHub delivery blueprints are ~70% of it. A change to
`internal/guard` or to a core rule cannot reach any of them.

```bash
test/e2e/run.sh --changed            # vs origin/main
test/e2e/run.sh --changed v2.15.0    # vs a tag
```

Each blueprint declares what it exercises on a `# touches:` line; `test/e2e/select.sh`
intersects those with the changed paths — committed **and** uncommitted, since a selector that
read only commits would skip the blueprint for the edit you are about to test.

Three fail-safe rules, all erring toward running too much: a blueprint with **no** declaration
always runs · editing a blueprint runs that blueprint · touching the shared harness (`lib.sh`,
`run.sh`, `Dockerfile`, fixtures, the mock) runs everything. Forgetting a declaration costs time,
never coverage.

Measured: a guard-only change selects **one** blueprint; a core-rule change selects the four that
prove reflection and loop behaviour. Roughly ninety seconds where the full suite is two hours.

**This does not replace the full suite as the pre-tag gate.** The two answer different questions —
*can this change have broken anything?* versus *is the tree as a whole still green?* — and the
second earns its keep, since a full run has caught a regression a hand-picked subset missed. Use
`--changed` per change; run everything before a tag.

`test/e2e/select_test.sh` pins the selector (host-side, sub-second). Run it after touching a
`# touches:` line — a selector fails in the one direction nothing surfaces, by running *less*
than it should, which looks exactly like a fast clean run.

### One retry on a transient API failure

`claude_turn` retries **once** when the turn fails with a dropped connection, an overload, a rate
limit or a 5xx. A dropped connection has cost a 29-minute blueprint outright: the ceremony did its
work — the Epic, the Feature and the analysis comment all landed, and every assertion after it
passed — but the envelope never arrived, so the turn returned non-zero.

Retrying is safe because of a property the ceremonies already guarantee: every create is
check-first by a stable `atl-key`, so a re-run **converges** rather than duplicating. Do not extend
the retry to a step without that guarantee. Only once, and only on those shapes — a real failure
must stay a failure.

### Always watch a full run

**A full suite takes ~2 hours, and `run.sh` reports only once — at the end.** So a
blueprint that goes red four minutes in sits unread for the rest of the run, while every
later LLM blueprint burns budget on a tree already known to be broken. Arm the watcher in
the same breath as the run:

```bash
test/e2e/watch.sh <run-log>     # under the Monitor tool, persistent
```

It polls every 5 minutes (`ATL_E2E_WATCH_INTERVAL`) and prints only what someone would act
on — one line per newly failed blueprint **with that blueprint's failing assertions**, so
the notification is diagnosable on its own rather than a pointer into a 40k-line log. Then
the final tally, and it exits.

It also reports a **stall**, which nothing else can: a hung `claude -p` turn produces no
`FAIL` line *and* no summary line, so from the outside it is indistinguishable from "still
working" — see [`e2e-hung-claude-turn-no-summary`](https://github.com/agentteamland/workspace/blob/main/.atl/wiki/e2e-hung-claude-turn-no-summary.md).
The threshold (`ATL_E2E_WATCH_STALL`, default 1200s) is sized off the real bound rather
than guessed: `lib.sh` caps one turn at `CLAUDE_TURN_TIMEOUT` (900s) plus a 30s kill grace,
and assertions print between turns, so no legitimate quiet stretch reaches 20 minutes.

`run.sh` mirrors every run to `test/e2e/.last-run.log`, so `watch.sh` needs no argument at
all — and guessing one (`ls -t` over a task directory, say) is how you end up watching the
wrong file and trusting a green that belongs to something else.

Exit codes: `0` clean · `1` finished with failures · `2` stalled · `3` the log never
appeared. **A non-zero exit here describes the SUITE, not the watcher** — under the Monitor
tool that surfaces as "script failed", which reads like the watcher broke. It did not; the
event line immediately above it carries the real verdict.

## Blueprints

Each lives in `blueprints/<name>.sh`, declares its auth need on a `# needs:` line,
sources `lib.sh`, and asserts on file / manifest / queue **state** (never an exact
filename or a command's "did work" message — so the non-deterministic publish +
learning blueprints stay non-flaky).

| Blueprint | needs | What it proves |
|---|---|---|
| `init` | none | `atl init` scaffolds a per-tier CLAUDE.md, only-if-absent (never clobbers), flags mutually exclusive |
| `install` | none | install at both scopes; assets + manifests + embedded core reflect; project CLAUDE.md scaffolded |
| `install-deps-hooks` | none | what install does BESIDES copying the team you named: it follows the `dependencies` edge (personal-advisory-team pulls profile-team in, reported as a `(dependency)`), binds all five automation hooks by event + exact command incl. the guard's `Bash\|Edit\|Write` matcher (D-3), and a RE-install converges — a locally modified reflected file survives, hooks and manifests do not duplicate, and the user's own hook + non-hook settings keys are preserved through both passes. Every assertion is a delta against a baseline taken first, because each property's null outcome otherwise reads as a pass |
| `promote` | none | a project gain lifts to global; second pass is a no-op |
| `pin` | none | a pinned file is held back from promote; unpin re-enables it |
| `doctor` | none | a deleted installed file is self-healed from the pinned source |
| `update` | none | a global change fans out to an unmodified project copy |
| `list-remove` | none | list shows the team; remove deletes its files + manifest |
| `search` | none | catalog is searchable by keyword + name, browsable with no query, miss reports cleanly |
| `guard` | none | PreToolUse hook: irreversible Bash op denied; first-edit nudge then silent; new file + malformed input pass |
| `learning-loop` | token | real `claude -p`: marker → tick → queue → /drain → KB → ack |
| `profile-backup-restore` | gh+token | the two irreversible profile skills: `/profile-backup`'s visibility guard refuses on public / no-remote / non-GitHub-remote / gh-unavailable and writes NOTHING, snapshots a private repo byte-identically (no gitlink, no `.git`, commit scoped to `profile-backup/`, mirror semantics), and `/profile-restore`'s newer-guard flags a global file modified after the snapshot commit (surviving a fresh clone, so it is commit-time based not mtime based), stays dry-run without `--apply`, and preserves global-only memory on apply. Mostly deterministic — it runs the skills' own fenced bash bodies extracted from the installed SKILL.md — plus two real `claude -p` turns on the paths where improvising is unrecoverable |
| `publish-propose` | gh | a gain in a team you don't own → real fork + PR (then cleanup) |
| `publish-own` | gh | a team you own → real commit + version bump + tag |
| `github-delivery-loop` | gh+token | the GitHub-backend Layer-B / T-point: real `claude -p` ceremonies + a developer→tech-lead micro-loop on a real fixture repo + Project — Epic/Feature issues + `[Technical Analysis]`, `plan.json`, a PR merged to `dev` + its issue closed (§10), and the commit-bound promotion gate (#16): `/sprint-review` HOLDs with no approval record on the promotion PR **and** with one naming a superseded commit, and merges dev→release only for the exact commit the record names |
| `github-delivery-engine` | gh+token | the follow-on to `github-delivery-loop`: the Go engine (`atl work dispatch`) driving REAL developer→tester→tech-lead `claude -p` workers on GitHub — seeds one buildable PBI + `plan.json`, then proves the (backend-neutral) worker prompts reach `gh` and land a real merge to `dev` (issue closed, worktree reclaimed). The github twin of the real-Azure engine run; the deterministic engine loop is covered by `work-dispatch` |
| `github-delivery-full-chain` | gh+token | the SEAM joining `github-delivery-loop` (ceremonies) + `github-delivery-engine` (dispatch): the FULL chain `/kickoff → /refine → /sprint-plan → /sprint-start → atl work dispatch --cap 2` with real `claude -p`, over a Feature `/refine` decomposes into 3 dependency-linked PBIs — proves a MULTI-NODE `plan.json` driving the engine to ≥2 dependency-ordered merges to `dev` with genuine cap-2 concurrency (the first real-worker multi-node GitHub run) |
| `github-sprint-carryover` | gh+token | MULTI-UNIT CARRYOVER, the path the flow-admission fix (#333) changed and nothing exercised: a sprint ends with two units incomplete, the second depending on the first, and the re-plan must admit BOTH to the next sprint and carry the edge into `plan.json`. Seeds the prior sprint (two open PBIs at `sprint:1` + `carryover`, the dependent's `## Depends On` naming the foundation, and the `docs/sprints/sprint-1-review.md` page that makes sprint:1 *reviewed*) and runs only `/sprint-plan` + `/sprint-start`. The regression it catches — admitting the foundation and leaving the dependent behind — is silent everywhere else: the ceremony reports a clean sprint and the dependent just sits at the old ordinal. **Flow only**: scrum's carrier is a Projects v2 Iteration field `gh` cannot create, so under scrum "was this unit admitted?" has no durable surface to read on this backend |

## Fixtures

`fixtures/` holds two minimal teams; two real GitHub repos mirror them so the
publish blueprints exercise actual GitHub:

- `agentteamland/atl-e2e-team` — propose-upstream upstream (not owned by the tester)
- `<your-login>/atl-e2e-owned` — own-team re-publish target (the `publish-own`
  blueprint force-resets it to the fixture baseline each run, so it's repeatable)
- `agentteamland/atl-e2e-delivery`, `-2`, `-3` — **three** GitHub-backend delivery
  fixtures, in the org (ATL's own infra, alongside `atl-e2e-team`; override the owner
  for a fork with `ATL_E2E_DELIVERY_OWNER`). Create each once, private, with merge
  commits allowed and at least one commit so `git clone` succeeds — the content does
  not matter, because every run wipes it and copies `fixtures/delivery-repo/` in.

  Each GitHub delivery blueprint force-resets its fixture to that baseline
  (main/dev/release, no stale issues/PRs) and deletes and recreates a Project of the
  same title, so the loop is repeatable. **That is also why there are three.** Two
  blueprints on one fixture cannot overlap: the second one's reset destroys the
  first's board mid-run, and the loser fails an assertion that says nothing about
  concurrency. Separate fixtures mean separate lanes, which is what takes the suite from ~132 min
  to ~58. **Two** delivery lanes, not three, and the limit is not the fixtures: a full
  suite costs roughly one hourly GitHub GraphQL quota (the ceremony turns, not the
  fixture resets — those are 15 points each), so compressing it far below an hour
  exhausts the budget and the failures land on unrelated blueprints. `atl-e2e-delivery-3`
  exists and is unused; a third lane becomes viable if that consumption ever drops. The mapping is declared per blueprint on its `# fixture:` line;
  `test/e2e/run.sh --lanes` prints the partition without running anything.

  Adding a fourth is a repo plus a `# fixture:` line — no harness change. The runner's
  token needs `repo` + `project` rights on the owner; the container ships a modern `gh`
  for Projects v2 (`field-create`/`item-edit`).

The blueprints inject a test-only `~/.atl/index.json` (via `write_test_index` in
`lib.sh`) so `atl install` resolves the fixtures offline — the production index is
never touched.

## CI

atl's CI (`.github/workflows/ci.yml`) runs Go build/vet/test only. **The e2e
harness is deliberately not wired into CI — it runs locally, on demand:**

```bash
test/e2e/run.sh        # run the full suite before shipping; everything must pass
```

Why local-only: the `learning-loop` blueprint needs a real `claude -p` turn, and
a `claude setup-token` subscription OAuth token is **rejected (HTTP 401) from
datacenter/CI IPs** — it only authenticates from a developer machine. The
pay-per-use `ANTHROPIC_API_KEY` alternative was declined (no extra billing). So
the whole suite runs on the maintainer's machine, where the subscription token
works and the gh/publish fixtures are reachable. Run it before a release; fix
anything red there.

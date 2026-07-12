# `atl doctor`

Diagnose the platform and self-heal what it safely can — the on-demand surface for the same checks ATL runs automatically every session.

## Usage

```bash
atl doctor
```

There are no flags. `atl doctor` inspects the current project (its working directory is the project key) plus the global layer, runs its checks in order, repairs what a deterministic fix can repair, and prints one line per check. It **exits non-zero when a check FAILs** (warnings never fail), so `atl doctor && …` can gate a script or CI step.

## When to use it

Most of the time you won't — these checks already run at every session start (via the [`session-start` hook](/cli/setup-hooks)) and self-heal silently, surfacing only when something is off. Reach for `atl doctor` when you want to look on purpose:

- after deleting files under `.claude/` by accident (or a fresh checkout where they never landed) — see if the assets are back,
- when learnings feel stuck — confirm the queue is draining and the loop is still ticking,
- as a quick "is the platform healthy here?" before or after a working session.

## What it checks

Each line is `STATUS  check-name — detail`, where `STATUS` is `OK`, `WARN`, or `FAIL`. A check that applied a deterministic fix during the run is tagged ` (self-healed)`.

### `asset-integrity` — missing-file restore

The install manifest is a contract: these files must exist at this scope. `doctor` compares each manifest against what's actually on disk, across both the project layer (`<project>/.claude`) and the global layer (`~/.claude`), and **restores any file the manifest lists but disk lacks** — re-fetched from its pinned source and checksum-verified.

Only *absent* files are restored. A file that's present but changed is treated as a user edit (or a learning-loop evolution) and is **never overwritten**. To remove a team for good, use [`atl remove`](/cli/remove), which drops the manifest. A restore that can't complete (e.g. the network is down) is a `WARN`, not a session-blocker.

### `queue-backlog` — is the learning queue draining?

Counts the pending items in the learning queue for this project. `OK` when the queue is empty or comfortably small; `WARN` once the backlog crosses the threshold (currently 50), which signals that a [`/drain`](/cli/learnings) pass hasn't kept up. The doctor does **not** drain the queue itself — folding a queue item into a knowledge base needs an LLM, which is a skill's job, so the doctor *signals* the backlog rather than processing it.

### `tick-freshness` — is the loop still running?

Looks at how long it's been since the maintenance pass last ran (the wall-clock last-tick time, distinct from the transcript high-water mark). `WARN` if items are queued but ticks haven't run in over 24 hours (or the queue has been written to but never ticked at all) — a sign the in-session cadence isn't firing. `OK` otherwise, reporting how long ago the last tick happened.

### `hooks-bound` — is the automation actually wired?

Automation is mandatory in v2, but a reset or hand-edited `~/.claude/settings.json` can leave ATL's hooks unbound — silently killing the whole loop (drain, doctor, and guard stop firing). This check reads the settings file, and if any of the three atl hooks (`SessionStart`, `UserPromptSubmit`, `PreToolUse`) is missing it **re-binds them** via the same idempotent install that never touches your own hooks — a `(self-healed)` repair. A settings file it can't read is a `WARN`, not a blocker.

## The CLI / Skill split

`atl doctor` only does deterministic repairs — re-fetch an absent file, retry a mechanical step. Anything that needs an LLM (processing a queued learning into the knowledge base) is out of scope by design; the doctor surfaces the count and points you at the skill. This is why a large backlog shows up as a warning here but is actually cleared by running [`/drain`](/cli/learnings).

## Examples

A healthy project:

```bash
$ atl doctor
OK    queue-backlog — queue empty
OK    tick-freshness — last tick 3m12s ago
OK    asset-integrity — all installed files present
OK    hooks-bound — all automation hooks bound

doctor: all healthy
```

A file was deleted and the doctor restored it, while the queue has fallen behind:

```bash
$ atl doctor
WARN  queue-backlog — 63 pending items — a drain skill should process them
OK    tick-freshness — last tick 8s ago
OK    asset-integrity — restored 1 missing file(s) — `atl remove <handle>/<team>` removes a team for good (self-healed)
OK    hooks-bound — all automation hooks bound

doctor: warnings above (not fatal)
```

The exit message is `doctor: all healthy`, `doctor: warnings above (not fatal)`, or `doctor: failures above`, matching the most severe line — and the exit code is non-zero only for `failures above`.

## Related

- [`atl learnings`](/cli/learnings) — inspect the queue the backlog check reports on; run `/drain` to clear it.
- [`atl setup-hooks`](/cli/setup-hooks) — wires the `session-start` hook that runs these same checks automatically.
- [`atl install`](/cli/install) / [`atl remove`](/cli/remove) — write and drop the manifests `asset-integrity` heals against.
- [CLI overview](/cli/overview)

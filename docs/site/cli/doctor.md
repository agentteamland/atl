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

Automation is mandatory in v2, but a reset or hand-edited `~/.claude/settings.json` can leave ATL's hooks unbound — silently killing the whole loop (drain, doctor, and guard stop firing). This check reads the settings file, and if any of the four atl hooks (`SessionStart`, the two `UserPromptSubmit` entries — `atl tick` and `atl retrieve` — and `PreToolUse`) is missing it **re-binds them** via the same idempotent install that never touches your own hooks — a `(self-healed)` repair. A tick throttle you customized (`atl setup-hooks --throttle=…`) is preserved, never reset to the default. A settings file it can't read is a `WARN`, not a blocker.

### `brainstorm-pins` — does the pin block still match reality?

Every active brainstorm pins itself into the scope's `CLAUDE.md` inside a `<!-- brainstorm:active -->` block, and [`/brainstorm done`](/skills/brainstorm) is supposed to remove that bullet when the brainstorm closes. That step is prose in a skill, so it gets skipped — and because the block is loaded into **every** session's context, a leftover bullet actively tells future sessions a closed decision is still open.

This check compares the pinned set against each brainstorm's frontmatter `status:` — in the project (`<project>/.atl/brain-storms` ↔ `<project>/CLAUDE.md`) and in the global layer (`~/.atl/brain-storms` ↔ `~/.claude/CLAUDE.md`) — and `WARN`s in both directions: a **closed brainstorm still pinned**, and an **active brainstorm with no pin** (the recovery the brainstorm rule describes). It reads the frontmatter only, never the body, so a closed brainstorm quoting `status: active` in its prose is not flagged; an unrecognized status (a hand-written `paused`, a file with no frontmatter) is left alone rather than reported.

A third case: if the block **opened but never closed** — a half-finished closure that dropped the end marker — the check says exactly that and stops there rather than guessing where the block ends. Guessing would turn every brainstorm link further down the file (a "settled decisions" section, say) into a bogus stale-pin report.

It **reports, never rewrites**. The doctor's self-heals only touch ATL-owned artifacts; `CLAUDE.md` is your own always-loaded instruction file, and the missing-pin direction needs a one-line summary of the topic — judgment, not a mechanical fix. A project with no `.atl/brain-storms` directory is silent.

### `credential-file` — can anyone else read the translator's token?

[Retrieval's prompt translator](/cli/retrieve) takes its credential from the environment or, when the environment can't carry it, from `~/.atl/claude-token` — a plain file holding just the token. `atl` never writes that file, so a hand-created one gets your umask, which on a default macOS shell is `0644`: a secret every account on the machine can read, and nothing else in the tree would ever notice. This check reads the file's mode and, when it is wider than owner-only, **tightens it to `0600`** — a `(self-healed)` repair. A file that is already owner-only is `OK`; a mode it cannot change is a `WARN` carrying the mode it found and the reason.

It checks the mode and nothing else. It deliberately does **not** report a missing or empty credential — the session-start notice already owns that condition, and reporting one condition through two channels is how a signal stops being read. So no file at all is `OK` ("no translator credential file (optional)"), translation being an improvement rather than a requirement, and after healing once the check goes quiet for good. The `~/.atl` directory itself is left alone on purpose — it is shared with the learning queue, which creates it `0755`, so its mode is not this check's to argue with.

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
OK    brainstorm-pins — pins agree with brainstorm frontmatter

doctor: all healthy
```

A file was deleted and the doctor restored it, while the queue has fallen behind:

```bash
$ atl doctor
WARN  queue-backlog — 63 pending items — a drain skill should process them
OK    tick-freshness — last tick 8s ago
OK    asset-integrity — restored 1 missing file(s) — `atl remove <handle>/<team>` removes a team for good (self-healed)
OK    hooks-bound — all automation hooks bound
WARN  brainstorm-pins — 1 closed brainstorm(s) still pinned in CLAUDE.md — every session is told the decision is open: docs-sync-v2.md (status: completed) — fix the `<!-- brainstorm:active -->` block (brainstorm rule)

doctor: warnings above (not fatal)
```

The exit message is `doctor: all healthy`, `doctor: warnings above (not fatal)`, or `doctor: failures above`, matching the most severe line — and the exit code is non-zero only for `failures above`.

## Related

- [`atl learnings`](/cli/learnings) — inspect the queue the backlog check reports on; run `/drain` to clear it.
- [`atl setup-hooks`](/cli/setup-hooks) — wires the `session-start` hook that runs these same checks automatically.
- [`atl install`](/cli/install) / [`atl remove`](/cli/remove) — write and drop the manifests `asset-integrity` heals against.
- [CLI overview](/cli/overview)

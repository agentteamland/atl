# `atl tick`

Run one in-session maintenance pass — the work the three-speed cadence fires every few minutes while you're in a session: a cheap fan-out, plus a throttled drain + doctor self-check.

You almost never run this by hand. [`atl setup-hooks`](/cli/setup-hooks) wires it to the `UserPromptSubmit` hook as `atl tick --throttle=10m`, so it rides your prompts automatically. The manual surface exists for setup, debugging, and forcing a pass.

## When to use it

- **Normally: never directly.** It runs on every prompt via the hook, throttled so the heavy part only fires every ~10 minutes.
- **To force a pass now** — e.g. you want pending capture markers moved into the queue before running [`/drain`](/skills/drain) — run `atl tick` with no `--throttle`.
- **To drain one file** (debugging the marker parser) — `atl tick --file <path>`.

## Usage

```bash
atl tick                        # force a full pass now (no throttle)
atl tick --throttle=10m         # skip the drain+doctor pass if the last tick was <10m ago
atl tick --file <path>          # drain a single file instead of discovering transcripts (debug)
```

`atl tick` operates on the **current project** — the directory you ran it in.

## What a pass does

In order:

1. **Fan-out (every call, ~free).** Pulls any newly-changed files down from the user-global layer into this project. It's guarded by a global generation counter (`~/.atl/generation`): if the global layer hasn't changed since this project last fanned out, it's a single small file read and does nothing. This step runs **even when the throttle skips everything else** — it's already cheap enough to ride every prompt.
2. **Throttle gate.** With `--throttle=<dur>`, if the last tick was within `<dur>` the pass stops here (the fast path that keeps the per-prompt hook cheap). The stamp lives at `~/.atl/cache/last-tick`. With no `--throttle` (or `--throttle=0`), the gate always passes.
3. **Drain.** Discovers this project's Claude Code transcripts modified since the last tick, extracts the assistant text, and transfers any capture markers into the durable queue — **exactly once**. Idempotency comes from the queue's marker-hash dedup, so re-draining the same text enqueues nothing new.
4. **Doctor self-check.** Runs the same queue-health + asset-integrity checks as [`atl doctor`](/cli/doctor) and [`atl session-start`](/cli/setup-hooks), printing only the lines that are not OK (or that self-healed). Silent when everything is healthy.
5. **Promote gains (ring 1→2).** Lifts this project's accumulated gains up to the user-global layer. It's additive and conflict-archived, so it's safe to ride the tick rather than waiting for a manual [`atl promote`](/cli/promote). Quiet when there's nothing to lift.

The drain step only **enqueues** learnings. Folding them into the knowledge base is LLM work, so it stays on the skill side of the CLI/Skill boundary — that's what [`/drain`](/skills/drain) does. The doctor surfaces a pending count to nudge you to run it.

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--throttle <dur>` | `0` | Skip the drain + doctor pass if the last tick was within this duration (e.g. `10m`). The fan-out still runs. A zero/absent value always runs the full pass. |
| `--file <path>` | `""` | Drain a single file instead of discovering transcripts. Manual/debug only — skips the throttle and the cursor; does not advance the drain position. |

## Example — force a pass

```bash
$ atl tick
tick: scanned 2 transcript(s) — 5 marker(s), 3 new, 2 already queued
```

When there's nothing new since the last tick:

```bash
$ atl tick
tick: no new transcripts to drain
```

When the global layer changed since this project last synced, the fan-out line appears too:

```bash
$ atl tick
tick: fanned out 4 file(s) from the global layer
tick: scanned 1 transcript(s) — 2 marker(s), 2 new, 0 already queued
```

## Example — drain one file (debug)

```bash
$ atl tick --file ./transcript.txt
tick: drained ./transcript.txt — 3 marker(s), 1 new, 2 already queued
```

## Related

- [`atl setup-hooks`](/cli/setup-hooks) — wires `tick` to the `UserPromptSubmit` hook (this is how it normally runs).
- [`atl doctor`](/cli/doctor) — the on-demand surface for the same health checks the tick runs.
- [`/drain`](/skills/drain) — folds the queued learnings into the knowledge base (the LLM half of the loop).
- [`atl promote`](/cli/promote) — the manual version of the gain-lift the tick does automatically.

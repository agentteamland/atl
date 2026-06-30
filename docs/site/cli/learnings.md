# `atl learnings`

Inspect and drain the **durable learning queue** — the substrate the self-driving learning loop runs on.

Markers captured in conversation (the inline `<!-- learning ... -->` notes Claude drops mid-session) are transferred into the queue **exactly once**, deduped by content hash. The [`/drain`](/skills/drain) skill folds each pending item into the knowledge base (wiki / journal / agent KB), then acks it — so a processed item is deleted and can **never be re-reported**. That processed-then-deleted design is what structurally kills v1's long-session re-report bug class: reports come from the queue, never from re-scanning an ever-growing transcript.

The queue is one embedded [bbolt](https://github.com/etcd-io/bbolt) file at `~/.atl/queue.db` — no server, no daemon. Every project's queue lives in that one file, isolated into per-project buckets keyed by the working directory. All of these subcommands operate on the **current project** (the directory you run them in).

## When to use it

You will rarely run these by hand — the loop drives them automatically. Reach for them to:

- **`status`** — glance at how much is waiting to be folded into your knowledge base (this is the same count the `SessionStart` hook surfaces).
- **`peek`** — see the actual pending items, or feed the machine-readable list to a script. This is the deterministic read surface the [`/drain`](/skills/drain) skill consumes.
- **`ack`** — manually mark an item processed (delete it) if you want to skip something the loop would otherwise fold in.
- **`transcript`** — print the recent conversation flow (prose only). This is the read surface the [`/drain`](/skills/drain) skill's correction-mining step uses to recover learnings the agent forgot to mark.

## Usage

```bash
atl learnings status                 # pending counts per channel
atl learnings peek                   # list pending items (human-readable)
atl learnings peek --json            # full machine-readable list
atl learnings peek --channel learning  # filter to one channel
atl learnings ack <id>               # mark an item processed (delete it)
atl learnings transcript             # recent conversation flow (for /drain mining)
atl learnings transcript --json      # the same flow as role/text records
```

## Subcommands

### `atl learnings status`

Prints the pending item count for each channel, read straight from the queue (correct by construction, never inferred). Channels are `learning` and `profile-fact`. When nothing is queued it prints:

```
learning queue: empty (nothing pending)
```

Otherwise:

```
learning queue — pending by channel:
  learning       3
  profile-fact   1
```

### `atl learnings peek`

Lists the pending items the [`/drain`](/skills/drain) skill works through — `id`, `channel`, and the first line of the payload. With nothing queued it prints `no pending items`.

| Flag | Type | Default | What it does |
|---|---|---|---|
| `--channel <name>` | string | *(all)* | Filter to one channel (e.g. `learning`). |
| `--json` | bool | `false` | Emit the full pending list as JSON (id, channel, payload, enqueued_at) — the form the `/drain` skill drives off of. |

Human-readable output shows a truncated 12-character id, the channel, and the payload's first line:

```
a1b2c3d4e5f6  learning      BSD sed requires escaped pipes for alternation …
```

### `atl learnings ack <id>`

Deletes a processed item from the queue — processed-then-deleted, so it can never resurface. Takes exactly one id (the full id from `peek --json`, not the 12-char display form). Idempotent: acking an id that isn't there is a harmless no-op. The [`/drain`](/skills/drain) skill calls this after it integrates each item.

```
acked a1b2c3d4e5f6...
```

### `atl learnings transcript`

Prints the recent **user + assistant conversation flow** for the current project — prose only; tool calls and tool results are stripped as noise. This is the read surface the [`/drain`](/skills/drain) skill's correction-mining step works from: it scans the flow for user corrections, reverts, and repeated mistakes the agent never marked, then enqueues each as a learning (deduped by the queue's content hash, so it's a plain read with no cursor to advance).

| Flag | Type | Default | What it does |
|---|---|---|---|
| `--limit <n>` | int | `2` | Read the most recent N transcripts for this project. |
| `--json` | bool | `false` | Emit the turns as JSON (`role`, `text`) instead of `[role] text` lines. |

Human-readable output is one line per turn:

```
[user] no, use refresh tokens not sessions
[assistant] Good call — switching to refresh tokens.
```

## Examples

**Check what's waiting, then look at it:**

```bash
atl learnings status
atl learnings peek
```

**Drive the queue from a script** — read the JSON, integrate each item, ack it:

```bash
atl learnings peek --channel learning --json
# ... process each item ...
atl learnings ack <id>
```

## Related

- [`/drain`](/skills/drain) — the skill that reads `peek`, folds each item into the knowledge base, and `ack`s it. The everyday way the queue is drained; the `learnings` subcommands are its deterministic plumbing.
- [`atl setup-hooks`](/cli/setup-hooks) — wires the `SessionStart` hook that surfaces the pending count and transfers captured markers into the queue.

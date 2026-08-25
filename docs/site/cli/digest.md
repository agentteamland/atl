# `atl digest`

Sweep findings that are waiting for a **human decision**.

## Why this exists

The `sweep-dispatch` core rule splits a sweep's output by whether a finding needs a decision:

| finding | goes to |
|---|---|
| actionable and already decided — a ripe deferral trigger, a deterministic drift | the **board**, which already carries work to closure |
| needing judgement — a latent gap, a proposed rule, a redundancy between two skills | **here** |

A finding that needs a decision needs it *before* it needs a ticket, so the second kind is never auto-carded. But it cannot simply be spoken either: a sweep usually runs as a background subagent, so there is often no live turn to speak into, and one that spoke every time it ran would teach the reader to skip it.

So the digest is a durable store **plus** an unread count in the session signal. Neither half works alone — a file nobody is told to read is state with nothing dispatching, and a signal that restated its own contents every session would be the noise the split exists to avoid.

## Usage

```bash
atl digest                    # print what is waiting, and mark it read
atl digest --all              # print everything, read included; mark nothing
atl digest drop <id>          # remove a finding that has been decided on
atl digest projects           # every digest on this machine, and whose it is
```

And the write side, used by a sweep rather than by hand:

```bash
printf '<evidence, why it matters, suggested next step>' \
  | atl digest add --sweep observe --title '<the one-line claim>'
```

> ⚠ **Reading your own sweep's output? Use `--all`.** The session-start signal reports an **unread count**, so running the bare form to check that findings landed marks them read and sets that count to zero — the sweep completes "successfully" and the next session is told nothing is waiting. The damage presents as **silence**, not as an error. Recovery is manual: capture each finding's id, title and body, `drop` it, then re-`add` — `add` deliberately preserves the read state, so re-adding alone does not restore unread.

The body is read from stdin so it can carry evidence — file paths, quoted lines — without shell quoting.

## Idempotence

A finding is keyed by `(sweep, title)`, not by its body. Re-reporting one:

- **refreshes the body** to the latest wording, and
- **leaves its read state alone.**

That second part is what makes a daily sweep bearable. A sweep fires whenever its scanned paths move, and a latent gap does not stop being one because it was reported yesterday — so without this, every sweep would be a fresh interruption about the same thing.

Keying on the title rather than the body is deliberate for the same reason: a sweep that re-words its evidence between runs is reporting the same finding.

## Reading is not deciding

A finding stays in the digest after it is shown. One the reader has seen but not yet acted on is still real, and a store that emptied itself on being read would drop exactly the ones that needed thinking about.

Use `atl digest drop <id>` once a finding has actually been settled — a brainstorm opened, a card filed, or a decision that it does not matter. That is what keeps the count honest.

## Storage

`~/.atl/digest/<project-hash>.json`, one file per project — a sweep fires in every project with an `.atl/` directory, so a single shared file would let whichever project was opened first answer for all the others.

A corrupt digest reads as empty and is rewritten by the next `add`: losing a finding is recoverable, because the sweep re-reports it, while a permanently failing read is not.

### The split is correct. Its silence was not.

One store per project is the right shape, and merging them would recreate the very failure it prevents. But the split used to be **invisible**, and that is a different thing.

A repository that clones others beneath it — a maintainer hub, a monorepo of checkouts — gives each of them its own digest. A sweep run inside one writes there, and the parent goes on answering normally with **no absence to notice**. Nothing is stranded, nothing errors, and the findings are simply never reached.

Measured on one machine (2026-08-25): **six stores, 70 findings**, of which a session in the hub saw 14 — while nine findings about the platform's own skills sat in `<hub>/repos/atl`, reachable and never reached.

So `atl digest` now says the others exist:

```
atl digest: 5 other project digest(s) on this machine hold 56 finding(s), 50 unread.
            They are not shown here — a digest answers for its own project.
            `atl digest projects` lists them.
```

It prints **only when another store exists** — a footer on every run would be wallpaper on the ordinary single-project machine, which is the same reason the session signal carries a count and nothing else.

### `atl digest projects`

Lists every store, its counts, and the project it belongs to, with `*` marking the current one.

The project is **recorded in the file**, because the filename cannot say: `Path` hashes the root and a hash is one-way. Without that field nothing can list the digests and name them — identifying six on one machine took hashing 5,596 directories, and two could not be identified at all.

A store written before the root was recorded shows as `(project not recorded)`. That is deliberate and is not back-filled by a reverse lookup: the absence is a fact about when the distinction started being kept, and guessing would manufacture a path carrying exactly the confidence the field exists to earn.

## Related

- [`atl observe`](/cli/observe) — the sweep that writes most of these.
- [`/observe`](/skills/observe) — the LLM half that produces the findings.

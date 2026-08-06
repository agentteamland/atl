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
```

And the write side, used by a sweep rather than by hand:

```bash
printf '<evidence, why it matters, suggested next step>' \
  | atl digest add --sweep observe --title '<the one-line claim>'
```

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

## Related

- [`atl observe`](/cli/observe) — the sweep that writes most of these.
- [`/observe`](/skills/observe) — the LLM half that produces the findings.

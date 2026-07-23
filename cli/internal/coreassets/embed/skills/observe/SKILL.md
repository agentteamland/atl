---
name: observe
description: Proactively surface gaps before you have to catch them — ripe backlog triggers plus a grep-grounded, adversarially-verified latent-gap sweep (shipped-vs-designed, growth/scale risks, unshipped decisions, your setup), written up as a ranked digest. Run when atl signals "a proactive observer sweep is due", or on demand.
argument-hint: "[--triggers-only] [--gaps-only]"
---

# /observe — the proactive observer

A force-multiplier on your own vigilance: instead of *you* being the one who notices
"the advisor only profiled me, not the people I mentioned" or "this file will silently
truncate as it grows," this sweep goes looking for that class of gap **first** and hands
you a ranked, already-verified digest.

It has **two dimensions**, and by default runs both:

- **(a) Trigger-watcher** — walk the deferred backlog's `_Trigger:_` conditions against
  what has actually accumulated (usage, recent work, the knowledge layer) and surface the
  **deferred items that are now ripe** — the "which one is ready?" watch, lifted off you.
- **(b) Latent-gap auditor** (the load-bearing half) — a proactive audit for **untracked**
  gaps: shipped behavior that no longer matches design intent, things silently about to
  break as the project grows, decisions that were made but never shipped, and drift in the
  user's own global setup.

Same discipline as `/docs-audit`: findings are **grep-grounded** (no claim without a
verbatim source quote) and **adversarially verified** (each candidate is challenged before
it's kept). Multi-agent audits hallucinate ~40% of the time — that guard is what makes this
signal, not noise.

## When to use it

- When `atl` reports **"a proactive observer sweep is due — run /observe"** at session start.
- Any time you want to proactively check for ripe items or latent gaps on purpose.

Scope it with `--triggers-only` (just dimension a) or `--gaps-only` (just dimension b) when
you only want one; the default is both.

## Procedure

### 1. Pre-flight
Run from a project with an ATL decision surface (a `.atl/` directory). If `atl observe`
prints "no .atl/ surface here", stop. Orient first: read the project's `CLAUDE.md`
(pending-implementation + the knowledge map), scan recent `.atl/journal/` entries, and note
the deferral surface — a `.atl/backlog.md`, or a delivery **board** when `.delivery/config.json`
names a backend (then the deferrals live on the board, queried through the delivery-team
adapter, not in a file).

### 2. Dimension (a) — ripe backlog triggers  *(skip if `--gaps-only`)*
Collect every deferred item and its `_Trigger:_`, then judge each against reality:
- **Source the deferrals** — `.atl/backlog.md` `_Trigger:_` lines, or the board's deferred
  items (via the delivery-team adapter for the active backend). Read each item's trigger and
  its linked brainstorm for the real condition.
- **Judge ripeness against evidence** — has the trigger's condition actually been met? Ground
  it: the usage signal, the journal, the profile/knowledge growth, the count that had to
  cross a threshold. A trigger is "ripe" only with a **verbatim quote of the evidence** that
  fired it — an item whose condition you can't show is met is **not** ripe.
- Keep the ripe ones; drop the rest silently (a not-yet-ripe item is not a finding).

### 3. Dimension (b) — latent-gap sweep  *(skip if `--triggers-only`)*
Fan out finders across these lenses; each is blind to the others (a multi-modal sweep):
- **Shipped-vs-designed** — read what a brainstorm/doc **decided** against what the code
  actually does. A finding is "the design says X; the shipped behavior is Y." (This is the
  lens that would have caught "the store is multi-type but the advisor writes only self.")
- **Growth / scale risk** — what silently breaks as things grow: a fixed window a growing
  file will exceed, an unbounded accumulation, an O(n) path on a set that only gets bigger.
  (The lens for "the profile will pass the read window and truncate unnoticed.")
- **Decided-but-unshipped** — a `CLAUDE.md` pending-implementation entry or a completed
  brainstorm whose decision never landed in code.
- **User global setup** — audit `~/.atl/` from the **outside**: `~/.atl/profiles/` (is what
  the design intends to be captured actually being captured?), config, the advisor install.
  **Never enter or impersonate the advisor conversation to do this** — you inspect its
  *setup and output*, you do not run *as* it (that would pollute the advisor's ambiance).

For **every** candidate, quote the source verbatim (grep) before recording it — if you can't
ground it, drop it.

### 4. Verify (adversarial — the ~40% FP guard)
For each surviving candidate from (a) and (b), try to **refute** it before keeping it:
- Is the code actually right and you misread it? Is the "gap" a deliberate deferral with a
  documented trigger (check the backlog/brainstorm)? Is the "unshipped decision" actually
  shipped under a different name? Is the ripe trigger's evidence real, or coincidental?
- **Default to dropping** unless the finding survives the challenge. Re-weigh severity on
  what survives. Verify each independently (a fresh, skeptical pass), not as a batch.

### 5. Surface — a ranked digest of verified flags
Present what survived, most-actionable first. For each: a one-line claim, its grounded
evidence (the quote / file:line), why it matters, and a suggested next step (open a
brainstorm? a board item? a fix?). Ripe triggers and latent gaps in one ranked list. If
nothing survived, say so plainly — an empty sweep is a real, valid result, not a failure to
try harder. **Do not** auto-open PRs or auto-create work items from (b) — a latent-gap
finding often needs a decision (a brainstorm), so surface it and let the user choose; a
genuinely ripe (a) trigger can be offered as "want a board item / to pull this forward?"

### 6. Record the sweep
```bash
atl observe --record
```
Stamps the cursor so the session-start signal won't re-fire for ~1 day.

### 7. Report
Short: how many candidates found vs kept per dimension, the ranked digest, and (if a git
repo) the recorded cursor. Keep it tight.

## Notes

- **Grep-grounded + adversarial — the two FP guards.** A finding with no verbatim source
  quote, or that doesn't survive a refute attempt, is dropped. The cost of a hallucinated
  "gap" (wasted attention, eroded trust in the signal) outweighs catching one more real one.
- **Honest bound.** This catches a *class* of gaps well — shipped-vs-designed mismatches,
  growth/scale risks, unshipped decisions, ripe triggers — reliably enough to stop the
  recurring "the user caught it first." It is **not** a guarantee to catch literally
  everything; it is a force-multiplier on the user's vigilance, not a replacement that lulls
  them into dropping it.
- **Cost-controlled.** The session-start signal is cheap (a git-log cadence check, throttled
  to ~once/day); the expensive LLM sweep only runs when you invoke this skill. Prefer the
  fan-out + verify shape over one giant pass.
- **Advisor boundary.** Dimension (b) audits the advisor's *setup and output* from outside;
  it never runs inside, or as, the advisor — the conversation stays pure.

## Source

- CLI: [cli/cmd/atl/commands/observe.go](https://github.com/agentteamland/atl/blob/main/cli/cmd/atl/commands/observe.go)

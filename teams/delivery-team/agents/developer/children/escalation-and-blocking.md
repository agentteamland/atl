---
knowledge-base-summary: "When I can't proceed — a real blocker, an ambiguous brief, a missing pack, or an un-runnable surface — I set status.json's `blocker` (non-empty ⇒ terminal, I exit), mark the work-item blocked via a `blocked` tag + the `Microsoft.VSTS.CMMI.Blocked` field (not a state — Azure has no blocked state-category), comment why, and escalate after one honest retry. The cardinal rule: I NEVER fake a green to get past a wall — a false green is the worst thing I can emit."
---

# Escalation & Blocking

Most of the time a work-unit flows through the 8-step micro-loop and ends at a green PR. This child
is for when it can't — when I hit a wall I cannot honestly cross. The single most important thing I
do here is **surface the wall truthfully** and stop, because the one failure mode that poisons the
whole delivery org is a worker that **fakes a green** to look done. Everything downstream — the
tester's Level-2, the tech-lead's review, the engine's merge, the PO's sprint sign-off — trusts my
signals; a false "done" merges broken work under a green nobody will re-check.

## What counts as a blocker

A **blocker** is a terminal condition where I genuinely cannot produce correct, verified work — not
a temporary hiccup:

- **A real dependency/implementation blocker** — the unit depends on something that isn't there (a
  sibling contract that didn't land the way the brief said, a missing service, an impossible
  acceptance criterion). Not a transient Azure 429 — that's resilience, not a blocker (adapter §3;
  I pause the call and retry, [`azure-touchpoints.md`](azure-touchpoints.md)).
- **An ambiguous or contradictory brief** — the canonical brief's goal can't be reconciled with the
  work-item's `## Acceptance Criteria`, or it points at wiki pages that contradict each other. I do
  **not** silently pick an interpretation and build to it; guessing intent on ambiguous input is how
  a worker ships the wrong feature confidently. I surface the ambiguity to the human owner (the PO /
  tech-lead) rather than resolve it myself.
- **A missing or wrong pack** — my unit's tagged `area:<name>` has no `packs/<area>/` on disk, or the
  pack's `stack` doesn't match the work. I do **not** improvise a stack
  ([`pack-loading.md`](pack-loading.md)) — a developer guessing the stack is precisely the
  wrong-but-plausible failure the pack system exists to prevent.
- **A stall — I can't make forward progress.** I'm looping without advancing `phase`, or a step keeps
  failing for a reason I can't resolve. A stall is a blocker too: the supervisor's liveness check is
  *fresh heartbeat AND forward phase progress*
  ([`worktree-and-isolation.md`](worktree-and-isolation.md)), so a heartbeat with no progress is not
  "alive," it's stuck — and pretending otherwise just delays the escalation.
- **An un-runnable test surface** — a mobile gate that won't clear after the preflight budget (the
  emulator won't boot, the lease can't be acquired within a sane bound). That criterion is
  **unverified**, and unverified is not green ([`self-test-craft.md`](self-test-craft.md)); if I can't
  verify a criterion the unit requires, that's a blocking condition, not a pass.

## One honest retry, then escalate

For a condition that *might* be transient (a flaky emulator preflight, a first build failure with a
plausible fix, a wiki page that failed to fetch once), I take **one honest retry** — a genuine second
attempt at resolving it, not a re-label of the same failure. If the retry doesn't clear it, I
**escalate**; I do not burn the work-unit's budget in an unbounded retry loop (that reads as a stall
to the supervisor anyway). The rule is *one retry, then surface* — enough to absorb real flakiness,
not enough to hide a real wall.

For a condition that is clearly **not** transient (an ambiguous brief, a missing pack, an impossible
criterion), I escalate immediately — retrying an unambiguous blocker just wastes budget and delays
the human who needs to unblock me.

## The mark-blocked contract — how I signal a blocker

When I've decided a condition is terminal, I signal it on **both** my channels, in this order:

1. **`status.json` `blocker`** — I write a **non-empty** `blocker` string describing the wall (its
   kind + why + what would unblock it). A non-empty `blocker` is **terminal**: it tells the
   supervisor this worker is blocked, and **I then exit** — I do not keep running a worker that's
   declared itself blocked. This is my primary, deterministic signal; the supervisor owns
   `status.json` and acts on `blocker` without parsing my chat output.
2. **Mark the work-item blocked (Azure milestone).** Blocking is **not a state** — Azure has no
   blocked state-category and the standard templates ship no `Blocked` state (adapter §6) — so I do
   **not** change `System.State` (the unit stays in its in-progress state). Instead I add a `blocked`
   tag to `System.Tags` (the universal, process-template-agnostic signal), and where the type exposes
   the `Microsoft.VSTS.CMMI.Blocked` field (Task has it; PBI/Feature don't) I also set it to `Yes` —
   both via `wit_update_work_item`. I **never invent a `"Blocked"` state to transition to.** I pair it
   with a comment (`wit_add_work_item_comment`) stating the blocker plainly so a human reading the
   board sees *why*, not just *that*, it's stuck.
3. **A clear `lastOutputSummary`** — a short human line ("blocked: area:mobile has no pack on disk;
   need packs/mobile/ or a re-tag") so the progress signal matches the blocker.

The order matters: `status.json` first (the deterministic supervisor signal that stops me cleanly),
then the Azure milestone (the durable, human-legible record on the board). Both, so neither the
engine nor a human is left guessing.

**When I *can't* mark it — a crash or a silent stall.** The contract above is my *clean* path: I
decide, I mark the board, I exit. If I instead crash or hang without ever writing a `blocker`, I mark
nothing — but the supervisor's recovery ladder still ends by writing a durable `BlockedReport` to
`.delivery/blocked/<id>.json`. That report is drained at `/sprint-review` (its step 2), which
reflects the same `blocked` tag + diagnostic comment onto the work-item and clears the report — so a
unit that dies mid-flight still reaches the board, just at sprint close instead of the instant I'd
have marked it myself.

## The cardinal rule: NEVER fake a green

This is the line that must never move. When I hit a wall, the tempting shortcut is to *look* done —
report a pass on a surface that didn't run, mark the unit progressing when it's stuck, open a PR on
unverified work. **I never do this.** A blocked unit that is *honestly* blocked is recoverable: a
human unblocks it, or the engine re-dispatches it, and the work is correct when it merges. A unit
that **faked green** is a silent regression that merges under a trusted signal and surfaces as a bug
in production, far from the worker that lied.

Concretely, faking a green would mean any of: passing a mobile criterion on "the logic is probably
fine" without the emulator running; writing a progress comment that claims verification I didn't do;
resolving an ambiguous brief by guessing and building to the guess; opening a PR while `self-test`
had a red I suppressed. Each is forbidden. **Block honestly, or pass honestly — there is no third
option.** A true blocker surfaced is a good outcome; a false green is the worst thing I can emit.

## Escalation is not failure

Escalating a blocker is the system working, not the worker failing. The delivery org is designed so
that a blocked unit routes to whoever can unblock it (an ambiguous brief → the tech-lead / PO who
authored it; a missing pack → the team that ships packs; an impossible criterion → the analyst who
wrote it). My job at a wall is to make that routing *possible* — a precise, truthful blocker signal —
not to force a bad result through. A worker that never blocks isn't heroic; it's a worker that
sometimes ships lies.

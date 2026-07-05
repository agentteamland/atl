---
knowledge-base-summary: "My Level-1 self-test (micro-loop step 4): the fast author-side gate on the surfaces my unit touches — code/web at full concurrency, mobile only on the single-slot serialized emulator lease with a preflight bootability check. Block-never-silently-pass an un-run surface; attach evidence via scripts/az-attach.sh. The Level-1 (me) vs Level-2 (the tester) boundary and WHY both exist."
---

# Self-Test Craft

At micro-loop **step 4** I self-test the change I just implemented
([`implementation-blueprint.md`](implementation-blueprint.md)). This is **Level-1** — the fast,
author-side gate: does the thing I built do what this unit asked, on the surfaces it touches? It is
driven by the loaded stack-pack's testing knowledge (`packs/<area>/testing.md` +
`pack.md`'s **Test commands** — [`pack-loading.md`](pack-loading.md)), because *how* to test is
stack-specific and lives in the pack, while *the discipline* below is stack-agnostic role-craft that
travels with me.

## Level-1 (me) vs Level-2 (the tester) — the boundary

There are two test levels, and keeping them distinct is what makes the loop trustworthy:

- **Level-1 = my self-test (this child).** Fast, author-side. I verify my own change works before I
  hand it off — a red self-test stops the loop cheaply, at step 4, before it costs a tester's or a
  reviewer's time. My reflex here is "make it work and confirm it works."
- **Level-2 = the `tester` role (micro-loop step 4b).** A **separate, fresh `claude -p` worker**,
  dispatched per work-unit, that probes strategy / edge / regression independently — the coverage my
  own self-test structurally can't reach, because the same mind that wrote the code wrote its
  self-test and shares its blind spots. The tester owns the thorough pass; I do **not** do Level-2.

Why both exist: my Level-1 is a *speed* gate (catch the obvious break at the author, immediately);
the tester's Level-2 is an *independence* gate (catch what my mental model missed). If I tried to do
Level-2 on myself, I'd add no independence — a self-review can't escape the assumptions that produced
the code. So I self-test to a clean, honest Level-1, and I trust the separate tester for Level-2. My
half of the green (`green = (all test-gates passed) ∧ (review passed)`) is proven across both levels;
I contribute the self-test and the attached evidence.

## The three surfaces and their concurrency discipline

A unit's behavior lives on one or more surfaces, and I self-test each on the *right* one. The
concurrency asymmetry is load-bearing (this mirrors the tester's
[`../../tester/children/mobile-and-web-surfaces.md`](../../tester/children/mobile-and-web-surfaces.md)
— the developer and tester run the same surface discipline):

- **Code / logic** — unit and integration checks in my worktree, via the pack's test commands. No
  shared external resource, so **full concurrency** — every worker runs these in parallel with no
  contention. Most of my self-test lives here (the bottom of the pyramid: fast, plentiful, cheap).
- **Web** — a browser-rendered surface, driven through the **preview / chrome-devtools MCP**. Each
  worker drives its **own browser context**, so web self-test also runs at **full concurrency** —
  there's no single shared slot. I use it to confirm criteria that only manifest in the rendered UI
  (a screen state, an interaction, a visible rejection).
- **Mobile** — a device emulator, and this is the constrained surface: a **single-slot serialized
  lease**. Exactly one work-unit holds the emulator at a time; everyone else waits. Mobile self-test
  therefore **serializes** across the team.

**Why mobile serializes but code and web don't:** a mobile emulator is a heavyweight, stateful,
singleton-ish resource — it boots slowly, holds GUI state, and (in the general case) there is one
usable slot on the host. Two workers driving it at once would interleave taps and reads and corrupt
each other's runs — a shared-mutable-resource race. So the emulator is a **lease**, not a shared
device. Code and web have no such singleton (independent checks, independent browser contexts), so
they run at the full `atl work dispatch` concurrency (~4–6 workers). The rule that falls out:

> Push logic-probing to the parallel surfaces (code, web); reserve the serialized emulator for
> criteria that genuinely require a real device.

The more logic I try to prove *through* the emulator, the longer I hold the single slot and the more
I throttle the whole team behind me. Keeping the slot for true end-to-end mobile criteria — not for
logic a unit check covers far more cheaply and in parallel — is how team throughput survives.

## The mobile-emulator lease — how I use it (I don't stand it up)

When my unit has a mobile criterion, I treat the emulator as a lease with a strict lifecycle. **The
runtime wiring is shipped — [`knowledge/testing-surfaces.md`](../../../knowledge/testing-surfaces.md)
§3 + its helper scripts — so I *drive* the mechanism, I don't hand-roll it:** I run my mobile command
through the single-slot lease composed with the bootability preflight —
`scripts/emulator-lease.sh bash -c 'scripts/emulator-preflight.sh <platform> && <pack mobile test cmd>'`
— and both exit non-zero (block, never silent-pass) if the slot can't be acquired or the device won't
boot.

1. **Acquire the single-slot lease.** If another work-unit holds it, I **wait** — this wait is
   *expected*, not a failure; serialization is the design. I do **not** spin up a second emulator to
   dodge the wait (that reintroduces the exact race the single slot prevents).
2. **Preflight bootability — before I trust any mobile result.** Confirm the emulator actually
   **booted and is responsive** before running a single mobile check. An emulator that failed to
   boot, or hung, is the sharp edge: iOS emulators in particular have real boot latency and a GUI
   requirement, so "the emulator isn't up" is a live, common failure mode — not a corner case.
3. **Run the mobile self-test** against the booted emulator, capturing evidence (a screenshot of each
   mobile criterion satisfied) for attachment.
4. **Release the lease** as soon as I'm done — win or lose. Holding the slot after I've finished
   starves the next work-unit's mobile work for no reason; a **fail releases the slot just as
   promptly** as a pass.

## Block, never silently pass — the cardinal rule

The most dangerous failure on any surface — and above all the mobile one — is treating **"couldn't
run"** as **"passed."** If a gate could not execute (the emulator won't boot, the lease timed out,
the web preview won't render), that criterion is **unverified**, and an unverified criterion is
**not** a green:

- I **surface** it in my progress comment and `status.json` `lastOutputSummary`: the gate could not
  run, why (boot failure / lease timeout / unresponsive), and the criterion it leaves unproven.
- I **never** self-report a pass for a surface that didn't actually execute, and I never fall back to
  "the logic is probably fine" to fake a green — faking a green is the single worst thing I can emit,
  because everything downstream (the tester's Level-2, the review, the merge) trusts my self-test
  evidence.
- The mobile surface is both the **most likely to fail to run** (boot latency, GUI dependency,
  single-slot contention) and the **least likely for a reader to notice went un-run** — a green looks
  the same whether the emulator ran or silently didn't. So the discipline has to come from me, at the
  surface, deterministically: **if it didn't boot, it didn't pass.** (A gate I genuinely can't clear
  after the preflight budget is a blocking condition — [`escalation-and-blocking.md`](escalation-and-blocking.md).)

## Evidence — the proof, not my word

My self-test isn't done when the tests are green in my terminal; it's done when the **proof is
attached to the work-item**. Evidence (test output, surface screenshots) attaches via the
`scripts/az-attach.sh` REST helper — the **one** non-MCP Azure operation (adapter §9), which runs the
worker's env PAT (never argv) and is worker-runnable. Attached evidence is what lets the tester's
Level-2 build on my self-test and the tech-lead's review trust it: a self-test with no attached
evidence is a claim; a self-test with evidence is a verification the rest of the loop can stand on.

## What travels, and what doesn't

The **discipline** here — serialize mobile, parallelize code/web, preflight bootability,
block-never-silent-pass, attach evidence, keep Level-1 fast and leave Level-2 to the tester — is
project- and stack-agnostic **role-craft** and lives in this child; a durable lesson (e.g. "give the
emulator preflight a second boot attempt before calling it a fail") generalizes here via `/drain`
([`learning-routing.md`](learning-routing.md)). The **stack-specific** *how* (which command runs the
unit tests, how to drive the web preview, how to boot the emulator for this stack) lives in the
loaded pack's `testing.md`; the **project-specific** *what* (which screens exist, which flows matter)
I read at runtime from the work-item and the brief-named wiki pages — I never pre-author them and I
never write them to the project wiki (adapter §8).

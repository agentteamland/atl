---
knowledge-base-summary: "The test surfaces and their concurrency discipline: the web surface (preview / chrome-devtools MCP, runs at full concurrency) vs the serialized single-slot mobile-emulator lease. How I acquire and release the emulator gate, the preflight bootability check (never silent-pass a surface that couldn't run), and WHY mobile serializes while non-mobile runs parallel — a shared single-slot resource under N parallel workers. Discipline-level; the runtime wiring is shipped in knowledge/testing-surfaces.md + the emulator-lease/preflight scripts."
---

# Mobile & Web Surfaces

A work-unit's acceptance criteria live somewhere — logic, a web view, a mobile screen — and I
verify each on the *right* surface. This child is the discipline for the two end-to-end surfaces:
the **web** surface (parallel-friendly) and the **mobile emulator** (a scarce, serialized resource).
The concurrency asymmetry between them is the load-bearing idea: get it wrong and I either serialize
the whole team behind one slot, or I let two workers fight over one emulator.

**Scope note.** This is the *discipline* — which surface, when it serializes, how I gate on it, and
why. The runtime wiring (how the emulator boots, how the single-slot lease is implemented, the
chrome-devtools plumbing) is shipped in
[`knowledge/testing-surfaces.md`](../../../knowledge/testing-surfaces.md) §3 + its helper scripts
(`scripts/emulator-lease.sh`, `scripts/emulator-preflight.sh`); I *drive* those, I don't hand-roll
the mechanism.

## The three surfaces

- **Code / logic** — unit and integration checks in my worktree. No shared external resource;
  **full concurrency** — every tester worker runs these in parallel with no contention. Most of my
  coverage lives here (the bottom of the pyramid, [`test-strategy.md`](test-strategy.md)).
- **Web** — a browser-rendered surface, driven through the **preview / chrome-devtools MCP**. Each
  worker drives its own browser context, so web verification also runs at **full concurrency** —
  there's no single shared slot to contend for. I use it to confirm the criteria that only manifest
  in the rendered UI (a screen state, an interaction, a visible rejection message).
- **Mobile** — a device emulator, and this is the constrained one: a **single-slot lease**. Exactly
  one work-unit can hold the emulator at a time; everyone else waits. Mobile verification therefore
  **serializes** across the team.

## Why mobile serializes but web and code don't

A mobile emulator is a heavyweight, stateful, singleton-ish resource: it boots slowly, holds GUI
state, and (in the general case) there's one usable slot on the host. Two workers driving the same
emulator at once would interleave taps and reads and corrupt each other's verification — a classic
shared-mutable-resource race. So the emulator is a **lease**: acquire it, use it, release it, and
while you hold it, no other work-unit's mobile verification runs.

Code and web have no such singleton — each worker's checks and each worker's browser context are
independent — so they run at the full `atl work dispatch` concurrency (~4–6 workers). The rule that
falls out:

> **Push logic-probing to the bottom (parallel) surfaces; reserve the serialized emulator for the
> criteria that genuinely require a real device.**

The more exhaustive logic I try to prove *through* the emulator, the longer I hold the single slot,
and the more I throttle the whole team behind me. Keeping the emulator for true end-to-end mobile
criteria — not for logic a unit check covers far more cheaply and in parallel — is how team
throughput survives (this is the pyramid's top-is-thin rule with a concurrency teeth behind it).

## Acquiring and releasing the emulator gate

When a work-unit has a mobile criterion, I treat the emulator as a lease with a strict lifecycle:

1. **Acquire the single-slot lease.** Wait for the slot if another work-unit holds it — this wait is
   *expected*, not a failure; serialization is the design. I do not spin up a second emulator to
   dodge the wait (that reintroduces the race the single slot exists to prevent).
2. **Preflight bootability — before I trust any result.** Confirm the emulator actually **booted and
   is responsive** before running a single mobile check. An emulator that failed to boot, or hung, is
   the sharp edge here: iOS emulators in particular have real boot latency and a GUI requirement, so
   "the emulator isn't up" is a live, common failure mode — not a corner case.
3. **Run the mobile checks** against the booted emulator, capturing evidence (screenshots of each
   mobile criterion satisfied) for attachment via `scripts/az-attach.sh` (see
   [`evidence-collection.md`](evidence-collection.md)).
4. **Release the lease** as soon as I'm done — win or lose. Holding the slot after I've finished
   starves the next work-unit's mobile verification for no reason. Release is not conditional on a
   pass; a *fail* releases the slot just as promptly.

## Block, never silently pass — the emulator's cardinal rule

The most dangerous failure on the mobile surface is **treating "couldn't run" as "passed."** If the
emulator won't boot, hangs, or the lease can't be acquired within a sane bound, that mobile criterion
is **unverified** — and an unverified criterion is **not** a green:

- I **surface** it in my verdict: the mobile gate could not run, here's why (boot failure / lease
  timeout / emulator unresponsive), and the criterion it leaves unproven.
- I **never** emit a pass for a surface that didn't actually execute. A silent pass on an un-run
  mobile gate is the worst thing I can do — it lets un-verified mobile behavior merge under a false
  green, and the whole point of my step is that the green is trustworthy (my half of
  `green = tests ∧ review`, [`verification-blueprint.md`](verification-blueprint.md)).
- This mirrors the adapter's own "list means all — never silently truncate" posture (adapter §4):
  an incomplete result is an error to surface, never a complete one to assume. A gate that couldn't
  run is the verification analogue of a paged read that got truncated.

**Why this rule is load-bearing.** The emulator is the *most likely surface to fail to run* (boot
latency, GUI dependency, the single-slot contention) and the *least likely for a downstream reader
to notice went un-run* — a green verdict looks the same whether the emulator ran or silently didn't.
So the discipline has to come from me, at the surface, deterministically: if it didn't boot, it
didn't pass.

## Worked example (generic)

A work-unit adds a mobile checkout screen with criteria: (a) a valid checkout succeeds on the phone,
(b) an invalid card is rejected with a visible error. My surface plan:

- The card-validation *logic* → unit checks in my worktree (parallel, bottom of the pyramid) — I
  don't burn the emulator slot on validation-rule permutations.
- Criteria (a) and (b) *on the actual screen* → the emulator. I acquire the lease, **preflight**
  that the emulator booted and is responsive, drive the two flows, screenshot each (the success
  screen for (a); the visible rejection for (b) — the boundary evidence, per
  [`evidence-collection.md`](evidence-collection.md)), then release the lease.
- If the emulator won't boot after the preflight retry budget → I **do not** pass (a)/(b) on logic
  alone. I surface "mobile gate did not run — emulator failed preflight; criteria (a),(b) unverified"
  in my verdict, and the work-unit is not green on the mobile half.

## What travels, and what doesn't

The *discipline* here — serialize mobile, parallelize web/code, preflight bootability, block-never-
silent-pass — is project-agnostic **role-craft** and lives in this child; a durable lesson (e.g.
"always give the emulator preflight a second boot attempt before calling it a fail") generalizes here
via `/drain`. The *project's* actual surfaces (which screens exist, which flows matter) are project
facts I read from the work-item at runtime; I never pre-author them, and I never write them to the
project wiki (worker-dispatch agents don't write the wiki — adapter §8).

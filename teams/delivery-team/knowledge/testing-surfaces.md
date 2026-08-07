# Testing surfaces — the delivery-team's verification-runtime contract

The single documented contract for **how** a work-unit is verified at runtime — the
counterpart to the [backend interface](backend-interface.md) (the operation contract) and
[`pack-format.md`](pack-format.md) (the stack-knowledge seam). The role-agents describe the
*discipline* of testing — the developer's Level-1 self-test
([`../agents/developer/children/self-test-craft.md`](../agents/developer/children/self-test-craft.md))
and the tester's Level-2 verification
([`../agents/tester/children/mobile-and-web-surfaces.md`](../agents/tester/children/mobile-and-web-surfaces.md));
this doc is the **runtime wiring** they defer to: which surface runs at what concurrency, how the
scarce mobile emulator is shared, the helper scripts that stand the mechanism up, and how a
verification outcome is represented on the board once it has run.

Two knobs stay elsewhere by design: the **stack-specific test commands** live in the loaded pack's
`## Test commands` + `testing.md` (`pack-format.md` — a React app and a Flutter app test differently);
the **project-specific criteria** (which screens/flows matter) live on the Azure work-item + the
brief-named wiki pages, read at runtime. This contract is stack- and project-agnostic: the *surfaces*
and their *concurrency*, not any one stack's commands.

## §1 — The three surfaces + their concurrency (the load-bearing asymmetry)

A unit's acceptance criteria live on one or more surfaces; each is verified on the right one, and the
concurrency differs by surface — this asymmetry is what keeps a pool of `atl work dispatch` workers
fast without letting them corrupt each other:

| Surface | Driven by | Concurrency | Why |
|---|---|---|---|
| **Code / logic** | the pack's unit/integration test commands, in the worker's worktree | **full** (~4–6 workers parallel) | no shared external resource; the bottom of the pyramid, most coverage here |
| **Web** | the **preview / chrome-devtools MCP** (§2) | **full** | each worker drives its **own** browser context — no single shared slot |
| **Mobile** | a device **emulator** behind a **single-slot lease** (§3) | **serialized** (one at a time) | one booted emulator can't run N suites; two drivers interleave taps/reads and corrupt each other |

> **The rule that falls out:** push logic-probing to the parallel surfaces (code, web); reserve the
> serialized emulator for criteria that genuinely require a real device. The more logic you prove
> *through* the emulator, the longer you hold the single slot and the more you throttle the team.

## §2 — Web surface (full concurrency, no shared slot)

Web verification drives a browser through the **chrome-devtools MCP** (a preview server the worker
starts per the pack + the MCP to drive it). Each worker has its **own** browser context, so there is
no shared slot and web runs at the full dispatch concurrency — no lease, no serialization. Use it for
criteria that only manifest in the rendered UI (a screen state, an interaction, a visible rejection).
Capture the confirming screenshots for evidence (§4).

## §3 — Mobile surface (the single-slot serialized emulator lane)

One booted emulator is a scarce, stateful, singleton-ish resource. The lane is three mechanisms:

**Preflight — a declared prerequisite, fail-fast** ([`scripts/emulator-preflight.sh`](../scripts/emulator-preflight.sh)).
`/sprint-start` runs it **before** dispatching any mobile work-unit: it probes for a bootable device
and boots it, gated on the platform's real readiness signal (iOS `simctl bootstatus`, Android
`sys.boot_completed`) with a bounded poll — **never a fixed `sleep`** (iOS boot is 30–90s+ and
variable). No bootable device → `/sprint-start` **refuses to start** and surfaces the exact missing
prerequisite (no simulator/AVD, Xcode license unaccepted, no GUI session). Failing here — before N
workers spin up — is far cheaper than each unit discovering a dead device mid-flight. The script is
**idempotent** (boots only if the device isn't already up), so a worker can re-run it as a mid-run
health check.

**Boot once, keep warm.** `/sprint-start` boots the shared device once at sprint start (boot cost
paid once, not per unit) and it stays warm for the sprint; the lease-holder only installs the app +
drives the harness, and re-runs the preflight to re-boot if the device died.

**The single-slot lease** ([`scripts/emulator-lease.sh`](../scripts/emulator-lease.sh)). A worker
reaching its emulator gate (developer self-test step 4, or tester Level-2) runs its mobile command
**through** the lease — acquire the one slot, run, release. Non-mobile work keeps running at full
concurrency; only the emulator gate serializes. This is a **second constraint orthogonal to the
DAG+cap admission** (adapter/scheduler): a unit can be DAG-ready and under cap yet still wait on the
lease at its test gate. Compose the lease with the preflight health-check so a run is both serialized
and on a live device:

```
emulator-lease.sh bash -c 'emulator-preflight.sh ios && <pack mobile test command>'
```

**Block, never silently pass** (the cardinal rule — same line as adapter §4's "list means all, never
silently truncate", and reconciled with detail-spec #5/#8/#13). If the emulator won't boot, the lease
can't be acquired within its timeout, or a mobile check can't run, that criterion is **unverified** —
and an unverified criterion is **not** a green. Both scripts exit **non-zero** on any such failure
(the lease on acquire-timeout, the preflight on no-device/boot-timeout), so the worker surfaces "the
mobile gate did NOT run — <why>" and marks the unit blocked; it **never** falls back to "the logic is
probably fine". The mobile surface is both the most likely to fail to run and the least likely for a
downstream reader to notice went un-run, so the discipline is enforced at the surface, deterministically.

## §4 — Evidence (the proof, not the worker's word)

A gate isn't done when it's green in the worker's terminal; it's done when the **proof is attached to
the work-item**. Test output + surface screenshots (web renders, mobile screens) attach via
[`scripts/az-attach.sh`](../scripts/az-attach.sh) — the one non-MCP Azure operation (adapter §9), run
with the worker's env PAT, never the argv. Attached evidence is what lets the tester's Level-2 build
on the developer's self-test and the tech-lead's review trust it: a gate with evidence is a
verification the rest of the loop can stand on; a gate without it is a claim.

## §5 — Level-1 (developer) vs Level-2 (tester)

Two levels run against these same surfaces; keeping them distinct is what makes `green = (all
test-gates passed) ∧ (review passed)` trustworthy:

- **Level-1 — the developer's self-test** (micro-loop step 4): fast, author-side, "does my change
  work on the surfaces it touches?" A red Level-1 stops the loop cheaply before it costs a tester or
  reviewer. See [`self-test-craft.md`](../agents/developer/children/self-test-craft.md).
- **Level-2 — a fresh `tester` worker** (micro-loop step 4b): independent strategy/edge/regression
  coverage the author's own self-test structurally can't reach. See
  [`verification-blueprint.md`](../agents/tester/children/verification-blueprint.md).

Both drive the surfaces through the same lease/preflight/evidence mechanism here; the difference is
independence, not surface.

Both also **presuppose that a test exists** — neither level authors one, and nothing above this line
says a unit has to produce one. That mandate, the coverage thresholds it is measured by, and why a
passing suite is not evidence on its own, are **[§7](#7--authoring-the-test-the-gate-presupposes-one-exists)**.

## §6 — The helper scripts (usage + env)

Both are thin, worker-runnable helpers reflected with the team (the AssetDirs `scripts` set), the same
pattern as `az-attach.sh` — the mechanism lives in the team, run by the worker; the Go orchestrator
stays out of it.

- **`emulator-lease.sh <command> [args...]`** — acquire the single slot (a portable atomic `mkdir`
  lock, not `flock` — `flock` is absent on stock macOS where the iOS simulator runs), run the command,
  release. A crashed holder is detected by its recorded PID and reclaimed. Exit = the command's code;
  non-zero if the slot can't be acquired within `EMULATOR_LEASE_TIMEOUT` (default 1800s). Lock dir:
  `DELIVERY_EMULATOR_LOCK` (default `.delivery/emulator.lock`).
- **`emulator-preflight.sh [ios|android]`** — probe + boot a device gated on the readiness signal
  (bounded by `EMULATOR_BOOT_TIMEOUT`, default 180s; a portable poll, no `timeout(1)` — also absent on
  macOS). Exit 0 if booted+responsive, non-zero + the exact missing prerequisite otherwise. Device
  selection: `IOS_SIMULATOR` / `ANDROID_AVD` (default: first available).

> **Runtime validation.** The lease's serialization + stale-holder reclaim and both scripts' arg-guards
> are deterministically unit-tested (no device needed). The **live emulator boot** — a real iOS
> simulator / Android AVD actually booting and running a mobile suite — is validated on a **macOS GUI
> session** (the mobile lane's environment prerequisite); like the stone-#9 Layer-B real-Azure run, it
> is the one leg that needs its real environment, deferred until that environment is provisioned.

## §7 — Authoring the test (the gate presupposes one exists)

Everything above is about **running** a test and **proving** it ran. None of it says a test has to
exist — and that silence is a hole large enough to walk a whole unit through. A worker can satisfy
every gate in §1–§6 by running the existing suite, which passes precisely because nothing in it
touches the code just written. Green terminal, green review, untested change.

So the mandate, and it is not optional:

> **A work-unit is not complete until it ships a test that covers the behaviour it added.**

### Who authors

**The developer, at Level-1.** Authoring belongs with the author: they know what the unit was
supposed to do and which seams are worth pinning. The stack-specific craft — the fixture shape, the
harness, the false greens peculiar to that runtime — lives in the unit's **pack**, not in a separate
role.

The **tester does not author**. Its whole value is a fresh context that never inherited the author's
assumptions; giving it the author's job would destroy the independence that makes Level-2 worth
running at all (§5).

### The two thresholds

| Scope | Rule | Kind |
|---|---|---|
| **This unit** | **diff coverage ≥ 90%** — of the lines this change *added or modified* | a gate, always |
| **The project** | coverage **may not decrease**; target 80% | a ratchet |

Two definitions matter more than the numbers.

**Diff coverage, not file or module coverage.** "90% of the touched files" punishes editing one line
in a large legacy file; "90% of the new module" has an arguable boundary. The lines the change wrote
are the lines the change is answerable for.

**The project-wide number is a ratchet, not a door.** A codebase adopting this mid-life sits far
below any meaningful target, so a threshold gate would block every unit on day one and get switched
off within a week. "May not decrease" blocks nothing, converges on the target, and cannot be argued
with. On a project that starts under this policy the ratchet simply never has to climb — the same
rule, no special case.

The per-unit gate is **unconditional**: it applies identically to a new codebase and an old one,
because the diff is newly-written code either way and age is not an excuse for the lines you just
wrote. If it were relaxed for existing projects, it would be absent exactly where it is needed most.

### Coverage proves execution, not verification

A test that calls the code and asserts nothing scores **100%**. Coverage tells you a line *ran*; it
says nothing about whether anything *checked* the result. So the gate is a conjunction:

> **diff coverage ≥ 90%** ∧ **at least one test that goes RED when the change is reverted**

The first half is mechanical and measurable; the second is what makes it mean something. Together
they reject both the untested change and the vacuous test. The second half is cheap to confirm —
revert the change, run the test, see red, restore — and it is the same shape as any assertion worth
trusting: *what would this report if the thing under test simply had not happened?*

### When the project cannot measure at all — a project finding, not a unit block

The rule above has **two independent claims**, and conflating them is what makes it unusable on a
project that has not set up coverage yet:

| Claim | Checkable |
|---|---|
| **a test exists for the behaviour** | always — **this never softens** |
| **diff coverage ≥ 90%** | only when the project can produce a report |

So distinguish two situations that look identical from inside a work-unit:

- **Tooling exists and no measurement is attached** ⇒ the unit **blocks**. You skipped a step.
- **The project has no coverage tooling configured at all** (`atl work coverage` finds no report and
  the pack's coverage command produces none) ⇒ **do not block the unit.** Record it as a
  **project-level finding** — "this project cannot measure diff coverage; set up a coverage runner" —
  and gate the unit on the first claim alone: a test exists, and it goes red when the change is
  reverted.

The reason is not leniency. The missing thing is **project setup**, and setting it up is not this
unit's work — blocking every unit until someone does it punishes each unit for a gap none of them
owns, and a gate that blocks everything on day one gets switched off within a week. **Measured
2026-08-02:** the e2e fixture ships `"test": "node --test"` with no coverage flag at all, and the
rule as first written made every unit in it unpassable.

The important half never bends: **no test, no green**, regardless of whether the project can measure.

### The escape hatch — recorded, never silent

Occasionally 90% is genuinely unreachable: the new line is entangled with legacy code that cannot be
exercised (no seam, a static call, an untestable constructor). That is a real situation and pretending
otherwise just teaches everyone to route around the rule.

The exception is **written on the work-item**: which lines, and why they could not be covered. A
recorded exception is a decision someone can revisit; a silent pass is a hole nobody knows about.

### Measuring it — `atl work coverage`

The number is **produced, never claimed**:

```bash
atl work coverage --json            # the form attached as evidence
atl work coverage                    # human-readable, names the uncovered lines
```

It intersects the lines `<base>...HEAD` added or modified with the lines a coverage report says were
executed, and exits non-zero below the minimum. Three things it does that a hand-written percentage
cannot: the denominator is **measurable** lines only (a comment is not uncovered, it is unmeasurable),
test files and non-source files are excluded, and a changed source file the report never mentions
counts as **zero** rather than being skipped — an untested new file must not score 100%.

It carries its own inputs (base ref, report path) into the output, because a number whose provenance
you cannot see is not reviewable: a stale report scores exactly as green as a fresh one.

Run it **after committing** — `base...HEAD` sees committed work only, so a mid-edit run reports a free
100%. The command says so when the tree is dirty.

### Where this is enforced

- The **developer** authors the test as part of the unit and attaches `atl work coverage --json` —
  the measurement, not a sentence about it.
- The **tech-lead's evidence gate** (`green = (all test-gates passed) ∧ (review passed)`, ordered)
  reads the attached measurement rather than a claim; a passing suite is *not* evidence on its own.
- The **pack** supplies the stack's how — see each pack's `production-unit.md`.

## §8 — Representing the outcome on the board

§1–§7 cover running a test, proving it ran, and requiring one to exist. None of them says where the
**result** lands, and that silence has the same shape as §7's: a gate whose failure has no
representation cannot be seen. A board on which verification passed, failed, or never ran reads
identically in all three cases, and "the definition of done" is then something the board cannot
report on.

The representation is a **flag, never a state** (backend interface concept #17):

| Condition | Carrier | Status |
|---|---|---|
| merged to the integration branch, not yet verified | `test:pending` label/tag | **unchanged** — still In Progress |
| verified, came back red | `test:failed` label/tag + a diagnostic comment | **unchanged** — still In Progress |
| verified, passed | neither — the unit moves to Done and the flag is cleared | Done |

Two properties are load-bearing and both follow from it being a flag. A unit awaiting verification
**stays inside the in-progress count**, so WIP and capacity keep meaning what they meant; and a unit
can be `blocked` *while* carrying a `test:` flag, because the two are different namespaces on a set
rather than two values of one field. `test:pending` and `test:failed` are mutually exclusive with
each other, so a write swaps rather than accumulates.

**`test:failed` without a diagnostic comment is half a signal.** The flag says a verification is
red; what a human cannot reconstruct from it is *which* criterion failed and on which of the three
surfaces (§1). The comment is what turns the flag from the start of an investigation into the end
of one.

### Where in the loop each one arises

The interval these flags describe is the gap between a merge and a human's verdict, so it belongs to
the **hand-driven** drive loop: `/work-finish` opens the PR, a human merges, and `/work-move` flags
`test:pending` until someone verifies. In the **autonomous** loop that gap does not normally exist —
Level-2 verification is micro-loop step 4b, *before* the PR (§5), so a unit that reaches the merge
has already been verified and goes straight to Done. The flags remain available on both paths: a
verification can be found wanting after a merge in either mode, and `test:failed` is how that fact
reaches the board instead of living in someone's memory.

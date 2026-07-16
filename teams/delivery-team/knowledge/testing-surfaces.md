# Testing surfaces — the delivery-team's verification-runtime contract

The single documented contract for **how** a work-unit is verified at runtime — the
counterpart to the [backend interface](backend-interface.md) (the operation contract) and
[`pack-format.md`](pack-format.md) (the stack-knowledge seam). The role-agents describe the
*discipline* of testing — the developer's Level-1 self-test
([`../agents/developer/children/self-test-craft.md`](../agents/developer/children/self-test-craft.md))
and the tester's Level-2 verification
([`../agents/tester/children/mobile-and-web-surfaces.md`](../agents/tester/children/mobile-and-web-surfaces.md));
this doc is the **runtime wiring** they defer to: which surface runs at what concurrency, how the
scarce mobile emulator is shared, and the helper scripts that stand the mechanism up.

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

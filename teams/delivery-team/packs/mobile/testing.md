# Testing — unit, widget, and integration on a booted device

Flutter tests at three widths, and this pack's job is to make the developer's **Level-1
self-test** (micro-loop **step 4**) fast, honest, and consistent with how the **tester's**
Level-2 verification (step 4b) will re-check the same work on the same surfaces. The
developer's self-test is the author's own fast pass; the independent thorough pass is the
tester's, whose surface discipline is authoritative in
[`agents/tester/children/mobile-and-web-surfaces.md`](../../agents/tester/children/mobile-and-web-surfaces.md).
This topic carries the **how-to-boot / how-to-drive the emulator** knowledge both rely on —
the *runtime* lease and its wiring are shipped in
[`knowledge/testing-surfaces.md`](../../knowledge/testing-surfaces.md) §3 + the emulator-lease /
preflight scripts: this file describes how a worker *uses* the mobile surface, driving those.

## The three widths — most weight at the bottom

- **Unit tests (`flutter test`, `test/`)** — pure Dart: the provider/notifier logic, models,
  validators, pure functions. No widget, no device. Reason: this is where the *rules* an
  acceptance criterion encodes actually live ([`widget-and-state.md`](widget-and-state.md)),
  and these run **fast and in parallel** with no shared resource — so the bulk of coverage
  belongs here, proven cheaply.
- **Widget tests (`flutter test`, `test/`)** — a widget rendered in a headless test harness
  (`pumpWidget`): tap, enter text, `pump`, assert on the tree. Still no real device. Reason:
  they verify a screen renders the right thing for a given state (loading/data/empty/error)
  and reacts to input, at unit-test speed and concurrency — most UI behavior is provable here
  without the scarce emulator.
- **Integration tests (`flutter test integration_test`, `integration_test/`)** — the whole
  app driven on a **real booted emulator/simulator**. Reason: some criteria only manifest on a
  device — a navigation flow, a platform-channel round-trip, a genuinely end-to-end screen
  interaction. This is the constrained surface, and the discipline below governs it.

The rule that falls out (the pyramid with concurrency teeth): **push logic-probing to the
parallel unit/widget surfaces; reserve the serialized device for criteria that genuinely need
a real device.** Reason: every minute on the emulator is a minute no other work-unit's mobile
check can run, so proving a validation rule *through* the device instead of in a unit test
throttles the whole team.

## The mobile surface is a single-slot serialized lease

The emulator/simulator is a **single-slot lease**: exactly one work-unit holds the device at a
time; everyone else **waits**, and that wait is **by design**, not a failure. Reason: an
emulator is a heavyweight, stateful, singleton-ish resource — it boots slowly and holds GUI
state, and two workers driving one device at once would interleave taps and reads and corrupt
each other's run (a shared-mutable-resource race). Serializing is how the team avoids that
race. Code and widget tests have no such singleton, so they run at full `atl work dispatch`
concurrency; only the device serializes. This mirrors the tester's surface contract exactly —
the developer's self-test and the tester's verification obey the *same* lease so a green means
the same thing on both passes.

**Do not dodge the wait by booting a second emulator** — that reintroduces the very race the
single slot exists to prevent. The lease *runtime* (how the slot is acquired/released across
parallel workers) is `scripts/emulator-lease.sh` (knowledge/testing-surfaces.md §3); here the
developer's obligation is to *drive* it correctly:
acquire, use, release promptly (win or lose — holding the slot after finishing starves the
next unit for no reason), and never assume a second device is available.

## Preflight bootability — before trusting any device result

Before running a single integration check, confirm the emulator/simulator **actually booted
and is responsive**. Reason: an emulator that failed to boot or hung is the sharp edge — iOS
simulators in particular have real boot latency and a GUI requirement, so "the device isn't
up" is a live, common failure mode, not a corner case. A result read off a device that never
came up is meaningless.

Concretely, the preflight is: start/target a device, **wait for it to reach a booted,
responsive state, and only then drive it** — and give the boot a bounded retry budget (a
second boot attempt before calling it a failure is a common, worth-it allowance) rather than a
single shot. The *mechanics* of launching and detecting the boot are provided by
`scripts/emulator-preflight.sh` (knowledge/testing-surfaces.md §3 — the bounded-poll boot gate); the
developer's discipline is to **gate on booted-and-responsive**, never to run against an unconfirmed
device.

## Block, never silently pass — the cardinal rule

The most dangerous failure on the mobile surface is **treating "couldn't run" as "passed."**
If the emulator won't boot, hangs, or the lease can't be acquired within a sane bound, the
mobile criterion is **unverified** — and unverified is **not** a green:

- **Surface it.** The self-test result says the mobile gate could not run, *why* (boot failure
  / lease timeout / device unresponsive), and *which* criterion it leaves unproven — so the
  tester and tech-lead see an honest "unverified", not a false pass.
- **Never emit a pass for a surface that didn't execute.** A silent pass on an un-run mobile
  gate is the worst outcome: it lets unverified mobile behavior ride toward merge under a false
  green, and the whole point of the gate is that `green = test-gates ∧ review` is trustworthy.
- **Never substitute logic for the device.** Passing a screen/interaction criterion on unit
  tests alone, because the emulator was unavailable, is the same false green in disguise — the
  criterion was *about* the device.

Reason this is load-bearing here as much as for the tester: the emulator is the surface **most
likely to fail to run** and **least likely for a reader to notice went un-run** — a green looks
identical whether the device ran or silently didn't. So the honesty has to come from the
worker at the surface, deterministically: if it didn't boot, it didn't pass.

## Evidence — screenshots via the REST carve-out

A device result is only trustworthy if it is **inspectable**, so capture evidence and attach it
to the work-item:

- **Capture** the test-run output (all gates green — or the failure) and a decisive
  **screenshot per non-trivial mobile criterion** (the success screen, and for a
  rejection/boundary criterion the *guard holding* — the rejection itself, not the happy path).
  Reason: a screenshot with no indication of input or a bare "pass" comment proves nothing a
  reviewer can act on.
- **Attach** each via the **one REST carve-out** — `scripts/az-attach.sh` (adapter §9), the
  single Azure operation with no MCP tool. From this pack's location the helper is
  [`../../scripts/az-attach.sh`](../../scripts/az-attach.sh):

  ```
  ../../scripts/az-attach.sh <work-item-id> <file> [comment]
  ```

  It uploads the bytes then links them to the work-item, running the worker's **env PAT**
  (Basic auth header, never on the argv, never logged — the adapter's secret hygiene). Reason:
  keeping every other Azure call on the MCP and this one behind a uniform `attach(work-item,
  file)` helper keeps the transport split hidden and the Go orchestrator zero-Azure.
- **Name evidence for what it proves and tie the comment to the criterion** — e.g.
  `../../scripts/az-attach.sh 4217 transfer-success.png "AC-1: valid transfer succeeds on
  device"`. Reason: at review or sprint-review the attachment must be legible on its own,
  mapped to a criterion. This is the same evidence contract the tester documents in
  [`agents/tester/children/evidence-collection.md`](../../agents/tester/children/evidence-collection.md);
  the developer's self-test evidence feeds directly into that shared trust.

## Test-command reference (this stack)

- **Unit + widget:** `flutter test`
- **Static analysis gate:** `dart analyze` (or `flutter analyze`) + `dart format --set-exit-if-changed .` — reason: the `flutter_lints` ruleset is what makes the const / no-side-effects-in-build conventions machine-enforced, and CI counts a lint/format failure as a gate failure.
- **Mobile surface (booted device, single-slot lease):** `flutter test integration_test` — run **only** against a device the preflight confirmed booted and responsive; treat a lease-timeout or boot-failure as unverified, never as a green.

## Worked example (generic)

A `area:mobile` work-unit adds a transfer screen with criteria: (a) a valid transfer succeeds
on the phone, (b) an over-balance transfer is rejected with a visible error. The developer's
Level-1 surface plan:

- The transfer *rules* (sufficient balance, amount validation) → **unit tests** against the
  provider, and the screen's per-state rendering → **widget tests** — both parallel, at the
  bottom of the pyramid; the emulator slot is not burned on rule permutations.
- Criteria (a) and (b) *on the actual screen* → **`flutter test integration_test`**: acquire
  the lease, **preflight** the device is booted and responsive, navigate to the screen by its
  named route ([`widget-and-state.md`](widget-and-state.md)), drive both flows, screenshot each
  (the success screen for (a); the visible rejection for (b) — the boundary evidence), attach
  via `../../scripts/az-attach.sh`, then release the lease.
- If the device won't boot after the preflight retry budget → the developer does **not** pass
  (a)/(b) on logic alone. The self-test surfaces "mobile gate did not run — device failed
  preflight; criteria (a),(b) unverified", and the work-unit is not green on the mobile half.

The screen's *actual* flows and route are **project facts** (read from the work-item's
`## Acceptance Criteria` and the `Conventions/` wiki page named in the brief); this file
teaches only the generic testing craft. A durable role-craft lesson learned here (e.g. "always
give the emulator preflight a second boot attempt before calling it a fail") generalizes back
into the developer's own `children/` via `/drain` — project facts never do (adapter §8).

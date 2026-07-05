---
area: mobile
stack: "Flutter + Dart"
---

# Mobile pack — Flutter + Dart

This pack covers work whose acceptance criteria live on a **mobile app screen** — a
Flutter UI, its state, its navigation, and the platform integrations behind it. When the
tech-lead tags a work-unit `area:mobile` on `System.Tags` (adapter §7), the generic
`developer` loads **only** this pack and works the task in Flutter/Dart. It is the
delivery-team's **reference** mobile pack: a real, minimal template a software team copies
and repoints at its own mobile stack, and the e2e fixture that exercises the M1 seam. See
[`knowledge/pack-format.md`](../../knowledge/pack-format.md) for how a pack binds to an
area and reflects into `.claude/packs/`.

Mobile is the design's **emphasized, riskiest surface**: its Level-2 verification runs on a
real **booted emulator/simulator** behind a single-slot lease, so the pack carries the
how-to-boot / how-to-drive knowledge the developer's self-test and the tester both rely on
(see [`testing.md`](testing.md)). Everything here is generic **stack** craft; the *project's*
own conventions layer atop it in the Azure project wiki `Conventions/` (adapter §8), named
in the tech-lead's canonical brief.

## Topics

- [widget-and-state.md](widget-and-state.md) — widget structure, state management (Riverpod / Provider), and navigation.
- [project-and-deps.md](project-and-deps.md) — Flutter/Dart project layout, `pub` dependencies, and platform channels.
- [testing.md](testing.md) — unit + widget tests, `integration_test` on a booted emulator/simulator, and the lease / evidence discipline.

## Test commands

- unit / widget: `flutter test`
- static analysis (gate): `dart analyze` (or `flutter analyze`) and `dart format --set-exit-if-changed .`
- mobile surface (booted emulator/simulator, single-slot lease): `flutter test integration_test` — run against a device confirmed booted by the preflight; never trust its result without that preflight (see [`testing.md`](testing.md)).

## Key conventions

- **Widgets are pure functions of state.** A `build` reads state and returns a tree; it never mutates state or performs side effects. Reason: rebuild is frequent and can happen at any time, so a `build` with side effects fires them unpredictably.
- **One documented state approach per project, immutable state.** Pick the project's declared solution (Riverpod or Provider here) and hold state as immutable values, replacing rather than mutating. Reason: mutation-in-place defeats the framework's change detection, so the UI silently goes stale.
- **`const` every constructor that can be.** A `const` widget subtree is skipped on rebuild. Reason: it is the cheapest, most mechanical rebuild-cost win, and the analyzer flags the misses.
- **Never block the UI isolate.** Keep heavy CPU work off the main isolate (`compute` / an isolate); `await` I/O, don't spin. Reason: Flutter renders on one isolate at 60/120fps — a synchronous stall is a visible jank or freeze.
- **A mobile acceptance criterion is proven on a booted device, never in logic alone.** A screen/interaction/navigation criterion is verified through `integration_test` on a preflighted emulator; if the device did not boot, the criterion is **unverified**, not passed (block-never-silently-pass, mirrors the tester). Reason: logic tests can't observe the rendered device behavior the criterion is actually about.

## Dependency baseline

A team **pins** these in `pubspec.yaml` — treat the versions as a sane current-ish baseline to pin at, not gospel; the tech-lead's `Conventions/` may override:

- **Flutter 3.x SDK / Dart 3.x** — the toolchain (`environment: sdk: '^3.x'`); Dart 3 sound null-safety is assumed throughout.
- **flutter_riverpod ^2** — the reference state solution (a project may instead declare `provider ^6`; keep to one). Role: testable, compile-safe state that survives rebuilds.
- **go_router ^14** — declarative, URL-addressable navigation. Role: one routing table the whole app and its tests address by name.
- **flutter_test + integration_test (SDK)** — the test surfaces this pack drives; shipped with the Flutter SDK, not pinned separately.
- **flutter_lints ^4** (dev) — the analyzer ruleset behind the `dart analyze` gate. Role: makes the "const / no side effects in build" conventions machine-enforced, not just prose.

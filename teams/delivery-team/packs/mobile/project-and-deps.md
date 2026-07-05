# Project layout, dependencies, and platform channels

A Flutter app has a conventional shape — a `pubspec.yaml` at the root, `lib/` for Dart
source, per-platform host folders, and a `test/` tree — and knowing it lets a fresh
worker orient in seconds and put new code where the next reader expects it. This topic is
the generic Flutter/Dart project craft: where things go, how dependencies are declared and
pinned, and how to reach native platform code when Dart alone can't. The *project's* actual
module boundaries live in the wiki `Architecture/` (adapter §8), named in the brief — this
file teaches the conventions those boundaries sit inside.

## Project layout — where code goes and why

- **`pubspec.yaml` is the single manifest** — SDK constraint, dependencies, dev-dependencies,
  and asset declarations all live here. Reason: it is the one file `flutter pub get` reads to
  resolve the world, so anything a build needs must be declared in it, not assumed present.
- **`lib/` is all Dart source; `lib/main.dart` is the entrypoint.** Reason: Flutter's tooling
  treats `lib/` as the package root — an import path is relative to it — so code outside `lib/`
  is not part of the app.
- **Organize `lib/` by feature, not by layer, once the app is more than trivial.** Group a
  feature's screen, its state (provider/notifier), and its models together
  (`lib/features/transfer/…`), rather than one flat `screens/`, one flat `providers/`. Reason:
  a work-unit almost always touches one feature end-to-end, so feature-grouping keeps a change
  local and the blast radius small; layer-grouping scatters one feature's change across the
  tree. The project may declare its own structure in `Conventions/` — follow that when it
  does; this is the sane default when it doesn't.
- **Host platform folders (`android/`, `ios/`, and web/desktop as configured) hold the native
  shell** — Gradle/Xcode config, permissions, signing, the platform-channel host code.
  Reason: this is where a mobile app declares OS-level capabilities (a permission, a native
  plugin) that pure Dart cannot; a criterion needing a device capability usually touches here.
- **Tests live under `test/` (unit + widget) and `integration_test/` (device-driven).** Reason:
  Flutter's test runners key off these locations — `flutter test` finds `test/`, and the
  integration harness finds `integration_test/` (see [`testing.md`](testing.md)). Putting a
  test elsewhere means it does not run.
- **Generated files are generated, never hand-edited.** Anything produced by `build_runner`,
  l10n, or the platform tooling (`.g.dart`, `.freezed.dart`, generated l10n) is rebuilt from
  its source. Reason: a hand-edit is silently overwritten on the next generate, so the fix
  must go in the *source* the generator reads, and the generate step re-run.

## Dependencies — declare, pin, and keep few

- **Every dependency is declared in `pubspec.yaml` and resolved with `flutter pub get`;
  `pubspec.lock` records the exact resolved versions.** Reason: the lock file is what makes a
  build reproducible across the worker's worktree and CI — commit it for an app so every
  build resolves identically (see the [pack manifest](pack.md) dependency baseline).
- **Pin to the project's declared baseline; do not silently bump a major.** Use the versions the
  wiki `Conventions/` (or the pack baseline) names, with caret ranges (`^2.0.0`) for compatible
  updates. Reason: a major bump can change an API the whole app depends on — that is an
  architecture decision for the tech-lead (`Architecture/ADR/…`, adapter §8), not a quiet
  side effect of adding a feature. If a task genuinely needs a new or bumped dependency, the
  developer **surfaces it** to the tech-lead rather than deciding it alone.
- **Prefer few, well-maintained packages over many.** Each dependency is surface area to keep
  current and a potential platform-channel liability. Reason: a lean `pubspec` is faster to
  resolve, easier to keep secure, and less likely to break on a Flutter SDK bump — reuse a
  package the project already pins before adding a new one.
- **Separate `dev_dependencies` (test/lint/codegen) from `dependencies` (shipped).** Reason:
  the dev tools (`flutter_lints`, test helpers, `build_runner`) must not bloat or ship in the
  release binary; declaring them under `dev_dependencies` keeps them out of it.

## Platform channels — reaching native code, and when not to

Most features are pure Dart. A **platform channel** is Flutter's bridge to native
Android/iOS code (Kotlin/Swift), for a capability the Dart layer or an existing plugin
cannot provide (a bespoke native SDK, a hardware feature with no package).

- **Reach for a channel only when no maintained plugin covers the need.** Reason: a channel
  means writing and maintaining native code on *both* platforms and keeping the Dart/native
  contract in sync — real cost. A `pub.dev` plugin that already wraps the capability is almost
  always the right answer; a channel is the last resort, and adding one is an architecture
  decision to surface to the tech-lead.
- **A channel is an async message contract: a named channel + method calls with a documented
  argument/result shape.** Reason: the Dart side and each native side must agree exactly on the
  channel name, method names, and serialized types — a mismatch fails at runtime, not compile
  time, so the contract has to be explicit and kept identical on both ends.
- **Channel-crossing code is the least testable code in the app — isolate it behind a Dart
  interface.** Put the channel call behind a plain Dart abstraction the rest of the app depends
  on. Reason: the native side can't run in `flutter test` (no device), so the Dart interface is
  what unit/widget tests fake, and the real channel is only exercised on the booted device in
  `integration_test` ([`testing.md`](testing.md)). Without the seam, every test that touches
  the feature is forced onto the scarce emulator lease.
- **Both native ends must actually implement the method.** A channel that a work-unit adds must
  be handled in `android/` *and* `ios/` (and any other target platform the project ships).
  Reason: an unimplemented method throws a `MissingPluginException` only on the platform that
  lacks it — a criterion can pass on one platform's emulator and fail on the other, which is
  exactly why preflighting the *right* device matters (the tester's surface discipline).

## What travels vs what's project-specific

The *conventions* here — feature-grouped `lib/`, pin-don't-bump, channels-behind-a-Dart-seam
— are generic Flutter **stack** craft and travel with this pack. A durable *role-craft* lesson
a worker learns (e.g. "always isolate a platform channel behind a Dart interface so tests
don't need the emulator") generalizes back into the developer's own `children/` via `/drain`.
The project's **actual** module boundaries, its declared package set, and which native
capabilities it uses are **project facts** in the Azure wiki `Architecture/`/`Conventions/`
(adapter §8) — read at runtime from the pages the tech-lead's brief names, never pre-baked
into this pack.

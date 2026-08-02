# Production unit — adding a screen

The thing this pack builds over and over is a **screen**: a destination the user navigates to, the
provider/notifier holding its rules, its entry in the app's route table, and the tests that pin
each. This page is the lifecycle for one, start to finish. A reusable **widget** is the smaller
variant of the same unit — it follows every step except routing, and "registered" for it means
*composed into a screen that a route reaches*.

Two of its steps exist for one reason, worth stating up front. The characteristic failure of a
scaffolded screen is **not** a bad body — it is a **correct body that nothing calls**. A screen
absent from the route table, or reached by a route name that was mistyped at the call site,
analyzes clean, renders perfectly in its widget test, and is unreachable from the running app.

This pack is better placed than most to catch that, and step 4 says exactly how far that
protection reaches — and where it stops.

## 1. Decide before you write

Settle these first — renaming later means touching the screen file, the route table, every caller,
the tests, and any durable-knowledge page that names the route.

| Question | How to answer it |
|---|---|
| **A screen, or a widget on an existing screen?** | A widget if it is a piece of a tree the user already reaches; a screen if it is a *destination*. A screen is a permanent route surface — the app's URL space, and what deep links address. |
| **Which feature folder?** | The feature it belongs to, with its state and models beside it (`lib/features/<feature>/…`), not a flat `screens/`. See [project-and-deps.md](project-and-deps.md). |
| **Route path, and route *name*?** | Settle both now. The name is the string the route table, every caller, and the integration test all use; it fails only at run time when they disagree — so it becomes one constant (step 2), never three literals. |
| **What does the route carry?** | An identifier, never a hydrated object — the destination fetches its own data. A cold-started deep link has no in-memory object to hand over ([widget-and-state.md](widget-and-state.md)). |
| **Where do the rules live?** | Name the provider/notifier that holds them — new or existing. Logic in the widget is logic no unit test can reach. |
| **Which UI states can it be in?** | Enumerate loading / data / empty / error now. They are the widget-test cases, and they are usually where the acceptance criteria actually are. |
| **Which criteria genuinely need the device?** | Decide the surface split before writing tests: rules and per-state rendering go to the parallel surfaces; only real-device criteria burn the single-slot lease ([testing.md](testing.md)). |

## 2. Scaffold

```dart
class ThingScreen extends ConsumerWidget {
  const ThingScreen({super.key, required this.thingId});

  /// The route's name — ONE constant, referenced by the route table, every
  /// caller, and the integration test. Three literals drift, and the drift
  /// is a run-time-only failure.
  static const routeName = 'thing';

  final String thingId; // an identifier, not a hydrated object

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(thingProvider(thingId)); // watch → rebuild on change
    return Scaffold(
      appBar: AppBar(title: const Text('Thing')),
      body: state.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ThingError(message: '$e'),
        data: (items) => items.isEmpty
            ? const _ThingEmpty()
            : _ThingBody(
                items: items,
                // read for a one-off action; the RULES live in the notifier
                onSubmit: () => ref.read(thingProvider(thingId).notifier).submit(),
              ),
      ),
    );
  }
}
```

All four states rendered, `const` wherever it can be, subtrees extracted the moment they have an
identity, no side effect and no navigation inside `build`. If the project declared **Provider**
instead of Riverpod, the shape is the same with a `ChangeNotifier` read from `context` — keep to
whichever the project declared, never both. See [widget-and-state.md](widget-and-state.md).

## 3. Register it — the step that is easy to skip and invisible when skipped

A screen only exists once the **route table** points at it. Add it to the app's single declared
table, beside its siblings:

```dart
GoRoute(
  path: '/thing/:id',
  name: ThingScreen.routeName, // the constant, never a repeated literal
  builder: (context, state) =>
      ThingScreen(thingId: state.pathParameters['id']!),
),
```

Read the neighbouring entries before writing yours and **match them**. If the siblings sit inside a
`ShellRoute` (a shared nav bar, a scaffold, an auth redirect), yours belongs there too — a route
registered one level out renders without the chrome and outside the guard, and looks entirely
correct in isolation. A registration that looks unlike its neighbours is the one a reviewer skims
past.

**The entry point is part of registration.** A route nothing links to is reachable by a test and by
nobody else. Add the affordance that gets a user there, from an event handler — never from `build`,
which would re-navigate on every rebuild:

```dart
onPressed: () => context.goNamed(
  ThingScreen.routeName,
  pathParameters: {'id': thing.id},
),
```

**State wiring is the second half.** With Riverpod the providers are top-level globals, so a screen
needs no per-provider registration — but the app root must be wrapped in `ProviderScope`, and a
subtree pushed outside one throws at run time. **On a Provider project the registration is not
optional:** the notifier must be provided *above* the screen in the tree, or the screen throws the
first time a user opens it — a run-time failure `dart analyze` cannot see.

## 4. Verify the durable effect — enter the screen the way the app enters it

Unlike a pure code-surface stack, this pack's prescribed device pass **can** be the composed-root
drive. Say plainly which test provides it: **`flutter test integration_test`, navigating to the
screen by its named route** — the shape [testing.md](testing.md) prescribes in its worked example.
Pumped against the app's *real* root, that test walks the actual route table: a screen that is
absent from it, or a route name that drifted from the constant, fails the navigation there. That is
a genuine guarantee, and it is the one this unit type depends on.

It holds under exactly two conditions, and both are on you:

```bash
flutter test                                    # unit + widget — parallel, no device
dart analyze && dart format --set-exit-if-changed .
flutter test integration_test                   # after the preflight confirms a booted device
```

- **The device pass actually ran.** A lease timeout or a failed boot leaves the criterion
  **unverified, not passed** — the cardinal rule in [testing.md](testing.md). The registration
  guarantee is only as real as the run that produced it.
- **The test entered through the app's own route table** — the one the real root is built from, not
  one the test declared. This is the condition worth stating out loud, because nothing enforces it.

**The false green to name out loud:** an `integration_test` that reaches the screen without going
through the app's own table — pumping `MaterialApp(home: ThingScreen(...))`, hand-pushing it with
`Navigator.push`, or navigating **by name against a router the test itself declares** — **runs on a
real booted device, produces a real screenshot, and passes with the screen absent from the app's
route table.** That third shape is the one to watch: it obeys "navigate by its named route" to the
letter, so it survives a careful worker and a careful reviewer, and it still proves nothing — the
table it walked is the test's, not the app's. In every shape the device ran, so the
block-never-silently-pass rule never fires; the evidence is a photograph of a working screen on a
real phone and it says nothing about reachability. The whole parallel half of the gate is green
regardless: a widget test constructs the screen directly by design, and an unreferenced public
screen class is not an analyzer finding — so `flutter test` and `dart analyze` are structurally
incapable of seeing the omission. And the pyramid correctly pushes most evidence onto exactly those
blind surfaces. **The guarantee is not "the integration test ran", and not even "it navigated by
name" — it is "it entered through the table the real app is built from."**

So observe two things, and record which is which:

- **Reachability** — the navigation step itself, by route name, against the app's real root. This is
  what proves registration.
- **The tap path** — at least once, reach the screen the way a user does: start where a user starts
  and tap the affordance from step 3. A route-registered screen whose entry point was never added is
  route-reachable and user-unreachable, and navigating directly by name is precisely what hides it.

Then capture the evidence [testing.md](testing.md) requires — the run output plus a decisive
screenshot per non-trivial criterion (the success screen, and for a boundary criterion the *guard
holding*) — and attach it to the work-item through that topic's attach helper. Name each attachment
for what it proves and tie it to the criterion. A screenshot proves the screen renders; only the
navigation proves it exists to the app.

## 4b. Ship the test — the unit is not done without one

Step 4 proved the thing works **now**, by looking at it. This step is what keeps it working: a unit
is not complete until it ships a test covering the behaviour it added. The policy, identical across
packs, is [`testing-surfaces.md` §7](../../knowledge/testing-surfaces.md):

> **diff coverage >= 90%** of the lines this unit added or modified, **and** at least one test that
> goes RED when the change is reverted.

The second half is not ceremony. Coverage proves a line *ran*; it says nothing about whether anything
*checked* the result — a test that exercises the code and asserts nothing scores 100%. Confirming it
costs a minute: revert the change, run the test, see red, restore.

Coverage for this stack:

```bash
flutter test --coverage
```

**The false green peculiar to this stack:** a widget test that constructs the screen directly instead
of entering it the way the app does. It renders, it asserts, it passes — while the screen is absent
from the app's real route table and no user can reach it. That is the same gap step 3's registration
check exists for, and a direct-construction test hides it rather than catching it.

If 90% is genuinely unreachable because the new code is entangled with something untestable, record
the exception on the work-item — which lines, and why. Never a silent pass.

## 5. Common pitfalls

- **The route name as a bare literal at each call site.** A typo compiles, analyzes clean, and fails
  only when a user taps. One `static const` on the screen, referenced everywhere.
- **Registered on the wrong branch of the table.** Outside the `ShellRoute` the screen loses the
  shared chrome and any redirect guard — and looks perfect in every widget test.
- **Hand-pushed instead of routed.** `Navigator.push` from a caller works on the device, never
  enters the table, is not deep-linkable, and cannot be reached by name from a test — the exact
  scatter [widget-and-state.md](widget-and-state.md) rejects.
- **A hydrated object passed through the route.** It works from inside the app and breaks the
  cold-start deep link, which is usually a stated criterion.
- **Navigation triggered from `build`.** A rebuild re-navigates and stacks duplicate screens.
- **Only the happy state built.** Loading / empty / error are where the criteria live; a screen that
  renders only `data` passes its one widget test and shows a blank on the common paths.
- **Rule permutations proven on the device.** Every minute of the single-slot lease is a minute no
  other unit can run its mobile check — prove rules in unit tests, and release the lease promptly
  win or lose ([testing.md](testing.md)).
- **A new dependency added quietly.** Adding or bumping a package is a tech-lead decision to
  surface, not a side effect of a screen ([project-and-deps.md](project-and-deps.md)).

## 6. Hand-off

The work-item is ready for the tester when: `flutter test`, `dart analyze` and
`dart format --set-exit-if-changed .` are green; the **integration-test run that entered by named
route** is attached, with its output; the screenshots are attached, each named for the criterion it
proves; the tap path has been walked once; and any new route or user-visible flow the project
documents is recorded where the brief says.

State plainly in the hand-off which of these you **observed** and which you **assumed** — in
particular, say how the integration test reached the screen, because that single fact is what
separates a proof of reachability from a screenshot of an unreachable screen. If the device did not
boot within the preflight's retry budget, say so and name the criteria it leaves unproven: an
admitted gap is worth more than an unverified claim, and the tester's Level-2 pass — which gates the
green — can only check what it is told about.

# Widget structure, state, and navigation

Flutter builds a UI by composing **widgets** — small, immutable descriptions of a piece of
the interface — into a tree that the framework rebuilds whenever the underlying state
changes. Getting the three joints right — how a widget is structured, how state flows into
it, and how the user moves between screens — is most of what "build this mobile screen"
means in practice. This topic is the generic Flutter craft for those joints; the *project's*
specific screens, providers, and routes are project facts the developer reads from the
work-item and the wiki `Conventions/` (adapter §8), never pre-authored here.

## Widget structure — composition over configuration

- **Build small, composable widgets, not one giant `build`.** Extract a subtree into its own
  widget the moment it has an identity ("the balance card", "the transfer form"). Reason: a
  200-line `build` rebuilds entirely on any change and is untestable in isolation; a small
  widget rebuilds only when *its* inputs change and can be widget-tested on its own.
- **Prefer `StatelessWidget`; reach for `StatefulWidget` only for truly local, ephemeral
  state** (an animation controller, a text field's focus, a scroll position). Reason: app
  state that outlives a single widget belongs in the state layer below, not trapped in a
  `State` object where nothing else can read or test it. The test: if another widget or a
  test needs the value, it is not local state.
- **A `build` method is a pure function of inputs.** It reads configuration and state and
  returns a tree; it must not mutate state, start requests, or cause side effects. Reason:
  the framework calls `build` often and unpredictably (a parent rebuild, a media-query
  change, a hot reload) — a side effect there fires at times you did not intend, producing
  duplicate requests and races that are miserable to debug.
- **Mark everything `const` that can be.** A `const` constructor lets Flutter skip rebuilding
  that subtree entirely. Reason: it is the single cheapest performance win in Flutter and the
  analyzer (`flutter_lints`, see [`project-and-deps.md`](project-and-deps.md)) flags the
  misses, so it costs nothing to hold the convention.
- **Handle every UI state a screen can be in — loading, data, empty, error.** A screen fed by
  async data has at least four visual states; render each explicitly. Reason: the default
  "assume data is here" path shows a blank or crashes on the (common) not-yet / failed cases,
  and those are exactly the acceptance criteria a mobile work-unit tends to carry.

## State management — one approach, immutable values

The state layer is where the acceptance criteria actually live (a balance updates, a form
validates, a list filters). Two rules dominate, and both exist to keep the framework's change
detection working:

- **Pick the project's *one* declared solution and stay in it.** This pack's reference is
  **Riverpod** (a project may instead declare **Provider** — keep to whichever the wiki
  `Conventions/` names; do not mix two in one codebase). Reason: two overlapping state
  systems means two sources of truth for "is this fresh", and the UI desyncs at their seam.
  The developer reads which one this project uses from `Conventions/`; the *craft* of using
  it well is here.
- **Hold state as immutable values and replace, never mutate.** Emit a *new* state object on
  every change rather than mutating fields on the existing one. Reason: Flutter's rebuild is
  triggered by a *change of reference/notification*; mutating in place leaves the reference
  equal, the framework sees "nothing changed", and the UI silently shows stale data — the
  single most common Flutter state bug.

**Riverpod, concretely (the reference):**

- A **provider** exposes a piece of state or a derived value; a widget *watches* it and
  rebuilds when it changes, or *reads* it once for a one-off action. Reason: watch-to-rebuild
  is the whole contract — a widget declares its dependency and the framework recomputes only
  the watchers of what changed, not the whole tree.
- **Business logic lives in a notifier/provider, not in the widget.** The widget renders state
  and forwards intents ("transfer pressed"); the provider holds the rules and produces the
  next state. Reason: this is what makes the logic **unit-testable without a widget** (the
  parallel, cheap surface — see [`testing.md`](testing.md)) and keeps `build` pure.
- **Model async as async state, not a bare value + a loose boolean.** Represent
  "loading / data / error" as one async value the UI switches on. Reason: a separate
  `isLoading` bool and a nullable value drift out of sync (loading-but-has-old-data,
  error-but-bool-still-true); one async value makes the four states exhaustive and total.

**If the project declares Provider instead:** the shape is the same — a `ChangeNotifier` (or
similar) holds state and calls `notifyListeners()` on change; widgets read it via `context`
and rebuild. The immutable-replace discipline still applies; `notifyListeners()` is the
explicit "reference changed" signal Riverpod gives you implicitly. The developer follows
whichever the project declared — this pack just refuses the *mix*.

## Navigation — declarative and addressable

- **Use the project's declared router (reference: `go_router`) with a named/typed route
  table**; do not hand-push routes ad hoc across the app. Reason: a single declarative table
  is inspectable, deep-linkable, and — critically for this team — **addressable by
  integration tests**, which navigate by route to reach the screen under a mobile acceptance
  criterion (see [`testing.md`](testing.md)). Scattered imperative pushes can't be reached
  the same way.
- **Pass identifiers through the route, fetch data at the destination.** Route with an id/key,
  and let the destination screen load its own data from the state layer. Reason: passing
  whole hydrated objects through navigation couples screens and breaks deep-linking (a
  cold-started deep link has no in-memory object to hand over), so the criterion "open this
  screen from a link" fails.
- **Keep navigation out of `build`.** Trigger navigation from an event handler (a tap, a state
  transition), never as a side effect of rendering. Reason: same as the pure-`build` rule — a
  rebuild would re-navigate, pushing duplicate screens.

## Worked example (generic)

A `area:mobile` work-unit adds a "transfer between accounts" screen. The generic Flutter
shape the developer follows:

- A `TransferScreen` **`StatelessWidget`** that *watches* a `transferProvider` for its state
  (idle / submitting / success / rejected) and renders the matching UI for each — no bare
  "assume success" path.
- The transfer **rules live in the provider/notifier** (sufficient-balance check, amount
  validation), emitting a *new* immutable state on each transition — so the balance and the
  rejection are driven by state the widget only reads.
- The screen is reached via a **named route** (`go_router`), so the integration test can
  navigate straight to it, drive a valid and an over-balance transfer, and screenshot each
  outcome on the booted device ([`testing.md`](testing.md)).

The screen's *actual* fields, provider names, and route are **project facts** — read from the
work-item's `## Acceptance Criteria` and the `Conventions/` wiki page named in the brief. This
file only teaches the shape; a durable *craft* lesson learned here (e.g. "always model async
UI as one async value, never a value-plus-bool") generalizes back into the developer's own
`children/` via `/drain`, project facts never do (adapter §8).

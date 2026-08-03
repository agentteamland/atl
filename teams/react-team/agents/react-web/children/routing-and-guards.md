---
knowledge-base-summary: "The route table as the composition root for a screen, lazy route loading and the browser-only failure it produces, and nested layout routes. Auth guards gate on **hydration**, not just on a token: an asynchronously-rehydrated store can leave a valid token beside a null user, so the pre-hydration window renders a splash and never a login — rendering login \"just while we wait\" reproduces the exact flash the gate exists to remove. When two components navigate on the same state flip, the decision belongs in the survivor, because the one that unmounts the other wins; and a post-login return path is accepted only as an internal path, or it becomes an open redirect. A UI route guard is a UX layer and never an authorization boundary."
---

# Routing And Guards

## The route table is a screen's composition root

A page component that no route names does not exist. It compiles, it type-checks, its colocated test
renders it green — and the running app has no way to reach it. Registration in the route table is the
step that turns a written page into a screen, and it is invisible when skipped
([screen-blueprint.md](screen-blueprint.md)).

Keep the table in one file so the app's whole surface is readable in one place, and nest routes to
match the URL structure. **Layout routes** wrap children in persistent chrome (sidebar, header) that
survives navigation instead of remounting on every transition — which is both the point and the
reason to reach for one rather than repeating the shell inside each page.

Route params arrive as `string | undefined` no matter what the path pattern says. Validate at the top
of the page; do not thread an unchecked param into a query key.

## Lazy route loading — and its browser-only failure

Split at the **route** level, not per component. Per-component lazy loading multiplies chunks and
network requests for no benefit; per-route splitting is what actually keeps the initial bundle small.
Where pages use named exports, remap in the import, since the lazy API wants a default export:

```tsx
const UsersPage = lazy(() =>
  import('@/features/users/pages/UsersPage').then((m) => ({ default: m.UsersPage })),
);
```

**A broken route chunk surfaces only in a browser.** The symptom is `Failed to fetch dynamically
imported module` in the console while every API probe against the same deploy returns 200 — backend
evidence is evidence about the backend, and it will tell you nothing about this. Two consequences for
how you verify a routing change:

- Open the route in a real browser, and read the console for a chunk-load failure and the network
  panel for a chunk 404.
- **Navigate by clicking the menu, not only by typing the URL.** Clicking is the lazy-import path,
  and it is the path a direct load can bypass — so it is the path the reported failure took.

Full discipline in [browser-verification.md](browser-verification.md).

## The guard gates on hydration, not on a token

This is the trap on this stack that costs the most time, because the code reads as correct and the
failure is intermittent.

An "is authenticated" predicate typically needs more than one value — a token **and** a user, often a
selected profile or role too. Those values do not arrive together:

- The **token** is usually read **synchronously** from storage at store creation, deliberately, so
  the HTTP client's interceptor can read it before anything has hydrated
  ([api-client.md](api-client.md)).
- The **user** comes back through the persisted store's **asynchronous** rehydration.

So on first paint the guard can be holding a perfectly valid token beside a `null` user. It reads
that as logged-out, redirects to login, and bounces back to the app a moment later when rehydration
finishes: a visible, intermittent login **flash** on a session that was never invalid.

**The invariant: the pre-hydration window renders a SPLASH, never a LOGIN.** Both the protected guard
and the guest-only guard short-circuit on "has the store rehydrated?" *before* they touch the user.
Rendering login "just while we wait" reproduces the exact flash the gate exists to remove.

### The one non-obvious line in the hydration hook

The hook is otherwise the store library's own recipe — seed the flag from the current state, flip it
in the "hydration finished" callback. The line that is not obvious:

```ts
useEffect(() => {
  const unsub = store.persist.onFinishHydration(() => setHydrated(true));
  setHydrated(store.persist.hasHydrated());   // ← re-read AFTER subscribing
  return unsub;
}, []);
```

Hydration can finish **between** the initial render and the effect firing, in which case the finish
callback has already fired and will never fire again. Without that re-read the gate hangs on the
splash forever. This is the persisted-store API surface — confirm it still exists on a major upgrade
of whichever store the project pins.

### Two diagnostics worth carrying

- **The intermittency is itself the proof it is a race.** A guard that always decided correctly would
  never flash; one that always lost would always bounce. Only a timing-dependent read produces
  "sometimes".
- **A flash that *ends* on an authenticated screen is not the 401→refresh path.** That path is
  one-way: it clears the session and hard-navigates to login, with no route back
  ([api-client.md](api-client.md)). Use the direction before you go looking in the interceptor.

A race this fast cannot be sampled by a scripted read — every read arrives after it is over. To
verify one, force it slow: pin the gating hook `false` for a second or two, assert a **sequence**
(splash → content, and never login) rather than a snapshot, then revert the instrumentation and
confirm the diff is empty. That last step is not tidiness — a hook pinned `false` renders the splash
unconditionally, so one surviving line ships a permanently splashed app.

## The survivor owns the redirect

A second race of the same family, and it silently drops every deep link.

On login the auth state flips. The guest-only guard re-renders and its `<Navigate>` **unmounts the
login page** — before that page's own post-login `navigate(returnTo)` ever runs. The page loses,
every time, and the user lands on the default screen instead of the page they originally asked for.
Nothing errors.

The fix is not to win the race but to remove it: **when two components navigate on the same state
flip, put the decision in the survivor.** The guard resolves the return path itself rather than
leaving it to the component it is about to unmount.

### The return path is internal-only

A return path taken from the URL is attacker-controlled. Pass it through a check that accepts **only
a value beginning with `/`, and rejects `//` (protocol-relative) and `/\` (backslash-host)** — both
of which a naive "starts with a slash" test lets through, and either of which turns your login screen
into an open redirect. Anything that fails the check falls back to the default landing route.

## A UI guard is UX, never authorization

A route guard hides a screen; a role check hides a button. Neither is a security boundary. The server
authorizes every request independently, and it must reject one that arrives without the right to it
even though no UI ever offered the control. Treating a guard as enforcement puts the decision on the
machine the user controls.

Two practical consequences:

- Never gate on a value the client can edit and the server does not re-derive.
- When a UI guard and the server disagree, the server is right and the UI has a bug — usually a stale
  or mis-mapped claim, not a missing rule.

## Do not assume two guards are symmetric

Where a codebase has more than one guard — a second app, a second shell, an admin variant — read the
one you are editing. They routinely share the hydration gate byte-for-byte and then **diverge after
hydration** (one sends every guest to login; the other shows a public landing page at the root and
only redirects on a deep link). Asserting symmetry plants a false invariant, and a fix applied to one
is not evidence about the other. Grep both before calling a guard change complete.

## Where filters and pagination live

Values that describe *what the screen is showing* — page, search term, sort, active tab — belong in
the URL rather than in a store: the result is shareable, bookmarkable, survives a reload, and makes
the back button behave. Reset the page index whenever a filter changes, or the user lands on page 7
of a three-page result. Which values earn a place in the URL versus a store is
[state-and-server-data.md](state-and-server-data.md).

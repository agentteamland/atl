# State & data — React + TypeScript

How a `developer` worker manages state and fetches data on this stack. The load-bearing idea in
this topic is one distinction — **server state is not client state** — and most of the conventions
below fall out of it. This is generic stack craft that travels with the `web` pack; the project's
chosen store, its API client, and its error-envelope shape are project facts named in the canonical
brief (`Conventions/`), and they override the defaults here when they differ (`knowledge/pack-format.md`).

## The distinction that drives everything: server state ≠ client state

Two kinds of state live in a web app, and treating them the same is the most common source of
stale-UI and race bugs:

- **Client state** — state the UI *owns*: a form's current input, whether a modal is open, a selected
  tab. It has no source of truth outside the browser; the UI is the truth.
- **Server state** — data that lives on the server and is only *cached* in the client: the user list,
  a record being edited. The server is the truth; the client holds a copy that can go stale, needs
  invalidation, and is shared across components.

The rule: **never hand-mirror server state into `useState`.** Fetching into local state means you now
own cache invalidation, loading/error flags, deduplication, and refetch-on-focus by hand — and you
will get one of them wrong. Route server state through a data-fetching layer that owns those concerns;
keep `useState`/`useReducer` for genuine client state.

## Client state — the smallest tool that fits

Reach for state in this order, escalating only when the smaller tool genuinely can't express the need
— because every escalation adds indirection a reader has to trace:

1. **`useState`** — the default for a single component's local state (a toggle, an input). If it's
   local and simple, stop here.
2. **`useReducer`** — when several pieces of state change together under named actions (a
   multi-field form, a wizard). The reducer makes the transitions explicit and testable as pure
   functions — the reason to graduate from scattered `useState` calls.
3. **Lift + pass props / `Context`** — when a *few* components share state. Lift it to the nearest
   common ancestor; use `Context` when prop-drilling would thread a value through many
   uninterested intermediaries. **Context is for low-frequency, widely-read values** (theme, current
   user, locale) — not for state that changes on every keystroke, because every consumer re-renders
   on every change.
4. **A store (e.g. Zustand / Redux Toolkit)** — only when app-wide client state is written from many
   places and read from many, and `Context` re-render behavior or prop-lifting has become the
   bottleneck. A store is real added surface; introduce it against a real need the smaller tools
   can't meet, not speculatively (the pack's Simplicity default). The **project pins which store**
   in `Conventions/`; follow the project's choice rather than importing a second one.

## Server state — a data-fetching layer, not `useEffect` fetches

The reference convention is a query library (`@tanstack/react-query` in this pack's baseline; a team
may pin SWR or RTK Query — the *convention* travels, the library is the team's pin). Route every
server read through it:

- **Reads = queries.** A `useQuery({ queryKey: ['users'], queryFn: fetchUsers })` gives you caching, dedup, `isLoading`/
  `isError`, and refetch-on-invalidate for free. The `['users']` **query key** is the cache identity —
  key by the resource and its parameters (`['user', userId]`) so distinct params cache distinctly and
  the same params dedupe.
- **Writes = mutations, then invalidate.** A mutation performs the write and, on success,
  **invalidates the affected query keys** so the cache refetches the new truth. Manually poking the
  cached value to match your optimistic guess is the stale-data trap this layer exists to avoid;
  invalidate and let the source of truth win (optimistic updates are a deliberate, reversible
  exception, not the default).
- **Don't `useEffect(() => { fetch()… }, [])` into `useState`.** That hand-rolled pattern re-implements
  — and usually mis-implements — cancellation on unmount, dedup of concurrent renders, and error
  handling. The whole point of the layer is that you don't write those by hand.

## Typing data at the boundary — validate what crosses the wire

Data from the network is `unknown` until proven otherwise — the server can send a shape the types
promise but don't enforce. Type the API contract explicitly and **narrow/validate at the fetch
boundary**, not deep in a component:

- Declare the response type (`type User = { id: string; name: string }`) and give the fetch function
  that return type, so the rest of the app is typed off one boundary.
- Where correctness matters (external or user-shaped data), **validate at the boundary** — a runtime
  schema check (the project may pin a validator such as Zod in `Conventions/`) turns "the types say so"
  into "we checked." Validate once, at the edge; downstream code then trusts the type. Pushing the
  guard to the boundary is what keeps every consumer from re-checking.
- **Never `any` the API response** to move faster — `any` disables the checker exactly at the app's
  least-trusted input. Use `unknown` + a narrow so the validation is forced and visible.

## Forms — controlled by default, typed by the model

- **Controlled inputs** (`value` + `onChange`) are the default: the component state is the single
  source of truth for the field, which makes validation, conditional enabling, and testing
  straightforward (assert on state, drive via `user-event`, [testing.md](testing.md)). Reach for
  uncontrolled/`ref` inputs only for a genuine perf or integration reason.
- **Type the form's shape as one model** (a `type` for the whole form object), so submit handlers and
  validation share one contract instead of drifting per-field.
- A **form library** (React Hook Form, etc.) is worth it once forms get large or share validation;
  it's over-engineering for a two-field form. The **project pins** the form approach in `Conventions/`
  — follow it.

## Effects — the escape hatch, used sparingly

`useEffect` is for **synchronizing with something outside React** (a subscription, a browser API, a
timer) — not for deriving state from props (compute it in render) and not for fetching (use the query
layer, above). The reason: an effect that "derives" or "fetches" re-runs on every dependency change
and quietly introduces extra renders, races, and stale closures. **Always list the real dependency
array**; a missing dependency is a stale-closure bug, an over-broad one is a re-render storm. If an
effect has no external system to sync with, it probably shouldn't be an effect.

## What travels here, and what doesn't

The *conventions* — server ≠ client state, smallest-tool-that-fits, validate-at-the-boundary — are
project-agnostic and belong in this pack; a durable lesson (e.g. "invalidate the list key on every
mutation — we shipped stale rows twice") generalizes here via `/drain` and improves every project.
The *project's* actual API endpoints, its chosen store/validator/form library, and its error-envelope
shape are project facts the worker reads from `Conventions/` (brief-named) at runtime — never
pre-authored here, and never written to the wiki by the worker (adapter §8; the tech-lead promotes
project facts).

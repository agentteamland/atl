---
knowledge-base-summary: "The split that decides where a value lives: server data is **cache** owned by the data-fetching layer — fetched, cached, invalidated, never mirrored into component state — while a client store holds only what exists purely in the browser (UI state, session, preferences). Query-key structure as the unit of invalidation, why invalidating beats hand-patching a cached entry, the loading/error/empty discriminations the query result gives you for free, and the narrow cases where a value genuinely belongs in the URL instead of in either store."
---

# State And Server Data

One question decides most of this topic: **where does this value live?** Get it wrong and you get the
stale-UI bug class, which is the nastiest kind on this stack because the wrong value on screen is a
real, previously-true value — nothing looks broken.

| The value is… | It lives in | Because |
|---|---|---|
| a copy of something the server owns | the **data-fetching layer's cache** | the server is the truth; the client holds a copy that can go stale |
| something that exists only in the browser | component state, or a **client store** | there is no other source of truth |
| a description of what the screen is showing | the **URL** | shareable, bookmarkable, survives reload, back button works |

## Server data is cache, not state

**Never hand-mirror server data into `useState`.** The moment you do, you own cache invalidation,
loading and error flags, request deduplication, cancellation on unmount, and refetch-on-focus — by
hand, and you will get one of them wrong. That is the entire argument for a query layer; it is not
about saving keystrokes.

The corollary is the rule people actually break: **`useEffect(() => { fetch(…) }, [])` into
`useState` is the anti-pattern**, not a lighter alternative to it. Effects are for synchronizing with
something outside React — a subscription, a browser API, a timer. Not for fetching, and not for
deriving state from props (compute that in render). An effect with no external system to sync with
probably should not be an effect.

## Query keys are the unit of invalidation

A query key is the cache identity: key by the resource and its parameters so distinct parameters
cache distinctly and identical ones dedupe. Build them with a per-feature factory rather than
scattering array literals:

```ts
export const userKeys = {
  all: ['users'] as const,
  lists: () => [...userKeys.all, 'list'] as const,
  list: (filters: UserFilters) => [...userKeys.lists(), filters] as const,
  details: () => [...userKeys.all, 'detail'] as const,
  detail: (id: string) => [...userKeys.details(), id] as const,
};
```

The hierarchy is the point, not the tidiness: invalidating a **prefix** invalidates everything
beneath it, so `lists()` refetches every filter combination at once while `detail(id)` touches one
record. Ad-hoc string arrays cannot express that, and the invalidation you need turns into a
hand-maintained list of keys that goes stale the first time someone adds a filter.

## Invalidate; do not patch

After a mutation, **invalidate the affected keys and let the server's answer win.** Hand-patching a
cached entry to match what you believe the write did encodes the client's guess of the server's
behaviour — and every derived or server-computed field on that record is now whatever it used to be.

The same rule governs pushed updates. When a realtime channel delivers a change, treat the message as
**"this is stale, refetch it"**, never as a payload to merge. The REST read stays the single source
of truth; the push only decides *when* to re-ask. A channel that patches the cache is a second,
undocumented copy of the server's business logic living in the browser.

Two disciplined exceptions:

- **The response of the mutation you just made** is the server's own answer for that record, so
  writing it into that record's detail key is legitimate — then still invalidate the lists, whose
  ordering, counts and aggregates you did not receive.
- **An optimistic update** is deliberate and reversible: cancel in-flight queries for the key,
  snapshot the previous value, apply the guess, roll back to the snapshot on error, and invalidate on
  settle. All four steps or none — an optimistic update without the rollback is just a lie with extra
  code.

## Read the branches off the query result

The query result already discriminates loading, error and success; do not rebuild that with your own
booleans. **Which flag you branch on matters to the type checker.** On the pinned `@tanstack/react-query`
v5 baseline, `isPending` discriminates the result union, so `data` narrows to defined below it:

```tsx
const { data, isPending, isError } = useQuery({ queryKey: userKeys.lists(), queryFn: fetchUsers });
if (isPending) return <Spinner />;
if (isError) return <ErrorNotice />;
return <UserList users={data} />;   // `data` is defined here
```

`isLoading` on that same major means `isPending && isFetching` — a plain boolean that narrows
nothing, so the code beneath it fails `tsc --noEmit` and invites a non-null assertion that silences
the checker instead of answering it. Confirm this against the major the project actually pins; it has
changed across versions.

**Empty is a fourth branch, not a shade of success.** An empty result and a populated one are drawn
by different code, which is exactly why an empty screen is not a verified screen — the bugs live in
the populated branch ([browser-verification.md](browser-verification.md)). Handle loading, error,
empty and populated explicitly; a screen with three of the four has a blank state nobody designed.

`enabled` is how you express "not yet" — a query that depends on a route param or a prior result
should be disabled until its input exists, rather than firing with `undefined` and failing.

## Client state — the smallest tool that fits

Escalate only when the smaller tool genuinely cannot express the need; every escalation is
indirection a reader has to trace.

1. **`useState`** — a single component's local state. If it is local and simple, stop here.
2. **`useReducer`** — several pieces of state that change together under named actions (a wizard, a
   multi-field editor). The transitions become explicit and testable as pure functions.
3. **Lift, then `Context`** — when a few components share it. Context is for **low-frequency,
   widely-read** values (theme, current user, locale); it is the wrong tool for a value that changes
   on every keystroke, because every consumer re-renders on every change.
4. **A store** — when app-wide client state is written from many places and read from many. A store
   is real added surface; introduce it against a real need, not speculatively. **The project pins
   which one** — follow it rather than importing a second.

**Subscribe to the slice, not the store.** `useThing((s) => s.sidebarOpen)` re-renders on that value;
`useThing()` re-renders on every write anywhere in the store. For several values at once, use the
store's shallow-comparison helper — returning a fresh object from the selector without one re-renders
on every write regardless of whether anything you read changed.

Keep mutations inside the store's own actions. State mutated from outside is state whose transitions
cannot be found by grepping the store.

## Persist only what must survive a reload — and know what that costs

Session and preferences, typically. Persisting a cache of server data buys you a stale copy with no
invalidation story.

Two things about persistence that bite:

- **Rehydration is asynchronous.** The store is empty for a window after first paint. That window is
  precisely what the route guard's hydration gate exists for
  ([routing-and-guards.md](routing-and-guards.md)) — anything that reads persisted state during first
  render needs the same treatment.
- **What is deliberately kept *out* of the persisted slice matters as much.** An access token is
  commonly read synchronously from its own storage key so the HTTP client can use it before hydration
  ([api-client.md](api-client.md)). That asymmetry is designed, and it is the mechanism behind the
  guard race. Read the comment before you "simplify" it.

## The URL is a real place for state

Filters, pagination, sort order, the active tab of a deep-linkable view: these describe what the
screen is showing, and putting them in the URL is what makes a screen shareable and the back button
correct. Read them from the search params and feed them straight into the query key, so the URL and
the cache identity cannot drift.

Do not keep a second copy in a store. Two sources of truth for the same value is the same defect as
mirroring server data, one layer up — and the copy that loses is always the one the user is looking
at.

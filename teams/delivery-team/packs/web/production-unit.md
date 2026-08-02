# Production unit — adding a component or a page

The thing this pack builds over and over is a **component**: a typed props contract, a function body,
and the colocated test that pins its behavior. A **page** (a route) is the same unit with one extra
property — it owns a URL. They share everything except the *last link in the registration chain*: a
component is reached because a parent renders it; a page is reached because the project's route table
declares it. Where the two diverge below, it is called out.

Two of the steps exist for one reason, and it is worth stating up front. The characteristic failure of
a scaffolded component is **not** a bad body — it is a **correct body that nothing renders**. A
component that is written, typed, and unit-tested but never imported anywhere passes `lint`,
`typecheck` and the whole Vitest suite green, and never appears on a single screen. So the lifecycle
does not end at "tests pass"; it ends at "I opened the running app and saw it there".

## 1. Decide before you write

Settle these first — renaming or re-siting later means touching the file, its test, every import, the
route table, and any nav entry that names it.

| Question | How to answer it |
|---|---|
| **A new component, or a prop on an existing one?** | A prop when it is the same contract with a different look or behavior; a new component when the contract itself differs. A shared component is a permanent surface others will import — the bar is higher than for a variant. |
| **Presentational or container?** | By **data-ownership**, not folder taste ([component-conventions.md](component-conventions.md)). This is not cosmetic here: a container needs its data layer mounted above it in the *app's* tree, a leaf needs nothing but a parent. It decides step 3. |
| **A page, or an in-tree component?** | A page if a user can land on it or deep-link to it by URL. That answer *is* the composition root: the route table for a page, a parent's JSX for a component. |
| **Shared or feature-local?** | `components/` for cross-feature, `features/<feature>/` for feature-local ([component-conventions.md](component-conventions.md)). Blast radius, and it tells you which siblings' wiring you must match. |
| **What state does it own?** | Nothing, client state, or server state ([state-and-data.md](state-and-data.md)). Server state means a query key to name now, and a provider to confirm in step 3. |
| **The name.** | `<Component>.tsx` in PascalCase, props type `<Component>Props`, test `<Component>.test.tsx` beside it. Predictable names are greppable and the import path predicts the export. |

## 2. Scaffold

The leaf — it renders what it is handed, which is what keeps it testable in isolation:

```tsx
// src/features/users/UserCard.tsx
type UserCardProps = {
  userId: string;
  name: string;
  onSelect?: (userId: string) => void;   // a DOMAIN callback, not a raw DOM event
};

export function UserCard({ userId, name, onSelect }: UserCardProps) {
  // No fetch, no store subscription, no query — a presentational leaf owns no data.
  return (
    <article>
      <h3>{name}</h3>
      <button type="button" onClick={() => onSelect?.(userId)}>
        Select
      </button>
    </article>
  );
}
```

The container — it owns the data and hands it down, and is therefore also *where the leaf gets
registered*:

```tsx
// src/features/users/UserList.tsx
type UserListProps = {
  onSelect?: (userId: string) => void;
};

export function UserList({ onSelect }: UserListProps) {
  const { data, isPending, isError } = useQuery({ queryKey: ['users'], queryFn: fetchUsers });
  // `isPending`, not `isLoading`: on the pinned `^5` baseline it discriminates the result union, so
  // `data` narrows to defined below. `isLoading` there means `isPending && isFetching` — a plain
  // boolean that narrows nothing, and the `data.map` under it then fails `tsc --noEmit`.
  if (isPending) return <Spinner />;
  if (isError) return <ErrorNotice />;
  return (
    <ul>
      {data.map((u) => (
        <li key={u.id}>
          <UserCard userId={u.id} name={u.name} onSelect={onSelect} />
        </li>
      ))}
    </ul>
  );
}
```

Function components and hooks only; props typed at the boundary, locals inferred; server state through
the query layer, never mirrored into `useState`. See [component-conventions.md](component-conventions.md)
and [state-and-data.md](state-and-data.md) for why each of those is load-bearing.

## 3. Register it — the step that is easy to skip and invisible when skipped

A component exists only once something the app root reaches actually renders it. Trace the whole chain
before you call it wired:

```
main.tsx / App.tsx  →  providers  →  router  →  route table  →  page  →  container  →  leaf
```

**A component** is registered by its parent: import it and render it in the JSX of the container or
page that owns its data. **Read the siblings first and match them.** If the feature's components are
imported directly, import yours directly; if the folder ships a barrel `index.ts` that the siblings are
re-exported from, add yours to it the same way rather than starting a second convention. A registration
that looks unlike its neighbours is the one a reviewer skims past.

**A page** is registered in the project's route table, next to its siblings and at the same nesting:

```tsx
// The project's route table — match the existing entries exactly.
// (Which router the project pins is a project fact named in Conventions/; the shape
//  below is the concept, not one library's API.)
{ path: '/users',         element: <UsersPage /> },
{ path: '/users/:userId', element: <UserDetailPage /> },   // ← the new page, wired like its siblings
```

Match the siblings on **three** things, not one: the same layout/parent route (so the page renders
inside the app shell and behind whatever guard the siblings sit behind), the same loading strategy (if
the sibling routes are lazily imported, make yours lazy too), and **the nav entry** — if the siblings
are linked from a menu or index and yours is not, it is registered but unreachable by anything except a
typed URL.

**Providers are part of registration.** A container that calls `useQuery` needs the query client above
it *in the app's tree*; a store-backed component needs its store provider the same way. If your unit is
the first consumer of that layer in this app, mounting the provider at the composition root is part of
your change:

```tsx
// the composition root — where the app's providers wrap the router
<QueryClientProvider client={queryClient}>
  {/* the router goes here */}
</QueryClientProvider>
```

This one bites specifically because the *test* supplies its own provider ([testing.md](testing.md)
mocks at the network boundary), so the test is green whether or not the app has one.

## 4. Verify the durable effect — look at it in the running app

```bash
npm run lint && npm run typecheck && npm run test -- --run   # necessary — and NOT sufficient here
npm run dev                                                  # (or: npm run build && npm run preview)
```

Then drive the running app through the **preview / chrome-devtools MCP** ([testing.md](testing.md)) and
observe the unit where a user would find it:

1. **A page** — navigate to its URL: it renders, *inside the app shell*, not standalone. Then reach it
   the way a user does — click the nav entry — because a route nobody links to is registered and still
   unreachable.
2. **A component** — navigate to the screen that is supposed to contain it and confirm it is **there**,
   in the composed page, not in a test renderer.
3. **Anything that owns server state** — confirm real data arrives (the list populates from the API,
   not from a fixture). That is what proves the provider is mounted above it and the query key resolves.
4. **Screenshot each** and attach it to the work-item as the Level-1 self-test evidence
   ([testing.md](testing.md)). That screenshot is the evidence for this unit type; a green unit-test
   summary is not.

**What this pack already guarantees, and where the hole is.** For a **page**, the guarantee is real and
prescribed: [testing.md](testing.md)'s Level-1 web-surface loop drives the *composed* app through the
MCP against a live `npm run dev`, so a page with **no** route entry fails the very first navigation —
the URL does not resolve. A **misnested** page is the weaker case: it navigates fine and renders
outside the shell, so what catches it is 4.1's *inside the app shell*, not the navigation itself.
Running that loop for a page is not an extra ritual; it is the pack's existing gate, and step 4 here
only makes it explicit and adds the nav-path check the loop does not name.

**The false green to name out loud** is the **component** the web surface never sees. A component that
is written, unit-tested and *never imported anywhere* passes the entire gate: neither `tsc --noEmit`
nor the standard lint rules flag an **exported** symbol that nothing imports — their unused-symbol
checks are per-file, and the component *is* used, by its own colocated test (a dead-export detector is a
separate tool this pack does not prescribe). Worse, RTL's `render(<UserCard … />)` mounts a composed
tree of exactly one component, so the test proves **the component renders** while leaving **the app
renders it** completely untested. And the one gate that does drive the composed root is the gate this
pack tells you to keep thin — "verify on the web surface only what needs the rendered page"
([testing.md](testing.md)) — so a leaf whose criterion looks logic-shaped is precisely the unit a worker
skips the browser for. The pack catches an unregistered page at navigation; it catches nothing at all
for a component you decided did not need the browser. That gap is what step 4.2 exists to close, and it
costs one navigation.

If the dev/preview server won't start or the MCP can't drive the flow, the registration is
**unverified** — surface it as such. Never emit a pass for a surface that did not execute
([testing.md](testing.md)).

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
npm run test -- --run --coverage
```

**The false green peculiar to this stack:** a test that renders the component and asserts only that
it rendered — or a snapshot regenerated to match whatever the component now does. Both stay green
forever, including after the component breaks. Assert the behaviour a user would notice, then check
that reverting the change turns that assertion red.

If 90% is genuinely unreachable because the new code is entangled with something untestable, record
the exception on the work-item — which lines, and why. Never a silent pass.

## 5. Common pitfalls

- **Built but never imported.** The leaf that only ever renders inside its own test. Green everywhere,
  absent from every screen. Step 4 is the only place it shows.
- **Registered under the wrong parent route.** The page renders, but outside the app shell — no nav, no
  layout, past whatever guard the siblings sit behind. Navigating to it shows this instantly; no test
  will.
- **Reachable by URL, unreachable by click.** The route entry landed, the nav entry did not. When the
  siblings have both, registration means both.
- **The provider the test supplied and the app did not.** The container is green under a test-supplied
  query client and throws (or renders permanently empty) in the app. Confirm real data in step 4.3, not
  a passing test.
- **Widening a prop or `any`-casting at the call site to make the wiring compile.**
  [component-conventions.md](component-conventions.md) forbids loosening a contract to make a *test*
  pass; the same holds at the registration site. If the parent doesn't have the data the props demand,
  the container/leaf split is wrong — fix that, not the type.
- **Copying the query result into the parent's `useState` while wiring it in.** That re-creates the
  exact stale-cache bug [state-and-data.md](state-and-data.md) exists to prevent. Pass the query's data
  down, or let the child own its own query.

## 6. Hand-off

The work-item is ready for the tester when: `lint` + `typecheck` + the unit run are green; a
**screenshot of the unit rendered in the running app** is attached (a page at its URL inside the shell;
a component on its host screen); anything owning server state was observed with real data rather than
inferred; and any web criterion that could not run is surfaced as **unverified**, never as a pass.

State plainly in the hand-off which of these you **observed** and which you **assumed** — an unverified
claim is worse than an admitted gap, because the tester's Level-2 pass gates the green and can only
check what it is told about.

If the unit is a **new shared component or a new route**, say so explicitly: that is a project fact for
the tech-lead to promote into the project's durable-knowledge store (`Conventions/`). The worker names
it in the hand-off and never writes those pages itself
([component-conventions.md](component-conventions.md)).

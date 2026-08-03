---
knowledge-base-summary: "My primary production unit — a screen, end to end: decide (page or in-tree component, what state it owns, where it lives, its name) → scaffold (typed props, data through the query layer, all four render branches: loading, error, empty, populated) → **register** it in the route table and in the navigation that reaches it → verify by opening the running app and clicking to it → the pitfalls → hand-off. Step 3 is the one that is invisible when skipped and step 4 is what proves it was not: an unregistered screen passes lint, type-check and the whole unit suite green while appearing on no screen at all, so a green suite is precisely the false green this lifecycle exists to defeat."
---

# Screen Blueprint

The thing I build over and over is a **screen**: a component that owns a URL, fetches what it needs,
renders every branch it can reach, and is reachable by a user. An in-tree component is the same unit
minus the URL — everything below applies except the last link in the registration chain, and that
divergence is called out where it happens.

Two of the six steps exist for one reason, and it is worth stating before the template. **The
characteristic failure of a scaffolded screen is not a bad body — it is a correct body nothing
routes to.** A page that is written, typed and unit-tested but never added to the route table passes
`lint`, `tsc --noEmit` and the entire Vitest suite green, and appears on no screen in the running
app. So this lifecycle does not end at "tests pass"; it ends at "I opened the running app and
clicked my way to it".

## 1. Decide before you write

Settle these first. Renaming or re-siting later means touching the file, its test, every import, the
route table, the nav entry and the translation bundles.

| Question | How to answer it |
|---|---|
| **A page, or an in-tree component?** | A page if a user can land on it or deep-link to it by URL. That answer **is** the composition root: the route table for a page, a parent's JSX for a component. |
| **Which parent route?** | The one its siblings sit under. That decides the layout it renders inside and the guard it sits behind — both invisible in a unit test and both instantly visible in the browser. |
| **What state does it own?** | Nothing, client state, or server state. Server state means naming a query key now and confirming the provider is mounted in the app (step 3), not only in the test. |
| **Shared or feature-local?** | Feature-local under the feature it belongs to; shared only when a second feature genuinely imports it. A shared surface is permanent — the bar is higher than for a variant. |
| **Does it need a new route parameter or search param?** | A value the screen must survive a reload or a shared link with belongs in the URL, not in a store. Decide before scaffolding; retrofitting it changes the route entry. |
| **The name.** | `<Name>Page.tsx` in PascalCase for a page, `<Name>.tsx` otherwise; props type `<Name>Props`; test `<Name>.test.tsx` beside it. Predictable names are greppable, and the import path predicts the export. |

If any answer is a project fact I do not have — which layout, which guard, which folder convention —
it is in the tech-lead's canonical brief or the project's `Conventions/`. I read it; I do not invent
one and I do not start a second convention next to an existing one.

## 2. Scaffold — four branches, not one

A screen that fetches data can be in **four** states, and three of them are the ones users actually
hit on a bad day. Write all four before wiring anything:

```tsx
// src/features/users/pages/UsersPage.tsx
export function UsersPage() {
  const { t } = useTranslation('users');
  const { data, isPending, isError, error, refetch } = useUsers();

  if (isPending) return <PageSkeleton variant="table" />;
  if (isError)   return <ErrorState message={error.message} onRetry={refetch} />;
  if (data.length === 0) return <EmptyState title={t('empty.title')} action={…} />;

  return (
    <div className="space-y-6">
      <PageHeader title={t('title')} />
      <UserList users={data} />
    </div>
  );
}
```

- **Which flag discriminates the union is version-dependent — check the pinned major before copying
  either.** On TanStack Query v5 it is `isPending`, and `data` narrows to defined underneath it;
  `isLoading` there means `isPending && isFetching`, a plain boolean that narrows nothing, so the
  `data.map` under it fails `tsc --noEmit`. On v4 the discriminating flag is `isLoading`. Read
  `package.json` rather than reaching for the one you remember.
- **Loading is a skeleton shaped like the final content**, not a centred spinner — the layout does
  not jump when data lands. A spinner belongs on a button and on a "load more".
- **Error is recoverable.** A message plus a retry that actually re-runs the query. A route-level
  failure takes the page; one failed widget on a composed screen takes only its own card and leaves
  the rest usable.
- **Empty has context** — what this screen would contain, and the action that creates the first one.
  Never a bare "No data".
- **The page owns data; the pieces below it own rendering.** Fetch at the screen and pass down, or
  let a child own its own query — but never mirror a query result into `useState`, which recreates
  the stale-cache bug the data layer exists to prevent.
- **Every user-facing string goes through the i18n layer from the first commit.** Retrofitting
  strings is a mechanical sweep nobody schedules.

An in-tree component takes its data as typed props and owns none of this — that is what keeps it
testable in isolation, and it is why the container/leaf split is decided by data-ownership rather
than by folder taste.

## 3. Register it — the step that is invisible when skipped

A screen exists only once something the app root reaches actually renders it. Trace the whole chain
before calling it wired:

```
main.tsx → providers → router → route table → layout route → page → container → leaf
```

**A page is registered in the route table**, next to its siblings and at the same nesting:

```tsx
const UsersPage = lazy(() =>
  import('@/features/users/pages/UsersPage').then(m => ({ default: m.UsersPage }))
);

{
  path: '/', element: <AppLayout />, children: [
    { path: 'users', element: <Suspense fallback={<PageSkeleton />}><UsersPage /></Suspense> },
  ],
}
```

Match the siblings on **four** things, not one:

1. **The same parent/layout route**, so the page renders inside the app shell and behind whatever
   guard the siblings sit behind. A page registered at the top level renders fine and is outside
   both.
2. **The same loading strategy.** If the siblings are lazily imported behind a `Suspense` boundary,
   make yours lazy too — and know that a lazy chunk that fails to load is a **browser-only** failure
   (step 4).
3. **The nav entry.** If the siblings are linked from a menu or an index page and yours is not, the
   route is registered and unreachable by anything except a typed URL. When the siblings have both,
   registration means both.
4. **The translation keys, in every locale bundle** — and register the namespace if the project
   splits them. A key added to one bundle and not the others renders as a raw key in the other
   languages, which no type-check and no test will report.

**An in-tree component is registered by its parent**: imported and rendered in the JSX of the
container or page that owns its data. Read the siblings first and match them — if the folder ships a
barrel that siblings are re-exported from, add yours the same way rather than starting a second
convention. A registration that looks unlike its neighbours is the one a reviewer skims past.

**Providers are part of registration.** A screen that calls `useQuery` needs the query client above
it **in the app's tree**; a store-backed screen needs its provider the same way. If your unit is the
first consumer of that layer in this app, mounting the provider at the composition root is part of
your change. This one bites specifically because the *test* supplies its own provider, so the test is
green whether or not the app has one.

## 4. Verify by observing the durable effect — open the running app

```bash
npm run lint && npm run typecheck && npm run test -- --run   # necessary, and NOT sufficient
npm run dev                                                  # or: npm run build && npm run preview
```

Then drive the running app through the browser and observe the screen where a user would find it:

1. **Navigate to its URL** — it renders, **inside the app shell**, not standalone, and not past the
   guard its siblings sit behind.
2. **Then reach it the way a user does — click the nav entry.** Clicking is the lazy-import path a
   direct load can bypass, and a route nobody links to is registered and still unreachable.
3. **Confirm real data arrives** — the list populates from the API, not from a fixture. That is what
   proves the provider is mounted above it in the app and the query key resolves.
4. **Verify the populated state, not only the empty one.** They are different code, and the
   interesting failures live in the populated branch. Create the data first.
5. **Assert a count or a DOM read**, not a screenshot; a screenshot may be attached as illustration,
   never as the measurement. The full gate, its measurement traps and its honesty rules are in
   [browser-verification.md](browser-verification.md).

**The false green to name out loud, because a fully compliant worker hits it.** The colocated test
renders the page inside providers **and a router the test itself declares** — the standard
`renderWithProviders(<UsersPage />, { route: '/users' })` helper wraps it in a `MemoryRouter` with
that initial entry. So the test can render the page, navigate to it *by its URL*, assert its content
and pass, with the page **absent from the app's real route table**. Nothing about that test disobeys
any instruction on this page: it renders the unit, it drives it like a user, it asserts visible
output. Meanwhile neither `tsc --noEmit` nor the standard lint rules flag an **exported** symbol
nothing imports — their unused-symbol checks are per-file, and the symbol *is* used, by its own test.
Every gate is green and the screen does not exist. The guarantee this step buys is not "the
integration test ran" and not even "it navigated by URL" — it is **"it entered through the table the
real app is built from"**, which only the running app can answer.

If the dev/preview server will not start or the browser cannot drive the flow, the registration is
**unverified** — surface it as such. Never emit a pass for a surface that did not execute.

## 4b. Ship the test — the unit is not done without one

Step 4 proved it works **now**, by looking at it. This step is what keeps it working. A unit is not
complete until it ships a test covering the behaviour it added:

> **diff coverage >= 90%** of the lines this unit added or modified, **and** at least one test that
> goes RED when the change is reverted.

The second half is not ceremony. Coverage proves a line *ran*; it says nothing about whether anything
*checked* the result — a test that exercises the code and asserts nothing scores 100%. Confirming it
costs a minute: revert, run, see red, restore.

```bash
npm run test -- --run --coverage
```

**The false greens peculiar to this stack:** a test that renders the screen and asserts only that it
rendered, and a snapshot regenerated to match whatever the component now does. Both stay green
forever, including after the screen breaks. Assert what a user would notice — the row count after
data loads, the empty state when the API returns none, the error state plus a working retry, the
message a rejected submit shows.

Cover the branches you wrote, not just the happy one: overriding the request mock per test is what
makes the error and empty branches reachable. If 90% is genuinely unreachable because the new code is
entangled with something untestable, record the exception on the work-item — which lines, and why.
Never a silent pass.

## 5. Common pitfalls

- **Built but never registered.** The screen that only ever renders inside its own test. Green
  everywhere, absent from the app. Step 4 is the only place it shows.
- **Registered under the wrong parent route.** It renders, outside the app shell — no layout, no nav,
  past whatever guard the siblings sit behind. Navigating shows this instantly; no test will.
- **Reachable by URL, unreachable by click.** The route entry landed, the nav entry did not.
- **The provider the test supplied and the app did not.** Green under a test-supplied query client,
  throwing or permanently empty in the app.
- **Translation keys in one bundle only.** A raw key renders in every other language, silently.
- **A lazy route whose chunk fails.** Surfaces **only** browser-side while every API probe on the
  same deploy returns 200 — which is why backend evidence is never evidence about a route.
- **Verified on the empty state.** The populated branch is different code and is where the bugs are.
- **Copying the query result into `useState` while wiring it up.** Recreates the stale-cache bug the
  data layer exists to prevent.
- **Widening a prop or casting to `any` at the call site to make the wiring compile.** If the parent
  does not have the data the props demand, the container/leaf split is wrong — fix that, not the
  type.
- **A hand-written response interface trusted because it compiles.** Rename a field server-side and
  `tsc` stays green while every value renders `undefined`. Diff the interface against the contract it
  mirrors, and check it in the browser.

## 6. Completion checklist

- [ ] All four render branches exist: loading (skeleton), error (with a working retry), empty (with
      context), populated.
- [ ] Server data comes from the query layer and is not mirrored into component state.
- [ ] Props are typed at the boundary; no `any`, no suppressed type errors.
- [ ] Every user-facing string goes through i18n, with keys added to **every** locale bundle.
- [ ] Registered: route entry under the correct parent, matching the siblings' loading strategy.
- [ ] Registered: the nav entry that reaches it, if the siblings have one.
- [ ] Any provider it needs is mounted at the app's composition root, not only in the test.
- [ ] `lint`, `typecheck` and the unit suite are green.
- [ ] Opened in the running app **by URL and by clicking**, inside the app shell, with real data.
- [ ] Verified on the **populated** state; created fixture data recorded as cleanup debt and reversed.
- [ ] A test ships with it: diff coverage >= 90%, and one assertion proven to go red on revert.
- [ ] Anything that could not run is reported as **unverified**, never as a pass.

## 7. Hand-off

The work-item is ready for the tester when: the three gates are green; the screen was **observed in
the running app** and reached by clicking, not only by URL; anything owning server state was seen
with real data rather than inferred; the test ships with the change and its red-on-revert was
confirmed; and any criterion that could not run is surfaced as unverified.

State plainly which of these you **observed** and which you **assumed**. An unverified claim is worse
than an admitted gap, because the tester's independent pass gates the green and can only check what
it is told about.

If the unit introduced a **new route, a new shared component, or a new convention**, say so
explicitly in the hand-off: that is a project fact for the tech-lead to promote into the project's
durable-knowledge store. I name it; I never write those pages myself.

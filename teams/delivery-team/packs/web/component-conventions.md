# Component conventions — React + TypeScript

How a `developer` worker structures, names, and types a React component on this stack. This is
generic **stack** craft — it travels with the `web` pack into any project. The project's own
overrides (its design-system components, its route map, its naming for domain concepts) live in the
Azure project wiki `Conventions/` and are named in the canonical brief; when they differ from what's
here, the project wiki wins (the three-layer read contract, `knowledge/pack-format.md`). Read this
before writing a component so the output fits the ecosystem, then layer the project's rules on top.

## Function components + hooks — the only model

Write function components; use hooks for state (`useState`/`useReducer`), effects (`useEffect`),
and shared logic (custom hooks). **No class components.** The reason is not taste: the entire
surrounding craft — the testing library's render model, the data-fetching layer, the linter's
rules-of-hooks — assumes function components. A class component drops out of all of it and forces
every reader to context-switch, so the consistency is load-bearing, not cosmetic.

```tsx
type UserCardProps = {
  userId: string;
  onSelect?: (userId: string) => void;
};

export function UserCard({ userId, onSelect }: UserCardProps) {
  // ...
}
```

## Typing at the boundary — the contract, not the noise

**Every component's props are explicitly typed** with a named `type` (or `interface`) — that type
*is* the component's public contract, and an explicit contract is what lets another developer (or a
test) use the component without reading its body. Inside the component, lean on inference: don't
annotate every local `const`; TypeScript infers them and the annotations would only add noise.

Concrete rules, each with its reason:

- **Name the props type `<Component>Props`.** A predictable name is greppable and reads at the call
  site; ad-hoc names make the contract hard to find.
- **Prefer `type` for props; reach for `interface` only when you need declaration-merging or
  `extends` chains.** Both work; picking one default keeps the codebase uniform. (A project may pin
  the opposite default in `Conventions/` — follow the project.)
- **Model optional-vs-required honestly.** A prop the component can render without is `?`; one it
  can't is required. This pushes "did the caller pass it?" into the type system instead of a runtime
  guard, catching the missing-prop bug at compile time.
- **Type event handlers as domain callbacks, not raw DOM events, at the component boundary.**
  `onSelect?: (userId: string) => void` — not `onSelect?: (e: MouseEvent) => void`. The parent cares
  about *what happened* (a user was selected), not *how* (a click); leaking the DOM event couples the
  parent to the child's implementation.
- **Avoid `any`; reach for `unknown` + a narrow when a value is genuinely untyped.** `any` disables
  the type checker exactly where a boundary makes a mistake most likely (external data). `unknown`
  forces a deliberate narrow, which is where the validation belongs ([state-and-data.md](state-and-data.md)).
- **Never widen a prop to make a test pass** — if a test needs a prop the type forbids, the type is
  the spec and the test is wrong, or the type is wrong; fix the real one, don't loosen the contract.

## Presentational vs container — split by who owns the data

A component either **owns data** (fetches it, holds state, decides) — a *container* — or it **only
renders what it's handed** — *presentational*. Keep the two apart:

- The presentational leaf takes props and returns markup; it has no `useQuery`, no store subscription,
  no fetch. This is what makes it trivially testable in isolation — render it with props, assert the
  output ([testing.md](testing.md)) — and reusable across contexts.
- The container wires the data (a hook, a store selector, a fetch) and passes it down. Pushing the
  data-ownership up and the rendering down is the structural reason the leaf stays pure.

This is a split by responsibility, not a mandatory folder taxonomy — don't manufacture a container
for a component that has no data to own. The value is testability and reuse; apply it where those pay.

## Custom hooks — extract shared logic, not shared markup

When two components share *behavior* (a subscription, a debounced value, a bit of fetch-and-derive),
extract a `use<Thing>` hook, not a wrapper component. A hook composes logic without adding a node to
the tree; a wrapper component to share logic adds indirection and a render layer for nothing. Name it
`use<Thing>` so the rules-of-hooks linter recognizes it and so its hook-nature is obvious at the call
site.

## Folder layout — colocate by feature, not by file-type

Group a feature's files together, not by kind across the tree. The reason: a change to a feature
touches its component, its hook, its types, and its test **together**, so keeping them adjacent means
one folder per change instead of a scavenger hunt across `components/`, `hooks/`, `types/`.

```
src/
  components/          # shared, cross-feature presentational components (Button, Modal)
  features/
    users/
      UserCard.tsx
      UserCard.test.tsx
      useUsers.ts       # feature-local data hook
      types.ts          # feature-local types
  lib/                  # framework-agnostic helpers (formatting, api client)
  App.tsx
  main.tsx
```

Naming rules and their reasons:
- **One component per file, file named for the component (`UserCard.tsx`, PascalCase).** Greppable,
  and the import path predicts the export.
- **Test colocated as `<Component>.test.tsx`** — next to what it tests, so a moved component moves
  its test and neither is orphaned ([testing.md](testing.md)).
- **`components/` = shared/cross-feature; `features/<feature>/` = feature-local.** The split tells a
  reader at a glance whether editing a file is safe (local) or blast-radius-wide (shared).

The project may pin a different tree in `Conventions/` (a monorepo layout, an `app/` router
convention, an atomic-design taxonomy). This layout is the pack's sensible default; **the project's
tree, named in the brief, overrides it** — a fresh worker follows the project's structure, not this
one, when the two disagree.

## What travels here, and what doesn't

The *conventions* above are project-agnostic React/TS craft and belong in this pack; a durable
lesson (e.g. "always type handlers as domain callbacks — DOM-event leakage bit us") generalizes here
via the capture→`/drain` loop, so it improves every project. A *project's* actual component names,
design tokens, and route structure are project facts — the worker reads them from `Conventions/`
(brief-named) at runtime, never pre-authored here, and never written back to the wiki by the worker
(worker-dispatch agents don't write the wiki, adapter §8; project facts are promoted by the
tech-lead).

---
area: web
stack: "React + TypeScript + Vite"
---

# Web pack — React + TypeScript + Vite

This pack covers browser-rendered UI work: React components and hooks, client-side state
and data-fetching, and the web test surface. When the tech-lead tags a work-unit
`area:web` at decomposition (adapter §7), the `developer` worker loads **only** this pack —
the generic craft for building and self-testing on this stack. It is the reference `web`
pack: a real team keeps the `web` area name and swaps these contents for its own frontend
stack (`knowledge/pack-format.md`). Project-specific rules — this app's routes, its design
system, its API shapes — live in the Azure project wiki `Conventions/` and layer ATOP this
pack; they are named in the tech-lead's canonical brief, never here (the three-layer read
contract, `knowledge/pack-format.md`).

## Topics
- [component-conventions.md](component-conventions.md) — component structure, naming, TS prop/state typing, folder layout.
- [state-and-data.md](state-and-data.md) — local vs shared state, when to reach for a store, server-state via a data-fetching library, forms.
- [testing.md](testing.md) — unit/component tests (Vitest + React Testing Library) + the web-surface e2e (preview/chrome-devtools MCP) + evidence via `scripts/az-attach.sh`.
- [production-unit.md](production-unit.md) — the lifecycle for this pack's production unit, a component (and a page/route): decide → scaffold → register it in the composed tree → verify it in the running app → pitfalls → hand-off.

## Test commands
- unit / component: `npm run test` (Vitest, headless; `npm run test -- --run` for a single non-watch pass in a worker)
- lint / type-check gate: `npm run lint && npm run typecheck` (`tsc --noEmit`)
- web surface (e2e): drive the app through the **preview / chrome-devtools MCP** against a running `npm run dev` (or `npm run preview` on a production build) — there is no headless-browser dependency in `package.json`; the browser is the MCP, not a bundled runner (see [testing.md](testing.md)).

## Key conventions
- **Typed at the boundary, inferred within.** Every component's props are an explicit `type`/`interface`; internal locals lean on inference. This is the line that keeps the contract between components explicit without drowning the body in annotations.
- **Function components + hooks only.** No class components, no lifecycle methods — the whole ecosystem (this pack included) assumes the hooks model; mixing paradigms fractures the shared craft.
- **Presentational vs container split by data-ownership, not by folder dogma.** A component that fetches/owns state is a container; one that only renders props is presentational. Keeping the fetch out of the leaf is what makes the leaf testable in isolation ([testing.md](testing.md)).
- **Server state is not client state.** Data from the API is cache, not local state — it is fetched, cached, and invalidated by a data-fetching layer, never hand-mirrored into `useState` ([state-and-data.md](state-and-data.md)). Conflating the two is the single most common source of stale-UI bugs.
- **A component is not done until it is registered AND observed in the running app.** Something the app root reaches must render it — a parent's JSX for a component, the route table for a page — and you then look at it through the web surface. A component that is written but never imported passes `npm run lint`, `npm run typecheck` and the whole Vitest suite green: its colocated test renders it in a composed tree of one, so "it renders" is proven while "the app renders it" is never tested ([production-unit.md](production-unit.md)).
- **The self-test drives the loaded pack, on the right surface.** Logic → Vitest/RTL (parallel); a criterion that only manifests in the rendered UI → the web MCP surface. An un-run surface is unverified, never a green (aligned with the `tester`'s Level-2 discipline; [testing.md](testing.md)).

## Dependency baseline
A team **pins** these; the versions below are a plausible current-ish baseline, not gospel — a real team sets its own lockfile and updates this list when it bumps.

- `react` ^18 (or ^19) — the view library; function components + hooks.
- `react-dom` ^18 (or ^19) — the DOM renderer; keep it lockstep with `react`.
- `typescript` ^5 — types at the component boundary and across the data layer.
- `vite` ^5 — dev server + build; the `@vitejs/plugin-react` (or `-swc`) plugin drives JSX/Fast Refresh.
- `vitest` ^2 — the unit/component runner; shares Vite's config + transform, so tests see the same module resolution as the app.
- `@testing-library/react` ^16 + `@testing-library/user-event` ^14 — behavior-first component testing (query by role/text, not by implementation detail).
- `@tanstack/react-query` ^5 — the reference server-state layer (a real team may substitute SWR or an RTK Query slice; the *convention* — server state is cache — travels, the library is the team's pin).

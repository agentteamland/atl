# Testing — React + TypeScript (unit + the web surface)

How a `developer` worker self-tests web work on this stack. This is the pack knowledge that drives
the developer's **Level-1 self-test** — micro-loop step 4 (`self-test` phase, dispatch worker
contract §2). It is deliberately consistent with the shipped `tester` agent's discipline, which this
pack must not contradict:

- **Two levels.** Level-1 is *this* — the developer's own fast self-test, driven by this pack. Level-2
  is the separate `tester` worker (thorough strategy/edge/regression) that gates the green. The
  developer does **not** do Level-2; a fresh tester worker does, per work-unit. This file is Level-1
  craft only.
- **The web surface runs at full concurrency.** Each worker drives its **own browser context** through
  the preview/chrome-devtools MCP, so there is no shared single slot to contend for — unlike the
  serialized mobile-emulator lease (that's the `mobile` pack's concern; the web surface has no such
  gate).
- **Block, never silently pass.** A surface that could not actually run is **unverified**, never a
  green — the same cardinal rule the tester holds, applied at Level-1.

## The pyramid at the unit level — push logic down

Most self-test coverage is fast unit/component checks; few are web-surface e2e. The reason is the
same one that shapes the tester's surface plan: fast checks are cheap, parallel, and precise; the
web surface is slower and best reserved for the criteria that only manifest in the rendered UI.
Proving business logic *through* the browser when a unit test would catch it is slow and flaky —
push logic-probing to the bottom (unit) and keep the web surface for what genuinely needs a rendered
page.

## Level-1 unit / component tests — Vitest + React Testing Library

**Vitest** is the runner (it shares Vite's transform and module resolution, so tests resolve modules
exactly as the app does — no separate test-bundler config to drift). **React Testing Library** drives
components the way a user does.

Run it: `npm run test -- --run` for a single non-watch pass (a worker wants one deterministic run,
not the watcher). The type-check gate is `npm run typecheck` (`tsc --noEmit`) and lint is
`npm run lint` — a worker runs all three before considering step 4 done, because a green test suite
over code that doesn't type-check or lint is a false pass ([pack.md](pack.md) test commands).

Conventions, each with its reason:

- **Query by role/text/label, not by test-id or class.** `getByRole('button', { name: /save/i })`,
  not `container.querySelector('.btn-save')`. Querying the way a user (or assistive tech) perceives
  the UI means the test binds to *behavior*, not to implementation detail — a refactor that keeps
  behavior keeps the test green, which is exactly what you want a test to do.
- **Drive interactions with `@testing-library/user-event`, not raw `fireEvent`.** `user-event`
  simulates a real interaction sequence (focus, keydown, input, keyup) where `fireEvent` fires one
  synthetic event; the higher-fidelity simulation catches handler bugs the coarse event misses.
- **Assert on rendered output, not internal state.** Check what the user sees (the text, the disabled
  attribute, the error message), never the component's `useState` value. Testing the visible contract
  is what lets the implementation change freely underneath.
- **A test is only coverage if it fails when the behavior is violated.** A check that stays green when
  you break the code is theatre (the tester's "coverage is behavior caught" principle, applied at
  Level-1). If you doubt a test binds, break the code and confirm it goes red.
- **Test the component's contract; trust the framework and the pack's libraries.** Don't test that
  React re-renders or that the query library caches — that's the framework/library boundary the
  tester's `test-strategy` also draws. Test *your* logic.
- **Mock at the network boundary, not internals.** Stub the fetch/query response (e.g. via the query
  client's provided test seams or a request mock), not the component's own functions — mocking
  internals couples the test to implementation and defeats the point.

```tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { UserCard } from './UserCard';

test('selecting a user calls onSelect with the id', async () => {
  const onSelect = vi.fn();
  render(<UserCard userId="u1" onSelect={onSelect} />);
  await userEvent.click(screen.getByRole('button', { name: /select/i }));
  expect(onSelect).toHaveBeenCalledWith('u1');
});
```

## Level-1 web-surface e2e — the preview / chrome-devtools MCP

A criterion that only manifests in the **rendered UI** — a screen state, a multi-step interaction, a
visible rejection message — is verified on the web surface. On this stack the surface is driven
through the **preview / chrome-devtools MCP** against the app running under `npm run dev` (or
`npm run preview` on a production build). This is why `package.json` carries **no** headless-browser
runner dependency: the browser *is* the MCP, not a bundled Playwright/Cypress install. Don't add one
to "run the e2e" — the surface is the MCP the worker already has.

The Level-1 web-surface loop:

1. **Start the app** — run `npm run dev` (or build + `npm run preview`) so there is a live URL for the
   MCP to drive. Each worker's dev server + browser context is its own; the web surface runs at full
   concurrency (no shared slot).
2. **Drive the criterion** through the MCP: navigate to the screen, perform the interaction, and read
   the resulting DOM/rendered state. Confirm the *user-visible* outcome the acceptance criterion
   names — the same behavior-first posture as the unit tests, one level up.
3. **Capture evidence** — a screenshot of each web criterion satisfied (and of a boundary rejection
   where that's the criterion), for attachment (below). The evidence is what makes the pass
   inspectable downstream, not just asserted.
4. **Keep it thin.** Verify on the web surface only what needs the rendered page; logic that a unit
   test covers stays at the bottom of the pyramid. The web surface is for the rendered-UI criteria,
   not a second home for logic checks.

**Block, never silently pass — the web-surface rule.** If the dev/preview server won't start, the
page errors on load, or the MCP can't drive the flow, that web criterion is **unverified** — surface
it ("web gate did not run — dev server failed to start / page errored; criterion X unverified"),
never emit a pass for a surface that didn't execute. A false green is the worst signal the self-test
can produce, because the tester's Level-2 and the tech-lead's review both build on top of it.

## Evidence — attach it to the work-item via `scripts/az-attach.sh`

Test evidence (screenshots of satisfied web criteria, saved test-result output) attaches to the Azure
work-item through **`scripts/az-attach.sh`** — the ONE Azure REST carve-out (adapter §9); there is no
MCP tool for attachment upload. The developer already has the env PAT and network the helper needs
(it runs the worker's env PAT via Basic auth, never on the argv, never logged — adapter §9). Invoke it
as `scripts/az-attach.sh <work-item-id> <file> [comment]`.

Why the developer attaches at Level-1 at all: the same reason the tester does at Level-2 — a pass with
no attached proof is a claim, not a verification, and the review gate is evidence-first (`green =
test-gates ∧ review`, tech-lead review-craft). The developer's self-test evidence is the first layer
of that proof; the tester adds the independent Level-2 evidence on top.

## What the developer does NOT do here

- **Not Level-2.** The developer self-tests; the independent thorough verification (edge/regression/
  strategy) is the `tester` worker's job at step 4b. Don't duplicate it — the two levels are
  deliberately separate perspectives.
- **Not the mobile surface.** `area:web` work is verified on code + web; the serialized
  emulator lease is the `mobile` pack's concern. A web work-unit has no mobile gate.
- **Not merge, not Done.** The self-test gates the developer's own `pr` handoff; it does not merge and
  does not set Done — the deterministic engine merges to `dev` after the PR is green (test-gates ∧
  review) and verifies the durable git state, then the Azure→Done transition follows (dispatch worker
  contract §2, adapter merge note). A self-merging worker would violate both NEVER-merge and the
  engine's durable-state verification.

## What travels here, and what doesn't

The *testing craft* — behavior-first queries, the pyramid-at-the-unit-level, block-never-silent-pass,
attach-evidence-via-`az-attach.sh` — is project-agnostic and belongs in this pack; a durable lesson
(e.g. "always run `--run` not the watcher in a worker — the watcher never exits") generalizes here via
`/drain`. The *project's* actual test scripts (if they differ from `npm run test`), its specific
screens and flows, and its coverage bar are project facts the worker reads from `Conventions/`
(brief-named) at runtime — never pre-authored here, and never written to the wiki by the worker
(adapter §8; the tech-lead promotes project facts).

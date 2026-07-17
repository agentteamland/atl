---
knowledge-base-summary: "My second production unit: the /sprint-review deliverable written to the Sprints/Sprint-<n>-Review durable-knowledge page (concept #9, my namespace). Completed vs carryover, per-PBI PR links + test evidence, a deployable dev preview note, actual velocity for the closed sprint, and integration findings (#14). Idempotent upsert into the durable-knowledge store (concept #9). Generic template + checklist."
---

# Sprint Review Report (blueprint)

My second production unit. At `/sprint-review` I write one durable page per closed sprint to the
durable-knowledge store at `Sprints/Sprint-<n>-Review` — the **`project-manager`-owned namespace**
(concept #9). This page is the sprint's outcome of record: what shipped, what didn't and why, the
evidence, and the number that feeds the next plan's velocity. It is the input the PO reads before
deciding what to promote from `dev` to `release`.

The page's *content* is project-specific (this sprint's items, this project's PRs) and is written
**at runtime** — I never pre-author it. What lives here is the reusable craft: the section shape,
where each fact comes from, and the placement contract.

## Where it goes — the placement contract

- **Page path:** `Sprints/Sprint-<n>-Review`, `<n>` = the closed sprint's number resolved at
  runtime (see [iteration-management.md](iteration-management.md) for resolving concrete iteration
  names).
- **Write:** an **idempotent upsert** into the durable-knowledge store (concept #9). Re-running
  `/sprint-review` overwrites the same page rather than appending a duplicate; safe under a
  re-run (concept #10). The store's locator (where the active backend needs one) is read from
  `config.json` (resolved once at `/delivery-init`, cached) — I never re-resolve it.
- **Owner discipline:** `Sprints/` is my namespace and mine alone (concept #9, one owner per
  namespace — no write races). I do **not** write `Domain/`, `Analysis/`, `Architecture/`, or
  `Conventions/` — those belong to the `business-analyst`, the `technical-analyst`, and the
  `tech-lead`. If a sprint surfaced a durable *architecture* fact, I note it in my review page and
  flag it for the `tech-lead` to promote to `Architecture/`; I stay in my lane.
- **First-write check:** listing the durable-knowledge store (concept #9) confirms the `Sprints/`
  namespace is ready before the first write of the project.

## The report's sections (fixed shape, generic content)

The page has a stable set of H2s so the PO — and a future me — read it back by location, the same
deterministic-read-back principle as the analysis contracts (concepts #2/#3).

### `## Completed`
Every item that reached the Completed state this sprint. Resolve the category at runtime
(concept #7) — never filter on the literal `"Done"`. One row per item:
id, title, story points. This is the honest "what shipped" list.

### `## Carryover`
Every admitted item that did **not** complete, with the reason (blocked on a dependency, ran out
of sprint, review not passed). Each **carries to the next sprint as top priority** (workable) or is
**surfaced-but-not-workable** (blocked) — I never silently drop them, and unfinished work is never
bumped by newer work (see [reject-and-carryover.md](reject-and-carryover.md)). Listing carryover
explicitly is what keeps the backlog honest and the next velocity mean correct. A `blocked` entry
may be one the dispatch engine surfaced: `/sprint-review`'s step 2 drains any
`.delivery/blocked/<id>.json` report, reflects it onto the work-item, and feeds it here with its
diagnostic — so a crashed/stalled unit is listed too, not just a dependency-blocked one.

### `## Per-item evidence`
For each completed PBI (or task, per the sprint's granularity): the **PR link** and the **test
evidence**. I gather these read-only:
- **PR links** — read the item's progress/PR comment (concept #3), and the work-item↔PR link
  created at the micro-loop's PR step (concept #11). The PR read (concept #11) resolves PR details
  if I need title/status.
- **Test evidence** — the `tester`'s verification result and any attached screenshots/result
  files. Attachments are read back via the active adapter (concept #12 — the upload is the
  `tester`'s job, not mine). I link or summarize the evidence; I do not re-run tests.

### `## Deployable dev preview`
A note on the state of the integration branch (`config.branchPair.dev`) at
sprint end: is `dev` green and deployable? This is the "here is what you can look at" pointer for
the PO's promotion decision. I read the branch/build state read-only through the active backend's
adapter (and pipeline/build status if the project runs one); I do not deploy or promote —
**promotion to `release` is the PO's sprint-approval decision**, not mine.

### `## Actual velocity`
The story points *actually completed* this sprint (the `## Completed` sum). This is the number
that feeds the next sprint's velocity mean ([capacity-and-velocity.md](capacity-and-velocity.md)).
Recording it on the review page makes the velocity history auditable — the next plan's ceiling is
traceable to concrete closed sprints, not a hidden running total.

### `## Integration findings` (#14)
Cross-cutting observations from stitching the sprint's work together: integration friction, a
convention that emerged, a dependency that was mis-ordered, a repeated review finding. This is the
sprint's *retrospective signal*. Project-specific findings that are durable knowledge get flagged
for the owning role to promote to their durable-knowledge namespace (architecture → `tech-lead`'s
`Architecture/`; domain → `business-analyst`'s `Domain/`). Role-craft lessons I learn (a better
way to schedule, a capacity mis-estimate pattern) route to my *own* `children/` via `/drain`, not
to the durable-knowledge store — the two-layer split (brief §5).

## Generic template

```markdown
# Sprint <n> Review

_Sprint <n> · <iteration-name> · closed <date>_

## Completed
| Id | Title | Points |
|----|-------|--------|
| #<id> | <title> | <pts> |
| …    | …     | …    |
**Total completed: <sum> points**

## Carryover
| Id | Title | Points | Reason |
|----|-------|--------|--------|
| #<id> | <title> | <pts> | <blocked-on / out-of-time / review-not-passed> |

## Per-item evidence
### #<id> — <title>
- PR: <pr-link>
- Tests: <tester verdict + evidence link>

## Deployable dev preview
- `dev` branch (`<branchPair.dev>`): <green/red> — <deployable? build/pipeline state>

## Actual velocity
- Completed this sprint: **<sum> points** (feeds the velocity window)

## Integration findings
- <cross-cutting observation>
- <flagged for tech-lead → Architecture/ | business-analyst → Domain/ …>
```

## Completion checklist

- [ ] the durable-knowledge store's locator read from `config.json` (not re-resolved); `Sprints/`
      namespace readiness confirmed (concept #9) before first write.
- [ ] Page written to exactly `Sprints/Sprint-<n>-Review` with `<n>` resolved at runtime.
- [ ] `## Completed` lists only items at the runtime-resolved Completed category (never literal
      `"Done"`); Done set read to exhaustion ("list means all", concept #10).
- [ ] `## Carryover` names every admitted-but-incomplete item with its reason — nothing silently
      dropped; each carries to the next sprint (top priority if workable, surfaced if blocked).
- [ ] `## Per-item evidence` has a PR link + test evidence per completed item (attachments read
      via the active adapter, concept #12; I read, I don't re-test).
- [ ] `## Deployable dev preview` reports `dev` state read-only; **no promotion** (PO owns that).
- [ ] `## Actual velocity` = the `## Completed` point sum, for the velocity history.
- [ ] `## Integration findings` captured; durable project facts flagged to their owning role's
      namespace, not written by me outside `Sprints/`.
- [ ] Written as an idempotent upsert into the durable-knowledge store (concept #9), wrapped in
      adapter backoff (the resilience policy).

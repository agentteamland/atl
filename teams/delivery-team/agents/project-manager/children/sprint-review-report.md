---
knowledge-base-summary: "My second production unit: the /sprint-review deliverable written to the Sprints/Sprint-<n>-Review wiki page (adapter §8, my namespace). Completed vs carryover, per-PBI PR links + test evidence, a deployable dev preview note, actual velocity for the closed sprint, and integration findings (#14). Idempotent upsert with wiki_create_or_update_page. Generic template + checklist."
---

# Sprint Review Report (blueprint)

My second production unit. At `/sprint-review` I write one durable page per closed sprint to the
project wiki at `Sprints/Sprint-<n>-Review` — the **`project-manager`-owned namespace** (adapter
§8). This page is the sprint's outcome of record: what shipped, what didn't and why, the evidence,
and the number that feeds the next plan's velocity. It is the input the PO reads before deciding
what to promote from `dev` to `release`.

The page's *content* is project-specific (this sprint's items, this project's PRs) and is written
**at runtime** — I never pre-author it. What lives here is the reusable craft: the section shape,
where each fact comes from, and the placement contract.

## Where it goes — the placement contract

- **Page path:** `Sprints/Sprint-<n>-Review`, `<n>` = the closed sprint's number resolved at
  runtime (see [iteration-management.md](iteration-management.md) for resolving concrete iteration
  names).
- **Tool:** `wiki_create_or_update_page` — an **idempotent upsert** (adapter §8). Re-running
  `/sprint-review` overwrites the same page rather than appending a duplicate; safe under a
  re-run (adapter §5). The `wikiId` is read from `config.json` (resolved once at `/delivery-init`,
  cached) — I never re-resolve it.
- **Owner discipline:** `Sprints/` is my namespace and mine alone (adapter §8, one owner per
  namespace — no write races). I do **not** write `Domain/`, `Analysis/`, `Architecture/`, or
  `Conventions/` — those belong to the `business-analyst`, the `technical-analyst`, and the
  `tech-lead`. If a sprint surfaced a durable *architecture* fact, I note it in my review page and
  flag it for the `tech-lead` to promote to `Architecture/`; I stay in my lane.
- **First-write check:** `wiki_list_pages` confirms the `Sprints/` namespace exists before the
  first write of the project (adapter §8).

## The report's sections (fixed shape, generic content)

The page has a stable set of H2s so the PO — and a future me — read it back by location, the same
deterministic-read-back principle as the analysis contracts (adapter §7).

### `## Completed`
Every item that reached the Completed state-category this sprint. Resolve the category at runtime
(`wit_get_work_item_type`, adapter §6) — never filter on the literal `"Done"`. One row per item:
id, title, StoryPoints. This is the honest "what shipped" list.

### `## Carryover`
Every admitted item that did **not** complete, with the reason (blocked on a dependency, ran out
of sprint, review not passed). These return to the backlog for the next `/sprint-plan` — I never
silently drop them (see [reject-and-carryover.md](reject-and-carryover.md)). Listing carryover
explicitly is what keeps the backlog honest and the next velocity mean correct. A `blocked` entry
may be one the dispatch engine surfaced: `/sprint-review`'s step 2 drains any
`.delivery/blocked/<id>.json` report, reflects it onto the work-item, and feeds it here with its
diagnostic — so a crashed/stalled unit is listed too, not just a dependency-blocked one.

### `## Per-item evidence`
For each completed PBI (or task, per the sprint's granularity): the **PR link** and the **test
evidence**. I gather these read-only:
- **PR links** — `wit_list_work_item_comments` for the item's progress/PR comment, and the
  work-item↔PR link created by `wit_link_work_item_to_pull_request` at the micro-loop's PR step.
  The `repo_*` tools resolve PR details if I need title/status.
- **Test evidence** — the `tester`'s verification result and any attached screenshots/result
  files. Attachments are read back via `wit_get_work_item_attachment` (the MCP read leg; only the
  *upload* leg is REST, adapter §9 — and the upload is the `tester`'s job, not mine). I link or
  summarize the evidence; I do not re-run tests.

### `## Deployable dev preview`
A note on the state of the integration branch (`config.branchPair.dev`, adapter/config) at
sprint end: is `dev` green and deployable? This is the "here is what you can look at" pointer for
the PO's promotion decision. I read the branch/build state read-only (`repo_get_branch_by_name`,
and pipeline/build status if the project runs one); I do not deploy or promote — **promotion to
`release` is the PO's sprint-approval decision**, not mine.

### `## Actual velocity`
The story points *actually completed* this sprint (the `## Completed` sum). This is the number
that feeds the next sprint's velocity mean ([capacity-and-velocity.md](capacity-and-velocity.md)).
Recording it on the review page makes the velocity history auditable — the next plan's ceiling is
traceable to concrete closed sprints, not a hidden running total.

### `## Integration findings` (#14)
Cross-cutting observations from stitching the sprint's work together: integration friction, a
convention that emerged, a dependency that was mis-ordered, a repeated review finding. This is the
sprint's *retrospective signal*. Project-specific findings that are durable knowledge get flagged
for the owning role to promote to their wiki namespace (architecture → `tech-lead`'s
`Architecture/`; domain → `business-analyst`'s `Domain/`). Role-craft lessons I learn (a better
way to schedule, a capacity mis-estimate pattern) route to my *own* `children/` via `/drain`, not
to the project wiki — the two-layer split (brief §5).

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

- [ ] `wikiId` read from `config.json` (not re-resolved); `Sprints/` namespace existence confirmed
      (`wiki_list_pages`) before first write.
- [ ] Page written to exactly `Sprints/Sprint-<n>-Review` with `<n>` resolved at runtime.
- [ ] `## Completed` lists only items at the runtime-resolved Completed category (never literal
      `"Done"`); Done set read to exhaustion ("list means all", adapter §4).
- [ ] `## Carryover` names every admitted-but-incomplete item with its reason — nothing silently
      dropped; each returns to the backlog.
- [ ] `## Per-item evidence` has a PR link + test evidence per completed item (attachments read
      via `wit_get_work_item_attachment`; I read, I don't re-test).
- [ ] `## Deployable dev preview` reports `dev` state read-only; **no promotion** (PO owns that).
- [ ] `## Actual velocity` = the `## Completed` point sum, for the velocity history.
- [ ] `## Integration findings` captured; durable project facts flagged to their owning role's
      namespace, not written by me outside `Sprints/`.
- [ ] Written via `wiki_create_or_update_page` (idempotent upsert), wrapped in adapter backoff
      (adapter §3).

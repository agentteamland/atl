---
knowledge-base-summary: "The primary production unit: at /refine I break an analyzed Feature into PBIs/Tasks and record a durable decomposition plan (a manifest on the parent) with STABLE plan-ordinals that feed the atl-key idempotency hash (adapter §5), stamp each unit with an area:<name> System.Tags tag (adapter §7 — I own area→pack binding), and add Dependency links so the DAG scheduler can order the work. Includes the read-in contract, the ordinal-stability rules, and a completion checklist."
---

# Decomposition Blueprint

This is my primary production unit. Given an analyzed Feature — a business analysis in its
`System.Description` and a technical analysis in its `**[Technical Analysis]**` comment — I
produce the work the developers will build: a set of PBIs and Tasks, each idempotently keyed,
area-tagged, and dependency-linked, plus a durable **decomposition plan** that makes the whole
thing re-runnable. I run as a `subagent` inside the `/refine` ceremony (which stone #6 owns; I
describe my contribution, not the orchestration).

Everything downstream depends on getting this right: the `project-manager` schedules over the
Dependency links I add, the `developer` loads the knowledge-pack keyed by the area tag I apply,
and the idempotency of a re-run rests on the plan-ordinals I assign. A sloppy decomposition is
not a local error — it corrupts scheduling, pack-binding, and resumability at once.

## What I read in first (the analysis read-back contract)

Before I decompose anything I load the analysis **by location, never by guessing** (adapter §7):

1. `wit_get_work_item` on the Feature → parse the `System.Description` Markdown under its fixed
   H2s: `## Problem`, `## Business Value`, `## Scope`, `## Acceptance Criteria`, `## Out of
   Scope`. This is the business "what & why" — my units must trace back to the Acceptance Criteria
   and stay inside Scope.
2. `wit_list_work_item_comments` → find the comment whose **first line is the exact sentinel**
   `**[Technical Analysis]**` (a **sentinel match**, not "the newest comment" — a later human
   comment must never shadow the analysis). Parse its H2s: `## Approach`, `## Feasibility &
   Risks`, `## NFRs`, `## Dependencies`, `## Suggested Areas`. This is the technical shape and,
   crucially, the `technical-analyst`'s **suggested** areas — suggestions I now decide on.

If either artifact is missing or the sentinel doesn't match, I do not invent the analysis — I
stop and surface it (the Feature is not ready to refine). The `business-analyst` owns the
Description, the `technical-analyst` owns the sentinel comment; I am the first consumer of both.

## The decomposition plan — a durable manifest, the source of idempotency

The single most important artifact I produce is not the work-items — it is the **decomposition
plan**: an ordered list of the units I *intend* to create, recorded durably so a re-run
converges instead of duplicating. I write it as a manifest on the parent Feature (a plan block in
a labeled comment, and — when the decomposition is architecturally significant — mirrored to the
`Architecture/` wiki page for the area; see [architecture-and-adr.md](architecture-and-adr.md)).

Each planned unit gets a **stable plan-ordinal** — a small integer that identifies the unit's
*position in the plan*, not its title and not a per-run id. The ordinal is the load-bearing part:

```
Feature #1234 — decomposition plan (v1)
  ordinal 1  PBI   "Authentication surface"          area:auth
  ordinal 2    Task  "Session lifecycle"             area:auth      depends-on: 1
  ordinal 3    Task  "Credential validation path"    area:auth      depends-on: 1
  ordinal 4  PBI   "Account settings surface"        area:profile
  ordinal 5    Task  "Settings read/write"           area:profile   depends-on: 4
```

(Generic example — the areas/titles are illustrative; real ones come from the analysis.)

### Why ordinals, not titles or GUIDs

The idempotency key (adapter §5) is `atl-key:<hash>` where `hash = hash(parent-id +
plan-ordinal)`. This choice is deliberate and I must protect its premise:

- **Not a per-run GUID/timestamp** — that would make every re-run mint a *new* key and duplicate
  every unit. Ordinals are stable across runs, so the same logical unit maps to the same key:
  that is what makes resume *convergent*, not merely dedup-attempted.
- **Not the title** — two units can share a title (distinct ordinals keep them from colliding),
  and a title edit during re-plan must NOT break convergence (the ordinal stays put, so the
  existing item is found and updated rather than re-created).

**Ordinal-stability rules (the discipline that keeps re-runs safe):**
- Once assigned in the plan, an ordinal **never** changes for a unit that still exists.
- Removing a unit **retires** its ordinal — I do not backfill or renumber. Renumbering would
  re-key surviving units and orphan/duplicate them.
- Adding a unit on re-plan gets a **fresh higher ordinal**, never a reused one.
- I bump a plan **version** (`v1` → `v2`) when I re-plan, so the manifest records history; the
  ordinals themselves are append-only within the Feature.

## Creating the units — check-first, then stamp (adapter §5)

For every planned unit, in ordinal order, I follow the stamp + check-before-create protocol from
the [Azure adapter](../../../backends/azure/adapter.md) (§5) so a re-run never duplicates:

1. Compute `atl-key = hash(parent-id + plan-ordinal)`.
2. **Check-first WIQL** — `wit_query_by_wiql` filtered to that `atl-key` tag.
   - **Found** → reuse + update the existing item (converge it to the intended state: title,
     description, links, area tag). Do NOT create a second one.
   - **Not found** → create it (`wit_create_work_item`, or `wit_add_child_work_items` to place it
     under the parent), then **stamp** `System.Tags` with `atl-key:<hash>` +
     `atl-run:<ceremony>:<sprint-id>` (provenance) as close to atomic as the API allows. A
     **409/duplicate on create is caught and resolved to the existing item**, not surfaced.
3. Apply the **area tag** (below) and the **Dependency links** (below).

I resolve the concrete work-item **type** at runtime — the `artifactHierarchy` in
[config-and-methodology.md](../../../knowledge/config-and-methodology.md) is the abstract ladder
(Epic → Feature → PBI → Task), but the real Azure type name is process-template-dependent
(`Product Backlog Item` vs `User Story`, etc.), so I resolve it via `wit_get_work_item_type`
(adapter §6) and **never hardcode a literal type or state string**.

## Area tagging — I own area→pack binding (adapter §7)

The `technical-analyst` only *suggests* areas (under `## Suggested Areas`); **I decide and
apply** them, because the area tag binds a unit to the knowledge-pack the `developer` will load
(`packs/<area>/`, stone #5). I write each unit's area to `System.Tags` in the exact
`area:<name>` convention.

Discipline for good area binding:
- **One primary area per unit.** A unit that genuinely spans two areas is usually two units — a
  smell that the decomposition is too coarse. If it truly must span, tag the dominant area and
  note the cross-area concern in the unit's description and a Dependency link.
- **Areas are project-shaped, not stack-shaped.** `area:auth`, `area:reporting`,
  `area:notifications` — a functional slice of *this system*. I keep the area vocabulary stable
  across a project (I own the `Architecture/` page that lists the areas), so the same
  `packs/<area>/` binds consistently sprint over sprint.
- **The suggested areas are input, not law.** If the analyst's suggested split doesn't match the
  system's real module boundaries (which I own — see architecture-and-adr.md), I re-slice and
  record why on the `Architecture/` page.

## Dependency links — the edges the scheduler orders over

The `project-manager`'s DAG scheduling and `atl work dispatch`'s worktree ordering both consume
the **Dependency links** I add between units (`wit_work_items_link`, a `Dependency` link).
Getting these right is what lets independent units run in parallel and prevents a worker from
building on a not-yet-merged sibling.

- Link a unit to a **prerequisite** only when it genuinely cannot start (or cannot pass review)
  until the prerequisite is merged — usually a shared surface, a schema, or a contract another
  unit produces.
- **No cycles.** A dependency cycle deadlocks the scheduler. If two units mutually depend, they
  are one unit or the boundary is wrong — I re-decompose rather than link a cycle.
- **Fewer edges is better.** Every unnecessary Dependency serializes work that could have run in
  parallel and slows the whole sprint. I add an edge only when the prerequisite is real.
- Parent/child (`Feature → PBI → Task`) is a *containment* link, not a *scheduling* edge —
  containment comes from `wit_add_child_work_items`; ordering comes from `Dependency`.

## Sizing — decompose to a worker-sized unit

Each Task should be a unit a single `developer` worker can implement, self-test, and take through
review in one isolated worktree. If a unit is too large to reason about as one PR it is too large
to key well — I split it (new ordinals), which also improves parallelism. If it's trivially
small, I fold it up, so the plan isn't noise. The right grain is "one coherent change, one PR,
one review."

## Completion checklist (run before I hand the plan to `/refine`)

- [ ] Read the Feature `System.Description` (fixed H2s) **and** the `**[Technical Analysis]**`
      sentinel comment (sentinel match, not newest comment) — both present, else stop + surface.
- [ ] Every unit traces to an Acceptance Criterion and stays inside Scope; nothing in Out of Scope.
- [ ] Decomposition plan recorded durably on the parent (manifest), with a plan **version**.
- [ ] Every unit has a **stable plan-ordinal**; retired ordinals not reused; new units get fresh
      higher ordinals; no renumbering of surviving units.
- [ ] For each unit: `atl-key = hash(parent-id + ordinal)` computed; **check-first WIQL** run;
      found → reuse+update, not-found → create-then-stamp; 409 resolved to existing.
- [ ] `atl-run:<ceremony>:<sprint-id>` provenance tag stamped alongside `atl-key`.
- [ ] Concrete type resolved at runtime (`wit_get_work_item_type`); no hardcoded type/state literal.
- [ ] Each unit tagged `area:<name>` (I decide; analyst only suggested); one primary area per unit.
- [ ] Dependency links added only for real prerequisites; **no cycles**; parent/child is
      containment, not a scheduling edge.
- [ ] Each Task is worker-sized (one coherent change, one PR, one review).

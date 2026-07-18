---
name: refine
description: /refine — the delivery-team's backlog-refinement ceremony. Reads analysis back by location (spec-field H2s + the [Technical Analysis] sentinel comment) for the Features/PBIs in scope, then adopts the business-analyst → technical-analyst → tech-lead subagents sequentially in one shared context to groom the backlog and decompose analyzed Features into keyed, area-tagged, dependency-linked work-units, recording a durable decomposition plan (the stable plan-ordinals that key idempotency) and a per-unit canonical brief. Writes PBI/Task items, area:<name> tags, the plan manifest + briefs, and Analysis/Architecture/Conventions durable-knowledge pages — all through the active backend's adapter. Recurring; runs between kickoff and sprint-plan.
---

# /refine — backlog refinement and decomposition

This is the delivery-team's **recurring grooming ceremony**: it takes the analyzed backlog and
makes it *plannable*. It reads each in-scope Feature/PBI's analysis back **by location** (the
`business-analyst`'s spec-field headings and the `technical-analyst`'s
`**[Technical Analysis]**` sentinel comment), re-grooms the business layer, then decomposes the
analyzed Features into the keyed, area-tagged, dependency-linked **work-units** the
`project-manager` will select from at `/sprint-plan` and `atl work dispatch` will build. It sits
after `/kickoff` (which seeds the analyzed backlog) and before `/sprint-plan` (which selects from
the refined units).

Unlike `/kickoff`'s live human dialogue, `/refine` runs its roles as **subagents adopted
sequentially in this session's shared context** (see [When to run](#when-to-run) and each step) —
it spawns no `developer`/`tester` workers; only `/sprint-start` hands work to `atl work dispatch`.

| Reads (by location) | Writes |
|---|---|
| Feature/PBI spec-field H2s (`## Problem` … `## Out of Scope`); the `**[Technical Analysis]**` sentinel comment (`## Approach` … `## Suggested Areas`); the `Domain/` + `Analysis/` durable-knowledge pages | PBI/Task work-units; `area:<name>` tags (concept #4); dependency links (concept #8); the durable **decomposition plan** manifest + per-unit **canonical briefs**; the `Analysis/`, `Architecture/`, `Architecture/ADR/`, `Conventions/` durable-knowledge pages |

Field semantics for the config the ceremony reads live in
[`config-and-methodology.md`](../../knowledge/config-and-methodology.md); the operation map,
content-placement, and idempotency rules live in the provider-neutral
[backend interface](../../knowledge/backend-interface.md), which the active backend's adapter
([`backends/<backend>/adapter.md`](../../backends)) binds to concrete tools; the pack contract the
area tags bind to lives in [`pack-format.md`](../../knowledge/pack-format.md). This skill orchestrates the roles; the
role-craft lives in each role-agent's `children/`, cited per step.

## When to run

- **Recurring** — `/refine` is a grooming ceremony, not a cadence ceremony. It has no
  `methodology.cadence` slot of its own (the `cadence.planningCeremonies` are `sprint-plan` +
  `sprint-start`, and `cadence.reviewCeremony` is `sprint-review`); run it **between** cycles
  whenever the analyzed backlog needs grooming and Features need decomposing into plannable units —
  typically ahead of `/sprint-plan`, so the `project-manager` selects from refined, keyed units.
- **Scope** — a Feature, a set of Features, an Epic's Features, or the top of the backlog. The
  scope selects which analyzed items this run reads back, grooms, and decomposes.
- **Re-run** — `/refine` is idempotent (see [Idempotent re-run](#idempotent-re-run)); re-running
  after a crash, a partial run, or an explicit re-plan **converges** the same units, never
  duplicating them.

## Procedure

Confirm `.delivery/config.json` and `.delivery/methodology.json` exist (written by
`/delivery-init`). Read `config.json` — the backend's coordinates (Azure `org`/`project`/`repo`; GitHub `owner`/`repo`/`projectNumber` — see [`config-and-methodology.md`](../../knowledge/config-and-methodology.md) §2), `branchPair` (the **authoritative**
dev/release names; config wins over `methodology.branches`), the selected `backend` (default
`azure`, which selects the active adapter), and the cached durable-knowledge-store handle the
active adapter needs (on the Azure backend, `wikiId`). Config is read-only to this ceremony. Resolve
the scope (which Features/PBIs to refine) with the user, then work the steps below. Every backend
touch names a real operation from the active backend's adapter operation map
(`backends/<backend>/adapter.md`); never invent a tool name.

> **Durable-knowledge locator is backend-specific:** `wikiId` is Azure-only — the GitHub backend's
> in-repo `/docs` store needs no such handle (see [`config-and-methodology.md`](../../knowledge/config-and-methodology.md)
> §2). The coordinate fields are now read per-backend; unifying the two config shapes into one
> neutral schema remains a tracked follow-up.

### 1. Read the analysis back — by location, never by guessing

**Deterministic read-back (concept #2/#3).** For each Feature/PBI in scope:

- Read the work-item (concept #2) → parse the spec-field Markdown under its fixed H2s
  (`## Problem`, `## Business Value`, `## Scope`, `## Acceptance Criteria`, `## Out of Scope`) —
  the business "what & why".
- List the comments (concept #3) → locate the technical analysis by **sentinel match**: the comment
  whose **first line is the exact `**[Technical Analysis]**` sentinel** — **never "the newest
  comment"**, so a later human comment can't shadow the analysis. Parse its H2s (`## Approach`,
  `## Feasibility & Risks`, `## NFRs`, `## Dependencies`, `## Suggested Areas`).
- Read the relevant durable-knowledge context (concept #9): the `Domain/` pages and any `Analysis/`
  page for the item. Use a store search / listing (concept #9) to discover a page whose path isn't
  pre-named.

If a Feature in scope is missing its spec-field headings or its sentinel comment, it is **not
ready to refine** — do not invent the analysis; stop and surface it. Prefer batched reads over a
loop of singles (the "list means all" rule, concept #10 — never silently truncate a list; treat a
query result at the cap as a truncation error).

### 2. Groom the business layer — as the `business-analyst`

Acting as the `business-analyst` (read [`../../agents/business-analyst/agent.md`](../../agents/business-analyst/agent.md)
and its `children/`, chiefly [`refine-participation.md`](../../agents/business-analyst/children/refine-participation.md)),
building on the read-back from step 1 **in this shared context**:

- **Sharpen the acceptance criteria** against real feedback — tighten vague criteria, add missing
  negative/boundary cases, remove non-conditions — and write them back into `## Acceptance
  Criteria` via an **idempotent update in place** (update the work-item, concept #2 spec field;
  idempotency, concept #10); never create a duplicate item.
- **Split oversized items along business seams** (independent slices of value — not technical
  seams, which are the tech-lead's cut in step 4). Each split child gets its own full five-H2
  spec field whose value ladders up to the parent, and follows the idempotency contract of step 4
  (check-first idempotency query + `atl-key` stamp on stable `parent + ordinal`, concept #10).
- **Keep `Domain/` and the business half of `Analysis/` current** where understanding deepened —
  an idempotent durable-knowledge upsert (concept #9); read-before-write, one owner, no
  write race. The `business-analyst` owns `Domain/` and co-owns `Analysis/` with the
  `technical-analyst`.

The `business-analyst` **does not** apply `area:<name>` tags and **does not** write the
`**[Technical Analysis]**` comment — those are neighbors' lanes (concept #3/#4).

### 3. Revisit the technical layer — as the `technical-analyst`

Then, still in the same context, acting as the `technical-analyst` (read
[`../../agents/technical-analyst/agent.md`](../../agents/technical-analyst/agent.md) and its
`children/`), building on the business grooming just done:

- Where a scope split or a sharpened criterion **changed what's feasible**, revise the
  `**[Technical Analysis]**` comment via the analyst's add-only convergence — the comment channel is
  **append-only by contract** (concept #3). List the comments (concept #3)
  and sentinel-match: if the analysis genuinely changed, add **one** fresh sentinel comment
  (add a comment, concept #3) that supersedes the earlier one, keeping the exact sentinel first line
  and the five fixed H2s (concept #3); if it is unchanged, do not re-add. Read-back always resolves
  the analysis by sentinel, so the latest sentinel comment *is* the analysis — never a second
  spec field; never "the newest comment".
- Keep the **`## Suggested Areas`** current — candidates only, prose only. The `technical-analyst`
  **suggests**; it never writes an `area:<name>` tag (that binding is the tech-lead's, step 4 —
  concept #4 and [`suggesting-areas.md`](../../agents/technical-analyst/children/suggesting-areas.md)).
- Record real technical **dependencies** as dependency links (concept #8), not
  just prose, so the scheduler's DAG is machine-sound.
- Co-author the **technical half** of the item's `Analysis/` page where the analysis exceeds a
  comment (an idempotent durable-knowledge upsert, concept #9; own section only, cross-link where the
  layers meet).
- Resolve any state/type reference at runtime (concept #7) — never a
  hardcoded literal.

### 4. Decompose into keyed, area-tagged, dependency-linked units — as the `tech-lead`

Then, still in the same context, acting as the `tech-lead` (read
[`../../agents/tech-lead/agent.md`](../../agents/tech-lead/agent.md) and its `children/`, chiefly
[`decomposition-blueprint.md`](../../agents/tech-lead/children/decomposition-blueprint.md)),
consuming the analysts' just-produced output. This is the ceremony's core write.

1. **Record the decomposition plan first — the idempotency substrate.** For each Feature being
   decomposed, produce an ordered list of the units you *intend* to create and record it
   **durably** as a plan manifest — a labeled comment on the parent Feature (add a comment,
   concept #3), mirrored to the area's `Architecture/` durable-knowledge page when the
   decomposition is architecturally significant (an upsert, concept #9). Give each intended
   unit a **stable plan-ordinal** (a small integer identifying its position in the plan — **not**
   its title, **not** a per-run GUID). Bump a plan **version** on re-plan; ordinals are
   append-only within the Feature (retired, never renumbered or reused). The plan-ordinal is the
   `atl-key` substrate step 5 depends on.
2. **Create the units — check-first, then stamp (concept #10).** For each planned unit, in ordinal
   order:
   - Compute `atl-key = hash(parent-id + plan-ordinal)`.
   - **Check-first idempotency query** (concept #10) filtered to that `atl-key` tag: **found →** reuse +
     update the existing item to the intended state (title, description, links, area) — never a
     second one; **not-found →** first run the brainstorm-provenance adoption check (next bullet); only
     if that too finds nothing do you create it (concept #1 — create the work-item, or nest it under
     the parent), then **stamp** its tags (concept #4) with `atl-key:<hash>` +
     `atl-run:refine:<sprint-or-scope>` (provenance), as close to atomic as the API allows. A
     409/duplicate on create is **caught and resolved to the existing item**, never surfaced.
   - **Adopt a brainstorm-sourced item in place — never duplicate it (concept #10).** A backlog item
     created by `/brainstorm done`'s board-sync carries the provenance label `atl-brainstorm:<slug>`
     but **no `atl-key`** (a brainstorm item has no parent/plan-ordinal), so the `atl-key` check-first
     above misses it. When a planned unit *is* such an in-scope item, run a check-first query filtered
     to `atl-brainstorm:<slug>` via the adapter (concept #10) and, on a title match to the planned
     unit, **adopt** the existing item — update it in place and stamp it with the computed
     `atl-key:<hash>` (+ `atl-run` provenance) — rather than creating a parallel unit. Adoption is
     one-time: once stamped, every later re-run converges through the normal `atl-key` check-first.
   - Resolve the concrete work-item **type** at runtime (concept #7) (the
     `artifactHierarchy` `Epic → Feature → PBI → Task` is abstract; the real backend type name is
     model-dependent) — **never** hardcode a type/state literal (concept #7).
3. **Apply the area tags — the tech-lead decides (concept #4).** Write each unit's `area:<name>`
   as a tag (concept #4). The `technical-analyst` only *suggested* areas under
   `## Suggested Areas`; the tech-lead **decides** them, because the tag *is* the pack binding — a
   `developer` loads exactly `packs/<area>/` for the tagged area (see
   [`pack-format.md`](../../knowledge/pack-format.md)). One primary area per unit; keep the area
   vocabulary stable on the `Architecture/` page.
4. **Add the dependency links — the edges the scheduler orders over.** Add a dependency link
   (concept #8) between units only for a real prerequisite (a shared surface, schema, or
   contract another unit produces); **no cycles**; parent/child containment (concept #1) is *not* a
   scheduling edge. Fewer edges is better — an unnecessary edge serializes work that could run in
   parallel.
5. **Promote worker-surfaced and analysis-durable project facts to the durable-knowledge store**
   (tech-lead write-lane, concept #9): fold the durable parts of the analysis into the
   `Architecture/` page (current-truth upsert of system shape, module boundaries, area vocabulary)
   and write an **ADR** at `Architecture/ADR/ADR-<n>-<slug>` **only** for a decision that is
   significant *and* hard-to-reverse (a reversed decision → a new ADR + supersede the old, never an
   in-place edit — see
   [`architecture-and-adr.md`](../../agents/tech-lead/children/architecture-and-adr.md)). Keep
   `Conventions/` current for project rules layered atop the pack's generics. All via idempotent
   durable-knowledge upserts (concept #9), verified against a store listing (concept #9) before a
   first write. Workers never write the store — the tech-lead promotes.

### 5. Write the per-unit canonical brief — as the `tech-lead`

Still as the `tech-lead`, for each work-unit created in step 4, write its **canonical brief** — the
artifact that lets a fresh, isolated `developer` worker load the right project knowledge without
carry-over (see [`canonical-brief.md`](../../agents/tech-lead/children/canonical-brief.md)). Record
each brief durably on its unit as a **single labeled comment** (add a comment, concept #3) whose
**first line is the exact sentinel `**[Canonical Brief]**`** — the same machine-locatable placement
the `**[Technical Analysis]**` comment uses (concept #3) — keyed to its `atl-key` so a re-run updates
in place. A brief bounds context; it does not dump it:

- The unit's **goal** restated in one or two sentences, traced to the Feature's Acceptance Criteria.
- The **area** (`area:<name>`) — binds the knowledge-pack (`packs/<area>/`).
- The **exact** `Architecture/` slice + `Conventions/` page paths for the unit's area (concept #9
  read contract) — specific paths the worker pulls from the durable-knowledge store (a store search
  is the fallback when a path isn't pre-named), never "read the whole store"; reference any
  constraining ADR by number.
- The **dependencies** (the dependency-linked prerequisites, concept #8) with "build against, don't
  re-declare" guidance.
- The **test-evidence expectation** — code + web + mobile-emulator evidence where the surface
  applies — so the worker knows the review gate ahead of time.

Leave out whole-store dumps, other units' internals, stack how-to (the pack's job), and methodology
mechanics. No `developer`/`tester` worker is spawned here — `/refine` only prepares the units and
briefs; `/sprint-start` hands the selected, briefed units to `atl work dispatch`.

### 6. Report

Summarize what the run refined and decomposed: the Features groomed, the work-units created (with
their `area:<name>` tags), the dependency links added (concept #8), the plan-manifest version, the canonical
briefs written, and any `Analysis/`/`Architecture/`/`ADR`/`Conventions/` durable-knowledge pages
touched. Note any Feature that was **not ready** (missing spec-field headings or sentinel comment)
so it can be analyzed before the next `/refine`. Point the user to `/sprint-plan` as the next step.

## Idempotent re-run

A re-run of `/refine` (after a crash, a partial run, or an explicit re-plan) must **never duplicate
work-units or double-write analysis** — the active backend's tags are the source of truth; there is
no local ledger (concept #10).

- **Every created unit carries two tags (concept #4):** `atl-run:refine:<sprint-or-scope>`
  (provenance) and `atl-key:<hash>` where `hash = hash(parent-id + plan-ordinal)`. The tech-lead's
  recorded **decomposition plan** gives each intended unit a **stable plan-ordinal** — not a per-run
  GUID, not `hash(title)` — so the same logical unit maps to the same key across runs (that is what
  makes resume *convergent*, not merely dedup-attempted). A **title edit must NOT mint a new key**
  (the ordinal is stable); two units sharing a title don't collide (distinct ordinals).
- **Before any create, run a check-first idempotency query** (concept #10) for that `atl-key`:
  **found → reuse + update** (converge to the intended state), **not-found → create-then-stamp**. A
  409/duplicate on create is caught and resolved to the existing item, never surfaced.
- **Brainstorm-sourced items are adopted, not duplicated.** An item created by `/brainstorm done`
  carries `atl-brainstorm:<slug>` and no `atl-key`; the first time it enters a decomposition plan the
  `atl-key` check-first misses it, so refine runs a second check-first on `atl-brainstorm:<slug>` and
  **adopts** it in place (stamp the computed `atl-key`) — a one-time bridge from the brainstorm-
  provenance key to the loop's `atl-key`, after which convergence is the normal `atl-key` path.
- **The spec-field / durable-knowledge writes are idempotent updates/upserts** — updating the
  work-item (concept #2) writes the spec field in place and a durable-knowledge upsert (concept #9)
  is idempotent. Comments are add-only by contract (concept #3): a re-run first **sentinel-matches**
  the existing `**[Canonical Brief]**` comment (list the comments, concept #3, not "the newest
  comment") — found → add **one** fresh superseding comment keyed to its unit's `atl-key`; not-found
  → add the first. The sentinel is the read-back locator; the `atl-key` is the convergence guard.
  Never append an uncontrolled second brief.
- **Ordinal-stability rules:** once assigned, an ordinal never changes for a surviving unit; a
  removed unit's ordinal is retired (never backfilled or reused); a re-plan adds units at fresh
  higher ordinals and bumps the plan version.

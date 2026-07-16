---
name: refine
description: /refine — the delivery-team's backlog-refinement ceremony. Reads analysis back by location (Description H2s + the [Technical Analysis] sentinel comment) for the Features/PBIs in scope, then adopts the business-analyst → technical-analyst → tech-lead subagents sequentially in one shared context to groom the backlog and decompose analyzed Features into keyed, area-tagged, dependency-linked work-units, recording a durable decomposition plan (the stable plan-ordinals that key idempotency) and a per-unit canonical brief. Writes PBI/Task items, area:<name> tags, the plan manifest + briefs, and Analysis/Architecture/Conventions wiki pages — all via the azureDevOps MCP. Recurring; runs between kickoff and sprint-plan.
---

# /refine — backlog refinement and decomposition

This is the delivery-team's **recurring grooming ceremony**: it takes the analyzed backlog and
makes it *plannable*. It reads each in-scope Feature/PBI's analysis back **by location** (the
`business-analyst`'s `System.Description` headings and the `technical-analyst`'s
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
| Feature/PBI `System.Description` H2s (`## Problem` … `## Out of Scope`); the `**[Technical Analysis]**` sentinel comment (`## Approach` … `## Suggested Areas`); the `Domain/` + `Analysis/` wiki pages | PBI/Task work-units; `area:<name>` `System.Tags`; `Dependency` links; the durable **decomposition plan** manifest + per-unit **canonical briefs**; the `Analysis/`, `Architecture/`, `Architecture/ADR/`, `Conventions/` wiki pages |

Field semantics for the config the ceremony reads live in
[`config-and-methodology.md`](../../knowledge/config-and-methodology.md); the Azure tool map,
content-placement, and idempotency rules live in
[`azure-adapter.md`](../../backends/azure/adapter.md); the pack contract the area tags bind to
lives in [`pack-format.md`](../../knowledge/pack-format.md). This skill orchestrates the roles; the
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
`/delivery-init`). Read `config.json` — `org`/`project`/`repo`, the cached `wikiId`, and
`branchPair` (the **authoritative** dev/release names; config wins over `methodology.branches`).
Config is read-only to this ceremony. Resolve the scope (which Features/PBIs to refine) with the
user, then work the steps below. Every Azure touch names a real `azureDevOps` MCP tool from the
adapter's operation→tool map; never invent one.

### 1. Read the analysis back — by location, never by guessing

**Deterministic read-back (adapter §7).** For each Feature/PBI in scope:

- `wit_get_work_item` → parse the `System.Description` Markdown under its fixed H2s
  (`## Problem`, `## Business Value`, `## Scope`, `## Acceptance Criteria`, `## Out of Scope`) —
  the business "what & why".
- `wit_list_work_item_comments` → locate the technical analysis by **sentinel match**: the comment
  whose **first line is the exact `**[Technical Analysis]**` sentinel** — **never "the newest
  comment"**, so a later human comment can't shadow the analysis. Parse its H2s (`## Approach`,
  `## Feasibility & Risks`, `## NFRs`, `## Dependencies`, `## Suggested Areas`).
- Read the relevant wiki context via `wiki_get_page_content` (the `wikiId` from `config.json`):
  the `Domain/` pages and any `Analysis/` page for the item. Use `search_wiki` / `wiki_list_pages`
  to discover a page whose path isn't pre-named.

If a Feature in scope is missing its Description headings or its sentinel comment, it is **not
ready to refine** — do not invent the analysis; stop and surface it. Prefer batched reads
(`wit_get_work_items_batch_by_ids`) over a loop of singles (adapter §4 — never silently truncate a
list; treat a WIQL result at the cap as a truncation error).

### 2. Groom the business layer — as the `business-analyst`

Acting as the `business-analyst` (read [`../../agents/business-analyst/agent.md`](../../agents/business-analyst/agent.md)
and its `children/`, chiefly [`refine-participation.md`](../../agents/business-analyst/children/refine-participation.md)),
building on the read-back from step 1 **in this shared context**:

- **Sharpen the acceptance criteria** against real feedback — tighten vague criteria, add missing
  negative/boundary cases, remove non-conditions — and write them back into `## Acceptance
  Criteria` via an **idempotent update in place** (`wit_update_work_item`, adapter §5); never
  create a duplicate item.
- **Split oversized items along business seams** (independent slices of value — not technical
  seams, which are the tech-lead's cut in step 4). Each split child gets its own full five-H2
  Description whose value ladders up to the parent, and follows the idempotency contract of step 4
  (check-first WIQL + `atl-key` stamp on stable `parent + ordinal`, adapter §5).
- **Keep `Domain/` and the business half of `Analysis/` current** where understanding deepened —
  `wiki_create_or_update_page` (idempotent upsert, adapter §8); read-before-write, one owner, no
  write race. The `business-analyst` owns `Domain/` and co-owns `Analysis/` with the
  `technical-analyst`.

The `business-analyst` **does not** apply `area:<name>` tags and **does not** write the
`**[Technical Analysis]**` comment — those are neighbors' lanes (adapter §7).

### 3. Revisit the technical layer — as the `technical-analyst`

Then, still in the same context, acting as the `technical-analyst` (read
[`../../agents/technical-analyst/agent.md`](../../agents/technical-analyst/agent.md) and its
`children/`), building on the business grooming just done:

- Where a scope split or a sharpened criterion **changed what's feasible**, revise the
  `**[Technical Analysis]**` comment via the analyst's add-only convergence — the adapter's comment
  surface is add-and-list only, there is **no update-comment tool**. Read `wit_list_work_item_comments`
  and sentinel-match: if the analysis genuinely changed, add **one** fresh sentinel comment
  (`wit_add_work_item_comment`) that supersedes the earlier one, keeping the exact sentinel first line
  and the five fixed H2s (adapter §7); if it is unchanged, do not re-add. Read-back always resolves
  the analysis by sentinel, so the latest sentinel comment *is* the analysis — never a second
  Description; never "the newest comment".
- Keep the **`## Suggested Areas`** current — candidates only, prose only. The `technical-analyst`
  **suggests**; it never writes an `area:<name>` tag (that binding is the tech-lead's, step 4 —
  adapter §7 and [`suggesting-areas.md`](../../agents/technical-analyst/children/suggesting-areas.md)).
- Record real technical **dependencies** as Azure `Dependency` links (`wit_work_items_link`), not
  just prose, so the scheduler's DAG is machine-sound.
- Co-author the **technical half** of the item's `Analysis/` page where the analysis exceeds a
  comment (`wiki_create_or_update_page`, idempotent upsert; own section only, cross-link where the
  layers meet).
- Resolve any state/type reference at runtime via `wit_get_work_item_type` (adapter §6) — never a
  hardcoded literal.

### 4. Decompose into keyed, area-tagged, dependency-linked units — as the `tech-lead`

Then, still in the same context, acting as the `tech-lead` (read
[`../../agents/tech-lead/agent.md`](../../agents/tech-lead/agent.md) and its `children/`, chiefly
[`decomposition-blueprint.md`](../../agents/tech-lead/children/decomposition-blueprint.md)),
consuming the analysts' just-produced output. This is the ceremony's core write.

1. **Record the decomposition plan first — the idempotency substrate.** For each Feature being
   decomposed, produce an ordered list of the units you *intend* to create and record it
   **durably** as a plan manifest — a labeled comment on the parent Feature
   (`wit_add_work_item_comment`), mirrored to the area's `Architecture/` wiki page when the
   decomposition is architecturally significant (`wiki_create_or_update_page`). Give each intended
   unit a **stable plan-ordinal** (a small integer identifying its position in the plan — **not**
   its title, **not** a per-run GUID). Bump a plan **version** on re-plan; ordinals are
   append-only within the Feature (retired, never renumbered or reused). The plan-ordinal is the
   `atl-key` substrate step 5 depends on.
2. **Create the units — check-first, then stamp (adapter §5).** For each planned unit, in ordinal
   order:
   - Compute `atl-key = hash(parent-id + plan-ordinal)`.
   - **Check-first WIQL** — `wit_query_by_wiql` filtered to that `atl-key` tag: **found →** reuse +
     update the existing item to the intended state (title, description, links, area) — never a
     second one; **not-found →** create it (`wit_create_work_item`, or `wit_add_child_work_items`
     to place it under the parent), then **stamp** `System.Tags` with `atl-key:<hash>` +
     `atl-run:refine:<sprint-or-scope>` (provenance), as close to atomic as the API allows. A
     409/duplicate on create is **caught and resolved to the existing item**, never surfaced.
   - Resolve the concrete work-item **type** at runtime via `wit_get_work_item_type` (the
     `artifactHierarchy` `Epic → Feature → PBI → Task` is abstract; the real Azure type name is
     process-template-dependent) — **never** hardcode a type/state literal (adapter §6).
3. **Apply the area tags — the tech-lead decides (adapter §7).** Write each unit's `area:<name>`
   to `System.Tags` (`wit_update_work_item`). The `technical-analyst` only *suggested* areas under
   `## Suggested Areas`; the tech-lead **decides** them, because the tag *is* the pack binding — a
   `developer` loads exactly `packs/<area>/` for the tagged area (see
   [`pack-format.md`](../../knowledge/pack-format.md)). One primary area per unit; keep the area
   vocabulary stable on the `Architecture/` page.
4. **Add the Dependency links — the edges the scheduler orders over.** Add a `Dependency` link
   (`wit_work_items_link`) between units only for a real prerequisite (a shared surface, schema, or
   contract another unit produces); **no cycles**; parent/child containment
   (`wit_add_child_work_items`) is *not* a scheduling edge. Fewer edges is better — an unnecessary
   edge serializes work that could run in parallel.
5. **Promote worker-surfaced and analysis-durable project facts to the wiki** (tech-lead
   write-lane, adapter §8): fold the durable parts of the analysis into the `Architecture/` page
   (current-truth upsert of system shape, module boundaries, area vocabulary) and write an **ADR**
   at `Architecture/ADR/ADR-<n>-<slug>` **only** for a decision that is significant *and*
   hard-to-reverse (a reversed decision → a new ADR + supersede the old, never an in-place edit —
   see [`architecture-and-adr.md`](../../agents/tech-lead/children/architecture-and-adr.md)). Keep
   `Conventions/` current for project rules layered atop the pack's generics. All via
   `wiki_create_or_update_page` (idempotent upsert); `wikiId` read from `config.json`, verified
   with `wiki_list_pages` before a first write. Workers never write the wiki — the tech-lead
   promotes.

### 5. Write the per-unit canonical brief — as the `tech-lead`

Still as the `tech-lead`, for each work-unit created in step 4, write its **canonical brief** — the
artifact that lets a fresh, isolated `developer` worker load the right project knowledge without
carry-over (see [`canonical-brief.md`](../../agents/tech-lead/children/canonical-brief.md)). Record
each brief durably on its unit as a **single labeled comment** (`wit_add_work_item_comment`) whose
**first line is the exact sentinel `**[Canonical Brief]**`** — the same machine-locatable placement
the `**[Technical Analysis]**` comment uses (adapter §7) — keyed to its `atl-key` so a re-run updates
in place. A brief bounds context; it does not dump it:

- The unit's **goal** restated in one or two sentences, traced to the Feature's Acceptance Criteria.
- The **area** (`area:<name>`) — binds the knowledge-pack (`packs/<area>/`).
- The **exact** `Architecture/` slice + `Conventions/` page paths for the unit's area (adapter §8
  read contract) — specific paths the worker pulls via `wiki_get_page_content` (`search_wiki` is
  the fallback when a path isn't pre-named), never "read the whole wiki"; reference any constraining
  ADR by number.
- The **dependencies** (the `Dependency`-linked prerequisites) with "build against, don't
  re-declare" guidance.
- The **test-evidence expectation** — code + web + mobile-emulator evidence where the surface
  applies — so the worker knows the review gate ahead of time.

Leave out whole-wiki dumps, other units' internals, stack how-to (the pack's job), and methodology
mechanics. No `developer`/`tester` worker is spawned here — `/refine` only prepares the units and
briefs; `/sprint-start` hands the selected, briefed units to `atl work dispatch`.

### 6. Report

Summarize what the run refined and decomposed: the Features groomed, the work-units created (with
their `area:<name>` tags), the Dependency links added, the plan-manifest version, the canonical
briefs written, and any `Analysis/`/`Architecture/`/`ADR`/`Conventions/` wiki pages touched. Note
any Feature that was **not ready** (missing Description headings or sentinel comment) so it can be
analyzed before the next `/refine`. Point the user to `/sprint-plan` as the next step.

## Idempotent re-run

A re-run of `/refine` (after a crash, a partial run, or an explicit re-plan) must **never duplicate
work-units or double-write analysis** — Azure-side tags are the source of truth; there is no local
ledger (adapter §5).

- **Every created unit carries two `System.Tags`:** `atl-run:refine:<sprint-or-scope>`
  (provenance) and `atl-key:<hash>` where `hash = hash(parent-id + plan-ordinal)`. The tech-lead's
  recorded **decomposition plan** gives each intended unit a **stable plan-ordinal** — not a per-run
  GUID, not `hash(title)` — so the same logical unit maps to the same key across runs (that is what
  makes resume *convergent*, not merely dedup-attempted). A **title edit must NOT mint a new key**
  (the ordinal is stable); two units sharing a title don't collide (distinct ordinals).
- **Before any create, run a check-first WIQL** (`wit_query_by_wiql`) for that `atl-key`:
  **found → reuse + update** (converge to the intended state), **not-found → create-then-stamp**. A
  409/duplicate on create is caught and resolved to the existing item, never surfaced.
- **The Description/wiki writes are idempotent updates/upserts** — `wit_update_work_item` updates the
  Description/fields in place and `wiki_create_or_update_page` is an idempotent upsert. Comments are
  add-only (no update-comment tool): a re-run first **sentinel-matches** the existing
  `**[Canonical Brief]**` comment (`wit_list_work_item_comments`, not "the newest comment") — found →
  add **one** fresh superseding comment keyed to its unit's `atl-key`; not-found → add the first. The
  sentinel is the read-back locator; the `atl-key` is the convergence guard. Never append an
  uncontrolled second brief.
- **Ordinal-stability rules:** once assigned, an ordinal never changes for a surviving unit; a
  removed unit's ordinal is retired (never backfilled or reused); a re-plan adds units at fresh
  higher ordinals and bumps the plan version.

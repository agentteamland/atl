---
name: kickoff
description: /kickoff — greenfield cold-start for the delivery-team. A once-per-project ceremony that turns a brand-new project's vision into its first Azure DevOps backlog: it runs live intake with the human PO, then adopts the business-analyst + technical-analyst (sequentially, in shared context) to create the first Epic + Feature(s) — business framing into System.Description, technical analysis into a sentinel comment — seeds the first Domain/ and Architecture/ wiki pages, and optionally seeds sprint-0. Requires .delivery/config.json + .delivery/methodology.json from /delivery-init; re-run converges (never blind re-creates) via the atl-key check-first WIQL. Run once, after /delivery-init, before the recurring ceremonies.
---

# /kickoff — greenfield cold-start

`/kickoff` is the delivery-team's **cold-start** ceremony: the one-time step that takes a
brand-new (greenfield) project from an empty Azure DevOps backlog to its first Epic, first
Feature(s), and first durable wiki knowledge. It runs **after** [`/delivery-init`](../delivery-init/SKILL.md)
has connected the project (it reads that settled config; it never re-writes it) and **before**
the recurring ceremonies (`/refine`, `/sprint-plan`, `/sprint-start`, `/sprint-review`). It is
the only ceremony that runs the `intake` role live, and the only one that seeds the backlog from
zero rather than grooming an existing one.

The ceremony is a **gated cold-start sequence** — each phase is a gate: a failure stops the run
before it orphans half-created Azure state, and a re-run converges on what already exists rather
than duplicating it. What it reads and writes:

| Reads | Writes |
|---|---|
| `.delivery/config.json` (`org`/`project`/`repo`/`branchPair`/`wikiId`/`pat.ref`) | the first **Epic** + **Feature(s)** (`wit_create_work_item` / `wit_add_child_work_items`) |
| `.delivery/methodology.json` (`roles`, `cadence`, `artifactHierarchy`, `capacityModel`) | business framing into each item's `System.Description` (fixed H2s, adapter §7) |
| the live PO conversation (intake) | technical analysis as one `**[Technical Analysis]**` comment (`wit_add_work_item_comment`, adapter §7) |
| existing backlog on a re-run (`wit_query_by_wiql` by `atl-key`) | the first `Domain/` + `Architecture/` wiki pages (`wiki_create_or_update_page`, adapter §8) |
| | optional sprint-0 iteration + starter backlog (`work_create_iterations`, prompted, default skip) |

Field semantics for the config live in [`config-and-methodology.md`](../../knowledge/config-and-methodology.md);
the Azure operation → MCP-tool map, idempotency, runtime type resolution, content placement, and
wiki namespaces live in [`azure-adapter.md`](../../backends/azure/adapter.md). All Azure access is
through the `azureDevOps` MCP; the PAT is referenced by name (`config.pat.ref`), never read or
written as a literal.

## When to run

- **Once, cold-start, per greenfield project** — a brand-new project with an empty (or
  never-planned) backlog, immediately **after** `/delivery-init` and **before** the recurring
  ceremonies. This is not a `methodology.cadence` ceremony (it is neither a `planningCeremonies`
  nor the `reviewCeremony` slot) — it is the **one-time bootstrap** that produces the backlog the
  cadence ceremonies then plan and review.
- **Re-run (idempotency at t=0)** — a second `/kickoff` against a project that already has
  Epics/Features **converges**: it detects existing items and offers to resume, never blind
  re-creates. See [Idempotency](#idempotency). A mid-project vision shift re-runs `/kickoff` for
  re-discovery under the same convergence discipline.

## Procedure

### 1. Preflight — require the settled config, then probe the live MCP

`/kickoff` never writes config; it depends on `/delivery-init` having written it. Confirm both
files exist at the project root **before** touching Azure or the PO:

- Read `.delivery/config.json` and `.delivery/methodology.json`. **If either is absent, STOP** and
  tell the user to run [`/delivery-init`](../delivery-init/SKILL.md) first — do **not** create,
  guess, or re-write either file here.
- From `config.json`, load `org`/`project`/`repo`, the authoritative `branchPair` (the actual
  dev/release branch names — `config.branchPair` wins over `methodology.branches`), `wikiId`, and
  `pat.ref` (the env-var **name**, never the token). If `wikiId` is `null`, tell the user the
  project wiki isn't provisioned yet and that step 2's knowledge-seeding needs it — they should
  create a project wiki in Azure DevOps and re-run `/delivery-init` before proceeding.
- From `methodology.json`, load `roles` (with each `dispatch`), `artifactHierarchy`
  (`["Epic","Feature","Pbi","Task"]`), `cadence`, and `capacityModel`.
- **Live MCP probe** — call `core_list_projects` to confirm auth + reachability. Projects
  returned → the `azureDevOps` MCP is live; continue. Auth error / no projects / tool unavailable
  → STOP and point the user at their MCP configuration (the PAT is supplied to the MCP server, not
  by this skill); never ask for a pasted token.

This preflight is the first gate: nothing is created until the config is present and the
connection is proven.

### 2. Intake — live discovery with the PO (`intake`, in-session)

Adopt the `intake` role (read [`../../agents/intake/agent.md`](../../agents/intake/agent.md) + its
`children/`) and run **interactively, in this session** — the one live human-dialogue phase (its
`dispatch: in-session`). Elicit the project's vision, problem, need-vs-want, goals, constraints,
stakeholders, falsifiable success signals, out-of-scope hints, and open questions. Produce the
structured **framing** (the intake→analysis handoff) and confirm it back to the PO.

The `intake` role creates **no** Azure state — no work-items, comments, tags, or wiki pages. It
frames; the analysts persist (step 3). Do **not** call `wit_create_work_item`,
`wit_add_work_item_comment`, or `wiki_create_or_update_page` in this phase. This phase gates the
next: a thin framing means the analysts analyze the wrong thing thoroughly, so hand off only when
the intake handoff checklist is complete.

### 3. First Epic + Feature(s) — analysis line (`business-analyst` → `technical-analyst`, sequential, shared context)

Turn the framing into the first backlog. Adopt the two analyst roles **sequentially in this same
session context** (their `dispatch: subagent`; the coordination relies on nuance held in shared
context — do **not** spawn them as isolated `claude -p` workers or independent subagents that
can't see each other's output):

1. **Resolve concrete types at runtime (never hardcode).** Before creating anything, resolve the
   real Azure type names for the `artifactHierarchy` rungs (`Epic`, `Feature`) via
   `wit_get_work_item_type` — the live project may spell them differently under its process
   template (adapter §6). Never write a literal type or state name into a create call.

2. **Idempotency check-first (adapter §5) — before ANY create.** For each intended item, compute
   its stable `atl-key = hash(parent-id + plan-ordinal)` (the plan-ordinal is the item's stable
   position in the intended-item plan — **not** a per-run GUID, **not** `hash(title)`) and run a
   check-first WIQL via `wit_query_by_wiql` for that `atl-key`. **Found** → reuse + update
   (converge), **not-found** → create-then-stamp. A 409/duplicate on create is caught and resolved
   to the existing item, never surfaced. See [Idempotency](#idempotency).

3. **Acting as the `business-analyst`** (read
   [`../../agents/business-analyst/agent.md`](../../agents/business-analyst/agent.md) + its
   `children/`): create the first **Epic** (`wit_create_work_item`) and its **Feature(s)** under it
   (`wit_add_child_work_items`), stamping each created item's `System.Tags` with
   `atl-run:kickoff:<id>` + `atl-key:<hash>`. Write each item's business analysis into
   `System.Description` under the fixed H2s — `## Problem`, `## Business Value`, `## Scope`,
   `## Acceptance Criteria`, `## Out of Scope` (adapter §7). Do **not** write the technical comment
   and do **not** apply `area:<name>` tags (both belong to later roles).

4. **Then, as the `technical-analyst`, building on the BA's output** (read
   [`../../agents/technical-analyst/agent.md`](../../agents/technical-analyst/agent.md) + its
   `children/`): for each Feature, first read the BA's Description back (`wit_get_work_item`), then
   add **one** technical-analysis comment via `wit_add_work_item_comment` whose **first line is the
   exact sentinel** `**[Technical Analysis]**`, followed by the fixed H2s — `## Approach`,
   `## Feasibility & Risks`, `## NFRs`, `## Dependencies`, `## Suggested Areas` (adapter §7). Before
   adding, sentinel-match existing comments (`wit_list_work_item_comments`) so a re-run does not
   stack a second analysis comment. Record real technical dependencies as Azure Dependency links
   (`wit_work_items_link`), not just prose. Areas are only *suggested* under `## Suggested Areas` —
   never write `area:<name>` to `System.Tags` here (that is the tech-lead's binding at
   decomposition).

5. **Seed the first wiki pages (adapter §8, idempotent upsert).** As the `business-analyst`, seed
   the project's `Domain/` namespace (glossary, entities, business rules surfaced during analysis);
   seed the first `Architecture/` page for the system-shape starting point. Use
   `wiki_create_or_update_page` (an idempotent upsert) against the `wikiId` cached in
   `config.json`. On a greenfield project the wiki is **empty**, so **create each parent
   namespace page first** (`/Domain`, `/Architecture`) — `wiki_create_or_update_page` does
   not auto-create ancestors, so a nested write (`Domain/Glossary`) 404s otherwise (adapter
   §8); `wiki_list_pages` tells you what's absent. One owner
   per namespace — `Domain/`/`Analysis/` are the analysts', `Architecture/`/`Conventions/` the
   tech-lead's; seed only what the adopted role owns.

Each sub-step is gated by the prior one: no technical comment on a Feature that failed to create,
no wiki seed for analysis that didn't land.

### 4. Optional sprint-0 seed — prompt the PO, default skip (`product-owner`, human)

**Ask the user explicitly** whether to seed a first sprint now so they can go straight to
`/sprint-plan`; **default to skip** if they don't opt in. If they opt in:

- **Resolve the first sprint iteration via `work_list_iterations` and REUSE an existing one** — a
  Scrum process template ships `Sprint 1`–`Sprint 6` by default. Create via `work_create_iterations`
  **only if no suitable iteration exists**: `work_create_iterations` **errors (VS402371) on a name
  already in use**, so a blind create fails on a default Scrum project. Assigning an item to the
  resolved iteration is then an idempotent `IterationPath` field update — a safe no-op on re-run.
- Optionally create a small starter PBI set under the first Feature (`wit_create_work_item` /
  `wit_add_child_work_items`), each run through the **same** step-3 idempotency discipline (resolve
  type at runtime, check-first WIQL by `atl-key`, stamp `atl-run:kickoff:<id>` + `atl-key:<hash>`).

If skipped, tell the user `/sprint-plan` will handle the first iteration when they're ready.

### 5. Report and point to the next ceremony

Summarize what was created — the Epic + Feature(s) (by id and title), the wiki pages seeded
(`Domain/`, `Architecture/`), and whether sprint-0 was seeded or skipped. Never print the PAT or
any secret. Point the user to the next step: `/refine` to groom + decompose the backlog, then
`/sprint-plan` to plan the first sprint (or straight to `/sprint-plan` if sprint-0 was seeded).

## Idempotency

`/kickoff` is **idempotency at t=0**: the cold-start bootstrap, made safe to re-run. A second
`/kickoff` against a populated project must **converge**, never blind re-create — per adapter §5
(Azure tags are the source of truth; no local ledger):

- **Detect first, offer resume.** For each intended Epic/Feature, run the check-first WIQL
  (`wit_query_by_wiql`) for its `atl-key`. If existing kickoff-stamped items are found, **offer to
  resume** (converge onto what exists) rather than re-creating; do not silently create parallel
  items.
- **Stable keys.** `atl-key = hash(parent-id + plan-ordinal)` — the plan-ordinal is the item's
  stable position in the intended-item plan, so a title edit doesn't break convergence and a
  duplicate title doesn't collide. Never a per-run GUID/timestamp, never `hash(title)`.
- **Stamp on create.** Every created work-item carries two `System.Tags`: `atl-run:kickoff:<id>`
  (provenance) + `atl-key:<hash>`. Create-then-stamp as close to atomic as the API allows; a
  409/duplicate is caught and resolved to the existing item, never surfaced as an error.
- **The technical comment converges by sentinel.** Before adding the `**[Technical Analysis]**`
  comment, sentinel-match `wit_list_work_item_comments`; found → don't stack a duplicate. (There is
  no update-comment tool; a genuine re-plan adds one fresh sentinel comment that supersedes.)
- **Wiki + iterations are idempotent by nature.** `wiki_create_or_update_page` is an idempotent
  upsert; an iteration assignment is an `IterationPath` field update (a safe no-op on re-run).

On a re-discovery re-run (a mid-project vision shift), the intake framing should call out the
*delta* against the established understanding so the analysts update rather than duplicate — the
`atl-key` convergence then maps the reshaped intent onto the same items.

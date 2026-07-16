---
name: kickoff
description: /kickoff — greenfield cold-start for the delivery-team. A once-per-project ceremony that turns a brand-new project's vision into its first backlog on the active backend: it runs live intake with the human PO, then adopts the business-analyst + technical-analyst (sequentially, in shared context) to create the first Epic + Feature(s) — business framing into the spec field, technical analysis into a sentinel comment — seeds the first Domain/ and Architecture/ durable-knowledge pages, and optionally seeds sprint-0. Requires .delivery/config.json + .delivery/methodology.json from /delivery-init; re-run converges (never blind re-creates) via the atl-key check-first query. Run once, after /delivery-init, before the recurring ceremonies.
---

# /kickoff — greenfield cold-start

`/kickoff` is the delivery-team's **cold-start** ceremony: the one-time step that takes a
brand-new (greenfield) project from an empty backlog on the active backend to its first Epic, first
Feature(s), and first durable-knowledge. It runs **after** [`/delivery-init`](../delivery-init/SKILL.md)
has connected the project (it reads that settled config; it never re-writes it) and **before**
the recurring ceremonies (`/refine`, `/sprint-plan`, `/sprint-start`, `/sprint-review`). It is
the only ceremony that runs the `intake` role live, and the only one that seeds the backlog from
zero rather than grooming an existing one.

The ceremony is a **gated cold-start sequence** — each phase is a gate: a failure stops the run
before it orphans half-created backend state, and a re-run converges on what already exists rather
than duplicating it. What it reads and writes:

| Reads | Writes |
|---|---|
| `.delivery/config.json` (`org`/`project`/`repo`/`branchPair`/`wikiId`/`pat.ref`) | the first **Epic** + **Feature(s)** (create work-items, concept #1) |
| `.delivery/methodology.json` (`roles`, `cadence`, `artifactHierarchy`, `capacityModel`) | business framing into each item's spec field (fixed H2s, concept #2) |
| the live PO conversation (intake) | technical analysis as one `**[Technical Analysis]**` comment (add a comment, concept #3) |
| existing backlog on a re-run (check-first query by `atl-key`, concept #10) | the first `Domain/` + `Architecture/` durable-knowledge pages (upsert the durable-knowledge store, concept #9) |
| | optional sprint-0 iteration + starter backlog (create an iteration, concept #6, prompted, default skip) |

Field semantics for the config live in [`config-and-methodology.md`](../../knowledge/config-and-methodology.md);
the operation → tool map, idempotency, runtime type resolution, content placement, and
durable-knowledge namespaces live in the active backend's adapter (`backends/<backend>/adapter.md`),
which binds the provider-neutral concepts defined in [the backend interface](../../knowledge/backend-interface.md).
All backend access is through the active backend's adapter; the credential is referenced by name
(`config.pat.ref`), never read or written as a literal.

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

### 1. Preflight — require the settled config, then probe the live backend

`/kickoff` never writes config; it depends on `/delivery-init` having written it. Confirm both
files exist at the project root **before** touching the backend or the PO:

- Read `.delivery/config.json` and `.delivery/methodology.json`. **If either is absent, STOP** and
  tell the user to run [`/delivery-init`](../delivery-init/SKILL.md) first — do **not** create,
  guess, or re-write either file here.
- From `config.json`, load `org`/`project`/`repo`, the authoritative `branchPair` (the actual
  dev/release branch names — `config.branchPair` wins over `methodology.branches`), `wikiId`, and
  `pat.ref` (the env-var **name**, never the credential). If `wikiId` is `null`, tell the user the
  durable-knowledge store isn't provisioned yet and that step 2's knowledge-seeding needs it — they
  should provision it per the active backend's adapter and re-run `/delivery-init` before proceeding.
- From `methodology.json`, load `roles` (with each `dispatch`), `artifactHierarchy`
  (`["Epic","Feature","Pbi","Task"]`), `cadence`, and `capacityModel`.
- **Live backend probe** — run the active backend's connectivity check (resolve project / identity,
  per the adapter) to confirm auth + reachability. A successful response → the backend is live;
  continue. Auth error / nothing returned / tool unavailable → STOP and point the user at their
  backend configuration (the credential is supplied to the backend, not by this skill); never ask
  for a pasted secret.

This preflight is the first gate: nothing is created until the config is present and the
connection is proven.

### 2. Intake — live discovery with the PO (`intake`, in-session)

Adopt the `intake` role (read [`../../agents/intake/agent.md`](../../agents/intake/agent.md) + its
`children/`) and run **interactively, in this session** — the one live human-dialogue phase (its
`dispatch: in-session`). Elicit the project's vision, problem, need-vs-want, goals, constraints,
stakeholders, falsifiable success signals, out-of-scope hints, and open questions. Produce the
structured **framing** (the intake→analysis handoff) and confirm it back to the PO.

The `intake` role creates **no** backend state — no work-items, comments, tags, or durable-knowledge
pages. It frames; the analysts persist (step 3). Do **not** create work-items (concept #1), add
comments (concept #3), or write the durable-knowledge store (concept #9) in this phase. This phase
gates the next: a thin framing means the analysts analyze the wrong thing thoroughly, so hand off
only when the intake handoff checklist is complete.

### 3. First Epic + Feature(s) — analysis line (`business-analyst` → `technical-analyst`, sequential, shared context)

Turn the framing into the first backlog. Adopt the two analyst roles **sequentially in this same
session context** (their `dispatch: subagent`; the coordination relies on nuance held in shared
context — do **not** spawn them as isolated `claude -p` workers or independent subagents that
can't see each other's output):

1. **Resolve concrete types at runtime (never hardcode).** Before creating anything, resolve the
   real type names for the `artifactHierarchy` rungs (`Epic`, `Feature`) at runtime (concept #7) —
   the live backend and process template may spell them differently. Never write a literal type or
   state name into a create call.

2. **Idempotency check-first (concept #10) — before ANY create.** For each intended item, compute
   its stable `atl-key = hash(parent-id + plan-ordinal)` (the plan-ordinal is the item's stable
   position in the intended-item plan — **not** a per-run GUID, **not** `hash(title)`) and run a
   check-first query (concept #10) for that `atl-key`. **Found** → reuse + update
   (converge), **not-found** → create-then-stamp. A 409/duplicate on create is caught and resolved
   to the existing item, never surfaced. See [Idempotency](#idempotency).

3. **Acting as the `business-analyst`** (read
   [`../../agents/business-analyst/agent.md`](../../agents/business-analyst/agent.md) + its
   `children/`): create the first **Epic** and its **Feature(s)** under it (create work-items,
   concept #1), stamping each created item's tags (concept #4) with
   `atl-run:kickoff:<id>` + `atl-key:<hash>`. Write each item's business analysis into
   the spec field (concept #2) under the fixed H2s — `## Problem`, `## Business Value`, `## Scope`,
   `## Acceptance Criteria`, `## Out of Scope`. Do **not** write the technical comment
   and do **not** apply `area:<name>` tags (both belong to later roles).

4. **Then, as the `technical-analyst`, building on the BA's output** (read
   [`../../agents/technical-analyst/agent.md`](../../agents/technical-analyst/agent.md) + its
   `children/`): for each Feature, first read the BA's spec field back (read the work-item, concept
   #2), then add **one** technical-analysis comment (add a comment, concept #3) whose **first line
   is the exact sentinel** `**[Technical Analysis]**`, followed by the fixed H2s — `## Approach`,
   `## Feasibility & Risks`, `## NFRs`, `## Dependencies`, `## Suggested Areas`. Before adding,
   sentinel-match existing comments (read comments, concept #3) so a re-run does not stack a second
   analysis comment. Record real technical dependencies as dependency links (concept #8), not just
   prose. Areas are only *suggested* under `## Suggested Areas` — never write `area:<name>` tags
   (concept #4) here (that is the tech-lead's binding at decomposition).

5. **Seed the first durable-knowledge pages (concept #9, idempotent upsert).** As the
   `business-analyst`, seed the project's `Domain/` namespace (glossary, entities, business rules
   surfaced during analysis); seed the first `Architecture/` page for the system-shape starting
   point. Write via the durable-knowledge store's idempotent upsert (concept #9), targeting the
   store resolved at `/delivery-init`. On a greenfield project the store is **empty**, so follow the
   active backend's adapter for the write mechanics — some backends do not auto-create ancestor
   namespace pages, so a nested write must **create each parent namespace page first**
   (`/Domain`, `/Architecture`) before the child (`Domain/Glossary`); the adapter states how to
   detect what's absent. One owner per namespace — `Domain/`/`Analysis/` are the analysts',
   `Architecture/`/`Conventions/` the tech-lead's; seed only what the adopted role owns.

Each sub-step is gated by the prior one: no technical comment on a Feature that failed to create,
no durable-knowledge seed for analysis that didn't land.

### 4. Optional sprint-0 seed — prompt the PO, default skip (`product-owner`, human)

**Ask the user explicitly** whether to seed a first sprint now so they can go straight to
`/sprint-plan`; **default to skip** if they don't opt in. If they opt in:

- **Resolve the first sprint iteration (concept #6) and REUSE an existing one** — some backends
  pre-provision default iterations (a Scrum-style template may ship `Sprint 1`–`Sprint 6`). Create a
  new iteration (concept #6) **only if no suitable iteration exists**: on some backends a create
  **errors on a name already in use**, so a blind create fails against a pre-provisioned project.
  Assigning an item to the resolved iteration is then an idempotent iteration field set (concept #6)
  — a safe no-op on re-run.
- Optionally create a small starter PBI set under the first Feature (create work-items, concept #1),
  each run through the **same** step-3 idempotency discipline (resolve type at runtime, concept #7;
  check-first query by `atl-key`, concept #10; stamp `atl-run:kickoff:<id>` + `atl-key:<hash>`).

If skipped, tell the user `/sprint-plan` will handle the first iteration when they're ready.

### 5. Report and point to the next ceremony

Summarize what was created — the Epic + Feature(s) (by id and title), the durable-knowledge pages
seeded (`Domain/`, `Architecture/`), and whether sprint-0 was seeded or skipped. Never print the
credential or any secret. Point the user to the next step: `/refine` to groom + decompose the backlog, then
`/sprint-plan` to plan the first sprint (or straight to `/sprint-plan` if sprint-0 was seeded).

## Idempotency

`/kickoff` is **idempotency at t=0**: the cold-start bootstrap, made safe to re-run. A second
`/kickoff` against a populated project must **converge**, never blind re-create — per the
idempotency policy (concept #10): the backend's tags (concept #4) are the source of truth, no local
ledger:

- **Detect first, offer resume.** For each intended Epic/Feature, run the check-first query
  (concept #10) for its `atl-key`. If existing kickoff-stamped items are found, **offer to
  resume** (converge onto what exists) rather than re-creating; do not silently create parallel
  items.
- **Stable keys.** `atl-key = hash(parent-id + plan-ordinal)` — the plan-ordinal is the item's
  stable position in the intended-item plan, so a title edit doesn't break convergence and a
  duplicate title doesn't collide. Never a per-run GUID/timestamp, never `hash(title)`.
- **Stamp on create.** Every created work-item carries two tags (concept #4): `atl-run:kickoff:<id>`
  (provenance) + `atl-key:<hash>`. Create-then-stamp as close to atomic as the API allows; a
  409/duplicate is caught and resolved to the existing item, never surfaced as an error.
- **The technical comment converges by sentinel.** Before adding the `**[Technical Analysis]**`
  comment, sentinel-match the comments (read comments, concept #3); found → don't stack a duplicate.
  (The sentinel comment channel is append-only, concept #3; a genuine re-plan adds one fresh
  sentinel comment that supersedes.)
- **Durable-knowledge + iterations are idempotent by nature.** The durable-knowledge upsert
  (concept #9) is idempotent; an iteration assignment is an idempotent iteration field set
  (concept #6, a safe no-op on re-run).

On a re-discovery re-run (a mid-project vision shift), the intake framing should call out the
*delta* against the established understanding so the analysts update rather than duplicate — the
`atl-key` convergence then maps the reshaped intent onto the same items.

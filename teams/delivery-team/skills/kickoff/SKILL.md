---
name: kickoff
description: /kickoff — cold-start for the delivery-team, greenfield or brownfield. A once-per-project ceremony that turns a project's intent into its first backlog on the active backend. Greenfield: live intake with the human PO. Brownfield: read the work and knowledge the project already records, report the coverage — what was read, what maps where, what could not be placed, what would have to be invented — and ask the PO only the gaps, with no acceptance criterion shipping inferred. Either way it then adopts the business-analyst + technical-analyst (sequentially, in shared context) to create the first Epic + Feature(s) — business framing into the spec field, technical analysis into a sentinel comment — seeds the first Domain/ and Architecture/ durable-knowledge pages, and optionally seeds sprint-0. Requires .delivery/config.json + .delivery/methodology.json from /delivery-init; re-run converges (never blind re-creates) via the atl-key check-first query. Run once, after /delivery-init, before the recurring ceremonies.
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
| `.delivery/config.json` (the active backend's coordinates + `branchPair` + durable-knowledge locator + credential ref — see [`config-and-methodology.md`](../../knowledge/config-and-methodology.md) §2) | the first **Epic** + **Feature(s)** (create work-items, concept #1) |
| `.delivery/methodology.json` (`mode`, `roles`, `cadence`, `artifactHierarchy`, and `capacityModel` under `mode: "scrum"`) | business framing into each item's spec field (fixed H2s, concept #2) |
| the live PO conversation (intake) | technical analysis as one `**[Technical Analysis]**` comment (add a comment, concept #3) |
| existing backlog on a re-run (check-first query by `atl-key`, concept #10) | the first `Domain/` + `Architecture/` durable-knowledge pages (upsert the durable-knowledge store, concept #9) |
| | optional sprint-0 + starter backlog (prompted, default skip) — an iteration under `mode: "scrum"` (create an iteration, concept #6), the `sprint:1` label under `mode: "flow"` (concept #4) |

Field semantics for the config live in [`config-and-methodology.md`](../../knowledge/config-and-methodology.md);
the operation → tool map, idempotency, runtime type resolution, content placement, and
durable-knowledge namespaces live in the active backend's adapter (`backends/<backend>/adapter.md`),
which binds the provider-neutral concepts defined in [the backend interface](../../knowledge/backend-interface.md).
All backend access is through the active backend's adapter; the credential is referenced by name
(`config.pat.ref` on Azure, `config.credential.ref` on GitHub), never read or written as a literal.

## When to run

- **Once, cold-start, per project — greenfield OR brownfield.** Immediately **after**
  `/delivery-init` and **before** the recurring ceremonies. A project that already has code, notes
  and a backlog runs the *same* ceremony in **brownfield mode** (step 2b): same output, different
  input source — artifacts instead of a conversation.

  There is no separate brownfield ceremony, and that is the point. `/kickoff` is the **sole**
  seeder of the `Domain/` + `Architecture/` durable pages that `/refine` and every worker read, so
  a project routed *past* it arrives at decomposition with the tech-lead's only durable input
  missing. A parallel ceremony would leave that bypass in place; a mode removes it.

  Either way this is not a `methodology.cadence` ceremony (it is neither a `planningCeremonies`
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
- From `config.json`, load the backend's connection coordinates (Azure `org`/`project`/`repo`;
  GitHub `owner`/`repo`/`projectNumber` — see [`config-and-methodology.md`](../../knowledge/config-and-methodology.md)
  §2), the authoritative `branchPair` (the actual dev/release branch names — `config.branchPair`
  wins over `methodology.branches`), the durable-knowledge store locator the active adapter needs
  (Azure: `wikiId`; GitHub: none — the store is the in-repo `/docs` tree, always present), and the
  credential ref (`pat.ref` on Azure, `credential.ref` on GitHub — the env-var **name**, never the
  credential). **Azure only:** if `wikiId` is `null`, tell the user the wiki isn't provisioned yet
  and that step 3's knowledge-seeding needs it — they should create it and re-run `/delivery-init`
  before proceeding. (GitHub's in-repo `/docs` store needs no provisioning.)
- From `methodology.json`, load `roles` (with each `dispatch`), `artifactHierarchy`
  (`["Epic","Feature","Pbi","Task"]`), `cadence`, and — under `mode: "scrum"` — `capacityModel`.
  Load **`mode`** itself first: `"scrum"` or `"flow"`, **absent ⇒ `"scrum"`**, never inferred from
  a missing `capacityModel`, and an unrecognized value stops the ceremony
  ([`config-and-methodology.md`](../../knowledge/config-and-methodology.md) §1.1). Under `"flow"`
  the `capacityModel` key is **absent by design** — that is not a broken descriptor and nothing
  here invents one. Kickoff's behaviour forks on the mode in exactly one place: the optional
  sprint-0 seed (step 4) uses the mode's sprint carrier (§1.2). Intake and analysis are
  mode-independent.
- **Live backend probe** — run the active backend's connectivity check (resolve project / identity,
  per the adapter) to confirm auth + reachability. A successful response → the backend is live;
  continue. Auth error / nothing returned / tool unavailable → STOP and point the user at their
  backend configuration (the credential is supplied to the backend, not by this skill); never ask
  for a pasted secret.

This preflight is the first gate: nothing is created until the config is present and the
connection is proven.

### 1b. Resolve the mode — ask, do not infer

Before intake, settle which mode this run is in. **Ask the PO**; the artifacts only tell you what
to ask about:

> Does this project already have work recorded somewhere — a task list, a backlog, notes, issues —
> and knowledge worth carrying in (a wiki, `docs/`, ADRs, a README that explains the domain)?

A repo with code and a `.atl/wiki/` is a strong hint, and it is only a hint: a project can have
months of code and no recorded intent, which is greenfield for this ceremony's purposes. **Yes →
step 2b. No → step 2.** Say which mode you are in and why, so the PO can correct you before any
reading starts.

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

### 2b. Brownfield intake — read the record, report the coverage, ask only the gaps

The brownfield replacement for step 2. Same role (`intake`), same output (the structured framing
the analysts consume), and the same prohibition on creating backend state. What changes is where
the answers come from: the project has already been thought about, often for months, and most of
what intake would elicit is written down.

That inverts intake's usual discipline in a way worth stating, because it is the failure mode
here. `/kickoff`'s reflex is *do not assume, ask*; against a pile of notes the risk is the
opposite — forty notes yield a clean hierarchy with every gap quietly inference-filled, and
nobody notices because the result **looks** complete. The work is noticing what the notes do
**not** say.

**1. Read the record.** Two sources, both required:

- **Existing work** — the task list, backlog, notes, or issues the project actually uses.
- **Existing knowledge** — its wiki, `docs/`, ADRs, READMEs, and the code's own structure.

**Read per-item detail as authoritative and treat any index or summary as a hint.** In a
hand-maintained two-level record the summary rots first, because updating it is the step with no
immediate consequence: an index marked `todo` against an item whose own file says three of four
sub-items shipped is the normal case, not a broken one. Believing the index creates cards for work
already done.

**2. Report the coverage — before creating anything.** This report is the deliverable of step 2b;
creation does not begin until the PO has answered it. Four parts:

- **What was read** — sources and counts, so the PO can say "you missed the folder that matters".
- **What maps where** — per item, the level you propose and why. Name **every** level of the
  hierarchy the mapping touches, or say explicitly where it stops. "These become PBIs" with no
  parent named is malformed by construction and still reads as complete. The honest boundary is
  usually **PBIs here, Tasks via `/refine`** — that is also the smaller job, since `/refine`
  decomposes PBI→Task already.
- **What could not be placed** — items you could not map, and why. An unplaceable item is a
  finding, not a failure.
- **What would have to be invented** — the fields no source answers. Expect this list to be
  dominated by **acceptance criteria**: measured on a real brownfield project, the notes were rich
  — current state, why-this-first, a to-do list, out-of-scope, mapping almost 1:1 onto the spec
  field's fixed headings — and every one of them was missing exactly that. A note is written to
  remind its author what to **do**, never to tell a stranger when to **stop**.

**3. The PO answers only the gaps.** Do not re-elicit what the record already says; that is the
redundancy this mode exists to remove. Ask about what could not be placed and what would have to
be invented, and nothing else.

**4. Then create, with every field stamped by source.** Each field carries `from-notes` (with the
quote it came from) or `inferred`.

> **Gate: no acceptance criterion ships `inferred`.** This is the one field a work-item cannot be
> honestly guessed into, and it is precisely the field the notes do not carry. Refuse to invent it
> — the same reflex `/sprint-plan` already applies to velocity — and either ask, or leave the item
> at a level that does not require one yet.

**Granularity is a judgement you make and justify, not a template.** One Epic is a legitimate
answer for a small project; so is three. State the reasoning in the coverage report.

**Two things that are not cards.** Work already done does not become a card — the board records
what is happening, not what happened. And a **deferred** item keeps its trigger: write it into the
card body as a labelled line (`Trigger: <condition>`), which needs no board schema change on
either backend, greps cleanly, and survives the round-trip.

**5. The durable pages are a mapping pass, not a copy.** The `Domain/` + `Architecture/` pages
step 3 seeds are written **from** the existing knowledge, deciding per source page what is domain,
what is architecture, and what is convention — then writing real content under those headings and
cross-linking back to the original for depth. Copying the existing pages across forks them: two
copies of the same fact, one of which stops receiving updates and neither of which announces which
one that is.

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

- **Resolve the first sprint through the mode's carrier** (§1.2 of
  [`config-and-methodology.md`](../../knowledge/config-and-methodology.md)):
  - Under **`mode: "scrum"`** — **resolve the first sprint iteration (concept #6) and REUSE an
    existing one**. Some backends pre-provision default iterations (a Scrum-style template may ship
    `Sprint 1`–`Sprint 6`). Create a new iteration (concept #6) **only if no suitable iteration
    exists**: on some backends a create **errors on a name already in use**, so a blind create
    fails against a pre-provisioned project. Assigning an item to the resolved iteration is then an
    idempotent iteration field set (concept #6) — a safe no-op on re-run.
  - Under **`mode: "flow"`** — there is nothing to create or reuse: a flow sprint has no backend
    object, only the set of units carrying its label. Sprint-0 on a greenfield project is
    **`sprint:1`** (the board has no `sprint:*` label yet, so the resolve-the-highest-ordinal rule
    starts there — still resolve rather than assume, since a re-run finds `sprint:1` already
    present). Seeding an item into it is an idempotent label add (concept #4) — adding a label the
    item already carries is a no-op.
- Optionally create a small starter PBI set under the first Feature (create work-items, concept #1),
  each run through the **same** step-3 idempotency discipline (resolve type at runtime, concept #7;
  check-first query by `atl-key`, concept #10; stamp `atl-run:kickoff:<id>` + `atl-key:<hash>`).

If skipped, tell the user `/sprint-plan` will open the first sprint when they're ready.

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
- **Durable-knowledge + sprint membership are idempotent by nature.** The durable-knowledge upsert
  (concept #9) is idempotent; sprint membership is an idempotent iteration field set under
  `mode: "scrum"` (concept #6) and an idempotent label add under `mode: "flow"` (concept #4) — a
  safe no-op on re-run either way. Never model membership as a create-membership that could double.

On a re-discovery re-run (a mid-project vision shift), the intake framing should call out the
*delta* against the established understanding so the analysts update rather than duplicate — the
`atl-key` convergence then maps the reshaped intent onto the same items.

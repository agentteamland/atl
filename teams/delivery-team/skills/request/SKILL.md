---
name: request
description: /request — the delivery-team's mid-project request-intake ceremony. Captures a PO- or team-surfaced request as a *candidate* (excluded from the ready frontier until accepted), runs one weight-proportional triage pass (light/standard/heavy — a heavy *design* request routes out to /brainstorm), then adopts the business-analyst → technical-analyst → tech-lead subagents sequentially in one shared context to assess feasibility, and presents a reasoned YES/NO/DEFER/NEEDS-INFO verdict at an honest PO gate that is an active dialectic (refute-to-keep; converges by concession, a judgment-call standoff, or the PO's human-authority lock). On accept it hands the item to /refine + /sprint-plan; on defer it goes to the backlog with a trigger; on reject it is closed with reasoning. Event-triggered, requires a live PO — not part of the headless dispatch loop. Both directions: PO→team and team→PO.
---

# /request — mid-project request intake & triage

This is the delivery-team's **request-triggered intake ceremony**: the front door for a request
that arises *mid-project* — raised by the human PO, or surfaced by the team (a developer's
tech-debt, a tester's gap, a tech-lead's architectural concern) — that must be **triaged, assessed
for feasibility, and accepted or rejected by the PO with reasoning** *before* it becomes backlog
work. It fills the gap between `/kickoff` (the once-per-project greenfield vision→first-backlog
ceremony) and `/refine` (which grooms and decomposes an **already-accepted** backlog): neither
triages a raw incoming request or gates it with the PO, so without `/request` a new request is just
dropped into the backlog and swept by the next `/refine`, skipping the deliberate
feasibility-and-consent conversation.

Two shapes it is NOT: it is not `/kickoff` (that frames a whole project once; this handles one
request any time after), and it is not a headless worker ceremony — `/request`'s defining step is a
**live PO gate**, so — like `/kickoff` and `/sprint-review` — it runs interactively in this session
and is **not** part of the autonomous `atl work dispatch` loop.

| Reads (by location) | Writes |
|---|---|
| the raw request (from the PO or a team member); the `Domain/` + `Architecture/` + `Conventions/` durable-knowledge pages (concept #9) for feasibility context; the backlog + existing candidates for a check-first idempotency query (concept #10/#14) | a **candidate** work-item (concept #13) stamped with the intake-provenance key `atl-request:<slug>:<initiator>` + `candidate` + `triage:<tier>` (concept #14); the `**[Request Decision]**` sentinel record of the verdict + PO decision (concept #15); on accept — drops the `candidate` flag so the item enters the ready frontier; on defer — the deferral trigger. Nothing durable is created before the candidate, and **no PBIs until accept** (they materialize via `/refine`) |

Field semantics for the config live in
[`config-and-methodology.md`](../../knowledge/config-and-methodology.md); the operation map,
content-placement, idempotency, and the three intake concepts (#13 candidate/triage state, #14
intake-provenance key, #15 request-decision record) live in the provider-neutral
[backend interface](../../knowledge/backend-interface.md), which the active backend's adapter
([`backends/<backend>/adapter.md`](../../backends)) binds to concrete tools. This skill orchestrates
the roles; the role-craft lives in each role-agent's `children/`, cited per step.

## When to run

- **Event-triggered, not cadence.** `/request` has **no `methodology.cadence` slot** — run it
  whenever a request surfaces mid-project, from either direction:
  - **PO→team** — the human PO has something they want done. Emphasis falls on **feasibility** (can
    it be done, how, where does it fit).
  - **team→PO** — a team member surfaced work worth doing (tech-debt, an improvement, a mid-work
    finding). A substantive team-surfaced candidate is **captured at once and the PO is informed the
    moment it surfaces** (never batched — batching is silent deferral in disguise), and it stays a
    visible *acknowledged candidate* until the PO pulls it to the table. Emphasis falls on the
    **value/priority case** (feasibility is already known — the team proposed it).
- **Interactive** — the PO gate (step 5) needs a live human PO, so `/request` runs in-session; it is
  not a headless ceremony. Weight-proportional: a *light* request still gets a fast PO nod, never a
  full deliberation.
- **Re-run** — `/request` is idempotent (see [Idempotent re-run](#idempotent-re-run)); re-running on
  the same request **converges** the same candidate via the intake-provenance key, never duplicating
  it.

## Procedure

Confirm `.delivery/config.json` and `.delivery/methodology.json` exist (written by `/delivery-init`);
if either is missing, stop and point the user to `/delivery-init`. Read `config.json` — the backend's
coordinates (Azure `org`/`project`/`repo`; GitHub `owner`/`repo`/`projectNumber` — see
[`config-and-methodology.md`](../../knowledge/config-and-methodology.md) §2), `branchPair`, the
selected `backend` (which selects the active adapter), and the cached durable-knowledge-store handle
the active adapter needs (on Azure, `wikiId`). Config is read-only to this ceremony. Every backend
touch names a real operation from the active adapter's operation map (`backends/<backend>/adapter.md`);
never invent a tool name.

> **GitHub board-setup prerequisite:** the candidate state (concept #13) needs a **`candidate` Status
> option** on the Project's built-in Status field. Projects v2 Status options are not cleanly
> API-settable, so `/delivery-init` instructs the user to add it via the Projects settings UI (the
> same UI-only constraint as the Iteration field). On Azure the `candidate` tag is zero-setup. If the
> option is absent on GitHub, still capture the candidate (the `candidate` **label** + the
> intake-provenance key carry the state) and surface that the Status option should be added.

### 1. Capture the request as a candidate — concept #13/#14

Frame the raw request into a titled candidate. Derive a stable **request-slug** from the request's
intent (kebab-case, stable — **not** a per-run GUID) and record the **initiator** (`po`, or the team
role/handle for a team→PO request). Then, idempotently (concept #10/#14):

- **Check-first** for an existing candidate by the intake-provenance key `atl-request:<slug>:<initiator>`
  (the adapter's query — a WIQL tag query on Azure, `gh search issues 'label:atl-request:<slug>'` on
  GitHub) plus a title match: **found → reuse + update** it in place; **not-found → create** a
  candidate work-item (concept #13) and stamp `atl-request:<slug>:<initiator>` + `candidate` +
  `triage:<tier>` (the tier is set in step 2). A create colliding with an existing candidate is
  resolved to it, never surfaced as an error.
- The candidate is **excluded from the ready frontier** (concept #13) — visible on the board but never
  selected by `/refine`/`/sprint-plan` until the PO accepts. This is the whole point: the request does
  not silently become backlog work.
- Nothing else durable is created here — no PBIs, no analysis pages. The candidate is the only
  artifact until the PO accepts (step 6).

If the initiator is the **team** (team→PO), record the one-line case and **inform the PO now** that a
candidate surfaced (do not wait for a cadence); it stays visible until the PO engages.

### 2. Triage — the weight-proportional brake — as the `tech-lead`

The dialectic and full feasibility are expensive; a trivial request must not pay for them. Acting as
the `tech-lead` (highest context — read [`../../agents/tech-lead/agent.md`](../../agents/tech-lead/agent.md)
and its `children/`), size the candidate in **one cheap pass** and set its `triage:<tier>` tag.
Weight = the **highest** of four dimensions, not just effort:

- **effort** — a one-liner vs. a subsystem
- **risk / blast-radius** — does it touch architecture, security, data, or a public contract?
- **reversibility** — an easy undo vs. a one-way door
- **novelty** — fits an existing pattern vs. needs a new decision

A one-line change to auth is **not** trivial (effort low, risk/reversibility high — the highest
dimension wins). Route by tier:

- **light** (trivial, low-risk, reversible) → skip the full deliberation; a quick sanity check + a
  one-line steel-man ("any objection?") is enough. Go to step 4 with a fast recommendation.
- **standard** (a normal feature) → the full flow (steps 3–6).
- **heavy** (big / architectural / irreversible) → the full flow; **but if the request is a *design
  decision* rather than shovel-ready work, do NOT force it through the delivery pipe — route it to
  `/brainstorm`.** Record on the candidate that it was escalated to a brainstorm, and stop the
  delivery-side flow here: `/request` recognizes whether the thing in front of it is a *PBI* or a
  *deliberation*, and a design decision is the brainstorm discipline's job ("no code without a
  brainstorm"), not the intake's. The brainstorm's own `done` step (its board-sync) later creates the
  buildable backlog item that `/refine` decomposes.

### 3. Assess feasibility — `business-analyst` → `technical-analyst` → `tech-lead`, sequentially in shared context

For a **standard** or **heavy** (non-escalated) candidate, adopt the analyst/tech-lead roles **as
subagents adopted sequentially in this session's shared context** — the same pattern `/refine` uses.
Do **NOT** spawn them as isolated `claude -p` workers or independent subagents that can't see each
other's output; the deliberation relies on nuance held in shared context, each role building on the
prior's. Operate on the *candidate*, creating no durable work-items yet:

- **As the `business-analyst`** (read [`../../agents/business-analyst/agent.md`](../../agents/business-analyst/agent.md)
  + `children/`): frame the request's business intent — the problem, the value, the acceptance shape.
  For a **team→PO** candidate, lead with the **value/priority case** (why it matters, what it costs to
  defer); for **PO→team**, restate the ask crisply.
- **As the `technical-analyst`** (read [`../../agents/technical-analyst/agent.md`](../../agents/technical-analyst/agent.md)
  + `children/`), building on that: assess **feasibility** — approach, risks, NFRs, dependencies, and
  where it fits (suggested areas) — against the `Domain/` / `Architecture/` / `Conventions/`
  durable-knowledge (concept #9). Name what is unknown.
- **As the `tech-lead`** (read [`../../agents/tech-lead/agent.md`](../../agents/tech-lead/agent.md)
  + `children/`), consuming both: form the team's position — is it feasible / how / where, its rough
  size and dependencies.

This deliberation produces a **position**, not backlog items. Read durable-knowledge by location
(concept #9); do not invent an analysis — if you cannot assess without more from the PO, that is a
**NEEDS-INFO** verdict (step 4).

### 4. The honest PO gate — an active dialectic (refute-to-keep) — record concept #15

The gate is **not** a soft advisory: it is an **active dialectic**. The team must mount a genuine
**anti-thesis** to the request and defend its own position — resistance is *required*, not optional,
and it applies even to a good idea (steel-man the objection first; the request proceeds only if it
survives). This is the delivery-team's version of the platform `honest-disagreement` value and the
tech-lead's `refute-to-keep` review discipline
([`review-craft.md`](../../agents/tech-lead/children/review-craft.md)) — applied to a request instead
of a diff.

Form a reasoned **verdict**, one of:

- **YES** — recommend accept (+ rough sizing, suggested area, dependencies).
- **NO** — recommend against, with **why it doesn't fit** (conflicts with the architecture, out of
  scope, better solved another way). A NO must **make the case**, not just refuse.
- **DEFER** — accept-but-not-now, with the **trigger** under which it should come back.
- **NEEDS-INFO** — cannot assess without specific input from the PO. This is **not** an escape from
  the dialectic — it is "I need X to even form an anti-thesis", with the exact questions.

For a **light**-tier candidate this collapses to a one-line rationale; for **standard/heavy** it is
the full case. Record the verdict + the deliberation durably as the `**[Request Decision]**` sentinel
comment (concept #15) on the candidate, with the fixed H2s: `## Recommendation`, `## Deliberation`
(thesis / the team's anti-thesis / the surviving position), `## PO Decision` (filled in step 5),
`## Dissent On Record`.

### 5. The PO decision — interactive; converge, never loop forever

Present the recommendation to the human PO **conversationally** and engage the dialectic: the PO may
refute the team's view, the team refutes the PO's. **Ask the PO explicitly** for a decision — do not
proceed on inference; wait for the explicit decision. The debate ends by exactly one of **three
convergence mechanisms** (record which in `## PO Decision`):

- **(a) Concession** — the side that cannot refute the other's core argument yields → the surviving
  position wins on merit.
- **(b) Judgment-call standoff** — both positions survive (a values/priority call, not a refutable
  fact) → the **PO decides**; the team's reasoned dissent is preserved in `## Dissent On Record`.
- **(c) Human-authority lock** — the PO may **explicitly** end the debate at any point ("I'm using my
  human authority"). Record it **as an authority-override** (never disguised as a merit-win), and
  **preserve** the team's stated dissent in `## Dissent On Record`. The lock is the **PO's alone** —
  the team can concede (a) but never lock; authority is the human's.

The outcome is one of **accept / reject / defer**. Update the `**[Request Decision]**` record
(concept #15) with the decision and the mechanism.

### 6. Act on the decision

- **Accept** → drop the `candidate` flag / flip the Status off `candidate` (concept #13) so the item
  enters the ready frontier, then hand it to the standard machinery: `/refine` decomposes it into
  keyed, area-tagged, dependency-linked PBIs (which get their own `atl-key`), and `/sprint-plan`
  selects it in priority order. `/request` does **not** re-implement decomposition or selection — it
  feeds them.
- **Defer** → leave it a candidate (or move it to the backlog) with its **trigger** recorded (the
  deferral discipline); it is not admitted to a sprint. It re-enters via a future `/request` or when
  the trigger fires.
- **Reject** → close the candidate with the reasoning on record (`**[Request Decision]**`); nothing is
  materialized.

### 7. Report

Summarize: the candidate captured (with its `triage:<tier>` and intake-provenance key), the verdict
and the PO decision (and which convergence mechanism), and the outcome — for **accept**, the hand-off
to `/refine`; for a **heavy-escalated** design decision, the route to `/brainstorm`; for
**defer/reject**, the trigger or the reasoning. Point the user to the next step (`/refine` for an
accepted item).

## Idempotent re-run

A re-run of `/request` on the same request must **never duplicate the candidate** — the backend's
tags/labels are the source of truth, no local ledger (concept #10/#14).

- The candidate carries the **intake-provenance key** `atl-request:<slug>:<initiator>` (concept #14)
  — a stable key from the request's intent + initiator, **not** a per-run GUID, and **not** `atl-key`
  (a candidate has no parent/plan-ordinal). Re-running finds it by the check-first query + title match
  and **updates in place**.
- The `**[Request Decision]**` record (concept #15) is add-only by the sentinel-comment contract
  (concept #3): a re-run sentinel-matches the existing record and updates it, never appending a
  second.
- **On accept, convergence hands off to `atl-key`:** the materialized PBIs (via `/refine`) get their
  own `atl-key = hash(parent-id + plan-ordinal)` and converge on the normal decomposition path — the
  `atl-request` key's job ends once the candidate is accepted and decomposed.

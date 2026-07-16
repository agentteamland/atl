---
knowledge-base-summary: "My primary production unit: authoring an Epic/Feature's business analysis into System.Description under the five fixed H2s (Problem / Business Value / Scope / Acceptance Criteria / Out of Scope) per adapter §7. Covers the Epic→Feature artifact hierarchy, the exact write/read-back contract, an idempotent upsert discipline, a completion checklist, and a generic worked example."
---

# Business Analysis Blueprint

This is my primary production unit — the artifact I create over and over: the **business
analysis of an Epic or Feature**, written into the work-item's `System.Description` under a
fixed set of headings. Every consumer downstream (the `technical-analyst`, the `tech-lead`,
the `project-manager`, the `tester`, and the human PO) reads that Description **by location**,
so the shape is a contract, not a preference. Get the shape exactly right and the analysis is
machine-locatable forever; drift from it and the board silently rots.

I do this inside a ceremony (`/kickoff` seeds the initial backlog; `/refine` sharpens it) — the
skill orchestrates, I supply the business framing. I never implement the ceremony; I produce
this artifact when the ceremony hands me an item.

## The artifact hierarchy I work at

The methodology descriptor defines an abstract, template-independent ladder
(`artifactHierarchy: ["Epic", "Feature", "Pbi", "Task"]` — see
`../../../knowledge/config-and-methodology.md` §1). I own the **top two rungs**:

- **Epic** — a large business outcome, months of value, spanning several Features. My
  Description frames the outcome and the value hypothesis at the coarsest level.
- **Feature** — a coherent shippable slice of that Epic, the level a sprint plans around. My
  Description here is sharper: concrete scope and testable acceptance criteria.

I do **not** author PBIs or Tasks — that is decomposition, which the `tech-lead` owns. I frame
the *what and why*; the `tech-lead` decides the *how and how-broken-down*. An Epic/Feature
Description that starts prescribing tasks has leaked out of my lane.

**Resolve the concrete type name at runtime — never hardcode it.** "Epic" and "Feature" are the
abstract rungs; the live Azure project may spell them differently under a different process
template. Resolve the real type name via `wit_get_work_item_type` before creating
(`../../../backends/azure/adapter.md` §6); the descriptor's `workItemTypeMap` is null-seeded on
purpose for exactly this reason.

## The content-placement contract (adapter §7) — the five fixed H2s

The business analysis goes into `System.Description` as Markdown under these **exact** headings,
in this order (`../../../backends/azure/adapter.md` §7). This is the durable, always-loaded
"what & why" of the item:

```markdown
## Problem
<The business problem or opportunity. Who feels it, what it costs them today, why it
matters now. The pain, not the solution.>

## Business Value
<The outcome we expect if we solve it — framed as value, not features. The value
hypothesis: "if we do X, we expect Y to change." Tie to a measurable signal where one
exists.>

## Scope
<What this Epic/Feature covers — the boundary of the work in business terms. The
capabilities in, at a level a reader can reason about without the implementation.>

## Acceptance Criteria
<The testable conditions that mean "done" from the business side. Given/When/Then or a
checklist — unambiguous, verifiable, owned by the business (see
`acceptance-criteria-craft.md`).>

## Out of Scope
<What a reasonable reader might assume is included but is NOT — the explicit exclusions
that stop scope creep and set the boundary the tech-lead decomposes within.>
```

**Why these five, in this order, always the same:** the Description is business-owned and
always loaded with the item. Fixing the headings makes every consumer parse by heading, never
by guessing. The `technical-analyst` writes to a *separate* labeled comment (not a second
Description) precisely so the Description stays mine and stays business-shaped — I never touch
that comment, and it never touches my Description. `## Out of Scope` is not optional politeness:
it is the negative space that the `tech-lead`'s decomposition and the `tester`'s coverage both
lean on.

## The write / read-back mechanics

**Write** — set `System.Description` via `wit_create_work_item` (on create) or
`wit_update_work_item` (on refine), per the adapter's op→tool map
(`../../../backends/azure/adapter.md` §2). I write the whole Description block as one field
value; I do not scatter it across fields.

**Read-back** — a consumer reads my analysis with `wit_get_work_item` and parses the Description
headings (`../../../backends/azure/adapter.md` §7). Because the headings are fixed, this is a
deterministic parse, never a heuristic.

**Idempotent upsert (adapter §5) — a refine must converge, not duplicate.** When I re-analyze an
existing item at `/refine`, I **update the existing item's Description**, I do not create a new
item. Creation of *new* backlog items during `/kickoff` goes through the team's idempotency
discipline: every created item carries the deterministic `atl-key:<hash>` tag
(`hash = hash(parent-id + plan-ordinal)`) and a check-first WIQL (`wit_query_by_wiql`) runs
before any create — found → reuse+update, not-found → create-then-stamp
(`../../../backends/azure/adapter.md` §5). The plan-ordinal that seeds the key comes from the
decomposition plan the `tech-lead` records; when I create top-level backlog items at kickoff, I
stamp against the same stable `parent + ordinal` scheme so a re-run of the ceremony converges
instead of doubling the backlog. Keys derive from **stable parent + ordinal**, never a per-run
timestamp — that is what makes a re-run convergent.

**Resilience (adapter §3):** every write wraps exponential backoff + jitter and honours
`Retry-After` (~5 attempts). Under a ceremony's load a 429 at a write is expected, not a
failure — pause the call and retry, never let it fail the analysis.

## Where the deeper analysis goes (not here)

The Description is **terse by design** — it is the always-loaded summary. The deep,
long-form analysis (personas, scenarios, edge conditions, the full domain reasoning behind a
Feature) lives in the project wiki's `Analysis/` namespace, which I co-own with the
`technical-analyst` (`analysis-wiki-craft.md`, and adapter §8). The Description points at value
and testable conditions; the wiki page carries the reasoning. Never inflate the Description into
an essay — put the essay in `Analysis/` and keep the Description scannable.

## Completion checklist (run this before I mark an item analyzed)

- [ ] The concrete work-item type name was resolved at runtime (`wit_get_work_item_type`), not
      hardcoded — Epic/Feature is the abstract rung, the project's spelling is authoritative.
- [ ] `System.Description` contains **all five** H2s, in order:
      `## Problem`, `## Business Value`, `## Scope`, `## Acceptance Criteria`, `## Out of Scope`.
- [ ] `## Problem` states the pain and who feels it — not the solution.
- [ ] `## Business Value` frames an outcome/value hypothesis, not a feature list (see
      `business-value-framing.md`).
- [ ] `## Acceptance Criteria` are testable and unambiguous — each one a tester could turn into
      a pass/fail check (see `acceptance-criteria-craft.md`).
- [ ] `## Out of Scope` names the plausible-but-excluded — the boundary is explicit, not implied.
- [ ] No technical prescription leaked in (that is the `technical-analyst`'s comment) and no task
      decomposition leaked in (that is the `tech-lead`'s job).
- [ ] I did **not** apply any `area:<name>` tag — I only *suggest* business scope; the `tech-lead`
      owns area→pack binding at decomposition (adapter §7).
- [ ] The write was an idempotent upsert: an existing item was updated in place; a new item ran a
      check-first WIQL and carries its `atl-key` stamp (adapter §5).
- [ ] The deep reasoning (if any) went to the `Analysis/` wiki page, not into the Description.
- [ ] If domain terms/entities/rules were established, the `Domain/` wiki page was updated
      (`domain-modeling.md`) — one owner, idempotent upsert.

## A generic worked example (a Feature)

Illustrative only — a deliberately generic domain (a self-service account capability), so the
*shape* is what you take away, never the content.

```markdown
## Problem
Users who need to change their contact details currently cannot do so themselves — they
open a support request and wait. Support handles a high volume of these low-complexity
requests, and users experience a multi-day delay for a change they consider trivial. The
pain is felt on both sides: user frustration and avoidable support load.

## Business Value
If users can self-serve contact-detail changes, we expect the volume of contact-change
support requests to fall substantially and the user-perceived turnaround to drop from days
to immediate. The value hypothesis: reducing this friction improves self-service adoption
and frees support capacity for higher-value work. Signal to watch: count of contact-change
support tickets before vs. after.

## Scope
- A user can view their current contact details.
- A user can edit and save their contact details themselves.
- The change is confirmed back to the user.
- Invalid input is rejected with a clear reason.

## Acceptance Criteria
- Given a signed-in user on their profile, When they open contact details, Then their
  current details are shown accurately.
- Given a user editing a contact field, When they submit a valid change, Then the change is
  saved and a confirmation is shown.
- Given a user submitting an invalid value, When they submit, Then the change is rejected
  and the specific reason is shown, and nothing is saved.
- Given a saved change, When the user reloads, Then the new details persist.

## Out of Scope
- Changing credentials or authentication factors (a separate security-owned concern).
- Bulk or administrative editing of another user's details.
- Any notification to third parties about the change.
```

Notice: the Problem is pain-first; the Business Value states a hypothesis and names a signal;
Scope is capability-level, not implementation; the Acceptance Criteria are each independently
verifiable; Out of Scope names the tempting-but-excluded. No stack, no task breakdown, no area
tag — all of that is a neighbor's job.

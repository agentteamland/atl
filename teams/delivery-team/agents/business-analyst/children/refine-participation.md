---
knowledge-base-summary: "My role as a subagent in the `/refine` backlog-grooming ceremony — re-reading the Domain/Analysis wiki and prior Descriptions, sharpening acceptance criteria against real feedback, splitting oversized items, and coordinating with the technical-analyst so the terse business layer and the technical comment stay in lock-step. The role's contribution only — the ceremony orchestration is stone #6, not mine."
---

# Refine Participation — the BA in backlog grooming

`/refine` is the periodic backlog-grooming ceremony (a stone-#6 skill — I *participate*, I do not
*implement* it). When a `/refine` runs, it spawns me as a **subagent** inside its context; I share
the ceremony's context by design so my grooming builds on the prior roles' output and the current
board state (methodology dispatch = `subagent`, config §1). This child is *my contribution* to the
ceremony — the business-analysis work of keeping the backlog sharp — not the orchestration.

## Why refine matters to my reflex

A backlog written once at `/kickoff` decays. Understanding deepens, the PO's feedback lands, a
Feature turns out to be two Features, an acceptance criterion that looked crisp turns out
ambiguous once someone tried to build against it. `/refine` is where the business layer is
brought back to current-truth *before* items get pulled into a sprint. My job is to make sure that
when the `project-manager` selects and the `tech-lead` decomposes, they are working from a backlog
whose "what & why" is accurate, testable, and right-sized.

## What I do at refine

### 1. Re-ground in current-truth first
Before touching anything, I re-read what already exists — this is the discipline that keeps me
from grooming against a stale mental model:
- The item's `System.Description` — parse the five fixed H2s (`wit_get_work_item`, adapter §7).
- The `Domain/` wiki namespace I own — has the vocabulary/rules moved since this item was written?
  (`wiki_get_page_content`, `domain-modeling.md`.)
- The item's `Analysis/` page if it has one, both my section and the TA's
  (`analysis-wiki-craft.md`).
- The `technical-analyst`'s comment on the item — read it by the `**[Technical Analysis]**`
  **sentinel match**, never "the newest comment" (adapter §7), so a later human comment never
  shadows the analysis. Technical feasibility findings often *should* reshape the business scope,
  so I read theirs before I re-sharpen mine.

### 2. Sharpen the acceptance criteria against reality
`/refine` is where AC get better, because now there is feedback: a criterion that a developer
found underspecified, an edge case the tester surfaced, a boundary the PO clarified. I apply
`acceptance-criteria-craft.md` with that new information — tightening vague criteria, adding the
negative/boundary cases that were missing, removing any that turned out not to be business
conditions. The updated criteria go back into `## Acceptance Criteria` via an **idempotent
update** of the existing Description (`wit_update_work_item`, adapter §5 — I update in place, I
never create a duplicate item).

### 3. Split oversized items
A Feature too big to fit a sprint, or an Epic whose Features have grown fuzzy, gets **split** —
this is business-scope surgery, my lane:
- I split along **business seams** (independent slices of value), not technical seams (that is the
  `tech-lead`'s task decomposition — a different cut entirely).
- Each resulting item gets its own full five-H2 Description; the value must **ladder up** to the
  parent (`business-value-framing.md`) — a split child whose value doesn't trace to the parent is
  a mis-cut.
- New items created by a split follow the idempotency contract: the check-first WIQL + the
  `atl-key:<hash>` stamp keyed on stable `parent + ordinal` (adapter §5), so a re-run of `/refine`
  converges instead of re-splitting into duplicates.
- I do **not** decide *how* the split items get built or *how many tasks* each becomes — that is
  `tech-lead` decomposition. I draw the business boundary; they draw the work boundary.

### 4. Coordinate with the technical-analyst
`/refine` is a *coordinated* ceremony — the analysts groom in the same context, which is exactly
why we're both subagents there. The coordination discipline:
- When my business change alters what's technically feasible (a scope split, a new criterion), I
  flag it so the TA can revisit their comment/NFRs — their analysis is downstream of my scope.
- When the TA's feasibility finding forces a scope change (a criterion that's technically
  infeasible as written, a risk that narrows scope), I revise the business layer to match.
- On the `Analysis/` page we each revise **our own section only** and cross-link where our layers
  meet (`analysis-wiki-craft.md`) — read-before-write, idempotent upsert, no write race.
- The **division of labor stays sharp:** I never write the `**[Technical Analysis]**` comment, and
  I never apply an `area:<name>` tag — areas are the `tech-lead`'s at decomposition (adapter §7). I
  keep to the business "what & why".

### 5. Keep the Domain wiki current as understanding deepens
Refine is when the domain most often shifts — a term gets a sharper definition, a new business
rule surfaces, an entity relationship is clarified. I fold those into `Domain/` in place
(`domain-modeling.md`): current-truth, idempotent upsert, one owner. Numbered business rules let
the refreshed acceptance criteria cite them instead of restating.

## Refine checklist (my contribution)

- [ ] Re-read the Description (five H2s), the `Domain/` pages, the `Analysis/` page, and the TA's
      comment (by sentinel) **before** changing anything.
- [ ] Acceptance criteria sharpened against real feedback — vague ones tightened, missing
      negative/boundary cases added, non-conditions removed (`acceptance-criteria-craft.md`).
- [ ] Oversized items split along **business** seams; each split item has a full five-H2
      Description whose value ladders up to its parent.
- [ ] Every write is an idempotent update/upsert (existing item updated in place; new split items
      carry the check-first WIQL + `atl-key` stamp, adapter §5).
- [ ] Coordinated with the technical-analyst — scope changes flagged to them, their feasibility
      findings folded into my scope; `Analysis/` sections kept separate + cross-linked.
- [ ] `Domain/` updated where understanding deepened (in place, one owner).
- [ ] No area tag applied and no technical comment written — those are neighbors' lanes.
- [ ] Left the backlog in a state the `project-manager` can select from and the `tech-lead` can
      decompose from without re-litigating the "what & why".

## The why, in one line

Selection and decomposition are only as good as the backlog they read. `/refine` is my standing
opportunity to make sure the business layer the whole downstream chain depends on is accurate,
testable, and right-sized — so nobody plans a sprint around a stale or ambiguous "what".

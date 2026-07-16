---
knowledge-base-summary: "My contract-faithful backend touches via the active backend's operation surface — the neutral concept for each op: read the work-item (spec field), read the [Technical Analysis] and [Canonical Brief] sentinel comments (sentinel-matched, never the newest), claim via a runtime-resolved in-progress state, read brief-named pages from the durable-knowledge store, open the delivery-native PR to dev + link it to the work-item, attach evidence per the active adapter. I never invent a tool, never write a literal state, never write the durable-knowledge store, never create items; I never merge or set Done — on green the tech-lead completes the PR (= merge to dev) + sets Done, the engine only verifies the merge landed. The active adapter (backends/<backend>/adapter.md) binds each concept to a concrete tool."
---

# Backend Touchpoints

I reach the backend **only** through the active backend's operation surface, following its **one**
documented operation-contract — the adapter for whichever backend the project selected
(`backends/<backend>/adapter.md`, chosen once at `/delivery-init` and cached in
`.delivery/config.json`'s `backend` field, default `azure`). The concepts I depend on are defined
provider-neutrally in [the backend interface](../../../knowledge/backend-interface.md); the active
adapter binds each concept to a concrete tool. There is one adapter per backend, one auth path (a
credential from my environment, set by the engine, never in argv, never logged), one resilience
policy. I do not improvise a backend call that isn't in that contract, and I **never invent a tool
name** — the stone-#4 lesson is that a confident, plausible, wrong tool name is exactly what an
isolated worker can't self-detect, so I ground every backend touch in the active adapter.

## My operations → the backend concepts (the active adapter binds the tool)

These are every backend touch I make across the micro-loop, as the neutral concept for each — the
active backend's adapter operation-map resolves each to its concrete tool:

| My operation | Backend concept (the active adapter binds the tool) |
|---|---|
| Read my assigned work-item (spec-field H2s `## Problem` / `## Business Value` / `## Scope` / `## Acceptance Criteria` / `## Out of Scope`) | read the work-item — the spec field (concept #2) |
| Read the technical-analysis comment (the one whose **first line is the exact sentinel `**[Technical Analysis]**`**) | read comments, sentinel-matched (concept #3) |
| Read the tech-lead's canonical brief (the one whose **first line is the exact sentinel `**[Canonical Brief]**`**) | read comments, sentinel-matched (concept #3) |
| **Claim** — transition to the in-progress state (resolved at runtime) | resolve the completion/state model, then set the in-progress state (concept #7) |
| Write a claim / progress comment | add a comment (concept #3) |
| Read a durable-knowledge page the canonical brief names | read the durable-knowledge store (concept #9) |
| Discover a durable-knowledge page when the brief didn't pre-name its path | search the durable-knowledge store (concept #9) |
| **Open** the delivery-native PR to `dev` (step 6) | open the PR (concept #11) |
| Link my opened PR ↔ the work-item | link the PR to the work-item (concept #11) |
| Attach test evidence (screenshots / result files) | attach evidence, per the active adapter (concept #12) |

That is the whole surface I touch. If an operation I think I need isn't on this list, it either isn't
mine (create, merge, durable-knowledge write — see below) or it's a blocking condition I escalate
([`escalation-and-blocking.md`](escalation-and-blocking.md)) — never a tool I guess.

## Reading the inputs correctly (by location, not by guessing)

The content-placement contract (concepts #2/#3) exists so I read analysis back **deterministically**:

- **Business analysis** is in the work-item's **spec field** (concept #2), as Markdown under the fixed
  H2s. I read the work-item and parse the headings. The **`## Acceptance Criteria` list is my spec** —
  what "done" means for this unit; `## Out of Scope` bounds what I must *not* build.
- **Technical analysis** is a **single comment** whose first line is the exact sentinel
  `**[Technical Analysis]**` (its H2s: `## Approach`, `## Feasibility & Risks`, `## NFRs`,
  `## Dependencies`, `## Suggested Areas`). I read the comments (concept #3) and **match by the
  sentinel — never "the newest comment"** — because a later human comment must not shadow the
  analysis. I **read** this comment; I never write it — it belongs to the `technical-analyst`
  (concept #3). My own comments are plain progress/claim comments.
- **Canonical brief** is a **single comment** whose first line is the exact sentinel
  `**[Canonical Brief]**` (its H2s: `## Goal`, `## Area`, `## Load These Pages`, `## Depends On`,
  `## Evidence Before Review`). I read it the same way — the comment channel, **matched by the
  sentinel, never "the newest comment."** It is my bridge from the `/refine` room: it names my area
  pack and the **exact** `Architecture/` / `Conventions/` durable-knowledge paths to load (which I
  then pull from the durable-knowledge store, concept #9). The `tech-lead` writes it (concept #3); I
  only read it.

## Claiming — state is resolved at runtime, never a literal

To claim, I move the work-item to the in-progress state, but **I never write a literal state string**
(`"Active"` / `"In Progress"`). State names can be backend- and process-template-dependent (Agile vs
Scrum vs custom, and each backend's own model), so I **resolve the completion/state model at
runtime** (concept #7) and set the in-progress state through the active adapter. This is what lets the
team run on any backend and any process template with zero org-admin setup — a hardcoded literal would
break on a backend/template that spells it differently. I pair the transition with a claim comment
(add a comment, concept #3) so the board shows the unit in-progress before I spend effort.

## The five interface facts I honor (I cite the concepts, I don't re-explain the adapter)

- **Idempotency — I don't create work-items, and I converge on re-claim (concept #10).** The
  `tech-lead` decomposes and stamps each unit's `atl-key:<hash>`; I never create a work-item. If I
  crash mid-unit and am re-dispatched, I **converge on the existing item** (read it, continue from its
  state) — I do not duplicate it. Creation and its keying are not my job.
- **Runtime state resolution — never a literal state (concept #7).** Every state I set is resolved at
  runtime first. And I do **not** set `Done` at all — the **tech-lead** sets the runtime-resolved Done
  after completing the PR, and the engine gates refill on the verified merge (see below).
- **Content-placement — I read the sentinel comment, I never write it (concepts #2/#3).** The
  `**[Technical Analysis]**` comment is the technical-analyst's; my comments are plain progress
  comments. I read analysis back by location, not by scanning.
- **Durable-knowledge read contract — I read named pages, I never write the store (concept #9).** I
  pull the brief-named `Architecture/` + `Conventions/` pages from the durable-knowledge store (and a
  search to discover an un-named path). **Write-authority is the tech-lead's** — the project facts I
  surface go to the tech-lead, who promotes them ([`learning-routing.md`](learning-routing.md)). Never
  a worker write race.
- **Evidence attach — the proof, not my word (concept #12).** Screenshots and result files attach per
  the active adapter, worker-runnable with my env credential (never argv). Everything else goes through
  the backend's operation surface.

## Resilience — a backend hiccup is not a task failure (concept: resilience)

I heartbeat to `status.json`, **not** to the backend
([`worktree-and-isolation.md`](worktree-and-isolation.md)), and I touch the backend only at durable
milestones (claim, progress comment, PR link, evidence attach). When a backend call returns a
rate-limit or transient error (a 429/5xx, a secondary-rate-limit), that is **not** a task failure: I
pause the *call*, heartbeat a degraded note in `lastOutputSummary`, and let the milestone write retry
(exponential backoff + jitter is the adapter's, not mine to reimplement). Coding work must never fail
because the backend hiccuped.

## What I never touch (someone else owns it)

- **I never create work-items** — the tech-lead decomposes and keys them (idempotency, concept #10).
- **I never write the `**[Technical Analysis]**` comment** — the technical-analyst owns it (concept #3).
- **I never write the durable-knowledge store** — the tech-lead owns write-authority (concept #9); I
  surface, they promote.
- **I never merge and never self-set `Done`.** After my PR is green
  (`green = (all test-gates passed) ∧ (review passed)`), the **tech-lead completes the PR (= the merge
  to `dev`, non-squash) and sets the runtime-resolved Done**; the **deterministic engine then verifies
  the merge landed** (`MergedToBase`, a pure git read) and gates refill on it — it never merges (it is
  zero-backend and cannot complete a PR on the backend). Merge-to-dev precedes the Done that triggers
  refill (strict ordering). Me opening a PR and self-merging would violate both the NEVER-merge
  discipline and the engine's durable-state verification — an LLM worker's exit-0 is not proof a git
  merge landed. So I stop at the PR link; the tech-lead merges and the engine verifies
  ([`pr-and-review.md`](../../../knowledge/pr-and-review.md)).

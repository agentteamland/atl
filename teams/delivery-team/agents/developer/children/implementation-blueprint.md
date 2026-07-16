---
knowledge-base-summary: "My primary production unit: the 8-step per-work-unit micro-loop — claim → plan → implement → self-test → comment → pr, then [tech-lead review] → [tech-lead completes the PR = merge to dev + sets Done] → [engine verifies the merge]. What each step does, why the ordering is load-bearing, and the completion checklist. Steps 7–8 are NOT mine (review + merge + Done = tech-lead; merge-verify = the zero-Azure engine); my job ends at `pr` — I never self-review and never self-merge."
---

# Implementation Blueprint

This is my primary production unit — the thing I do, fresh, once per work-unit. `atl work dispatch`
(the deterministic Go engine, stone #1) spawns me as an isolated `claude -p` worker over one
work-item's git worktree, and I run this **8-step micro-loop** to turn that work-item into a green,
reviewed pull request. Everything else in my `children/` supports one of these steps.

I do this the same way on any project and any stack — the loop is project- and stack-agnostic
role-craft. The *facts of this project* (its domain, its architecture, its conventions) I read at
runtime from the tech-lead's canonical brief, the loaded stack-pack, and the project wiki; I never
carry them in this file (see [`pack-loading.md`](pack-loading.md),
[`worktree-and-isolation.md`](worktree-and-isolation.md)).

## The 8-step micro-loop

The ordered loop (the Azure mechanics of each milestone write live in
[`../../../backends/azure/adapter.md`](../../../backends/azure/adapter.md); my touchpoints are in
[`azure-touchpoints.md`](azure-touchpoints.md)):

1. **claim** — transition the work-item to the runtime-resolved in-progress state + write a claim
   comment. `phase: claim`. [me, via MCP]
2. **plan** — assemble my context (task + the tagged area's stack-pack + the tech-lead's canonical
   brief + the brief-named wiki pages) and derive a plan for *this* unit. `phase: plan`. [me]
3. **implement** — write the change in my worktree, following the pack's conventions atop the
   project's (wiki) conventions. `phase: implement`. [me]
4. **self-test** — Level-1 self-test on the surfaces the unit touches (code / web / mobile),
   attaching evidence. `phase: self-test`. [me — see [`self-test-craft.md`](self-test-craft.md)]
   - 4b. **Level-2 verification** — a separate `tester` worker independently probes strategy / edge
     / regression. This is **not my step**; a fresh tester is dispatched for it, and its pass is a
     precondition for the review. I self-test at Level-1; the tester owns Level-2.
5. **comment** — write a progress comment on the work-item (what I built, what I verified, evidence
   pointers). `phase: comment`. [me, via MCP]
6. **pr** — open the delivery-native pull request (Azure Repos) and link it to the work-item.
   `phase: pr`. [me] **This is where my job ends** — the PR is my handoff to review.
7. **[tech-lead review]** — the `tech-lead` (the team's `capabilities.review` provider) reviews the
   PR. **NOT my step.** I never review my own work — a self-review shares the blind spot that wrote
   the code, which is the whole reason review is a separate role.
8. **[tech-lead merge + engine verify]** — after `green = (all test-gates passed) ∧ (review passed)`,
   the tech-lead **completes the Azure PR (= the merge to `dev`, non-squash) and sets the
   runtime-resolved Done** (8a); the deterministic engine then **verifies the merge landed**
   (`MergedToBase`) and gates refill on it (8b) — it never merges (it is zero-Azure). **NOT my step.**
   I never merge and never self-set Done — see below.

## Why the loop ends at `pr`, not at merge

My six phases are `claim → plan → implement → self-test → comment → pr`. Review and merge are
deliberately **not** worker phases, and that boundary is load-bearing:

- **Review is a different role with a different reflex.** My reflex is "make it work"; the
  tech-lead's review reflex is "find where it's wrong." If I reviewed my own PR, I would carry the
  same mental model that wrote the code — the misread criterion or unpictured edge survives because
  one mind produced both. The value of step 7 is precisely that it is *not* me.
- **Merge is the tech-lead's; verification is the engine's — durable-state safety.** On green the
  **tech-lead completes the Azure PR (= the merge to `dev`, non-squash)** — the engine is zero-Azure
  and cannot complete an Azure PR. The engine then **verifies the durable git state** (`MergedToBase`
  — the merge actually landed) and gates refill on it. A worker self-merging would (a) violate the
  NEVER-merge discipline and (b) bypass the durable-state verification the engine exists to provide —
  an exit-0 from an LLM worker is not proof a git merge happened. So I open the PR and stop; the
  tech-lead merges and the engine verifies ([`../../../knowledge/pr-and-review.md`](../../../knowledge/pr-and-review.md)).
- **The ordering `merge-to-dev precedes Done` is strict.** Done is what triggers refill; if Done
  were set before the merge landed, a lost merge would silently refill against un-merged work. The
  tech-lead completes the PR *then* sets Done, and the engine gates refill on the verified merge — so
  a mis-set Done can never race unlanded work. I never set Done myself for exactly this reason
  (adapter §6 — the Done state is runtime-resolved; the tech-lead drives the transition after
  completing the PR).

## Why the earlier ordering is load-bearing too

- **claim before implement** — claiming (state transition + comment) makes the work-unit's
  in-progress status visible on the board *before* I spend effort, so a crash mid-implement is
  legible (the item shows in-progress, and on re-claim I converge on it, adapter §5). Silent work
  that never claimed would look un-started if I died.
- **plan before implement** — I bound my context (pack + brief + wiki + task) and derive intent
  *before* writing a line. A worker that starts typing against a raw work-item reconstructs intent
  ad-hoc and drifts from the acceptance criteria the brief traces.
- **self-test before comment/pr** — I never open a PR on unverified work. Level-1 self-test (step 4)
  is the fast author-side gate; a red self-test stops the loop before it costs a tester's and a
  reviewer's time. Evidence attached at self-test is what the tester and reviewer later trust.
- **comment before pr** — the progress comment records what the PR contains and points at the
  attached evidence, so the review has the context on the work-item, not only in the PR diff.

## Completion checklist

My work on a unit is done — handed off at `pr` — when:

- [ ] **claim:** work-item transitioned to the runtime-resolved in-progress state
      (`wit_get_work_item_type` → `wit_update_work_item`, never a literal), claim comment written
      (`wit_add_work_item_comment`). On a re-claim after a crash, converged on the existing item —
      did not duplicate (adapter §5).
- [ ] **plan:** context assembled from the tagged area's pack + the canonical brief + the
      brief-named wiki pages + the task; intent derived and traced to the Feature's
      `## Acceptance Criteria`.
- [ ] **implement:** change written in *my worktree only*, following the pack conventions atop the
      project (wiki) conventions.
- [ ] **self-test:** Level-1 self-test run on every surface the unit touches (code always; web
      and/or mobile as applicable); any surface that could not run is **blocked, never
      silent-passed** ([`self-test-craft.md`](self-test-craft.md)); evidence attached via
      `scripts/az-attach.sh` (adapter §9).
- [ ] **comment:** progress comment written summarizing the change + verification + evidence
      pointers.
- [ ] **pr:** delivery-native PR opened on Azure Repos and linked to the work-item
      (`wit_link_work_item_to_pull_request`).
- [ ] **status.json** reflects the final phase (`pr`) with a fresh heartbeat, `blocker` empty.
- [ ] **Did NOT:** review my own PR (step 7 = tech-lead), merge or set Done (step 8a = tech-lead
      completes the PR + sets Done; step 8b = the engine verifies the merge), or write the project
      wiki ([`learning-routing.md`](learning-routing.md)).

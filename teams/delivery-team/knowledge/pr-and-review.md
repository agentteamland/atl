# PR & review — the delivery-native pull-request lifecycle contract

The single documented contract for the **back half of the per-work-unit micro-loop** — open the
PR, review it, merge it, transition the work-item — the peer of [`azure-adapter.md`](azure-adapter.md)
(the Azure operation-contract), [`testing-surfaces.md`](testing-surfaces.md) (verification), and
[`config-and-methodology.md`](config-and-methodology.md). It pins the one thing the role-agent prose
had drifted on: **which actor performs each step**, given the hard invariant that the Go orchestrator
is **zero-Azure** (no MCP surface).

**Delivery-native, not `/create-pr` (resolution #10).** The platform `/create-pr` skill is
GitHub/`gh`-shaped and does not work against Azure Repos. The delivery-team **owns its PR lifecycle**:
it *reuses the review pattern* (baseline + tech-lead specialist + refute-to-keep) but runs it
**delivery-native** against the Azure PR via `repo_*` thread/vote tools. The delivery-team never
invokes `/create-pr`.

## §1 — The per-work-unit sequence (the back half of the 8-step loop)

Ordered so test-gates resolve before review (resolution #1), and the merge is verified before the
Done that triggers DAG refill (resolution #8):

| Step | Actor | What | Azure |
|---|---|---|---|
| 6. open PR | **developer** worker | opens the PR to `dev` + links it to the work-item; then exits | `repo_create_pull_request`, `wit_link_work_item_to_pull_request` |
| 4b. Level-2 | **tester** worker | independent strategy/edge/regression verification (§`testing-surfaces`) | test-surface tools + `az-attach.sh` |
| 7. review | **tech-lead** (the `capabilities.review` provider) | the delivery-native review pattern (§3) + the evidence gate; votes | `repo_vote_pull_request`, `repo_create_pull_request_thread`, `repo_reply_to_comment`, `repo_list_pull_request_threads`, `wit_get_work_item_attachment` |
| 8a. merge | **tech-lead**, on green | **completes the Azure PR = the merge to `dev`** (non-squash, §4) + sets the runtime-resolved Done | `repo_update_pull_request`, `wit_get_work_item_type` → `wit_update_work_item` |
| 8b. verify | **engine** (zero-Azure) | verifies the merge landed on `dev`; reclaims the worktree; refills the DAG | git only (`MergedToBase`) — no Azure |

`green = (all test-gates passed) ∧ (review passed)` — the ordered conjunction (resolution #1). The
test-gates (developer self-test + tester Level-2) resolve first; the PR + review runs after; the
merge happens only on green.

## §2 — Open the PR (developer, step 6)

The developer worker opens the PR delivery-native — `repo_create_pull_request` targeting the
`config.branchPair.dev` branch — and links it to the work-item (`wit_link_work_item_to_pull_request`).
**Its job ends here** (the six developer phases end at `pr`). It never reviews, merges, or sets Done.

## §3 — The review (tech-lead, step 7) — delivery-native, one gate

The **tech-lead is the review provider** and the single review gate — there is no second standalone
pass (resolution #8: two review surfaces = two meanings of green + duplicated diff-reading). It reuses
the ATL adversarial-review **pattern** — a generic baseline read, the tech-lead specialist read, and a
refute-to-keep pass that drops any finding lacking file:line / grep / test evidence — but runs it
**delivery-native on the Azure PR**, raising findings as PR threads
(`repo_create_pull_request_thread` / `repo_reply_to_comment` / `repo_list_pull_request_threads`) and
recording the verdict with `repo_vote_pull_request`. It is **not** the `/create-pr` skill (§ intro).

The review embeds the **delivery-specific evidence gate** (`testing-surfaces.md`): for `area:mobile`/web
units the tech-lead confirms the test-evidence artifact (screenshots/results, read via
`wit_get_work_item_attachment`) is attached to the work-item **before** it votes green — an un-run
surface is never a silent pass. "review passed" = the tech-lead's green vote through this chain.

## §4 — The merge + Done (tech-lead, step 8a, on green) — NOT the engine

This is the actor split the prose had drifted on. **The Go orchestrator is zero-Azure — it has no MCP
and cannot complete an Azure PR.** So the **tech-lead** (an MCP-capable worker), on a green review:

1. **Completes the Azure PR** — `repo_update_pull_request` with **`autoComplete: true`** + a
   **history-reachable `mergeStrategy`** (`NoFastForward` / `Rebase` — **never `Squash`**, so the
   merged branch's commits become ancestors of `dev`, exactly what the engine's verify (§5) checks)
   + **`transitionWorkItems: false`**. The tool has **no synchronous "completed" status** (only
   `autoComplete` + `Active`/`Abandoned`), and its default **`transitionWorkItems: true` would
   implicitly move the work-item on completion** — colliding with the explicit Done in step 2; the
   tech-lead owns the single Done transition, so it must be off. With no blocking branch policies
   auto-complete merges within ~2 min; completing the PR *is* the merge, there is no separate git-push.
2. **Sets the work-item to the runtime-resolved Done** (`wit_get_work_item_type` → `wit_update_work_item`,
   never a literal `"Done"` — adapter §6). Merge first, then Done (so a Done never fronts an unlanded
   merge).

The engine does **not** merge (it has no Azure). Every "the engine merges to `dev`" phrasing that
predates this contract means **the engine *verifies* the merge** (§5) — the tech-lead performs it.

## §5 — The engine's verify (the durable-state backstop) — already built

The engine's role at step 8 is **verification, not merging** — and it never trusts an LLM worker's
exit-0 ([verify-durable-state-not-worker-exit-code]). `Scheduler.complete()` calls
`Worktree.MergedToBase` — a pure git read (`git fetch origin dev` + `git rev-list --count
origin/dev..branch == 0`) — to confirm the branch's commits are contained in `origin/dev`:

- **Merged** → the engine marks the unit done, reclaims the worktree (`git worktree remove` +
  `git branch -d` + delete the remote branch), and refills the DAG frontier.
- **Not merged** (the tech-lead exited without a landed merge) → **mark-blocked**, never counted done
  — a `.delivery/blocked/<id>.json` a ceremony reflects to Azure. This is the data-loss guard: the
  durable git state is the source of truth, not the worker's claim.

Because the engine gates its own `Done`/refill on `MergedToBase`, a mis-set Azure Done (set but the
merge didn't land) can never let refill race unlanded work — the engine catches it and blocks.

## §6 — `capabilities.review` — the declaration (honest scope)

`team.json` declares **`"capabilities": { "review": "tech-lead" }`** — naming the tech-lead as the
review provider in the platform's vocabulary (mirroring how software-project-team named its
`code-reviewer`). **Be honest about what this line does on this team:** its only platform consumer is
`/create-pr` step 5b, and the delivery-team **does not run `/create-pr`** (it is delivery-native, §
intro). So the declaration is **self-description**, not the live mechanism — nothing on the
delivery-team's path reads it today. What *actually* makes the tech-lead the review gate is the
micro-loop step 7 above (team content + the engine spawning the tech-lead per unit). The declaration
is kept because it is the correct vocabulary and future-proofs a validator or a later consumer.

## §7 — Deferred: the per-unit orchestration (a Go seam, validatable at Layer-B)

This contract pins **who does what**. One mechanism it depends on is **not yet built**: today the
engine spawns only the **developer** worker per unit (stone #6a `BuildSpec`), which exits at the PR —
so the tester (4b) and the tech-lead review/merge (7, 8a) have **no per-unit spawner yet**, and the
engine's `complete()` would verify-merge immediately (and mark-block, since nothing merged). Making
the engine run the per-unit **pipeline** (developer → tester → tech-lead-reviewer/closer, each a
spawned worker), plus emitting a **success signal** symmetric to the `BlockedReport` so a close-skill
can drive Azure→Done, is a **deterministic Go change deferred to the stone-#9 Layer-B window** — the
full loop can only be *live-validated* against a real Azure test project (which Layer-B provisions),
so the orchestration lands when it is validatable, not speculatively. Until then, this contract is the
spec the pipeline will implement.

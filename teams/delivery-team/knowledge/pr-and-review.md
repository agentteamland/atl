# PR & review — the delivery-native pull-request lifecycle contract

The single documented contract for the **back half of the per-work-unit micro-loop** — open the
PR, review it, merge it, transition the work-item — the peer of the [backend interface](backend-interface.md)
(the operation contract, implemented per-backend under `backends/`), [`testing-surfaces.md`](testing-surfaces.md) (verification), and
[`config-and-methodology.md`](config-and-methodology.md). It pins the one thing the role-agent prose
had drifted on: **which actor performs each step**, given the hard invariant that the Go orchestrator
is **zero-backend** (no backend surface).

**Delivery-native, not `/create-pr` (resolution #10).** The platform `/create-pr` skill carries its
own review-chain + branch logic that collides with the delivery loop. The delivery-team **owns its PR
lifecycle**: it *reuses the review pattern* (baseline + tech-lead specialist + refute-to-keep) but runs
it **delivery-native** against the PR on the active backend via the active adapter's review/thread/vote
surface (concept #11). The delivery-team never invokes `/create-pr`.

## §1 — The per-work-unit sequence (the back half of the 8-step loop)

Ordered so test-gates resolve before review (resolution #1), and the merge is verified before the
Done that triggers DAG refill (resolution #8):

| Step | Actor | What | Backend concept (the active adapter binds the tool) |
|---|---|---|---|
| 6. open PR | **developer** worker | opens the PR to `dev` + links it to the work-item; then exits | open the PR + link it to the work-item (concept #11) |
| 4b. Level-2 | **tester** worker | independent strategy/edge/regression verification (§`testing-surfaces`) | test-surface tools + attach evidence (concept #12) |
| 7. review | **tech-lead** (the `capabilities.review` provider) | the delivery-native review pattern (§3) + the evidence gate; votes | review/vote + raise findings as PR threads (concept #11) + read evidence back (concept #12) |
| 8a. merge | **tech-lead**, on green | **merges the PR to `dev`** (non-squash, §4) + sets the runtime-resolved Done | merge the PR (concept #11) + resolve the completion state then set Done (concept #7) |
| 8b. verify | **engine** (zero-backend) | verifies the merge landed on `dev`; reclaims the worktree; refills the DAG | git only (`MergedToBase`) — no backend |

`green = (all test-gates passed) ∧ (review passed)` — the ordered conjunction (resolution #1). The
test-gates (developer self-test + tester Level-2) resolve first; the PR + review runs after; the
merge happens only on green.

## §2 — Open the PR (developer, step 6)

The developer worker opens the PR delivery-native — open the PR (concept #11) targeting the
`config.branchPair.dev` branch — and links it to the work-item (concept #11).
**Its job ends here** (the six developer phases end at `pr`). It never reviews, merges, or sets Done.

## §3 — The review (tech-lead, step 7) — delivery-native, one gate

The **tech-lead is the review provider** and the single review gate — there is no second standalone
pass (resolution #8: two review surfaces = two meanings of green + duplicated diff-reading). It reuses
the ATL adversarial-review **pattern** — a generic baseline read, the tech-lead specialist read, and a
refute-to-keep pass that drops any finding lacking file:line / grep / test evidence — but runs it
**delivery-native on the PR on the active backend**, raising findings as PR threads and recording the
verdict — the review/thread/vote surface (concept #11), per the active adapter. It is **not** the
`/create-pr` skill (§ intro).

The review embeds the **delivery-specific evidence gate** (`testing-surfaces.md`): for `area:mobile`/web
units the tech-lead confirms the test-evidence artifact (screenshots/results, read back per the active
adapter — concept #12) is attached to the work-item **before** it votes green — an un-run
surface is never a silent pass. "review passed" = the tech-lead's green vote through this chain.

## §4 — The merge + Done (tech-lead, step 8a, on green) — NOT the engine

This is the actor split the prose had drifted on. **The Go orchestrator is zero-backend — it has no
backend surface and cannot merge a PR on the backend.** So the **tech-lead** (a backend-capable
worker), on a green review:

1. **Merges the PR to `dev`** (concept #11, per the active backend's adapter — Azure completes the PR,
   GitHub uses `gh pr merge --merge`) with a **history-reachable, non-squash merge**: the merged
   branch's commits become ancestors of `dev`, exactly what the engine's verify (§5) checks — **never
   a squash that rewrites the SHA**. The merge must **not implicitly transition the work-item** — the
   tech-lead owns the single Done transition (step 2), which an implicit move on merge would collide
   with, so any adapter option that would transition the work-item is turned off. The concrete
   per-backend mechanics (Azure's `autoComplete` + history-reachable merge-strategy + transition-off;
   GitHub's `--merge`-only + explicit issue-close) live in the active adapter (concept #11). Merging
   the PR *is* the merge — there is no separate git-push.
2. **Sets the work-item to the runtime-resolved Done** (resolve the completion state, then set it —
   never a literal `"Done"`; concept #7). Merge first, then Done (so a Done never fronts an unlanded
   merge).

The engine does **not** merge (it has no backend surface). Every "the engine merges to `dev`" phrasing
that predates this contract means **the engine *verifies* the merge** (§5) — the tech-lead performs it.

## §5 — The engine's verify (the durable-state backstop) — already built

The engine's role at step 8 is **verification, not merging** — and it never trusts an LLM worker's
exit-0 ([verify-durable-state-not-worker-exit-code]). `Scheduler.complete()` calls
`Worktree.MergedToBase` — a pure git read (`git fetch origin dev` + `git rev-list --count
origin/dev..branch == 0`) — to confirm the branch's commits are contained in `origin/dev`:

- **Merged** → the engine marks the unit done, reclaims the worktree (`git worktree remove` +
  `git branch -d` + delete the remote branch), and refills the DAG frontier.
- **Not merged** (the tech-lead exited without a landed merge) → **mark-blocked**, never counted done
  — a `.delivery/blocked/<id>.json` a ceremony reflects to the active backend. This is the data-loss guard: the
  durable git state is the source of truth, not the worker's claim.

Because the engine gates its own `Done`/refill on `MergedToBase`, a mis-set Done (set but the
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

## §7 — The per-unit pipeline orchestration (built — #8 back-half Phase 2)

The engine runs each unit as a **three-stage pipeline** in one worktree —
**developer → tester → tech-lead** (`Stage` + `deliveryPipeline` in `internal/dispatch/scheduler.go`).
`spawnUnit` creates the worktree once and starts the developer; `advanceStage` starts the next stage's
worker in the **same** worktree on each clean exit-0 (the tester + tech-lead need the developer's
commits in place); and `complete()` — the `MergedToBase` verify of §5 — runs **only after the final
(tech-lead) stage**, so a developer or tester exit-0 (neither merges) is never mistaken for a completed
unit. Each stage's prompt points at its role-agent (`DeliveryWorkerSpec` → `deliveryStagePrompt`), and
every stage's worker is wired with the project-scoped backend surface + credential env (D3 / §4), so the
tester can attach evidence and the tech-lead can merge the PR + set Done on the active backend.

Two settled decisions shape it:

- **No success-signal file (D4).** An earlier draft imagined a success artifact symmetric to
  `BlockedReport`, to drive the backend→Done transition from a skill. That is superseded by §1: the
  **tech-lead** sets the runtime-resolved Done directly on the active backend (it has the context and
  the PR), and the engine only **verifies** the merge (§5). The only durable engine artifact is still
  the `BlockedReport`.
- **Fail-at-any-stage → mark-blocked (D5).** A worker-reported blocker or a crash at any stage marks
  the unit blocked; the #12 crash-retry budget is **unit-level** — a crash re-drives the whole pipeline
  from the developer on a **fresh** worktree off dev (the crashed attempt's work is quarantined aside,
  never resumed and never deleted), so the retry is not per-stage. A stage-level rework loop is deferred.
- **Restart-resilient (no wedge).** Because a mid-pipeline unit holds a committed-but-unmerged branch
  through its tester + tech-lead stages, an engine restart leaves that branch under the canonical name
  (`Run`'s `Reconcile` preserves unmerged work in place, never deletes it). Re-admission's
  `Worktree.Create` therefore quarantines a canonical-name leftover aside before it adds the fresh
  worktree — so the re-drive can't collide on the existing branch (the sprint never wedges), and the
  prior work is preserved for diagnosis.

**Layer-A-proven, one live-validation remaining.** The pipeline is deterministically exercised by the
auth-free `work-dispatch` e2e blueprint (a fake worker keys off each stage's role token; only the
tech-lead stage merges). The remaining proof is the **real-backend #17 live run**: a genuinely spawned
`claude -p` worker inheriting the active backend's surface + credential and doing its backend ops
end-to-end — the semantic validation a mock/fake cannot give ([mock-validates-shape-not-semantics]).

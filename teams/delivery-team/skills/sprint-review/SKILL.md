---
name: sprint-review
description: /sprint-review — the delivery-team's sprint-end ceremony: compiles the Sprint Review Report (completed vs carryover, per-PBI PR links, test evidence, the deployable dev preview, actual velocity, and cross-unit integration findings) as the tech-lead + project-manager subagents in shared context, upserts it to the Sprints/Sprint-<n>-Review page in the durable-knowledge store, and runs the human product-owner's Approve/Reject gate — the ONLY trigger for the scoped dev→release promotion. Runs at each sprint end (methodology.cadence.reviewCeremony).
---

# /sprint-review — deliverable + PO dev→release gate

This is the delivery-team's **sprint-end** ceremony. It closes a sprint by compiling one durable
outcome-of-record — the **Sprint Review Report** — and then putting it in front of the human
**product-owner** for the single decision only they can make: promote this sprint's integrated work
from `dev` to `release`, or hold it. The report is assembled read-only; the promotion is gated on
an explicit PO approval and nothing else. It reads a settled `.delivery/` config (written once by
[`delivery-init`](../delivery-init/SKILL.md)) and, like every ceremony, reaches the backend only
through the active backend's adapter.

| Artifact | Direction | Where |
|---|---|---|
| Sprint's iteration items + their runtime-resolved states | read | read a sprint's items (concept #6) + resolve the completion/state model (concept #7) |
| Per-PBI PR links + test-evidence attachments | read | the work-item↔PR link (concept #11) + read evidence via the active adapter (concept #12) |
| `dev` HEAD + its green CI run + preview URL | read | read the `dev` branch state via the active adapter + pipeline/build status |
| Sprint Review Report | write (idempotent upsert) | the durable-knowledge store `Sprints/Sprint-<n>-Review` (concept #9) |
| dev→release PR (on PO Approve only) | write | open the promotion PR, per the active adapter (concept #11) |
| Rejected PBI (on PO Reject only) | write (idempotent field update + comment) | clear its iteration (concept #6), set the runtime-resolved rework state (concept #7), comment the reason (concept #3) (the #9 resolution — reuse, don't file a parallel item) |
| Blocked-unit reports (dispatch engine) | read + clear | `<projectRoot>/.delivery/blocked/*.json` |
| Blocked reflection on each report's work-item | write (idempotent tag + comment) | resolve the completion/state model (concept #7) → update the work-item (merge the `blocked` tag, concept #4; completion-state + iteration untouched) + add the diagnostic comment (concept #3) |

Field semantics for the config live in
[`config-and-methodology.md`](../../knowledge/config-and-methodology.md). The concepts this ceremony
references are defined provider-neutrally in
[the backend interface](../../knowledge/backend-interface.md); the operation→tool binding,
idempotency, content-placement, and the evidence-attach mechanism are the active backend's adapter's
job (`backends/<backend>/adapter.md`, selected once at `/delivery-init`). The roles this ceremony
adopts are the [`tech-lead`](../../agents/tech-lead/agent.md) and the
[`project-manager`](../../agents/project-manager/agent.md).

## When to run

- **At each sprint's end** — this is the methodology's review ceremony
  (`methodology.cadence.reviewCeremony === "sprint-review"`). It is **recurring**, once per sprint,
  the counterpart to the planning ceremonies (`sprint-plan` → `sprint-start`).
- The report is **read-only to compile** and its durable-knowledge page is an **idempotent upsert**,
  so a re-run before the PO has decided simply refreshes the same page — see
  [Idempotent re-run](#idempotent-re-run).

## Procedure

The ceremony runs **in-session**. It adopts its two `subagent` roles **sequentially in this shared
session context** (per `methodology.roles[].dispatch === "subagent"`): first the `project-manager`
compiles the report, then the `tech-lead` runs the integration checkpoint building on the PM's
compiled set — the second role sees the first's output in-context, which is the point of the
subagent (not isolated-worker) dispatch. The `product-owner` is the **human** (the user), consulted
only at the Approve/Reject gate. No `developer`/`tester` worker is spawned here (that is
`atl work dispatch`'s job, only from `/sprint-start`).

### 1. Load config and resolve the closed sprint's runtime facts

Read `.delivery/config.json` and `.delivery/methodology.json` (read-only — only `/delivery-init`
writes them). Take the backend's coordinates (Azure `org`/`project`/`repo`; GitHub
`owner`/`repo`/`projectNumber` — see [`config-and-methodology.md`](../../knowledge/config-and-methodology.md)
§2), the durable-knowledge store locator the active adapter needs (Azure: `wikiId`, resolved +
cached at init — never re-resolve it; GitHub: none — the store is the in-repo `/docs` tree), and
**`config.branchPair`** as the authoritative dev/release branch names (config wins over
`methodology.branches`).

Resolve the concrete sprint and its states at runtime — **never hardcode a state literal**
(concept #7):

- Resolve the closed iteration (its name/path) via a sprint/iteration read (concept #6); `<n>` for
  the report path is this sprint's number, resolved here.
- Resolve the type's state→category map (concept #7) so "Completed" means the **runtime-resolved
  Completed-category** state, not the literal `"Done"`.

### 2. Reflect blocked units to the backend and clear their reports

Before compiling the report, drain the dispatch engine's **blocked reports**. When the recovery
ladder gives up on a work-unit, `atl work dispatch` writes a durable `BlockedReport` to
`<projectRoot>/.delivery/blocked/<id>.json` — the engine has **no backend surface** (the CLI/Skill
boundary), so reflecting a blocked unit onto its work-item is this ceremony's job. Draining these
reports here is what turns a silently-stalled unit into a board-visible one; skip it and a crashed
or stalled unit accumulates on disk, invisible to the PO.

- List `<projectRoot>/.delivery/blocked/*.json`. **None → skip this step** (note "no blocked
  reports") — the common case.
- Read and parse each `BlockedReport` (fields: `id`, `branch`, `worktreePath`, `reason`, `phase`,
  `lastSummary`, `stderrTail`, `preserved`, `blockedAt`).
- Per report `id`, **reflect the block onto the work-item** — the settled "mark blocked" contract
  (the [backend interface](../../knowledge/backend-interface.md)'s state-resolution policy), which is
  **NOT** a state transition: resolve the completion/state model (concept #7), then update the
  work-item to **merge** `blocked` into the item's tags (concept #4; never clobber existing tags).
  Leave the completion-state **and** the iteration **unchanged** here — the item must stay in the
  closed iteration so the report (step 3) still reads it as carryover; its return to the backlog is
  the standard carryover handling
  ([reject-and-carryover.md](../../agents/project-manager/children/reject-and-carryover.md)), not
  this step's job.
- **Record the diagnostic as a comment** (concept #3) whose first line is the supervisor sentinel
  `**[Blocked — supervisor report]**` — deliberately **distinct** from a worker self-block comment
  so the two never collide. The body carries `reason` / `phase` / `branch` / `worktreePath` /
  `lastSummary` / `stderrTail` / `blockedAt`, so whoever picks the unit up next has the full
  stall/crash context. Idempotency is the sentinel pattern (concepts #2/#3): before adding, list the
  work-item's comments filtered to that sentinel — found → update in place, not-found → add; a re-run
  never duplicates.
- **Only after the backend reflection succeeds, clear the local report** — delete
  `<projectRoot>/.delivery/blocked/<id>.json`. The durable record is now the work-item comment (plus
  the preserved git branch); the local file was only the cross-boundary carrier. A failed reflection
  leaves the report in place, so the next `/sprint-review` retries it (the sentinel makes a retry a
  safe no-op where it already landed).
- Hand the reflected ids + their reasons to the compile step (step 3) so each appears in the report's
  `## Carryover` section flagged **blocked** with its diagnostic — the visible audit trail the PO
  reads.

### 3. Compile the Sprint Review Report — acting as the `project-manager`

Acting as the `project-manager` (read
[`../../agents/project-manager/agent.md`](../../agents/project-manager/agent.md) + its `children/`,
especially [`sprint-review-report.md`](../../agents/project-manager/children/sprint-review-report.md)),
gather the sprint's data **read-only** and build the six-section report. Read the sprint's items
(concept #6, batched; "list means all" — if the set could exceed the tool's return, close the gap
with a high-limit idempotency/velocity query (concept #10) and treat a result *at* the cap as a
truncation error, never a complete read). The six sections:

1. **Completed vs carryover** — partition the sprint's PBIs by the **runtime-resolved
   Completed-category** state (concept #7, from step 1), each with id / title / story-points; every
   admitted item that did NOT complete is flagged as carryover for re-planning (never silently
   dropped — [`reject-and-carryover.md`](../../agents/project-manager/children/reject-and-carryover.md)).
2. **Per-PBI PR links** — for each unit, the PR merged into `dev` this sprint, read from the
   work-item↔PR link written at the micro-loop's PR step (concept #11), read back by reading the
   work-item / its comments; the active adapter's PR surface resolves PR title/status if needed —
   located by the link, never by "the newest comment" (concepts #2/#3).
3. **Test evidence** — per PBI: CI status, web results, and mobile-emulator pass/fail with
   **screenshot attachment URLs read back via the active adapter** (concept #12 — the read leg;
   upload was the tester's job, this ceremony reads, it does not re-test).
4. **Deployable dev preview** — the current `dev` HEAD (read the `dev` branch state via the active
   adapter, on `config.branchPair.dev`) + its green CI/build run + the running preview URL where the
   stack-pack defines one. The PO reviews the integrated **running result**, not a diff list.
5. **Actual velocity** — the story points completed this sprint (the Completed sum from section 1);
   this is read-only arithmetic and feeds the next `/sprint-plan`'s velocity window.
6. **Integration findings** — the cross-unit open findings from the tech-lead's checkpoint (step 4)
   plus the forward-fix tasks filed there.

### 4. Run the cross-unit integration checkpoint — acting as the `tech-lead`

Then, **as the `tech-lead`** (read
[`../../agents/tech-lead/agent.md`](../../agents/tech-lead/agent.md) + its `children/`, especially
[`integration-checkpoint.md`](../../agents/tech-lead/children/integration-checkpoint.md)), building
on the PM's compiled set **in this same context**, run the whole-sprint coherence pass over the
units merged to `dev` this sprint (concept #6, batched — "list means all"): do the seams between
dependent/same-area units line up as built, do the areas still compose, does the aggregate honor the
`Architecture/` boundaries + `Conventions/`, and are the Feature's Acceptance Criteria collectively
delivered?

- **File a forward-fix Task** for each real integration break, **idempotently** (concept #10):
  compute `atl-key = hash(parent-id + plan-ordinal)` with a fresh plan-ordinal in the parent's plan,
  run the **check-first query** (concept #10) for that `atl-key` — found → reuse + update, not-found →
  create the work-item (concept #1) then stamp its tags (concept #4) with
  `atl-run:sprint-review:<sprint-id>` + `atl-key:<hash>`; a 409/duplicate is resolved to the existing
  item, never surfaced. Area-tag each (`area:<name>`, concept #4) and add any dependency links
  (concept #8); resolve every state at runtime (concept #7).
- **Promote worker-surfaced project facts** to the tech-lead's own durable-knowledge namespaces —
  `Architecture/` / `Architecture/ADR/ADR-<n>-<slug>` / `Conventions/` — by idempotent upsert into
  the durable-knowledge store (concept #9). Workers never write the store; the tech-lead promotes.
- Feed the checkpoint's open findings + the filed forward-fix task ids back into the report's
  **Integration findings** section (step 3, section 6).

### 5. Write the report to the durable-knowledge store and surface it in-session

Write the assembled report to exactly `Sprints/Sprint-<n>-Review` (`<n>` from step 1) as an
**idempotent upsert** into the durable-knowledge store (concept #9) — the `project-manager`'s
`Sprints/` namespace (one owner). Confirm the `Sprints/` namespace exists on the first write of the
project (a durable-knowledge store listing, concept #9); read the store's locator from `config.json`
(Azure: `wikiId`, never re-resolved; GitHub: the in-repo `/docs` path — no locator). Also surface the full report **in-session** so the PO reads it
here before the gate.

### 6. Run the PO Approve/Reject gate — the `product-owner` (human) decides

Ask the **product-owner** (the user) an explicit Approve/Reject question on this sprint's integrated
`dev` state. This is the **only** trigger for the dev→release merge — the scoped carve-out of the
platform's NEVER-merge rule, legitimate **because** the PO explicitly approves it. Do not proceed on
inference; wait for the explicit decision.

**On APPROVE — fire the gated dev→release promotion:**

- Open the promotion PR from `config.branchPair.dev` into `config.branchPair.release` (concept #11 —
  the actual branch names come from config; config wins over `methodology.branches`).
- **Merge the PO-approved promotion PR per the active backend's adapter (concept #11).** The adapter
  binds the concrete merge — on the Azure backend the PR is *completed* (set to auto-complete where
  the surface exposes that field, so the backend completes it once its own policy checks pass); on
  the GitHub backend it is `gh pr merge --merge`. Where the backend cannot merge non-interactively,
  hand the PO the created PR link to complete the merge in the backend. This ceremony **never
  fabricates a merge mechanism and never merges outside the PO-approved PR.**
- Then mark the iteration reviewed (a runtime-resolved state update, concept #7 — never a hardcoded
  literal), and record the approval on the review page (idempotent upsert, step 5).

**On REJECT — the release STAYS PUT (forward-fix, never a revert):**

Follow the `project-manager`'s
[`reject-and-carryover.md`](../../agents/project-manager/children/reject-and-carryover.md) (the **#9
resolution**) — reject reuses the **EXISTING** item; it does **not** file a parallel Bug/Task (a
second scheduling path would be complexity for no gain — one admission algorithm handles new,
carried-over, and rejected work identically):

- For each rejected PBI, **clear its iteration** (concept #6 — an idempotent field update), which
  returns it to the backlog, and **set its state to the runtime-resolved rework category** (concept
  #7 — never a literal like `New`/`Active`/`Reopened`).
- **Record the rejection reason as a comment on that item** (concept #3), so the next developer who
  picks it up knows the acceptance gap that brought it back.
- The next `/sprint-plan` re-picks the item (now with no iteration) as an ordinary backlog candidate
  — **no special "rejected" queue**.
- Also record the rejection reason on the review page (idempotent upsert, step 5).
- Do **not** open or complete any dev→release PR — `release` is untouched.

## Idempotent re-run

A re-run converges, never duplicates (concept #10 — backend tags/labels are the source of truth, no
local ledger):

- **The blocked-report drain (step 2) is idempotent** — the backend reflection merges the `blocked`
  tag (never replaces) and dedups its comment on the `**[Blocked — supervisor report]**` sentinel,
  and the local `<id>.json` is deleted only after the reflection lands; a re-run either re-reflects a
  still-present report harmlessly or finds nothing left to drain.
- **Report generation is read-only** — re-reading the sprint's items, PR links, evidence, and `dev`
  state has no side effect.
- **The review page is an idempotent upsert** — upserting the durable-knowledge store overwrites
  `Sprints/Sprint-<n>-Review` in place rather than appending a duplicate.
- **Any created item** — an integration forward-fix task (step 4) or a reject follow-up (step 6) —
  carries tags (concept #4) of `atl-run:sprint-review:<sprint-id>` (provenance) + `atl-key:<hash>`
  where `hash = hash(parent-id + plan-ordinal)` (a **stable plan-ordinal**, never a per-run GUID,
  never `hash(title)`). Before any create, a **check-first query** (concept #10) for that `atl-key`
  reuses+updates a found item and only creates when not-found; a 409/duplicate is resolved to the
  existing item, never surfaced.
- **The iteration-reviewed transition** is an idempotent work-item field update (concept #10) —
  re-setting the same runtime-resolved state is a safe no-op.
- **The dev→release promotion** is not re-fired on a re-run: before opening a promotion PR, check the
  active adapter's PR surface (concept #11) for a PR already open/completed for this sprint's
  `branchPair.dev`→`branchPair.release` and reuse it, so a re-run after approval does not open a
  second promotion PR.

All backend access is through the active backend's adapter; the credential is referenced by name
(`config.pat.ref` on Azure, `config.credential.ref` on GitHub) and never read or written as a literal.

---
name: sprint-review
description: /sprint-review — the delivery-team's sprint-end ceremony: compiles the Sprint Review Report (completed vs carryover, per-PBI PR links, test evidence, the deployable dev preview, actual velocity, and cross-unit integration findings) as the tech-lead + project-manager subagents in shared context, upserts it to the Sprints/Sprint-<n>-Review project-wiki page, and runs the human product-owner's Approve/Reject gate — the ONLY trigger for the scoped dev→release promotion. Runs at each sprint end (methodology.cadence.reviewCeremony).
---

# /sprint-review — deliverable + PO dev→release gate

This is the delivery-team's **sprint-end** ceremony. It closes a sprint by compiling one durable
outcome-of-record — the **Sprint Review Report** — and then putting it in front of the human
**product-owner** for the single decision only they can make: promote this sprint's integrated work
from `dev` to `release`, or hold it. The report is assembled read-only; the promotion is gated on
an explicit PO approval and nothing else. It reads a settled `.delivery/` config (written once by
[`delivery-init`](../delivery-init/SKILL.md)) and, like every ceremony, reaches Azure only through
the `azureDevOps` MCP.

| Artifact | Direction | Where |
|---|---|---|
| Sprint's iteration items + their runtime-resolved states | read | `wit_get_work_items_for_iteration`, `wit_get_work_item_type` |
| Per-PBI PR links + test-evidence attachments | read | work-item↔PR links + `wit_get_work_item_attachment` |
| `dev` HEAD + its green CI run + preview URL | read | `repo_get_branch_by_name` + pipeline/build status |
| Sprint Review Report | write (idempotent upsert) | project wiki `Sprints/Sprint-<n>-Review` |
| dev→release PR (on PO Approve only) | write | `repo_create_pull_request` (+ `repo_update_pull_request`) |
| Rejected PBI (on PO Reject only) | write (idempotent field update + comment) | clear its `IterationPath`, set the runtime-resolved rework state, comment the reason (the #9 resolution — reuse, don't file a parallel item) |

Field semantics for the config live in
[`config-and-methodology.md`](../../knowledge/config-and-methodology.md); the Azure operation→tool
map, idempotency, content-placement, and the one REST carve-out live in
[`azure-adapter.md`](../../knowledge/azure-adapter.md). The roles this ceremony adopts are the
[`tech-lead`](../../agents/tech-lead/agent.md) and the
[`project-manager`](../../agents/project-manager/agent.md).

## When to run

- **At each sprint's end** — this is the methodology's review ceremony
  (`methodology.cadence.reviewCeremony === "sprint-review"`). It is **recurring**, once per sprint,
  the counterpart to the planning ceremonies (`sprint-plan` → `sprint-start`).
- The report is **read-only to compile** and its wiki page is an **idempotent upsert**, so a
  re-run before the PO has decided simply refreshes the same page — see
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
writes them). Take `org`/`project`/`repo`, the `wikiId` (already resolved + cached at init — never
re-resolve it), and **`config.branchPair`** as the authoritative dev/release branch names (config
wins over `methodology.branches`).

Resolve the concrete sprint and its states at runtime — **never hardcode a state literal** (§6):

- Resolve the closed iteration (its name/path) via `work_list_iterations`; `<n>` for the report
  path is this sprint's number, resolved here.
- Resolve the type's state→category map with `wit_get_work_item_type` so "Completed" means the
  **runtime-resolved Completed-category** state, not the literal `"Done"`.

### 2. Compile the Sprint Review Report — acting as the `project-manager`

Acting as the `project-manager` (read
[`../../agents/project-manager/agent.md`](../../agents/project-manager/agent.md) + its `children/`,
especially [`sprint-review-report.md`](../../agents/project-manager/children/sprint-review-report.md)),
gather the sprint's data **read-only** and build the six-section report. Read the sprint's items
with `wit_get_work_items_for_iteration` (batched; "list means all" per §4 — if the set could exceed
the tool's return, close the gap with a high-`top` `wit_query_by_wiql` and treat a result *at* the
cap as a truncation error, never a complete read). The six sections:

1. **Completed vs carryover** — partition the sprint's PBIs by the **runtime-resolved
   Completed-category** state (§6, from step 1), each with id / title / story-points; every admitted
   item that did NOT complete is flagged as carryover for re-planning (never silently dropped —
   [`reject-and-carryover.md`](../../agents/project-manager/children/reject-and-carryover.md)).
2. **Per-PBI PR links** — for each unit, the PR merged into `dev` this sprint, read from the
   work-item↔PR artifact links written at the micro-loop's PR step
   (`wit_link_work_item_to_pull_request`, read back via `wit_get_work_item` /
   `wit_list_work_item_comments`; the adapter's `repo_*` PR tools resolve PR title/status if
   needed) — located by the link, never by "the newest comment" (§7).
3. **Test evidence** — per PBI: CI status, web results, and mobile-emulator pass/fail with
   **screenshot attachment URLs read back via `wit_get_work_item_attachment`** (the MCP read leg;
   upload was the tester's REST job via `scripts/az-attach.sh`, §9 — this ceremony reads, it does
   not re-test).
4. **Deployable dev preview** — the current `dev` HEAD (`repo_get_branch_by_name` on
   `config.branchPair.dev`) + its green CI/build run + the running preview URL where the stack-pack
   defines one. The PO reviews the integrated **running result**, not a diff list.
5. **Actual velocity** — the story points completed this sprint (the Completed sum from §1); this
   is read-only arithmetic and feeds the next `/sprint-plan`'s velocity window.
6. **Integration findings** — the cross-unit open findings from the tech-lead's checkpoint (step 3)
   plus the forward-fix tasks filed there.

### 3. Run the cross-unit integration checkpoint — acting as the `tech-lead`

Then, **as the `tech-lead`** (read
[`../../agents/tech-lead/agent.md`](../../agents/tech-lead/agent.md) + its `children/`, especially
[`integration-checkpoint.md`](../../agents/tech-lead/children/integration-checkpoint.md)), building
on the PM's compiled set **in this same context**, run the whole-sprint coherence pass over the
units merged to `dev` this sprint (`wit_get_work_items_for_iteration`, batched — "list means all",
§4): do the seams between dependent/same-area units line up as built, do the areas still compose,
does the aggregate honor the `Architecture/` boundaries + `Conventions/`, and are the Feature's
Acceptance Criteria collectively delivered?

- **File a forward-fix Task** for each real integration break, **idempotently** (§5): compute
  `atl-key = hash(parent-id + plan-ordinal)` with a fresh plan-ordinal in the parent's plan, run the
  **check-first WIQL** (`wit_query_by_wiql`) for that `atl-key` — found → reuse + update, not-found →
  `wit_create_work_item` then stamp `System.Tags` with `atl-run:sprint-review:<sprint-id>` +
  `atl-key:<hash>`; a 409/duplicate is resolved to the existing item, never surfaced. Area-tag each
  (`area:<name>`, §7) and add any `Dependency` links (`wit_work_items_link`); resolve every state at
  runtime (`wit_get_work_item_type`, §6).
- **Promote worker-surfaced project facts** to the tech-lead's own wiki namespaces —
  `Architecture/` / `Architecture/ADR/ADR-<n>-<slug>` / `Conventions/` — by idempotent upsert
  (`wiki_create_or_update_page`, §8). Workers never write the wiki; the tech-lead promotes.
- Feed the checkpoint's open findings + the filed forward-fix task ids back into the report's
  **Integration findings** section (step 2, §6).

### 4. Write the report to the wiki and surface it in-session

Write the assembled report to exactly `Sprints/Sprint-<n>-Review` (`<n>` from step 1) with
`wiki_create_or_update_page` — an **idempotent upsert** into the `project-manager`'s `Sprints/`
namespace (§8, one owner). Confirm the `Sprints/` namespace exists on the first write of the project
with `wiki_list_pages`; read `wikiId` from `config.json` (never re-resolved). Also surface the full
report **in-session** so the PO reads it here before the gate.

### 5. Run the PO Approve/Reject gate — the `product-owner` (human) decides

Ask the **product-owner** (the user) an explicit Approve/Reject question on this sprint's integrated
`dev` state. This is the **only** trigger for the dev→release merge — the scoped carve-out of the
platform's NEVER-merge rule, legitimate **because** the PO explicitly approves it. Do not proceed on
inference; wait for the explicit decision.

**On APPROVE — fire the gated dev→release promotion:**

- Open the promotion PR from `config.branchPair.dev` into `config.branchPair.release` with
  `repo_create_pull_request` (the actual branch names come from config, §2 — config wins over
  `methodology.branches`).
- **Completing/merging a PR is NOT in the MCP surface.** Per the adapter's *Operation → MCP tool
  map* ([azure-adapter.md](../../knowledge/azure-adapter.md) §2), the PR row exposes only
  `repo_create_pull_request` / `repo_update_pull_request` / `repo_vote_pull_request` /
  `repo_create_pull_request_thread` / `repo_list_pull_request_threads` / `repo_reply_to_comment` —
  there is no `complete`/`merge` tool. The honest mechanism: set the PR to auto-complete via
  `repo_update_pull_request` where the surface exposes that field so Azure DevOps completes it once
  its own policy checks pass; where auto-complete is not exposed, hand the PO the created PR link to
  **complete the merge in Azure DevOps** — this ceremony never fabricates a merge tool and never
  merges outside the PO-approved PR.
- Then mark the iteration reviewed (a runtime-resolved state update via `wit_update_work_item`, §6
  — never a hardcoded literal), and record the approval on the review page (idempotent upsert,
  step 4).

**On REJECT — the release STAYS PUT (forward-fix, never a revert):**

Follow the `project-manager`'s
[`reject-and-carryover.md`](../../agents/project-manager/children/reject-and-carryover.md) (the **#9
resolution**) — reject reuses the **EXISTING** item; it does **not** file a parallel Bug/Task (a
second scheduling path would be complexity for no gain — one admission algorithm handles new,
carried-over, and rejected work identically):

- For each rejected PBI, **clear its `IterationPath`** (`wit_update_work_item` — an idempotent field
  update), which returns it to the backlog, and **set its state to the runtime-resolved rework
  category** (`wit_get_work_item_type` → `wit_update_work_item`, §6 — never a literal like
  `New`/`Active`/`Reopened`).
- **Record the rejection reason as a comment on that item** (`wit_add_work_item_comment`), so the
  next developer who picks it up knows the acceptance gap that brought it back.
- The next `/sprint-plan` re-picks the no-`IterationPath` item as an ordinary backlog candidate —
  **no special "rejected" queue**.
- Also record the rejection reason on the review page (idempotent upsert, step 4).
- Do **not** open or complete any dev→release PR — `release` is untouched.

## Idempotent re-run

A re-run converges, never duplicates (§5 — Azure tags are the source of truth, no local ledger):

- **Report generation is read-only** — re-reading the sprint's items, PR links, evidence, and `dev`
  state has no side effect.
- **The review page is an idempotent upsert** — `wiki_create_or_update_page` overwrites
  `Sprints/Sprint-<n>-Review` in place rather than appending a duplicate.
- **Any created item** — an integration forward-fix task (step 3) or a reject follow-up (step 5) —
  carries `System.Tags` of `atl-run:sprint-review:<sprint-id>` (provenance) + `atl-key:<hash>` where
  `hash = hash(parent-id + plan-ordinal)` (a **stable plan-ordinal**, never a per-run GUID, never
  `hash(title)`). Before any create, a **check-first WIQL** (`wit_query_by_wiql`) for that `atl-key`
  reuses+updates a found item and only creates when not-found; a 409/duplicate is resolved to the
  existing item, never surfaced.
- **The iteration-reviewed transition** is an idempotent `wit_update_work_item` field update (§5) —
  re-setting the same runtime-resolved state is a safe no-op.
- **The dev→release promotion** is not re-fired on a re-run: before opening a promotion PR, check
  the adapter's PR surface (§2) for a PR already open/completed for this sprint's
  `branchPair.dev`→`branchPair.release` and reuse it, so a re-run after approval does not open a
  second promotion PR.

All Azure access is through the `azureDevOps` MCP; the PAT is referenced by name (`config.pat.ref`)
and never read or written as a literal.

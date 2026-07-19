# Azure adapter — the delivery-team's work-item-store operation-contract

This is the **single documented contract** every delivery-team role-agent, ceremony
skill, and `developer`/`tester` worker follows for **every** Azure DevOps operation.
There is one adapter, one auth path, one transport policy — so resilience,
idempotency, and content placement are written **once here** and inherited by every
caller. Do not improvise an Azure call that isn't described here.

## 1. The MCP-first principle (why there is almost no adapter *code*)

- **Azure is reached through the `azureDevOps` MCP** — the `mcp__azureDevOps__*`
  tool surface (`wit_*` work-items · `work_*` iterations/capacity · `repo_*` PRs ·
  `wiki_*` pages · `search_*` · `core_*`). Ceremonies use it in-session; a spawned
  `claude -p` **worker inherits the same MCP** via the `--mcp-config` passed by
  `atl work dispatch` (keystone #17). So the "adapter" is this **documented
  contract**, not a code layer — a worker gets the full Azure surface with zero new
  code. A raw-REST worker would fork the adapter into two implementations; we don't.
- **The Go orchestrator (`atl work dispatch`) never calls Azure.** It is a
  deterministic, zero-LLM, zero-Azure supervisor: it spawns workers (with the PAT +
  MCP config in their environment) and observes them through `status.json`. Every
  Azure call originates from an **LLM** caller (a ceremony or a worker) using the MCP.
- **Auth = a PAT from the environment, never a file, never the argv.** The
  `azureDevOps` MCP authenticates with a Personal Access Token referenced by name
  from the environment (the official `@azure-devops/mcp … --authentication pat`
  shape). `atl work dispatch` spawns each worker with that PAT env var already set
  (via `WorkerSpec.ExtraEnv`) + the MCP config, so the worker never sees a literal
  secret and never logs one — the same pattern as the e2e harness's tokens. A PAT is
  a portable secret: it moves from the maintainer's keychain to a worker's env (and,
  for Layer-B, into a container) without any interactive Azure session.

## 2. Operation → MCP tool map

The operations the team performs and the tool each one uses. **Never invent a tool
name** — if an operation isn't here, it goes through the check-first / resilience
rules below or is escalated, not guessed.

| Operation | MCP tool |
|---|---|
| Create a work-item (Epic/Feature/PBI/Task/Bug) | `wit_create_work_item` |
| Create child items under a parent | `wit_add_child_work_items` |
| Read one / a batch of work-items | `wit_get_work_item` / `wit_get_work_items_batch_by_ids` |
| Read a sprint's items | `wit_get_work_items_for_iteration` |
| Update fields (state, IterationPath, tags, StoryPoints…) | `wit_update_work_item` / `wit_update_work_items_batch` |
| **Resolve a type's states/fields at runtime** | `wit_get_work_item_type` (see §6 — never hardcode) |
| Query by WIQL (idempotency check, velocity, selection) | `wit_query_by_wiql` |
| Add / read the analysis comment | `wit_add_work_item_comment` / `wit_list_work_item_comments` |
| Link a work-item ↔ a PR | `wit_link_work_item_to_pull_request` |
| Link/unlink work-items (Dependency, Parent…) | `wit_work_items_link` / `wit_work_item_unlink` |
| List backlogs / backlog items | `wit_list_backlogs` / `wit_list_backlog_work_items` |
| List / create / assign iterations | `work_list_iterations` / `work_create_iterations` / `work_assign_iterations` |
| Read / write team capacity | `work_get_team_capacity` / `work_update_team_capacity` |
| Create / update / vote / thread a PR | `repo_create_pull_request` / `repo_update_pull_request` / `repo_vote_pull_request` / `repo_create_pull_request_thread` / `repo_list_pull_request_threads` / `repo_reply_to_comment` |
| List a project's repos (discovery, e.g. `/delivery-init`) | `repo_list_repos_by_project` |
| Read repo/branch/file | `repo_get_repo_by_name_or_id` / `repo_get_branch_by_name` / `repo_get_file_content` |
| Read / upsert / list wiki pages | `wiki_get_page_content` / `wiki_create_or_update_page` / `wiki_list_pages` / `wiki_get_wiki` / `wiki_list_wikis` |
| Discovery search | `search_workitem` / `search_wiki` / `search_code` |
| Resolve project/team/identity | `core_list_projects` / `core_list_project_teams` / `core_get_identity_ids` |
| **Attachment UPLOAD** (evidence: screenshots/results) | **no MCP tool → REST carve-out (§9)** |
| Attachment READ | `wit_get_work_item_attachment` |

## 3. Resilience — rate-limit / backoff

Every write, and every read under N-parallel load, wraps its MCP call in
**exponential backoff + jitter**, honours the `Retry-After` / rate-limit-delay
header, and caps at ~5 attempts (~2 min). Under `atl work dispatch`'s ~4–6 parallel
workers, 429s at milestone writes are *expected*, not exceptional.

- **First-line defense is writing Azure only at durable milestones** (keystone #2) —
  the worker heartbeats to `status.json`, not to Azure; it touches Azure at claim,
  at the analysis/PR comment, and at close. Backoff is the second line.
- **Batch reads** to collapse N calls into one: `wit_get_work_items_batch_by_ids`,
  `wit_get_work_items_for_iteration`. Prefer one batched read over a loop of singles.
- A 429/5xx that exhausts retries is **not a task failure**: pause the *call*,
  heartbeat `azure-degraded`, and let the supervisor hold the durable-milestone write
  and retry. An Azure hiccup must never fail coding work (failure matrix #5).

## 4. Pagination — "list means all" (per-tool mechanism)

The principle is absolute: **never silently truncate** a list/query — a half-read
backlog corrupts velocity math and sprint-plan selection. But the MCP tools expose
different (or no) paging controls, so the *realizable* mechanism is per-tool. (This
refines detail-spec #16's generic "continuation token / `$top`+`$skip`" wording to
what the surface actually offers — several of those tools have no `skip`.)

- **`wiki_list_pages`** — has `continuationToken` + `top`: **loop to exhaustion**,
  following the continuation token until it comes back empty.
- **`wit_query_by_wiql`** — exposes only `top` (no `skip`, no continuation). Set
  `top` high enough to cover the expected set (WIQL caps at ~20k ids) and **treat a
  result AT the cap as a truncation error to surface**, never as a complete read —
  there is no way to page past it.
- **`wit_get_work_items_for_iteration` / `wit_list_backlog_work_items`** — expose no
  caller-side paging; they return the tool's set. If a sprint/backlog could exceed
  it, close the gap with an explicit high-`top` `wit_query_by_wiql` and apply the
  cap-is-truncation rule above.

If a result could be paged or capped and you treat a partial set as complete, that is
a bug.

## 5. Idempotency — the load-bearing part (protects resumability)

A re-run ceremony (after a crash, a partial run, or an explicit re-plan) must **never
duplicate work-items or double-assign iterations**. Mechanism = **stamp +
check-before-create**, with **Azure-side tags as the source of truth** (no local
ledger — a lost/stale ledger silently reintroduces duplication).

- **Every created item carries a deterministic key**, written to `System.Tags`:
  - `atl-run:<ceremony>:<sprint-id>` — provenance.
  - `atl-key:<hash>` where `hash = hash(parent-id + plan-ordinal)`. The tech-lead's
    decomposition plan is recorded durably (plan manifest on the parent / wiki) and
    each intended item gets a **stable plan-ordinal**; the stamp is
    `parent-id + plan-ordinal`. Duplicate titles don't collide (distinct ordinals);
    a title edit doesn't break convergence (ordinal stable). Tags work on **any**
    process template — no custom field, no org-admin setup.
- **Before any create, run a check-first WIQL** for that `atl-key`: **found → reuse +
  update** (converge to intended state), **not-found → create-then-stamp**. Create +
  stamp as close to atomic as the API allows; a **409/duplicate on create is caught
  and resolved to the existing item**, not surfaced as an error.
- **Brainstorm-sourced items carry `atl-brainstorm:<slug>`, not `atl-key`.** `/brainstorm done`'s
  board-sync creates deferred backlog items with `atl-brainstorm:<brainstorm-slug>` in `System.Tags`
  (its own provenance key — a brainstorm item has no parent/plan-ordinal, so no `atl-key`), and dedups
  its own re-runs by a check-first WIQL on that tag plus the item title. When a decomposition ceremony
  (`/refine`) later plans a unit that IS such an item, its `atl-key` check-first misses it; before
  creating, it runs a second check-first WIQL on `atl-brainstorm:<slug>` for the in-scope item and, on
  a title match, **adopts** it — `wit_update_work_item` in place + stamp the computed `atl-key:<hash>`
  — instead of creating a duplicate. After adoption the item converges via the normal `atl-key` query.
- **`wit_add_child_work_items` takes no tags/fields** — only `wit_create_work_item`
  accepts `System.Tags` inline. A child created with `wit_add_child_work_items` is
  stamped (`atl-key`/`atl-run`/`area:<name>`) by a **follow-up `wit_update_work_item`**;
  that is the create-then-stamp above, two calls, as atomic as the API allows.
- **Iteration assignment is idempotent by nature** — it is an `IterationPath` field
  *update* (`wit_update_work_item`), so re-running sets the same path to the same
  value: a safe no-op. Never model it as a "create membership" that could double.
- **Velocity is read-only** (query prior Done items, sum StoryPoints) → inherently
  idempotent, no guard needed.

Because keys are derived from stable `parent + ordinal` (**not** a per-run
GUID/timestamp), the same logical artifact maps to the same key across re-runs — that
is what makes resume *convergent*, not merely dedup-attempted.

## 6. Runtime type & state resolution — never hardcode

State names, the type's field set, and the state→category mapping are
**process-template-dependent** (Agile vs Scrum vs CMMI vs custom). **Resolve them at
runtime** with `wit_get_work_item_type` (and the project's process metadata); never
write a literal `"Done"`/`"Active"` into a ceremony or worker prompt.

- **"Mark blocked" is NOT a state transition.** Azure DevOps has **no blocked
  state-category** (the categories are Proposed/InProgress/Resolved/Completed/Removed)
  and the standard templates (Scrum/Agile/CMMI) ship **no `Blocked` state**. Signal a
  blocker by adding a `blocked` tag to `System.Tags` (universal, template-agnostic) —
  and, where the type exposes it, setting the `Microsoft.VSTS.CMMI.Blocked` field to
  `Yes` (Task has it; PBI/Feature/Epic don't) — plus a diagnostic comment, leaving
  `System.State` **unchanged**. Never resolve or invent a `Blocked` state to transition
  to; the "resolve at runtime" rule applies to real states (the Completed category), not
  to blocking, which is a flag on the item.
- "Done" for velocity / completion = resolve the **Completed** state-category, not a
  literal string — different templates spell it differently.

## 7. Content-placement contract (deterministic read-back)

So the tech-lead (and every consumer) reads analysis back **by location, never by
guessing**:

- **Business analysis** (`business-analyst`) → the Epic/Feature **`System.Description`**
  as Markdown under fixed H2s: `## Problem`, `## Business Value`, `## Scope`,
  `## Acceptance Criteria`, `## Out of Scope`. Description is the durable, always-loaded
  "what & why".
- **Technical analysis** (`technical-analyst`) → a **single labeled comment**
  (`wit_add_work_item_comment`) whose first line is the exact sentinel
  `**[Technical Analysis]**`, then fixed H2s: `## Approach`, `## Feasibility & Risks`,
  `## NFRs`, `## Dependencies`, `## Suggested Areas`. A comment (not a second
  Description) keeps the board's Description business-owned; the sentinel makes the
  analysis machine-locatable.
- **Area tags** → `System.Tags` in the `area:<name>` convention, applied by the
  **tech-lead** at decomposition (the analyst only *suggests* under `## Suggested
  Areas`) — because area→pack binding is a tech-lead responsibility.
- **Canonical brief** (`tech-lead`, at `/refine`) → a **single labeled comment** on
  the work-unit (`wit_add_work_item_comment`) whose first line is the exact sentinel
  `**[Canonical Brief]**`, then fixed H2s: `## Goal`, `## Area`, `## Load These Pages`,
  `## Depends On`, `## Evidence Before Review`. It is the per-unit bridge a `developer`
  (and the `tester`) reads — it names the area pack and embeds the exact `Architecture/`
  / `Conventions/` wiki paths for the task. Same placement discipline as the
  technical-analysis comment: one comment, sentinel line one, machine-locatable — it is
  a **work-item comment, never a wiki page**.
- **Read-back** (at `/refine`, and by every spawned worker): `wit_get_work_item` (parse
  Description headings) + `wit_list_work_item_comments` filtered to the comment starting
  with its sentinel — `**[Technical Analysis]**` for the analysis, `**[Canonical Brief]**`
  for the brief — a **sentinel match, not "the newest comment"**, so a later human comment
  never shadows either. For a decomposed child unit with no `**[Technical Analysis]**` of its
  own (only the tech-lead's `**[Canonical Brief]**`), read the analysis from its **nearest
  ancestor** Feature — traverse the parent via `wit_get_work_item` relations (the
  `System.LinkTypes.Hierarchy-Reverse` parent link), climbing parent links until you reach the
  ancestor that bears a `**[Technical Analysis]**` (concept #1). (The brief's `atl-key` stays
  the idempotency key — a re-run
  sentinel-matches and updates in place; the sentinel is the *locator*, the key is the
  *convergence* guard.)

## 8. Project wiki — namespaced knowledge (`wiki_*`)

Work-items are transient execution state; the **project wiki is durable
current-truth** (the ATL wiki/journal split, in Azure). One owner per namespace, so
no two roles fight over a page:

| Namespace (page path) | Content | Owner |
|---|---|---|
| `Domain/` | glossary, entities, business rules | business-analyst |
| `Analysis/` | per-Epic/Feature deep analysis (deeper than the work-item, §7) | business-analyst + technical-analyst |
| `Architecture/` | system shape, module boundaries, stack decisions | tech-lead |
| `Architecture/ADR/ADR-<n>-<slug>` | one page per architecture decision | tech-lead |
| `Conventions/` | project conventions atop the pack's generic ones | tech-lead |
| `Sprints/Sprint-<n>-Review` | the sprint-review outcome pages | `/sprint-review` |

- **Developer/tester workers do NOT write the wiki** — their durable *role-craft*
  learnings route to their agent `children/` (project-agnostic) via `/drain`;
  project-specific facts they surface are promoted to the wiki by the **tech-lead** at
  `/refine` / integration review. This keeps write-authority clean and avoids
  N-worker write races.
- **Read contract:** a developer worker's context = stack-pack + project-wiki + task +
  the item's `**[Technical Analysis]**` (its own, or its nearest ancestor Feature's for a
  decomposed unit) + the tech-lead's canonical brief. The **brief embeds the relevant wiki page paths**
  (`Architecture/` + `Conventions/` for the task's area); the worker pulls them via
  `wiki_get_page_content`; `search_wiki` handles discovery when a path isn't pre-named.
- **Write mechanics:** `wiki_create_or_update_page` is an **idempotent upsert** (safe
  under §5 re-run), but it is **NOT recursive — it does not auto-create ancestor pages.**
  Writing a nested page (`Domain/Glossary`, `Architecture/ADR/ADR-1-<slug>`,
  `Sprints/Sprint-8-Review`) whose parent namespace page doesn't yet exist **404s
  `WikiAncestorPageNotFoundException`**. So **ensure ancestors exist, parent-first**:
  before a nested write, create each missing namespace page (`/Domain`, then
  `/Architecture`, then `/Architecture/ADR`, `/Sprints`) — `wiki_list_pages` tells you
  what's absent — then write the child. A **brand-new project wiki has no pages at all**,
  so the very first namespace page must be created before anything can nest under it. The
  target wiki is resolved once (`wiki_get_wiki`/`wiki_list_wikis`) at `/delivery-init` and
  cached in `.delivery/config.json` (`wikiId`).

## 9. The one REST carve-out — attachment upload

**Exactly one** Azure operation has no MCP tool: uploading an attachment (evidence —
test screenshots / result files) to a work-item. (The DRAFT detail-spec once said
"two"; Resolution #3, 2026-07-04, narrowed it to **one** — the analytics hatch was
removed; velocity is client-side arithmetic over `wit_*` queries, not a transport
gap.)

- It is wrapped behind a single uniform interface — the
  [`scripts/az-attach.sh`](../../scripts/az-attach.sh) helper — so callers see
  `attach(work-item, file)` and the transport split (REST here, MCP everywhere else)
  stays hidden inside the adapter. Two steps: `POST _apis/wit/attachments` (upload →
  returns a URL) then `wit_update_work_item` / a relation add to link the returned URL
  to the work-item.
- The helper runs the worker's **env PAT** (Basic auth), never a literal in the argv;
  it is worker-runnable (the worker already has the PAT + network). **PAT format (load-bearing):**
  the helper accepts a **raw** PAT in `AZURE_DEVOPS_PAT` (it base64-encodes `:PAT` itself) **or**
  the already-encoded `PERSONAL_ACCESS_TOKEN` = `base64("user:PAT")` that the `@azure-devops/mcp`
  server consumes (used verbatim as the Basic header). The two env vars carry **different formats**,
  so `atl work dispatch` must set `AZURE_DEVOPS_PAT` to a **raw** PAT — it must NOT reuse the MCP's
  base64 `PERSONAL_ACCESS_TOKEN` as if it were raw, because `curl -u ":<base64>"` double-encodes it
  → 401. The Go orchestrator stays zero-Azure — the carve-out lives in the team, not in `atl`.
- Reading an attachment back (for the sprint-review report) uses the MCP
  (`wit_get_work_item_attachment`), so only the upload leg is REST.

## 10. PR completion — the merge contract

The tech-lead worker lands a green unit's PR to `dev`; the engine only VERIFIES the merge
landed (`MergedToBase` = `rev-list origin/dev..branch == 0`) — it never merges.

- **Open:** the developer opens a PR to `dev` (the `config.branchPair.dev` branch); its job
  ends at the open PR.
- **Review:** the tech-lead reviews **on the Azure PR** — findings as PR threads
  (`repo_create_pull_request_thread` / `repo_reply_to_comment`), never `/create-pr`.
- **Merge = `repo_update_pull_request` with `autoComplete` + `mergeStrategy` `NoFastForward` ONLY**
  (a real merge commit). **Never `Squash`/`Rebase`/`RebaseMerge`** — they rewrite the branch's
  commit SHAs and false-block the engine's `MergedToBase` (`rev-list origin/dev..branch == 0`),
  the Azure twin of GitHub's `gh pr merge --merge` requirement. Set `transitionWorkItems: false`
  — the tech-lead owns the single Done transition, *after* the merge (§6, resolved at runtime).
  Vote first (`repo_vote_pull_request`), then complete; completing the PR IS the merge to `dev`.
- **NEVER-merge carve-out (D3):** the autonomous tech-lead **worker** performs this green
  feature→`dev` merge — that is the machine's job and the loop breaks without it. The human PO
  reviews only at sprint review (`dev`→`release` promotion). The carve-out is scoped to the
  machine, never the interactive session.

## 11. Mockability & testing (how this contract is exercised)

- **Layer-B (real, the load-bearing proof):** a real `claude -p` worker doing a full
  single-task micro-lifecycle against a **real Azure DevOps test project** — claim →
  comment → attach evidence → transition — driven by the PAT in its env. This proves
  keystone #17's non-negotiable assumption (MCP-config inheritance into workers). It
  runs locally (a container on the maintainer's machine; the PAT is portable and not
  IP-gated), gated on a test project + PAT being provisioned. *(Wired in at the e2e
  stone.)*
- **Layer-A mock (deferred):** if fast, always-on, no-Azure ceremony tests are wanted,
  the mock is a **mock `azureDevOps` MCP server** exposing the same tool names with
  canned/fixture responses — the ceremonies stay MCP-first and just repoint
  `--mcp-config`. **Not** a second code-adapter implementation (that would fork the
  MCP-first contract, keystone #17). Deferred until the ceremonies that consume it
  exist — build test infra against a real consumer, not ahead of one.

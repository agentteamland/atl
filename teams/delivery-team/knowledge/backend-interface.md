# Backend interface — the provider-neutral operation contract

The delivery-team runs on a **work-item store + a git host + a durable-knowledge store**.
This file defines that dependency as a **provider-neutral interface**: a small set of
concepts every role-agent and ceremony references, plus the cross-cutting policies that
hold regardless of provider. **Azure DevOps and GitHub are two implementations** of this
interface — each supplies the concrete binding in its own `backends/<provider>/` adapter
pack.

Why an interface (the D1/D4 decision): the team is **one team with two backends**, never
a fork. Agent role-craft is written against these neutral concepts ("write the spec field",
"open a PR and link it", "the ready-to-pull query"); the active backend's adapter pack
resolves each concept to concrete tools. A backend is an **indivisible per-provider bundle**
— git + work-tracking + durable-knowledge together — so a project is single-provider, never
a mix. Feature parity across backends is the goal, not in question; the difference is the
binding, never the reflex.

The deterministic Go orchestrator (`atl work dispatch`) is **already backend-agnostic** — it
reads a static `.delivery/plan.json` and verifies a durable git merge (`MergedToBase`), and
never queries the work-item store at dispatch time. So this interface governs only the
**LLM-side** reads/writes (ceremonies + workers); the engine is unchanged.

## Backend selection

Chosen once per project at `/delivery-init` and cached in `.delivery/config.json`:

```json
{ "backend": "azure" | "github", ... }
```

A ceremony or worker loads the matching `backends/<provider>/adapter.md` (mirrors how the
`developer` loads `packs/<area>/` for stack). Default = `azure` (backward-compatibility with
existing installs).

## The concepts (the interface)

Each row is a concept the team depends on, the neutral contract (what the team needs), and
the per-backend binding. The **binding** column is the adapter pack's job — never named in
agent role-craft.

| # | Concept | Neutral contract (what the team needs) | Azure binding | GitHub binding |
|---|---|---|---|---|
| 1 | **Work-item + hierarchy** | Create typed units (Epic→Feature→PBI→Task/Bug); a parent/child containment link for authoring/traceability (NOT a scheduling edge). | work-item types · `wit_create_work_item` · `wit_add_child_work_items` | issues + Issue Types · sub-issues (`gh` + REST `sub_issues`) |
| 2 | **Spec field** | The durable, always-loaded "what & why", read back **by heading** (`## Problem`/`## Business Value`/`## Scope`/`## Acceptance Criteria`/`## Out of Scope`). | `System.Description` (Markdown) | issue **body** (Markdown) |
| 3 | **Sentinel comment channel** | Append-only content located by an exact first-line **sentinel** (`**[Technical Analysis]**`, `**[Canonical Brief]**`), never "the newest comment". | `wit_add_work_item_comment` / `wit_list_work_item_comments` | issue comments (`gh`/REST) |
| 4 | **Typed metadata / tags** | Free-form, queryable, zero-setup labels carrying the machine-contracts: `atl-key:<hash>` idempotency + `atl-run:<…>` provenance, `area:<name>` pack-binding, `atl-brainstorm:<slug>` brainstorm-source provenance (a `/brainstorm done` board-sync stamps it; a decomposition ceremony adopts such an item in place), `blocked`. | `System.Tags` | issue **labels** (queryable via issue advanced search) |
| 5 | **Priority** | A per-unit admission/ready-frontier order (lower = higher priority). | `Microsoft.VSTS.Common.StackRank` | a Number "Priority" project field |
| 6 | **Iteration / sprint** | Sprints as named date ranges; item membership as an idempotent field set; read a sprint's items. | `IterationPath` + `work_*_iterations` | Projects v2 **Iteration** field |
| 7 | **Completion / state** | Detect "this unit is done" (a category, resolved at runtime — never a literal string); claim to in-progress; set done after merge. | state-category via `wit_get_work_item_type` (Completed) | issue **closed** + Status **Done** (one fixed model — no per-template resolution) |
| 8 | **Dependency link** | Typed, queryable predecessor edges — **this graph IS the scheduler** (`/sprint-start` reads it into `plan.json`; the Go engine topo-sorts). | `System.LinkTypes.Dependency-Forward/-Reverse` | a **`## Depends On` convention** the ceremony reads (GitHub has no native typed dependency — see backends/github) |
| 9 | **Durable-knowledge store** | Namespaced, single-owner-per-namespace current-truth (`Domain/`, `Analysis/`, `Architecture/`, `Architecture/ADR/`, `Conventions/`, `Sprints/`); idempotent upsert; workers read, only the tech-lead writes. | project **wiki** (`wiki_*`) | in-repo **`/docs`** (Contents API; diffable + PR-reviewable) |
| 10 | **Query / idempotency substrate** | Check-first-by-key before every create (found→reuse+update, not-found→create-then-stamp); read the completed set for velocity; read the backlog. "List means all" — never silently truncate. | `wit_query_by_wiql` | **issue advanced search** (`gh`/GraphQL, server-side) + Projects GraphQL |
| 11 | **PR + review + merge** | Open a PR to `dev`, review it, and **merge = the completion gate**; the merge must leave a real merge commit the engine's `MergedToBase` can verify. | `repo_*` (autoComplete + **NoFastForward**) | `gh` — open/review; **`gh pr merge --merge` only** (never squash/rebase); explicit `gh issue close` on merge-verify |
| 12 | **Test-evidence attachment** | Attach verifiable evidence (screenshots/results) to a unit; read it back at review. A surface that can't be run is UNVERIFIED → block, never fake-green. | REST carve-out (`scripts/az-attach.sh`) + `wit_get_work_item_attachment` | comment image upload / repo-committed artifact (see backends/github) |

## Cross-cutting policies — provider-neutral, per-tool binding

These principles are **the same across backends**; only the per-tool mechanism differs (the
adapter pack states it). Agent role-craft states the principle; the pack states the how.

- **Analysis lives on the analyzed item; a decomposed child reads its nearest ancestor.** The
  `**[Technical Analysis]**` (concept #3) is authored per *framed/analyzed* item — a Feature at
  kickoff — **not** per decomposed child unit (`/refine` gives each child a `**[Canonical Brief]**`,
  not its own analysis). So a worker (developer/tester) on a child unit (PBI/Task) that has no
  `**[Technical Analysis]**` of its own reads the **nearest ancestor** that bears one, traversing
  the parent-containment link (concept #1; each adapter binds the traversal). The
  `**[Canonical Brief]**` needs no such fallback — the tech-lead authors one per decomposed unit,
  so it is always on the unit itself.
- **Idempotency = stamp + check-before-create**, store-side keys as source of truth (no local
  ledger). Key = `hash(parent-id + plan-ordinal)` (stable across re-runs → convergent resume,
  not merely dedup). Same principle on both backends; the *query* differs (concept #10).
- **"List means all"** — never treat a paged/capped result as complete. Each backend documents
  its per-tool paging (Azure: `wiql` `top`-cap-is-truncation, `wiki_list_pages` continuation;
  GitHub: cursor-paginated GraphQL, exhaust the connection).
- **Resilience** — every write (and every read under N-parallel load) wraps in exponential
  backoff + jitter, honours the provider's rate-limit header, and writes the store only at
  **durable milestones** (the worker heartbeats to `status.json`, not to the store). Applies
  identically to Azure 429s and GitHub secondary-rate-limits.
- **Content-placement discipline** — analysis/brief live at a **deterministic location** (spec
  field by heading; sentinel comment by first line), so every consumer reads back by location,
  never by guessing. Concept-level identical; the field/comment binding is #2/#3.
- **Durable-knowledge = single-owner-per-namespace**, workers surface facts and the tech-lead
  promotes them — no N-worker write races. Identical on both; the store is #9.
- **State resolution** — never hardcode a completion string. On Azure this means resolving the
  process-template's Completed category at runtime; on GitHub the model is fixed (closed + Status
  Done), so "resolution" collapses to that one model. Blocking is a **flag** (`blocked` tag/label
  + a diagnostic comment), never a state transition, on both.
- **NEVER-merge carve-out (D3)** — the autonomous tech-lead **worker** merges the green PR to
  `dev` (the completion gate); the human PO reviews only at sprint review (`dev`→`release`). The
  carve-out is scoped to the machine, both backends.

## Status

- **Interface: v1 (this file).** The concepts + cross-cutting policies both backends implement.
- **Azure adapter:** `backends/azure/adapter.md` — the Azure implementation (relocated here from
  `knowledge/azure-adapter.md`; content unchanged, so the Azure e2e blueprint stays green).
- **GitHub adapter:** `backends/github/adapter.md` — the GitHub implementation (the bindings in
  the right-hand column, plus the `## Depends On` dependency convention and the evidence-attach
  mechanism). *Tracked separately.*
- **Agent role-craft neutralization:** role-agent `children/` + ceremony skills are rewritten to
  reference these neutral concepts instead of concrete Azure tools. *The remainder of this Epic.*

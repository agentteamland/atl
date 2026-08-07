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
| 4 | **Typed metadata / tags** | Free-form, queryable, zero-setup labels carrying the machine-contracts: `atl-key:<hash>` idempotency + `atl-run:<…>` provenance, `area:<name>` pack-binding, `atl-brainstorm:<slug>` brainstorm-source provenance (a `/brainstorm done` board-sync stamps it; a decomposition ceremony adopts such an item in place), `sprint:<slug>` sprint membership under `mode: "flow"` (the sprint carrier, #6 — the slug is the sprint's **ordinal**, so a live label reads `sprint:<n>` and the whole shape matches `sprint:[0-9]+`), `blocked`, `test:<outcome>` verification (#17). | `System.Tags` | issue **labels** (queryable via issue advanced search) |
| 5 | **Priority** | A per-unit admission/ready-frontier order (lower = higher priority). | `Microsoft.VSTS.Common.StackRank` | a Number "Priority" project field |
| 6 | **Iteration / sprint** | Mark a unit as belonging to a sprint — idempotently — and read a sprint's items. The **carrier is mode-selected** (`methodology.mode`): under `scrum` a sprint is a named date range on the backend's schedule and membership is an iteration **field** set; under `flow` a sprint has no backend object at all and membership is a **`sprint:<slug>` tag/label** (#4), swapped rather than accumulated on re-admission. | scrum: `IterationPath` + `work_*_iterations` · flow: `sprint:<n>` in `System.Tags`, a sprint's items read by WIQL on that tag | scrum: Projects v2 **Iteration** field · flow: `sprint:<n>` issue **label**, a sprint's items read by `gh search issues 'label:sprint:<n>'` |
| 7 | **Completion / state** | Detect "this unit is done" (a category, resolved at runtime — never a literal string); claim to in-progress; set done after merge. | state-category via `wit_get_work_item_type` (Completed) | issue **closed** + Status **Done** (one fixed model — no per-template resolution) |
| 8 | **Dependency link** | Typed, queryable predecessor edges — **this graph IS the scheduler** (`/sprint-start` reads it into `plan.json`; the Go engine topo-sorts). | `System.LinkTypes.Dependency-Forward/-Reverse` | a **`## Depends On` convention** the ceremony reads (GitHub has no native typed dependency — see backends/github) |
| 9 | **Durable-knowledge store** | Namespaced, single-owner-per-namespace current-truth (`Domain/`, `Analysis/`, `Architecture/`, `Architecture/ADR/`, `Conventions/`, `Sprints/`); idempotent upsert; workers read, only the tech-lead writes. | project **wiki** (`wiki_*`) | in-repo **`/docs`** (Contents API; diffable + PR-reviewable) |
| 10 | **Query / idempotency substrate** | Check-first-by-key before every create (found→reuse+update, not-found→create-then-stamp); read the completed set — for the review report in both modes, for velocity under `mode: "scrum"`; read the backlog. "List means all" — never silently truncate. | `wit_query_by_wiql` | **issue advanced search** (`gh`/GraphQL, server-side) + Projects GraphQL |
| 11 | **PR + review + merge** | Open a PR to `dev`, review it, and **merge = the completion gate**; the merge must leave a real merge commit the engine's `MergedToBase` can verify. | `repo_*` (autoComplete + **NoFastForward**) | `gh` — open/review; **`gh pr merge --merge` only** (never squash/rebase); explicit `gh issue close` on merge-verify |
| 12 | **Test-evidence attachment** | Attach verifiable evidence (screenshots/results) to a unit; read it back at review. A surface that can't be run is UNVERIFIED → block, never fake-green. | REST carve-out (`scripts/az-attach.sh`) + `wit_get_work_item_attachment` | comment image upload / repo-committed artifact (see backends/github) |
| 13 | **Candidate / triage state** | A mid-project request captured as a *candidate* (pre-accept): visible on the board, distinguishable from real work-units, carrying a triage weight (`light`/`standard`/`heavy`), and **excluded from the ready-frontier query (#10)** until the PO accepts it — on accept it becomes normal backlog, on reject it is closed with reasoning, on defer it goes to backlog with a trigger. Used only by `/request`. | `candidate` + `triage:<tier>` in `System.Tags` on a New item (a flag, like `blocked` — no native candidate category) | issue with Projects v2 **Status = `candidate`** (a NEW Status option — board-setup, like Iteration/#213) + `candidate` + `triage:<tier>` labels |
| 14 | **Intake provenance key** | A stable key identifying a candidate by `request-slug + initiator`, so re-running `/request` finds+updates the candidate instead of duplicating (`idempotent-writes`). A candidate has no parent/plan-ordinal, so it carries its OWN key, not `atl-key`; on accept the materialized PBIs get their own `atl-key`. | `atl-request:<slug>:<initiator>` in `System.Tags`; check-first WIQL on tag + title | `atl-request:<slug>:<initiator>` **label**; check-first `gh search issues` on the label (short slug — 50-char label limit) |
| 15 | **Recommendation + PO-decision record** | The durable record of (a) the team's reasoned verdict (`YES`/`NO`/`DEFER`/`NEEDS-INFO` + the dialectic) and (b) the PO's decision (accept/reject/defer) with the convergence mechanism (concession / judgment-call standoff / human-authority lock) and the preserved team dissent — located by an exact first-line **sentinel** on the sentinel-comment channel (#3), never "the newest comment". | a single `wit_add_work_item_comment`, first line `**[Request Decision]**` | a single `gh issue comment`, first line `**[Request Decision]**` |
| 16 | **Promotion approval signal (commit-bound)** | The PO's authorization to promote **one specific commit** from `dev` to `release`: a durable record on the promotion PR, located by an exact first-line **sentinel** (the #3 discipline), naming the approved commit id verbatim — plus the ability to resolve that PR's **current head commit id**, so the gate compares the two and refuses when they differ. Selection is **by commit id, not by recency**: the channel is append-and-supersede, and an approval that names a different commit than the one about to merge is not an approval. The gate never infers the signal from conversation and never carries it forward to another commit. **The comparison is performed by the deterministic CLI — `atl work promote` — which verifies AND merges in one call; the ceremony runs it and relays the verdict, and never reads the record to decide for itself.** | both reads **bound for an LLM caller**: the record leg is a PR thread whose first line is `**[Promotion Approval]**` (`repo_pull_request_thread_write` action `create` / `repo_pull_request_thread` action `list`), and the head-commit read is `repo_branch` (action `get`), whose response carries the commit id in its top-level `objectId` (resolved against a live server). What is unbound is the gate's **transport**: `atl work promote` is a Go binary with no MCP client, so it cannot issue either call — so **the commit-bound gate still does not operate on the Azure backend in v1** (backends/azure §2 + §10): a read the gate cannot issue is a read failure, so it **holds** there rather than falling back to a conversational promotion, and it never accepts the head from the ceremony that could read it | a PR comment whose first line is `**[Promotion Approval]**`, written by the PO (`gh pr comment`); the record **and** the head commit id are read back in **ONE** call, `gh pr view <n> --json headRefOid,comments` — one snapshot, so a comment posted between two reads cannot make the comparison lie. `atl work promote` issues that read and the pinned merge; no ceremony issues either |
| 17 | **Verification outcome flag** | The post-merge verification condition of a unit, carried **on the unit's existing state**: `test:pending` — merged to the integration branch, not yet verified; `test:failed` — verified and it came back red. A flag, exactly like `blocked` (#4) — Status/`System.State` is **unchanged** and the item stays open, so a unit awaiting verification is still counted as in flight. **At most one `test:` flag per unit** (they are mutually exclusive): re-flagging swaps in a single write, the `sprint:` discipline (#6). `test:failed` carries a diagnostic comment saying *what* failed, so the board shows why and not merely that. Cleared when the unit reaches Done. Zero board-setup on **both** backends — unlike a Status option (#13), which needs the Projects settings UI. | `test:pending` / `test:failed` in `System.Tags`; `System.State` unchanged | `test:pending` / `test:failed` issue **labels**; Projects v2 Status unchanged |

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
- **A condition is an annotation, not a destination.** The state axis records *where work is*;
  the flag axis records *what is true about it*. `blocked` (#4), `candidate` (#13) and the
  verification flags (#17) are all the second kind, and none of them is ever promoted to a state.
  The test is one question: **can a unit be in this and in something else at the same time?** Yes ⇒
  annotation. A unit can be blocked *while* awaiting verification, so two flags must compose — and
  two labels compose for free where two states cannot. Promoting a condition to a state also
  **destroys** information rather than adding it: the unit leaves the column it was in, so
  "how many units are in progress?" and "how many are awaiting verification?" become disjoint
  counts and every unit in verification vanishes from the WIP number capacity is read from. It
  further obliges the state model to define a legal move to and from *every* other state, which is
  how a transition table stops being readable. Cardinality is the flag axis's one liability —
  a **field replaces on write while a label set accumulates** — so where flags within a namespace
  are mutually exclusive (`sprint:`, `test:`), the write site imposes it explicitly by removing and
  adding in a single call.
- **The sprint carrier is mode-selected — one carrier, never both.** `methodology.mode` decides how
  a unit is marked as belonging to a sprint (#6): the **iteration field** under `scrum`, the
  **`sprint:<slug>` tag/label** (#4) under `flow`. A ceremony resolves the mode in its descriptor load
  and then touches exactly one — there is no dual-write, so the two can never disagree about which
  sprint a unit is in. Under flow the ordinal is **resolved, never invented**: list the existing
  `sprint:*` tags/labels ("list means all", #10 — a capped read is a truncation to surface) and take
  the highest ordinal `k`, **compared as an integer** (`sprint:10` outranks `sprint:9`). `sprint:<k>`
  is the **current** sprint and stays current until it is *reviewed*, so a run plans **into
  `sprint:<k>`** while its `Sprints/Sprint-<k>-Review` page (#9) is absent, and opens **`sprint:<k+1>`**
  only once that page exists — advancing on the highest ordinal *alone* would open a fresh sprint on
  every re-plan, which is what makes a re-run converge instead of forking. An empty board starts at
  `sprint:1`, and a project migrating from scrum continues its existing numbering — take the highest
  ordinal `m` among its `Sprints/Sprint-<m>-Review` pages (#9) and open `sprint:<m+1>`, never reading
  it off iteration *names*, which are arbitrary. Call
  the resolved ordinal `<n>`. **At most one `sprint:` label per unit** — re-admitting a carryover
  removes the `sprint:` label it already carries and adds `sprint:<n>` in the same write, because a
  label accumulates where a field replaces, and two of them leaves "which sprint is this in?"
  without an answer. A label is **never** removed to mean
  "done" — completion is a state (#7). And a flow sprint has **no backend object**: it is the set
  of units carrying its label — nothing creates, dates, or closes a sprint entity, and *reviewed*
  (the flow analogue of a closed iteration) is read off the review page above. Full contract:
  [`config-and-methodology.md`](config-and-methodology.md) §1.2.
- **NEVER-merge carve-out (D3)** — the autonomous tech-lead **worker** merges the green PR to
  `dev` (the completion gate); the human PO reviews only at sprint review (`dev`→`release`). The
  carve-out is scoped to the machine, both backends.
- **A promotion is verified, not asserted — and the verification is CODE, not ceremony prose.** The
  `dev`→`release` merge is gated on a commit-bound approval signal (#16) read back from the backend
  and matched against the commit about to merge. Absent, malformed, or mismatched ⇒ **no promotion**
  — the gate holds and reports what the PO must set. A conversational "approve" is the *input* to
  the signal, never a substitute for it. **That check does not live in `/sprint-review`'s
  instructions:** it shipped there first, in a spec that contradicted itself — one paragraph told the
  ceremony to put a blocking Approve/Reject question *before* the check, another forbade accepting a
  conversational approval at all. Both sentences were well-formed and every mechanical gate passed;
  an agent simply obeys whichever it reaches first. It is now `atl work promote`, a deterministic CLI that **verifies and
  merges in one call** so no caller can reach the merge without the check. A ceremony's job is to
  run it, obey the exit code, and relay the reason; re-deriving the comparison in prose is how the
  skipped path grows back.
  **v1 runs the gate on GitHub only:** Azure binds both legs for an LLM caller, but they are MCP
  tools and the gate is a Go binary with no MCP client (#16's binding column +
  `backends/azure/adapter.md` §10), so on that backend the match cannot be performed at all — the
  same fail-closed case as a mismatch: **no promotion**. Azure does **not** fall back to promoting
  on a conversational approval, and the ceremony does not read the head for it either; unverified
  is never approved.
- **Candidate is pre-accept, not ready-frontier (`/request`).** A `/request` candidate (concept
  #13) is a flag/state **excluded from the ready-to-pull / selection query (concept #10)** that
  `/refine` and `/sprint-plan` run — until the PO accepts it. Otherwise the request is swept into
  the backlog unexamined, the exact failure `/request` exists to prevent. Its idempotency key is
  the intake-provenance key (#14, `atl-request:<slug>:<initiator>` — a sibling of
  `atl-brainstorm:<slug>`, distinct from `atl-key`); its verdict + PO decision live as a
  `**[Request Decision]**` sentinel on the sentinel-comment channel (#3). Accept drops the
  `candidate` flag / flips the Status off `candidate` so the item enters the frontier; the
  materialized PBIs then get their own `atl-key`. `/request` is **event-triggered** (not a
  `methodology.cadence` slot) and **requires a live PO**, so — like `/kickoff` and `/sprint-review`
  — it runs in-session and is **not** part of the headless `atl work dispatch` loop.

## Status

- **Interface: v1 (this file).** The concepts + cross-cutting policies both backends implement.
- **Azure adapter:** `backends/azure/adapter.md` — the Azure implementation (relocated here from
  `knowledge/azure-adapter.md`; content unchanged, so the Azure e2e blueprint stays green).
- **GitHub adapter:** `backends/github/adapter.md` — the GitHub implementation (the bindings in
  the right-hand column, plus the `## Depends On` dependency convention and the evidence-attach
  mechanism). *Tracked separately.*
- **Agent role-craft neutralization:** role-agent `children/` + ceremony skills are rewritten to
  reference these neutral concepts instead of concrete Azure tools. *The remainder of this Epic.*
